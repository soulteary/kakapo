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

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/soulteary/kakapo/internal/config"
	"github.com/soulteary/kakapo/internal/translate"
	"github.com/wailsapp/wails/v3/pkg/application"
)

func isolatedConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("KAKAPO_HTTP_LOG", "off")
	return filepath.Join(dir, "Kakapo")
}
func writeConfig(t *testing.T, c config.Settings) {
	t.Helper()
	if err := config.Save(c); err != nil {
		t.Fatal(err)
	}
}
func request(t *testing.T, h http.Handler, method, path, body string, want int) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != want {
		t.Fatalf("%s %s: status=%d want=%d body=%s", method, path, w.Code, want, w.Body.String())
	}
	return w
}
func TestAppInfo(t *testing.T) {
	s := NewAppInfoService()
	v := s.Info()
	if v.Name != "Kakapo" || v.OS != runtime.GOOS || v.Arch != runtime.GOARCH || v.GoVersion != runtime.Version() || v.Version != AppVersion || v.Commit != AppCommit || v.BuildTime != AppBuildTime || v.StartedAt <= 0 {
		t.Fatalf("metadata: %+v", v)
	}
	if s.UptimeSeconds() < 0 {
		t.Fatal("negative uptime")
	}
}
func TestSettingsRoundTrip(t *testing.T) {
	isolatedConfig(t)
	a := NewTranslateApp()
	if err := a.SaveSettings(nil); err == nil {
		t.Fatal("nil settings accepted")
	}
	dto := &SettingsDTO{Providers: []ProviderDTO{{BaseURL: " https://api.deepseek.com ", Models: []string{" a ", " ", "b"}, Enabled: true, EndpointMode: " FULL ", SetAPIKey: "synthetic-key", ClearAPIKey: true}}}
	if err := a.SaveSettings(dto); err != nil {
		t.Fatal(err)
	}
	got, err := a.GetSettings()
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Providers) != 1 || got.Providers[0].ID == "" || got.Providers[0].Type != "deepseek" || got.Providers[0].EndpointMode != "full" || !reflect.DeepEqual(got.Providers[0].Models, []string{"a", "b"}) || got.Providers[0].APIKeySet || got.Providers[0].SetAPIKey != "" {
		t.Fatalf("settings: %+v", got)
	}
	if got.TimeoutSeconds != 30 || got.MaxInputChars != 5000 || got.SourceLanguage != "zh" || !reflect.DeepEqual(got.TargetLanguages, []string{"en"}) || got.Temperature != 0 {
		t.Fatalf("defaults: %+v", got)
	}
	if err := a.SaveSettings(&SettingsDTO{}); err != nil {
		t.Fatal(err)
	}
	got, err = a.GetSettings()
	if err != nil || got.Providers[0].ID != config.DefaultProviderID {
		t.Fatalf("default provider: %+v %v", got, err)
	}
	if err := a.SaveSettings(&SettingsDTO{Temperature: math.NaN()}); err == nil {
		t.Fatal("invalid JSON temperature accepted")
	}
	if got := maskAPIKey("abcdef"); got != "****cdef" {
		t.Fatal(got)
	}
	if maskAPIKey("x") != "****" {
		t.Fatal("short key mask")
	}
	id := genProviderID()
	if !strings.HasPrefix(id, "p_") || len(id) != 14 {
		t.Fatalf("provider id %q", id)
	}
	if got := normalizeModels([]string{" ", " a ", "b"}); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatal(got)
	}
}
func TestSettingsAndTranslationsStorageErrors(t *testing.T) {
	dir := isolatedConfig(t)
	a := NewTranslateApp()
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := a.GetSettings(); err == nil {
		t.Fatal("corrupt config ignored")
	}
	if _, err := a.TranslateZhToEn("x"); err == nil {
		t.Fatal("config load ignored")
	}
	if _, err := a.TranslateParallel("x", "", nil, nil); err == nil {
		t.Fatal("parallel config load ignored")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := a.SaveSettings(&SettingsDTO{}); err == nil {
		t.Fatal("write to directory accepted")
	}
}
func TestTranslationValidationAndMissingKey(t *testing.T) {
	isolatedConfig(t)
	a := NewTranslateApp()
	if _, err := a.TranslateZhToEn(" "); err == nil {
		t.Fatal("blank input accepted")
	}
	if _, err := a.TranslateParallel(" ", "", nil, nil); err == nil {
		t.Fatal("blank parallel input accepted")
	}
	c := config.Default()
	c.MaxInputChars = 1
	writeConfig(t, c)
	if _, err := a.TranslateZhToEn("你好"); err == nil {
		t.Fatal("input limit ignored")
	}
	if _, err := a.TranslateParallel("你好", "", nil, nil); err == nil {
		t.Fatal("parallel input limit ignored")
	}
	c.MaxInputChars = 5000
	c.Providers[0].Enabled = false
	writeConfig(t, c)
	if _, err := a.TranslateZhToEn("x"); !errors.Is(err, translate.ErrInvalidConfig) {
		t.Fatal(err)
	}
	if _, err := a.TranslateParallel("x", "", nil, nil); !errors.Is(err, translate.ErrInvalidConfig) {
		t.Fatal(err)
	}
	c.Providers[0].Enabled = true
	writeConfig(t, c)
	if _, err := a.TranslateZhToEn("x"); !errors.Is(err, translate.ErrNoAPIKey) {
		t.Fatal(err)
	}
	if _, err := a.TranslateParallel("x", "zh", []string{"en"}, []string{"absent"}); !errors.Is(err, translate.ErrInvalidConfig) {
		t.Fatal(err)
	}
	result, err := a.TranslateParallel(" x ", "", nil, []string{" kimi-k2.6 "})
	if err != nil || len(result.Results) != 1 || result.Input != "x" || result.CreatedAt <= 0 || result.Results[0].Error != translate.ErrorMessage(translate.ErrNoAPIKey) {
		t.Fatalf("missing key result: %+v %v", result, err)
	}
}
func TestRunTranslate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/endpoint" || r.Header.Get("Authorization") != "Bearer synthetic-key" {
			t.Errorf("wrong request")
		}
		io.WriteString(w, `{"choices":[{"message":{"content":" translated "}}]}`)
	}))
	defer srv.Close()
	pm := config.ProviderModel{Provider: config.ProviderConfig{BaseURL: srv.URL + "/endpoint", EndpointMode: "full"}, Model: "m"}
	out, latency, err := runTranslate(config.Default(), pm, "synthetic-key", "system", "source")
	if err != nil || out != "translated" || latency < 0 {
		t.Fatalf("translation: %q %d %v", out, latency, err)
	}
}
func TestHistoryBusinessRoundTrip(t *testing.T) {
	dir := isolatedConfig(t)
	a := NewTranslateApp()
	a.startup(context.Background())
	if err := a.AddHistory(nil); err != nil {
		t.Fatal(err)
	}
	body := &HistoryPostBody{Input: "source", SourceLanguage: "zh", TargetLanguages: []string{"en"}, Results: []SingleTranslationResult{{Provider: "p", Model: "m", TargetLanguage: "en", Output: "target", LatencyMs: 2}, {Model: "m", Error: "safe error", LatencyMs: 9}, {Model: "", Output: "other"}}}
	if err := a.AddHistory(body); err != nil {
		t.Fatal(err)
	}
	got, err := a.GetHistory("SOURCE")
	if err != nil || len(got) != 1 {
		t.Fatalf("history: %+v %v", got, err)
	}
	if got[0].CreatedAt <= 0 || got[0].LatencyMs != 9 || !reflect.DeepEqual(got[0].Models, []string{"m"}) || !reflect.DeepEqual(got[0].Results, body.Results) {
		t.Fatalf("history fields: %+v", got[0])
	}
	if err := a.AddHistory(&HistoryPostBody{Input: "legacy", Output: "result", Model: "old", CreatedAt: 1, LatencyMs: 4}); err != nil {
		t.Fatal(err)
	}
	got, err = a.GetHistory("legacy")
	if err != nil || len(got) != 1 || got[0].Models[0] != "old" || got[0].LatencyMs != 4 {
		t.Fatalf("legacy: %+v %v", got, err)
	}
	if err := a.ClearHistory(); err != nil {
		t.Fatal(err)
	}
	got, err = a.GetHistory("")
	if err != nil || len(got) != 0 {
		t.Fatalf("clear: %+v %v", got, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "history.json"), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := a.GetHistory(""); err == nil {
		t.Fatal("history read failure ignored")
	}
}
func TestEchoAPIRoutes(t *testing.T) {
	t.Setenv("KAKAPO_HTTP_LOG", "off")
	s := NewEchoService()
	if s.ServiceName() != "Echo API Service" || s.ServiceShutdown(context.Background()) != nil {
		t.Fatal("service lifecycle")
	}
	request(t, s, "GET", "/info", "", 200)
	request(t, s, "GET", "/users", "", 200)
	request(t, s, "GET", "/users/1", "", 200)
	request(t, s, "GET", "/users/bad", "", 400)
	request(t, s, "GET", "/users/999", "", 404)
	w := request(t, s, "POST", "/users", `{"name":"Synthetic","email":"nobody@example.invalid","id":123}`, 201)
	var u User
	if err := json.Unmarshal(w.Body.Bytes(), &u); err != nil || u.ID != 4 || u.Name != "Synthetic" || u.CreatedAt.IsZero() {
		t.Fatalf("created user: %+v %v", u, err)
	}
	request(t, s, "POST", "/users", `broken`, 400)
	request(t, s, "DELETE", "/users/bad", "", 400)
	request(t, s, "DELETE", "/users/999", "", 404)
	request(t, s, "DELETE", "/users/4", "", 200)
	request(t, s, "GET", "/users/4", "", 404)
}
func TestTranslateAPIReadinessAndAssets(t *testing.T) {
	isolatedConfig(t)
	s := NewTranslateWebService()
	if s.ServiceName() != "Translate Web Service" || s.ServiceShutdown(context.Background()) != nil {
		t.Fatal("service lifecycle")
	}
	for _, r := range []struct{ method, path string }{{"POST", "/api/translate"}, {"GET", "/api/settings"}, {"PUT", "/api/settings"}, {"GET", "/api/history"}, {"POST", "/api/history"}, {"DELETE", "/api/history"}} {
		request(t, s, r.method, r.path, `{}`, 503)
	}
	s.assetsFS = fstest.MapFS{"index.html": {Data: []byte("<title>fixture</title>")}, "style.css": {Data: []byte("body{}")}, "main.js": {Data: []byte("export {}")}, "translate-api.js": {Data: []byte("export {}")}, "kakapo.png": {Data: []byte("fixture")}, "assets/app.js": {Data: []byte("fixture")}, "assets/app.css": {Data: []byte("fixture")}, "assets/app.unknown": {Data: []byte("fixture")}}
	for _, path := range []string{"/", "/index.html", "/style.css", "/main.js", "/translate-api.js", "/kakapo.png", "/assets/app.js", "/assets/app.css", "/assets/app.unknown"} {
		w := request(t, s, "GET", path, "", 200)
		if w.Body.Len() == 0 {
			t.Fatalf("empty asset %s", path)
		}
	}
	s.assetsFS = fstest.MapFS{}
	for _, path := range []string{"/", "/index.html", "/style.css", "/main.js", "/translate-api.js", "/kakapo.png", "/assets/", "/assets/missing.js"} {
		request(t, s, "GET", path, "", 404)
	}
}
func TestTranslateAPIValidationAndHistory(t *testing.T) {
	isolatedConfig(t)
	s := NewTranslateWebService()
	if err := s.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	request(t, s, "GET", "/api/settings", "", 200)
	request(t, s, "PUT", "/api/settings", `{}`, 200)
	request(t, s, "PUT", "/api/settings", `broken`, 400)
	request(t, s, "POST", "/api/translate", `broken`, 400)
	request(t, s, "POST", "/api/translate", `{}`, 400)
	request(t, s, "POST", "/api/translate", `{"text":"x"}`, 400)
	request(t, s, "POST", "/api/translate", `{"text":" ","models":["m"]}`, 400)
	w := request(t, s, "POST", "/api/translate", `{"text":"x","targetLanguages":["en"]}`, 200)
	if !strings.Contains(w.Body.String(), "err") {
		t.Fatal("missing provider key not reported")
	}
	request(t, s, "GET", "/api/history", "", 200)
	request(t, s, "POST", "/api/history", `broken`, 400)
	request(t, s, "POST", "/api/history", `{"input":"x","output":"y"}`, 200)
	w = request(t, s, "GET", "/api/history?q=x", "", 200)
	if !strings.Contains(w.Body.String(), `"output":"y"`) {
		t.Fatal("saved history missing")
	}
	request(t, s, "DELETE", "/api/history", "", 200)
	for _, body := range []string{`broken`, `{}`, `{"text":"` + strings.Repeat("x", 5001) + `"}`} {
		request(t, s, "POST", "/api/speak", body, 400)
	}
	request(t, s, "POST", "/api/speak", `{"text":"hello","language":"en"}`, 501)
	wantSplash := 503
	if application.Get() != nil {
		wantSplash = 204
	}
	request(t, s, "POST", "/api/splash", `{}`, wantSplash)
}

type testCredentials struct {
	mu     sync.Mutex
	keys   map[string]string
	setErr error
}

func (s *testCredentials) Get(service, account string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.keys[account], nil
}
func (s *testCredentials) Set(service, account, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.setErr != nil {
		return s.setErr
	}
	s.keys[account] = key
	return nil
}
func (s *testCredentials) Delete(service, account string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.keys, account)
	return nil
}
func TestTranslationProviderBindingAndPartialFailure(t *testing.T) {
	isolatedConfig(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if r.Header.Get("Authorization") != "Bearer key-"+body.Model {
			t.Error("provider identity/key mismatch")
		}
		if body.Model == "bad" {
			w.WriteHeader(500)
			io.WriteString(w, "private synthetic body")
			return
		}
		io.WriteString(w, `{"choices":[{"message":{"content":"translated"}}]}`)
	}))
	defer srv.Close()
	c := config.Default()
	c.Providers = []config.ProviderConfig{{ID: "first", BaseURL: srv.URL, Models: []string{"good"}, Enabled: true}, {ID: "second", BaseURL: srv.URL, Models: []string{"bad"}, Enabled: true}}
	writeConfig(t, c)
	a := NewTranslateApp()
	a.credentials = &testCredentials{keys: map[string]string{"first": "key-good", "second": "key-bad"}}
	single, err := a.TranslateZhToEn(" source ")
	if err != nil || single.Output != "translated" || single.Input != "source" || single.Model != "good" {
		t.Fatalf("single translation: %+v %v", single, err)
	}
	result, err := a.TranslateParallel("source", "zh", []string{"en", "ja"}, nil)
	if err != nil || len(result.Results) != 4 {
		t.Fatalf("parallel: %+v %v", result, err)
	}
	if result.Results[0].Output != "translated" || result.Results[1].TargetLanguage != "ja" || result.Results[2].Error != translate.ErrorMessage(translate.ErrServer) || result.Results[2].Output != "" {
		t.Fatalf("partial results/order: %+v", result.Results)
	}
	c.Providers = c.Providers[1:]
	writeConfig(t, c)
	if _, err := a.TranslateZhToEn("x"); !errors.Is(err, translate.ErrServer) {
		t.Fatal(err)
	}
	dto, err := a.GetSettings()
	if err != nil || !dto.Providers[0].APIKeySet || dto.Providers[0].APIKeyMask != "****-bad" {
		t.Fatalf("key status: %+v %v", dto, err)
	}
	raw, _ := json.Marshal(dto)
	if strings.Contains(string(raw), "key-bad") {
		t.Fatal("plaintext key in settings")
	}
}
func TestCredentialSetFailureAndLazyHistory(t *testing.T) {
	dir := isolatedConfig(t)
	a := NewTranslateApp()
	a.credentials = &testCredentials{keys: map[string]string{}, setErr: errors.New("synthetic credential backend unavailable")}
	if err := a.SaveSettings(&SettingsDTO{Providers: []ProviderDTO{{ID: "x", BaseURL: "https://example.invalid", SetAPIKey: "synthetic-key"}}}); err == nil {
		t.Fatal("credential write failure ignored")
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "invalid"))
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "invalid"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	a.startup(context.Background())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := a.AddHistory(&HistoryPostBody{Input: "recovered"}); err != nil {
		t.Fatalf("lazy history initialization failed: %v", err)
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "invalid"))
	b := NewTranslateApp()
	if err := b.AddHistory(&HistoryPostBody{Input: "blocked"}); err == nil {
		t.Fatal("history init failure ignored")
	}
}
func TestDesktopConstruction(t *testing.T) {
	isolatedConfig(t)
	a := newApplication()
	if a.Config().Name != "Kakapo" || len(a.Config().Services) < 4 {
		t.Fatalf("desktop config: %+v", a.Config())
	}
	if a.Menu.GetApplicationMenu().FindByLabel("App") == nil {
		t.Fatal("application menu missing")
	}
	s := NewEchoService()
	if err := s.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	if s.app != a {
		t.Fatal("echo service bound to wrong application")
	}
	request(t, s, "POST", "/users", `{"name":"Synthetic"}`, 201)
	response := make(chan string, 1)
	a.Event.On("echo-api-response", func(event *application.CustomEvent) { response <- fmt.Sprint(event.Data) })
	s.app.Event.Emit("echo-api-event", map[string]string{"message": "synthetic"})
	select {
	case v := <-response:
		if !strings.Contains(v, "Response from Echo API Service") {
			t.Fatal(v)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("echo event response missing")
	}
	about := make(chan struct{}, 1)
	a.Event.On(EventSplashAbout, func(*application.CustomEvent) { about <- struct{}{} })
	a.Event.Emit(EventShowSplash, nil)
	select {
	case <-about:
	case <-time.After(2 * time.Second):
		t.Fatal("about event missing")
	}
	if shutdown := a.Config().OnShutdown; shutdown != nil {
		shutdown()
	}
}

func TestTranslateAPISuccessAndStorageErrors(t *testing.T) {
	dir := isolatedConfig(t)
	s := NewTranslateWebService()
	if err := s.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"choices":[{"message":{"content":"translated"}}]}`)
	}))
	defer srv.Close()
	c := config.Default()
	c.Providers[0].BaseURL = srv.URL
	writeConfig(t, c)
	s.translateApp.credentials = &testCredentials{keys: map[string]string{config.DefaultProviderID: "synthetic"}}
	w := request(t, s, "POST", "/api/translate", `{"text":"x"}`, 200)
	if !strings.Contains(w.Body.String(), `"output":"translated"`) {
		t.Fatal("legacy translation missing")
	}
	request(t, s, "POST", "/api/translate", `{"text":"x","models":["absent"]}`, 400)
	historyPath := filepath.Join(dir, "history.json")
	if err := os.Mkdir(historyPath, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(historyPath, "keep"), []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	request(t, s, "GET", "/api/history", "", 500)
	request(t, s, "POST", "/api/history", `{"input":"x"}`, 500)
	request(t, s, "DELETE", "/api/history", "", 500)
	settingsPath := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(settingsPath, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	request(t, s, "GET", "/api/settings", "", 500)
	if err := os.Remove(settingsPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(settingsPath, 0700); err != nil {
		t.Fatal(err)
	}
	request(t, s, "PUT", "/api/settings", `{}`, 400)
}
