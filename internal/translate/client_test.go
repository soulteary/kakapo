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
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_Translate_success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Errorf("missing or wrong Authorization")
		}
		body := ChatCompletionsResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{Message: struct {
					Content string `json:"content"`
				}{Content: "  Hello, world.  "}},
			},
		}
		_ = json.NewEncoder(w).Encode(body)
	}))
	defer server.Close()

	client := NewClient(server.URL, "sk-test", "gpt-4o-mini", 5, 0.2)
	out, err := client.Translate(SystemPrompt, "你好世界")
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}
	if out != "Hello, world." {
		t.Errorf("got %q", out)
	}
}

func TestClient_Translate_kimiK2_request_shape(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(ChatCompletionsResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{{Message: struct {
				Content string `json:"content"`
			}{Content: "ok"}}},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "sk", "kimi-k2.6", 5, 0.7)
	if _, err := client.Translate("sys", "hi"); err != nil {
		t.Fatalf("Translate: %v", err)
	}
	if _, ok := got["temperature"]; ok {
		t.Errorf("kimi-k2.6 request should omit temperature, got %v", got["temperature"])
	}
	if got["max_tokens"] == nil {
		t.Errorf("kimi-k2.6 request should include max_tokens")
	}
	thinking, ok := got["thinking"].(map[string]any)
	if !ok || thinking["type"] != "disabled" {
		t.Errorf("kimi-k2.6 request should set thinking.type=disabled, got %v", got["thinking"])
	}
}

func TestClient_Translate_deepseek_request_shape(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(ChatCompletionsResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{{Message: struct {
				Content string `json:"content"`
			}{Content: "ok"}}},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "sk", "deepseek-reasoner", 5, 0.7)
	if _, err := client.Translate("sys", "hi"); err != nil {
		t.Fatalf("Translate: %v", err)
	}
	if _, ok := got["temperature"]; ok {
		t.Errorf("deepseek-reasoner request should omit temperature, got %v", got["temperature"])
	}
	if got["reasoning_effort"] != "high" {
		t.Errorf("deepseek-reasoner request should set reasoning_effort=high, got %v", got["reasoning_effort"])
	}
	thinking, ok := got["thinking"].(map[string]any)
	if !ok || thinking["type"] != "enabled" {
		t.Errorf("deepseek-reasoner request should set thinking.type=enabled, got %v", got["thinking"])
	}
}

func TestClient_Translate_deepseekChat_sends_temperature(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(ChatCompletionsResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{{Message: struct {
				Content string `json:"content"`
			}{Content: "ok"}}},
		})
	}))
	defer server.Close()

	// deepseek-chat is a non-reasoning model: it must receive temperature and
	// must NOT receive reasoning_effort / thinking.
	client := NewClient(server.URL, "sk", "deepseek-chat", 5, 0.3)
	if _, err := client.Translate("sys", "hi"); err != nil {
		t.Fatalf("Translate: %v", err)
	}
	if _, ok := got["temperature"]; !ok {
		t.Errorf("deepseek-chat request should include temperature")
	}
	if _, ok := got["reasoning_effort"]; ok {
		t.Errorf("deepseek-chat request should not include reasoning_effort, got %v", got["reasoning_effort"])
	}
	if _, ok := got["thinking"]; ok {
		t.Errorf("deepseek-chat request should not include thinking, got %v", got["thinking"])
	}
}

func TestClient_Translate_nonKimi_sends_temperature(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&got)
		_ = json.NewEncoder(w).Encode(ChatCompletionsResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{{Message: struct {
				Content string `json:"content"`
			}{Content: "ok"}}},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "sk", "moonshot-v1-8k", 5, 0.3)
	if _, err := client.Translate("sys", "hi"); err != nil {
		t.Fatalf("Translate: %v", err)
	}
	if _, ok := got["temperature"]; !ok {
		t.Errorf("non-kimi request should include temperature")
	}
	if _, ok := got["thinking"]; ok {
		t.Errorf("non-kimi request should not include thinking")
	}
}

func TestClient_Translate_baseURL_with_v1(t *testing.T) {
	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(ChatCompletionsResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{{Message: struct {
				Content string `json:"content"`
			}{Content: "ok"}}},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL+"/v1", "sk", "m", 5, 0.2)
	_, _ = client.Translate("", "x")
	if path != "/v1/chat/completions" {
		t.Errorf("path = %s", path)
	}
}

func TestChatCompletionsURL(t *testing.T) {
	cases := []struct {
		name    string
		baseURL string
		mode    EndpointMode
		want    string
	}{
		{"base bare host", "https://api.openai.com", EndpointModeBaseURL, "https://api.openai.com/v1/chat/completions"},
		{"base with v1", "https://api.moonshot.cn/v1", EndpointModeBaseURL, "https://api.moonshot.cn/v1/chat/completions"},
		{"base trailing slash", "https://api.deepseek.com/", EndpointModeBaseURL, "https://api.deepseek.com/v1/chat/completions"},
		{"base v1 trailing slash", "https://api.moonshot.cn/v1/", EndpointModeBaseURL, "https://api.moonshot.cn/v1/chat/completions"},
		{"full verbatim", "https://gw.example.com/proxy/chat/completions", EndpointModeFull, "https://gw.example.com/proxy/chat/completions"},
		{"full trailing slash trimmed", "https://gw.example.com/v2/chat/completions/", EndpointModeFull, "https://gw.example.com/v2/chat/completions"},
		{"full does not append", "https://api.openai.com/v1", EndpointModeFull, "https://api.openai.com/v1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := chatCompletionsURL(tc.baseURL, tc.mode); got != tc.want {
				t.Errorf("chatCompletionsURL(%q, %q) = %q, want %q", tc.baseURL, tc.mode, got, tc.want)
			}
		})
	}
}

func TestNormalizeEndpointMode(t *testing.T) {
	cases := map[string]EndpointMode{
		"":              EndpointModeBaseURL,
		"base":          EndpointModeBaseURL,
		"openai":        EndpointModeBaseURL,
		"anything-else": EndpointModeBaseURL,
		"full":          EndpointModeFull,
		"FULL":          EndpointModeFull,
		" Full ":        EndpointModeFull,
		"endpoint":      EndpointModeFull,
		"full_endpoint": EndpointModeFull,
	}
	for in, want := range cases {
		if got := NormalizeEndpointMode(in); got != want {
			t.Errorf("NormalizeEndpointMode(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestClient_Translate_fullEndpoint_usesURLVerbatim(t *testing.T) {
	var path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_ = json.NewEncoder(w).Encode(ChatCompletionsResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{{Message: struct {
				Content string `json:"content"`
			}{Content: "ok"}}},
		})
	}))
	defer server.Close()

	// Full-endpoint mode: the user configured the complete path, so the client
	// must NOT append /v1/chat/completions.
	client := NewClient(server.URL+"/custom/chat/completions", "sk", "m", 5, 0.2)
	client.EndpointMode = EndpointModeFull
	if _, err := client.Translate("", "x"); err != nil {
		t.Fatalf("Translate: %v", err)
	}
	if path != "/custom/chat/completions" {
		t.Errorf("path = %s, want /custom/chat/completions", path)
	}
}

func TestClient_Translate_401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClient(server.URL, "sk", "m", 5, 0.2)
	_, err := client.Translate("", "x")
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestClient_Translate_429(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := NewClient(server.URL, "sk", "m", 5, 0.2)
	_, err := client.Translate("", "x")
	if !errors.Is(err, ErrRateLimited) {
		t.Errorf("expected ErrRateLimited, got %v", err)
	}
}

func TestClient_Translate_500(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL, "sk", "m", 5, 0.2)
	_, err := client.Translate("", "x")
	if !errors.Is(err, ErrServer) {
		t.Errorf("expected ErrServer, got %v", err)
	}
}

func TestClient_Translate_empty_choices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "sk", "m", 5, 0.2)
	_, err := client.Translate("", "x")
	if !errors.Is(err, ErrBadResponse) {
		t.Errorf("expected ErrBadResponse, got %v", err)
	}
}

func TestClient_Translate_no_api_key(t *testing.T) {
	client := NewClient("https://api.openai.com", "", "m", 5, 0.2)
	_, err := client.Translate("", "x")
	if !errors.Is(err, ErrNoAPIKey) {
		t.Errorf("expected ErrNoAPIKey, got %v", err)
	}
}

func TestErrorMessage(t *testing.T) {
	if msg := ErrorMessage(ErrNoAPIKey); msg == "" {
		t.Error("ErrorMessage(ErrNoAPIKey) should be non-empty")
	}
	if msg := ErrorMessage(ErrUnauthorized); msg == "" {
		t.Error("ErrorMessage(ErrUnauthorized) should be non-empty")
	}
}
