// Copyright 2026 Su Yang (soulteary)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package translate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ChatCompletionsRequest / Response match OpenAI API shape for parsing.
type (
	ChatCompletionsRequest struct {
		Model    string    `json:"model"`
		Messages []Message `json:"messages"`
		// Temperature is a pointer so it can be omitted entirely (some models,
		// e.g. kimi-k2.6 / deepseek reasoning models, do not allow modifying it
		// and require the cloud default).
		Temperature     *float64        `json:"temperature,omitempty"`
		MaxTokens       int             `json:"max_tokens,omitempty"`
		ReasoningEffort string          `json:"reasoning_effort,omitempty"`
		Thinking        *ThinkingOption `json:"thinking,omitempty"`
	}
	// ThinkingOption mirrors the extra_body {"thinking": {"type": ...}} used by
	// Moonshot and DeepSeek.
	ThinkingOption struct {
		Type string `json:"type"`
	}
	Message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	ChatCompletionsResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
)

// Client is an OpenAI-compatible Chat Completions client.
type Client struct {
	BaseURL string
	APIKey  string
	Model   string
	// Temperature controls generation randomness.
	Temperature float64
	Timeout     time.Duration
	Client      *http.Client
}

// NewClient creates a client. BaseURL is normalized (trailing slash trimmed; /v1/chat/completions appended when calling).
func NewClient(baseURL, apiKey, model string, timeoutSec int, temperature float64) *Client {
	baseURL = strings.TrimSuffix(strings.TrimSpace(baseURL), "/")
	if timeoutSec <= 0 {
		timeoutSec = 30
	}
	if temperature < 0 {
		temperature = 0
	}
	return &Client{
		BaseURL:     baseURL,
		APIKey:      strings.TrimSpace(apiKey),
		Model:       strings.TrimSpace(model),
		Temperature: temperature,
		Timeout:     time.Duration(timeoutSec) * time.Second,
		Client:      &http.Client{Transport: http.DefaultTransport},
	}
}

// kimiK2MaxTokens is the output token budget used for kimi-k2 family models
// (1024 * 32), matching Moonshot's reference usage.
const kimiK2MaxTokens = 1024 * 32

// modelProfile describes provider/model-specific Chat Completions parameters.
// Zero values mean "omit the parameter".
type modelProfile struct {
	sendTemperature bool   // when false, omit temperature -> use cloud default
	maxTokens       int    // 0 = omit
	thinking        string // "" = omit, otherwise "enabled" / "disabled"
	reasoningEffort string // "" = omit, otherwise "low" / "medium" / "high"
}

// isDeepSeekReasoner reports whether a deepseek-* model is a reasoning model
// (deepseek-reasoner / deepseek-r1) rather than a chat model (deepseek-chat /
// deepseek-v3). Reasoning models reject temperature and accept reasoning_effort/
// thinking; chat models behave like a standard Chat Completions model.
func isDeepSeekReasoner(m string) bool {
	return strings.HasPrefix(m, "deepseek-reasoner") || strings.HasPrefix(m, "deepseek-r1")
}

// profileForModel returns request parameters tuned for the given model family.
// Supported providers (selected by model-name prefix, base URL is configured
// separately in settings):
//   - Moonshot kimi-k2*           : cloud-default temperature, thinking disabled, large max_tokens
//   - DeepSeek reasoning (r1)     : cloud-default temperature, reasoning_effort=high, thinking enabled
//   - DeepSeek chat (deepseek-chat/v3), OpenAI gpt-*, Moonshot moonshot-v1-*, others: standard shape with temperature
func profileForModel(model string) modelProfile {
	m := strings.ToLower(strings.TrimSpace(model))
	switch {
	case strings.HasPrefix(m, "kimi-k2"):
		return modelProfile{sendTemperature: false, maxTokens: kimiK2MaxTokens, thinking: "disabled"}
	case strings.HasPrefix(m, "deepseek"):
		if isDeepSeekReasoner(m) {
			return modelProfile{sendTemperature: false, reasoningEffort: "high", thinking: "enabled"}
		}
		// deepseek-chat / deepseek-v3 and other non-reasoning models accept temperature.
		return modelProfile{sendTemperature: true}
	default:
		return modelProfile{sendTemperature: true}
	}
}

// extractErrorDetail reads an error response body and returns a concise,
// human-readable detail. It understands the common OpenAI/Moonshot error shapes
// ({"error":{"message":...}} / {"error":"..."} / {"message":...}) and falls back
// to the raw (truncated) body text.
func extractErrorDetail(body io.Reader) string {
	raw, err := io.ReadAll(io.LimitReader(body, 8192))
	if err != nil || len(raw) == 0 {
		return ""
	}
	var parsed struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    string `json:"code"`
		} `json:"error"`
		Message string `json:"message"`
	}
	if json.Unmarshal(raw, &parsed) == nil {
		if parsed.Error.Message != "" {
			return parsed.Error.Message
		}
		if parsed.Message != "" {
			return parsed.Message
		}
	}
	// error may be a bare string: {"error":"..."}
	var alt struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(raw, &alt) == nil && alt.Error != "" {
		return alt.Error
	}
	return truncateDetail(strings.TrimSpace(string(raw)), 300)
}

// truncateDetail shortens s to at most n runes, appending an ellipsis if cut.
func truncateDetail(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// chatCompletionsURL returns the full URL for POST /v1/chat/completions.
func (c *Client) chatCompletionsURL() string {
	if strings.HasSuffix(c.BaseURL, "/v1") {
		return c.BaseURL + "/chat/completions"
	}
	return c.BaseURL + "/v1/chat/completions"
}

// Translate calls the Chat Completions API and returns the assistant content (trimmed).
func (c *Client) Translate(systemPrompt, userText string) (string, error) {
	if c.APIKey == "" {
		return "", ErrNoAPIKey
	}
	if c.BaseURL == "" || c.Model == "" {
		return "", ErrInvalidConfig
	}

	reqBody := ChatCompletionsRequest{
		Model: c.Model,
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userText},
		},
	}
	prof := profileForModel(c.Model)
	if prof.sendTemperature {
		temp := c.Temperature
		reqBody.Temperature = &temp
	}
	if prof.maxTokens > 0 {
		reqBody.MaxTokens = prof.maxTokens
	}
	if prof.thinking != "" {
		reqBody.Thinking = &ThinkingOption{Type: prof.thinking}
	}
	if prof.reasoningEffort != "" {
		reqBody.ReasoningEffort = prof.reasoningEffort
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("encode request: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.chatCompletionsURL(), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return "", ErrTimeout
		}
		return "", fmt.Errorf("%w: %v", ErrNetwork, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		detail := extractErrorDetail(resp.Body)
		switch {
		case resp.StatusCode == http.StatusUnauthorized:
			return "", newAPIError(ErrUnauthorized, resp.StatusCode, detail)
		case resp.StatusCode == http.StatusTooManyRequests:
			return "", newAPIError(ErrRateLimited, resp.StatusCode, detail)
		case resp.StatusCode >= 500:
			return "", newAPIError(ErrServer, resp.StatusCode, detail)
		default:
			// 4xx other (e.g. 400 bad request) treat as bad response / config
			return "", newAPIError(ErrBadResponse, resp.StatusCode, detail)
		}
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("%w: read body: %v", ErrBadResponse, err)
	}
	var parsed ChatCompletionsResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", newAPIError(ErrBadResponse, resp.StatusCode, truncateDetail(strings.TrimSpace(string(raw)), 300))
	}
	if len(parsed.Choices) == 0 || parsed.Choices[0].Message.Content == "" {
		return "", newAPIError(ErrBadResponse, resp.StatusCode, truncateDetail(strings.TrimSpace(string(raw)), 300))
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}
