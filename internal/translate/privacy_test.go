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
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestUpstreamFailuresDoNotExposeSensitiveBody(t *testing.T) {
	const sensitive = "synthetic-api-key-and-private-text"
	for _, status := range []int{200, 400, 401, 429, 500} {
		for _, body := range []string{`{"error":{"message":"` + sensitive + `"}}`, sensitive, `{"choices":[],"message":"` + sensitive + `"}`} {
			t.Run(fmt.Sprintf("status=%d/body=%d", status, len(body)), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status); _, _ = io.WriteString(w, body) }))
				defer server.Close()
				c := NewClient(server.URL, "synthetic-key", "m", 5, .2)
				output, err := c.Translate("system", "private text")
				if err == nil || output != "" {
					t.Fatalf("invalid upstream response accepted: output=%q err=%v", output, err)
				}
				if strings.Contains(err.Error(), sensitive) || strings.Contains(ErrorMessage(err), sensitive) {
					t.Fatal("upstream body exposed in diagnostic")
				}
				var apiErr *APIError
				if !errors.As(err, &apiErr) || apiErr.Status != status {
					t.Fatalf("missing HTTP status: %v", err)
				}
				if apiErr.Detail != "" {
					t.Fatal("upstream body retained in APIError.Detail")
				}
			})
		}
	}
}

type privacyTransport func(*http.Request) (*http.Response, error)

func (f privacyTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type failingBody struct{}

func (failingBody) Read([]byte) (int, error) { return 0, errors.New("synthetic-sensitive-read-error") }
func (failingBody) Close() error             { return nil }

func TestTransportAndReadFailuresDoNotExposeDetails(t *testing.T) {
	c := NewClient("https://example.invalid/private?token=synthetic-secret", "key", "model", 5, .2)
	c.Client = &http.Client{Transport: privacyTransport(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("synthetic-sensitive-network-error")
	})}
	if _, err := c.Translate("sys", "input"); !errors.Is(err, ErrNetwork) || strings.Contains(err.Error(), "synthetic") {
		t.Fatalf("unsafe network error: %v", err)
	}
	c.Client.Transport = privacyTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: failingBody{}, Header: make(http.Header)}, nil
	})
	if _, err := c.Translate("sys", "input"); !errors.Is(err, ErrBadResponse) || strings.Contains(err.Error(), "synthetic") {
		t.Fatalf("unsafe read error: %v", err)
	}
	c.BaseURL = "http://[invalid/synthetic-secret"
	if _, err := c.Translate("sys", "input"); !errors.Is(err, ErrInvalidConfig) || strings.Contains(err.Error(), "synthetic") {
		t.Fatalf("unsafe invalid URL error: %v", err)
	}
}

func TestLegacyAPIErrorDetailNotFormatted(t *testing.T) {
	err := &APIError{Kind: ErrUnauthorized, Status: 401, Detail: "synthetic-sensitive-legacy-detail"}
	if strings.Contains(err.Error(), "synthetic") || strings.Contains(ErrorMessage(err), "synthetic") {
		t.Fatal("legacy detail exposed")
	}
	if !errors.Is(err, ErrUnauthorized) || !strings.Contains(err.Error(), "401") {
		t.Fatal("error kind/status lost")
	}
}

func TestClientValidationAndTimeout(t *testing.T) {
	c := NewClient("https://example.invalid", "key", "model", 0, -1)
	if c.Timeout != 30*time.Second || c.Temperature != 0 {
		t.Fatalf("constructor defaults: %+v", c)
	}
	c.Model = ""
	if _, err := c.Translate("", "input"); !errors.Is(err, ErrInvalidConfig) {
		t.Fatal(err)
	}
	c.Model = "model"
	c.Temperature = math.NaN()
	if _, err := c.Translate("", "input"); err == nil {
		t.Fatal("non-JSON temperature accepted")
	}
	c.Temperature = .2
	c.Timeout = time.Nanosecond
	c.Client.Transport = privacyTransport(func(r *http.Request) (*http.Response, error) { <-r.Context().Done(); return nil, r.Context().Err() })
	if _, err := c.Translate("", "input"); !errors.Is(err, ErrTimeout) {
		t.Fatalf("timeout: %v", err)
	}
}

func TestErrorKindsAndPrompts(t *testing.T) {
	if ErrorMessage(nil) != "" {
		t.Fatal("nil error must have no message")
	}
	for _, err := range []error{ErrNoAPIKey, ErrInvalidConfig, ErrNetwork, ErrTimeout, ErrUnauthorized, ErrRateLimited, ErrServer, ErrBadResponse, errors.New("local failure")} {
		if ErrorMessage(fmt.Errorf("wrapped: %w", err)) == "" {
			t.Fatalf("missing message for %v", err)
		}
	}
	if BuildSystemPrompt(" ZH ", " EN ") != SystemPrompt {
		t.Fatal("language normalization lost")
	}
	for _, pair := range [][2]string{{"en", "zh"}, {"zh", "ja"}, {"ja", "zh"}, {"en", "ja"}, {"ja", "en"}} {
		if BuildSystemPrompt(pair[0], pair[1]) == "" {
			t.Fatal("missing builtin prompt")
		}
	}
	if prompt := BuildSystemPrompt("fr", "de"); !strings.Contains(prompt, "fr") || !strings.Contains(prompt, "de") {
		t.Fatal("generic prompt omitted languages")
	}
}
