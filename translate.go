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
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/soulteary/kakapo/internal/config"
	"github.com/soulteary/kakapo/internal/history"
	"github.com/soulteary/kakapo/internal/secrets"
	"github.com/soulteary/kakapo/internal/translate"
)

// settingsMu protects configuration and credential snapshots within one process.
var settingsMu sync.RWMutex

// maxParallelTranslations bounds the number of concurrent upstream requests in
// TranslateParallel, preventing rate limiting when many languages × models are
// selected at once.
const maxParallelTranslations = 6

// App struct
type TranslateApp struct {
	ctx          context.Context
	historyStore *history.Store
	historyMu    sync.Mutex
	credentials  secrets.Store
}

// NewApp creates a new App application struct
func NewTranslateApp() *TranslateApp {
	return &TranslateApp{}
}

// secretStore returns the configured credential backend or the platform default.
func (a *TranslateApp) secretStore() secrets.Store {
	if a.credentials != nil {
		return a.credentials
	}
	return secrets.KeychainStore{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods. Initializes native menu bar tray on macOS.
func (a *TranslateApp) startup(ctx context.Context) {
	a.ctx = ctx
	if _, err := a.ensureHistoryStore(); err != nil {
		log.Printf("[Kakapo] 历史记录初始化失败，后续操作将重试 (%T)", err)
	}
}

// ensureHistoryStore retries failed initialization and publishes only a usable store.
func (a *TranslateApp) ensureHistoryStore() (*history.Store, error) {
	a.historyMu.Lock()
	defer a.historyMu.Unlock()
	if a.historyStore != nil {
		return a.historyStore, nil
	}
	store, err := history.NewStore()
	if err != nil {
		return nil, err
	}
	a.historyStore = store
	log.Printf("[Kakapo] 历史记录存储已初始化")
	return store, nil
}

// TranslationResult is returned by TranslateZhToEn (single result, legacy).
type TranslationResult struct {
	Input           string   `json:"input"`
	Output          string   `json:"output"`
	Provider        string   `json:"provider"`
	Model           string   `json:"model"`
	LatencyMs       int64    `json:"latencyMs"`
	CreatedAt       int64    `json:"createdAt"`
	SourceLanguage  string   `json:"sourceLanguage,omitempty"`
	TargetLanguages []string `json:"targetLanguages,omitempty"`
}

// SingleTranslationResult is one result in a parallel/multi-language run.
type SingleTranslationResult struct {
	Provider       string `json:"provider,omitempty"`
	Model          string `json:"model"`
	TargetLanguage string `json:"targetLanguage"`
	Output         string `json:"output"`
	LatencyMs      int64  `json:"latencyMs"`
	Error          string `json:"error,omitempty"`
}

// MultiTranslationResult is the response for parallel translate (grouped by target language on frontend).
type MultiTranslationResult struct {
	Input     string                    `json:"input"`
	Results   []SingleTranslationResult `json:"results"`
	CreatedAt int64                     `json:"createdAt"`
}

// HistoryEntry is one history record for the UI.
type HistoryEntry struct {
	Input           string                    `json:"input"`
	Output          string                    `json:"output"`
	CreatedAt       int64                     `json:"createdAt"`
	SourceLanguage  string                    `json:"sourceLanguage,omitempty"`
	TargetLanguages []string                  `json:"targetLanguages,omitempty"`
	Models          []string                  `json:"models,omitempty"`
	LatencyMs       int64                     `json:"latencyMs,omitempty"`
	Results         []SingleTranslationResult `json:"results,omitempty"`
}

// HistoryPostBody is the payload for POST /api/history (legacy single or extended multi-result).
type HistoryPostBody struct {
	Input           string                    `json:"input"`
	Output          string                    `json:"output"`
	Provider        string                    `json:"provider"`
	Model           string                    `json:"model"`
	LatencyMs       int64                     `json:"latencyMs"`
	CreatedAt       int64                     `json:"createdAt"`
	SourceLanguage  string                    `json:"sourceLanguage,omitempty"`
	TargetLanguages []string                  `json:"targetLanguages,omitempty"`
	Models          []string                  `json:"models,omitempty"`
	Results         []SingleTranslationResult `json:"results,omitempty"`
}

// ProviderDTO is one provider config for the UI. The API key itself is never
// sent to the frontend in plaintext: APIKeySet/APIKeyMask describe its state,
// and SetAPIKey/ClearAPIKey are write-only directives.
type ProviderDTO struct {
	ID                      string   `json:"id"`
	Type                    string   `json:"type"`
	BaseURL                 string   `json:"baseURL"`
	EndpointMode            string   `json:"endpointMode"`
	Models                  []string `json:"models"`
	Enabled                 bool     `json:"enabled"`
	APIKeyNeedsConfirmation bool     `json:"apiKeyNeedsConfirmation"`
	APIKeySet               bool     `json:"apiKeySet"`
	APIKeyMask              string   `json:"apiKeyMask"`
	SetAPIKey               string   `json:"setAPIKey,omitempty"`
	ClearAPIKey             bool     `json:"clearAPIKey"`
}

// SettingsDTO is the settings payload for GetSettings/SaveSettings.
type SettingsDTO struct {
	PendingCredentialIDs []string      `json:"pendingCredentialIDs,omitempty"`
	Providers            []ProviderDTO `json:"providers"`
	SourceLanguage       string        `json:"sourceLanguage,omitempty"`
	TargetLanguages      []string      `json:"targetLanguages,omitempty"`
	TimeoutSeconds       int           `json:"timeoutSeconds"`
	Temperature          float64       `json:"temperature"`
	MaxInputChars        int           `json:"maxInputChars"`
	AutoCopy             bool          `json:"autoCopy"`
	ClearOnOpen          bool          `json:"clearOnOpen"`
}

// TranslateZhToEn translates Chinese to English via OpenAI-compatible Chat Completions.
func (a *TranslateApp) TranslateZhToEn(text string) (*TranslationResult, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("请输入要翻译的内容")
	}

	var cfg config.Settings
	var pm config.ProviderModel
	var apiKey string
	if err := func() error {
		settingsMu.RLock()
		defer settingsMu.RUnlock()
		var err error
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("加载配置失败: %w", err)
		}
		if len(cfg.PendingCredentialIDs) > 0 {
			return translate.ErrSettingsPending
		}
		if text != "" && cfg.MaxInputChars > 0 && len([]rune(text)) > cfg.MaxInputChars {
			return fmt.Errorf("输入超过 %d 字，请缩短后重试", cfg.MaxInputChars)
		}

		var ok bool
		pm, ok = cfg.FirstProviderModel()
		if !ok {
			return translate.ErrInvalidConfig
		}
		store := a.secretStore()
		apiKey, err = store.Get(secrets.KeychainService, pm.Provider.ID)
		if err != nil || apiKey == "" {
			return translate.ErrNoAPIKey
		}

		return nil
	}(); err != nil {
		return nil, err
	}

	out, latency, err := runTranslate(cfg, pm, apiKey, translate.SystemPrompt, text)
	if err != nil {
		return nil, err
	}

	return &TranslationResult{
		Input:     text,
		Output:    out,
		Provider:  pm.Provider.Type,
		Model:     pm.Model,
		LatencyMs: latency,
		CreatedAt: time.Now().Unix(),
	}, nil
}

// TranslateParallel runs translation for each (model × targetLanguage) in parallel and returns all results.
// Partial success is allowed: each task's error is stored in SingleTranslationResult.Error.
func (a *TranslateApp) TranslateParallel(text, sourceLang string, targetLangs, models []string) (*MultiTranslationResult, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("请输入要翻译的内容")
	}
	var cfg config.Settings
	var providerModels []config.ProviderModel
	var keyCache map[string]string
	if err := func() error {
		settingsMu.RLock()
		defer settingsMu.RUnlock()
		var err error
		cfg, err = config.Load()
		if err != nil {
			return fmt.Errorf("加载配置失败: %w", err)
		}
		if len(cfg.PendingCredentialIDs) > 0 {
			return translate.ErrSettingsPending
		}
		if cfg.MaxInputChars > 0 && len([]rune(text)) > cfg.MaxInputChars {
			return fmt.Errorf("输入超过 %d 字，请缩短后重试", cfg.MaxInputChars)
		}
		if sourceLang == "" {
			sourceLang = cfg.EffectiveSourceLanguage()
		}
		if len(targetLangs) == 0 {
			targetLangs = cfg.EffectiveTargetLanguages()
		}

		providerModels = cfg.EffectiveProviderModels()
		// Optional filter: if the caller passes explicit model names, only run those.
		if len(models) > 0 {
			want := make(map[string]bool, len(models))
			for _, m := range models {
				want[strings.TrimSpace(m)] = true
			}
			var filtered []config.ProviderModel
			for _, pm := range providerModels {
				if want[pm.Model] {
					filtered = append(filtered, pm)
				}
			}
			providerModels = filtered
		}
		if len(providerModels) == 0 {
			return translate.ErrInvalidConfig
		}

		// Fetch (and cache) each provider's API key once.
		store := a.secretStore()
		keyCache = make(map[string]string)
		for _, pm := range providerModels {
			if _, ok := keyCache[pm.Provider.ID]; ok {
				continue
			}
			key, _ := store.Get(secrets.KeychainService, pm.Provider.ID)
			keyCache[pm.Provider.ID] = key
		}

		return nil
	}(); err != nil {
		return nil, err
	}

	// Build the full task list (provider×model × targetLang) so results keep a
	// stable order regardless of completion timing.
	type task struct {
		pm         config.ProviderModel
		targetLang string
	}
	var tasks []task
	for _, pm := range providerModels {
		for _, targetLang := range targetLangs {
			tasks = append(tasks, task{pm: pm, targetLang: targetLang})
		}
	}

	results := make([]SingleTranslationResult, len(tasks))
	createdAt := time.Now().Unix()

	// Bound concurrency so selecting many languages × models does not flood the
	// upstream API and trigger rate limiting.
	sem := make(chan struct{}, maxParallelTranslations)
	var wg sync.WaitGroup
	for i, tk := range tasks {
		i, tk := i, tk
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			sr := SingleTranslationResult{
				Provider:       tk.pm.Provider.Type,
				Model:          tk.pm.Model,
				TargetLanguage: tk.targetLang,
			}
			apiKey := keyCache[tk.pm.Provider.ID]
			if apiKey == "" {
				sr.Error = translate.ErrorMessage(translate.ErrNoAPIKey)
				results[i] = sr
				return
			}
			prompt := translate.BuildSystemPrompt(sourceLang, tk.targetLang)
			out, latency, err := runTranslate(cfg, tk.pm, apiKey, prompt, text)
			sr.LatencyMs = latency
			if err != nil {
				sr.Error = translate.ErrorMessage(err)
			} else {
				sr.Output = out
			}
			results[i] = sr
		}()
	}
	wg.Wait()

	return &MultiTranslationResult{
		Input:     text,
		Results:   results,
		CreatedAt: createdAt,
	}, nil
}

// AddHistory appends a translation to history (called by frontend after success).
// Supports both legacy single result (Output + Model + LatencyMs) and extended multi-result (Results + Models).
func (a *TranslateApp) AddHistory(body *HistoryPostBody) error {
	if body == nil {
		return nil
	}
	store, err := a.ensureHistoryStore()
	if err != nil {
		return err
	}
	createdAt := body.CreatedAt
	if createdAt == 0 {
		createdAt = time.Now().Unix()
	}
	e := history.Entry{
		Input:           body.Input,
		Output:          body.Output,
		CreatedAt:       createdAt,
		SourceLanguage:  body.SourceLanguage,
		TargetLanguages: body.TargetLanguages,
		Models:          body.Models,
		LatencyMs:       body.LatencyMs,
	}
	if len(body.Results) > 0 {
		e.Results = make([]history.ResultItem, len(body.Results))
		for i, r := range body.Results {
			e.Results[i] = history.ResultItem{
				Provider:       r.Provider,
				Model:          r.Model,
				TargetLanguage: r.TargetLanguage,
				Output:         r.Output,
				LatencyMs:      r.LatencyMs,
				Error:          r.Error,
			}
		}
		if len(e.Models) == 0 {
			seen := make(map[string]bool)
			for _, r := range body.Results {
				if r.Model != "" && !seen[r.Model] {
					e.Models = append(e.Models, r.Model)
					seen[r.Model] = true
				}
			}
		}
		if e.LatencyMs == 0 && len(body.Results) > 0 {
			for _, r := range body.Results {
				if r.LatencyMs > e.LatencyMs {
					e.LatencyMs = r.LatencyMs
				}
			}
		}
	} else if body.Output != "" {
		if body.Model != "" {
			e.Models = []string{body.Model}
		}
		e.LatencyMs = body.LatencyMs
	}
	return store.Add(e)
}

// GetHistory returns history entries, optionally filtered by search query.
func (a *TranslateApp) GetHistory(query string) ([]HistoryEntry, error) {
	store, err := a.ensureHistoryStore()
	if err != nil {
		return nil, err
	}
	list, err := store.Search(query)
	if err != nil {
		return nil, err
	}
	out := make([]HistoryEntry, len(list))
	for i, e := range list {
		out[i] = HistoryEntry{
			Input:           e.Input,
			Output:          e.Output,
			CreatedAt:       e.CreatedAt,
			SourceLanguage:  e.SourceLanguage,
			TargetLanguages: e.TargetLanguages,
			Models:          e.Models,
			LatencyMs:       e.LatencyMs,
		}
		if len(e.Results) > 0 {
			out[i].Results = make([]SingleTranslationResult, len(e.Results))
			for j, r := range e.Results {
				out[i].Results[j] = SingleTranslationResult{
					Provider:       r.Provider,
					Model:          r.Model,
					TargetLanguage: r.TargetLanguage,
					Output:         r.Output,
					LatencyMs:      r.LatencyMs,
					Error:          r.Error,
				}
			}
		}
	}
	return out, nil
}

// ClearHistory removes all history entries.
func (a *TranslateApp) ClearHistory() error {
	store, err := a.ensureHistoryStore()
	if err != nil {
		return err
	}
	return store.Clear()
}

// GetSettings returns current settings for the UI (API keys from Keychain, never plaintext).
func (a *TranslateApp) GetSettings() (*SettingsDTO, error) {
	settingsMu.RLock()
	defer settingsMu.RUnlock()
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	dto := &SettingsDTO{
		PendingCredentialIDs: cfg.PendingCredentialIDs,
		SourceLanguage:       cfg.SourceLanguage,
		TargetLanguages:      cfg.TargetLanguages,
		TimeoutSeconds:       cfg.TimeoutSeconds,
		Temperature:          cfg.EffectiveTemperature(),
		MaxInputChars:        cfg.MaxInputChars,
		AutoCopy:             cfg.AutoCopy,
		ClearOnOpen:          cfg.ClearOnOpen,
	}
	store := a.secretStore()
	providers := append([]config.ProviderConfig(nil), cfg.Providers...)
	seen := make(map[string]bool, len(providers))
	for _, p := range providers {
		seen[p.ID] = true
	}
	for _, id := range cfg.PendingCredentialIDs {
		if !seen[id] {
			providers = append(providers, config.ProviderConfig{ID: id, Type: "custom"})
			seen[id] = true
		}
	}
	dto.Providers = make([]ProviderDTO, len(providers))
	for i, p := range providers {
		pd := ProviderDTO{
			ID:           p.ID,
			Type:         p.Type,
			BaseURL:      p.BaseURL,
			EndpointMode: p.EndpointMode,
			Models:       p.Models,
			Enabled:      p.Enabled,
		}
		if key, err := store.Get(secrets.KeychainService, p.ID); err == nil && key != "" {
			pd.APIKeySet = true
			pd.APIKeyMask = maskAPIKey(key)
		}
		for _, id := range cfg.PendingCredentialIDs {
			if id == p.ID {
				pd.APIKeyNeedsConfirmation = true
			}
		}
		dto.Providers[i] = pd
	}
	return dto, nil
}

// SaveSettings persists settings with a durable marker around credential changes.
// A failed/ambiguous credential operation keeps translations blocked until the
// affected credentials are explicitly set, cleared or removed in a later save.
func (a *TranslateApp) SaveSettings(settings *SettingsDTO) error {
	settingsMu.Lock()
	defer settingsMu.Unlock()
	if settings == nil {
		return fmt.Errorf("settings is nil")
	}
	if math.IsNaN(settings.Temperature) || math.IsInf(settings.Temperature, 0) {
		return translate.ErrInvalidConfig
	}
	prev, err := config.Load()
	if err != nil {
		return err
	}
	prevIDs := make(map[string]bool)
	for _, p := range prev.Providers {
		if strings.TrimSpace(p.ID) == "" || prevIDs[p.ID] {
			return translate.ErrInvalidConfig
		}
		prevIDs[p.ID] = true
	}
	for _, id := range prev.PendingCredentialIDs {
		if strings.TrimSpace(id) == "" {
			return translate.ErrInvalidConfig
		}
	}
	temperature := settings.Temperature
	cfg := config.Settings{SourceLanguage: strings.TrimSpace(settings.SourceLanguage), TargetLanguages: settings.TargetLanguages, TimeoutSeconds: settings.TimeoutSeconds, Temperature: &temperature, MaxInputChars: settings.MaxInputChars, AutoCopy: settings.AutoCopy, ClearOnOpen: settings.ClearOnOpen}
	type change struct {
		clear  bool
		secret string
	}
	changes := make(map[string]change)
	keepIDs := make(map[string]bool)
	for _, pd := range settings.Providers {
		id := strings.TrimSpace(pd.ID)
		if id == "" {
			id = genProviderID()
		}
		if keepIDs[id] {
			return translate.ErrInvalidConfig
		}
		keepIDs[id] = true
		baseURL := strings.TrimSpace(pd.BaseURL)
		typ := strings.TrimSpace(pd.Type)
		if typ == "" {
			typ = config.ProviderTypeFromBaseURL(baseURL)
		}
		cfg.Providers = append(cfg.Providers, config.ProviderConfig{ID: id, Type: typ, BaseURL: baseURL, EndpointMode: string(translate.NormalizeEndpointMode(pd.EndpointMode)), Models: normalizeModels(pd.Models), Enabled: pd.Enabled})
		secret := strings.TrimSpace(pd.SetAPIKey)
		if pd.ClearAPIKey || secret != "" {
			changes[id] = change{pd.ClearAPIKey, secret}
		}
	}
	d := config.Default()
	if cfg.SourceLanguage == "" {
		cfg.SourceLanguage = d.SourceLanguage
	}
	if len(cfg.TargetLanguages) == 0 {
		cfg.TargetLanguages = d.TargetLanguages
	}
	if cfg.TimeoutSeconds <= 0 {
		cfg.TimeoutSeconds = d.TimeoutSeconds
	}
	if cfg.MaxInputChars <= 0 {
		cfg.MaxInputChars = d.MaxInputChars
	}
	if len(cfg.Providers) == 0 {
		cfg.Providers = d.Providers
		for _, p := range cfg.Providers {
			keepIDs[p.ID] = true
		}
	}
	for id := range prevIDs {
		if !keepIDs[id] {
			changes[id] = change{clear: true}
		}
	}
	for _, id := range prev.PendingCredentialIDs {
		if !keepIDs[id] {
			changes[id] = change{clear: true}
		} else if _, ok := changes[id]; !ok {
			return translate.ErrSettingsPending
		}
	}
	if len(changes) > 0 {
		ids := make([]string, 0, len(changes))
		for id := range changes {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		pending := prev
		pending.PendingCredentialIDs = ids
		if err := config.Save(pending); err != nil {
			return err
		}
		store := a.secretStore()
		for _, id := range ids {
			op := changes[id]
			if op.clear {
				if err := store.Delete(secrets.KeychainService, id); err != nil {
					return translate.ErrSettingsPending
				}
			}
			if op.secret != "" {
				if err := store.Set(secrets.KeychainService, id, op.secret); err != nil {
					return translate.ErrSettingsPending
				}
			}
		}
	}
	return config.Save(cfg)
}

// runTranslate performs one Chat Completions call for the given provider/model,
// returning the output and measured latency. Shared by the single and parallel
// translation paths.
func runTranslate(cfg config.Settings, pm config.ProviderModel, apiKey, prompt, text string) (string, int64, error) {
	client := translate.NewClient(pm.Provider.BaseURL, apiKey, pm.Model, cfg.TimeoutSeconds, cfg.EffectiveTemperature())
	client.EndpointMode = translate.NormalizeEndpointMode(pm.Provider.EndpointMode)
	start := time.Now()
	out, err := client.Translate(prompt, text)
	return out, time.Since(start).Milliseconds(), err
}

// maskAPIKey returns a masked representation of an API key for display.
func maskAPIKey(key string) string {
	if len(key) >= 4 {
		return "****" + key[len(key)-4:]
	}
	return "****"
}

// genProviderID returns a short random hex id for a newly added provider.
func genProviderID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("p_%d", time.Now().UnixNano())
	}
	return "p_" + hex.EncodeToString(b)
}

// normalizeModels trims and drops empty model names, preserving order.
func normalizeModels(models []string) []string {
	var out []string
	for _, m := range models {
		m = strings.TrimSpace(m)
		if m != "" {
			out = append(out, m)
		}
	}
	return out
}
