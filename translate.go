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
	"strings"
	"sync"
	"time"

	"github.com/soulteary/kakapo/internal/config"
	"github.com/soulteary/kakapo/internal/history"
	"github.com/soulteary/kakapo/internal/secrets"
	"github.com/soulteary/kakapo/internal/translate"
)

// maxParallelTranslations bounds the number of concurrent upstream requests in
// TranslateParallel, preventing rate limiting when many languages × models are
// selected at once.
const maxParallelTranslations = 6

// App struct
type TranslateApp struct {
	ctx             context.Context
	historyStore    *history.Store
	historyInitOnce sync.Once
}

// NewApp creates a new App application struct
func NewTranslateApp() *TranslateApp {
	return &TranslateApp{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods. Initializes native menu bar tray on macOS.
func (a *TranslateApp) startup(ctx context.Context) {
	a.ctx = ctx
	if store, err := history.NewStore(); err == nil {
		a.historyStore = store
	} else {
		log.Printf("[Kakapo] 历史记录初始化失败（将在首次使用时重试）: %v", err)
	}
}

// ensureHistoryStore initializes history store if nil (lazy init after startup failure).
func (a *TranslateApp) ensureHistoryStore() {
	a.historyInitOnce.Do(func() {
		if a.historyStore != nil {
			return
		}
		store, err := history.NewStore()
		if err != nil {
			log.Printf("[Kakapo] 历史记录初始化失败: %v", err)
			return
		}
		a.historyStore = store
		log.Printf("[Kakapo] 历史记录将保存至: %s", store.Path())
	})
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
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	BaseURL     string   `json:"baseURL"`
	Models      []string `json:"models"`
	Enabled     bool     `json:"enabled"`
	APIKeySet   bool     `json:"apiKeySet"`
	APIKeyMask  string   `json:"apiKeyMask"`
	SetAPIKey   string   `json:"setAPIKey,omitempty"`
	ClearAPIKey bool     `json:"clearAPIKey"`
}

// SettingsDTO is the settings payload for GetSettings/SaveSettings.
type SettingsDTO struct {
	Providers       []ProviderDTO `json:"providers"`
	SourceLanguage  string        `json:"sourceLanguage,omitempty"`
	TargetLanguages []string      `json:"targetLanguages,omitempty"`
	TimeoutSeconds  int           `json:"timeoutSeconds"`
	Temperature     float64       `json:"temperature"`
	MaxInputChars   int           `json:"maxInputChars"`
	AutoCopy        bool          `json:"autoCopy"`
	ClearOnOpen     bool          `json:"clearOnOpen"`
}

// TranslateZhToEn translates Chinese to English via OpenAI-compatible Chat Completions.
func (a *TranslateApp) TranslateZhToEn(text string) (*TranslationResult, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("请输入要翻译的内容")
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}
	if text != "" && cfg.MaxInputChars > 0 && len([]rune(text)) > cfg.MaxInputChars {
		return nil, fmt.Errorf("输入超过 %d 字，请缩短后重试", cfg.MaxInputChars)
	}

	pm, ok := cfg.FirstProviderModel()
	if !ok {
		return nil, translate.ErrInvalidConfig
	}
	var store secrets.Store = secrets.KeychainStore{}
	apiKey, err := store.Get(secrets.KeychainService, pm.Provider.ID)
	if err != nil || apiKey == "" {
		return nil, translate.ErrNoAPIKey
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
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}
	if cfg.MaxInputChars > 0 && len([]rune(text)) > cfg.MaxInputChars {
		return nil, fmt.Errorf("输入超过 %d 字，请缩短后重试", cfg.MaxInputChars)
	}
	if sourceLang == "" {
		sourceLang = cfg.EffectiveSourceLanguage()
	}
	if len(targetLangs) == 0 {
		targetLangs = cfg.EffectiveTargetLanguages()
	}

	providerModels := cfg.EffectiveProviderModels()
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
		return nil, translate.ErrInvalidConfig
	}

	// Fetch (and cache) each provider's API key once.
	var store secrets.Store = secrets.KeychainStore{}
	keyCache := make(map[string]string)
	for _, pm := range providerModels {
		if _, ok := keyCache[pm.Provider.ID]; ok {
			continue
		}
		key, _ := store.Get(secrets.KeychainService, pm.Provider.ID)
		keyCache[pm.Provider.ID] = key
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
	a.ensureHistoryStore()
	if a.historyStore == nil {
		return fmt.Errorf("历史记录未初始化，无法保存")
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
	return a.historyStore.Add(e)
}

// GetHistory returns history entries, optionally filtered by search query.
func (a *TranslateApp) GetHistory(query string) ([]HistoryEntry, error) {
	a.ensureHistoryStore()
	if a.historyStore == nil {
		return nil, nil
	}
	list, err := a.historyStore.Search(query)
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
	a.ensureHistoryStore()
	if a.historyStore == nil {
		return nil
	}
	return a.historyStore.Clear()
}

// GetSettings returns current settings for the UI (API keys from Keychain, never plaintext).
func (a *TranslateApp) GetSettings() (*SettingsDTO, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	dto := &SettingsDTO{
		SourceLanguage:  cfg.SourceLanguage,
		TargetLanguages: cfg.TargetLanguages,
		TimeoutSeconds:  cfg.TimeoutSeconds,
		Temperature:     cfg.EffectiveTemperature(),
		MaxInputChars:   cfg.MaxInputChars,
		AutoCopy:        cfg.AutoCopy,
		ClearOnOpen:     cfg.ClearOnOpen,
	}
	var store secrets.Store = secrets.KeychainStore{}
	dto.Providers = make([]ProviderDTO, len(cfg.Providers))
	for i, p := range cfg.Providers {
		pd := ProviderDTO{
			ID:      p.ID,
			Type:    p.Type,
			BaseURL: p.BaseURL,
			Models:  p.Models,
			Enabled: p.Enabled,
		}
		if key, err := store.Get(secrets.KeychainService, p.ID); err == nil && key != "" {
			pd.APIKeySet = true
			pd.APIKeyMask = maskAPIKey(key)
		}
		dto.Providers[i] = pd
	}
	return dto, nil
}

// SaveSettings persists settings (JSON for config, Keychain for per-provider API keys).
func (a *TranslateApp) SaveSettings(settings *SettingsDTO) error {
	if settings == nil {
		return fmt.Errorf("settings is nil")
	}

	// Load the previous config to detect removed providers (whose keys we delete).
	prev, _ := config.Load()
	prevIDs := make(map[string]bool, len(prev.Providers))
	for _, p := range prev.Providers {
		prevIDs[p.ID] = true
	}

	var store secrets.Store = secrets.KeychainStore{}
	temperature := settings.Temperature
	cfg := config.Settings{
		SourceLanguage:  strings.TrimSpace(settings.SourceLanguage),
		TargetLanguages: settings.TargetLanguages,
		TimeoutSeconds:  settings.TimeoutSeconds,
		Temperature:     &temperature,
		MaxInputChars:   settings.MaxInputChars,
		AutoCopy:        settings.AutoCopy,
		ClearOnOpen:     settings.ClearOnOpen,
	}

	keepIDs := make(map[string]bool)
	for _, pd := range settings.Providers {
		id := strings.TrimSpace(pd.ID)
		if id == "" {
			id = genProviderID()
		}
		keepIDs[id] = true

		baseURL := strings.TrimSpace(pd.BaseURL)
		typ := strings.TrimSpace(pd.Type)
		if typ == "" {
			typ = config.ProviderTypeFromBaseURL(baseURL)
		}
		cfg.Providers = append(cfg.Providers, config.ProviderConfig{
			ID:      id,
			Type:    typ,
			BaseURL: baseURL,
			Models:  normalizeModels(pd.Models),
			Enabled: pd.Enabled,
		})

		if pd.ClearAPIKey {
			_ = store.Delete(secrets.KeychainService, id)
		}
		if secret := strings.TrimSpace(pd.SetAPIKey); secret != "" {
			if err := store.Set(secrets.KeychainService, id, secret); err != nil {
				return fmt.Errorf("保存 API Key 失败: %w", err)
			}
		}
	}

	// Apply global defaults.
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
	}

	if err := config.Save(cfg); err != nil {
		return err
	}

	// Delete keys of providers removed in this save.
	for id := range prevIDs {
		if !keepIDs[id] {
			_ = store.Delete(secrets.KeychainService, id)
		}
	}
	return nil
}

// runTranslate performs one Chat Completions call for the given provider/model,
// returning the output and measured latency. Shared by the single and parallel
// translation paths.
func runTranslate(cfg config.Settings, pm config.ProviderModel, apiKey, prompt, text string) (string, int64, error) {
	client := translate.NewClient(pm.Provider.BaseURL, apiKey, pm.Model, cfg.TimeoutSeconds, cfg.EffectiveTemperature())
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
