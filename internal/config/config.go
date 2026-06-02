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

package config

import "strings"

// DefaultProviderID is the id (and Keychain account) used for the provider
// migrated from the legacy single-provider configuration.
const DefaultProviderID = "default"

// DefaultTemperature is used when no temperature is configured (nil).
const DefaultTemperature = 0.2

// ProviderConfig is one configured upstream provider. The API key is NOT stored
// here; it lives in the Keychain under account = ID.
type ProviderConfig struct {
	ID      string   `json:"id"`
	Type    string   `json:"type"` // kimi|deepseek|openai|custom (UI hint only)
	BaseURL string   `json:"baseURL"`
	Models  []string `json:"models"`
	Enabled bool     `json:"enabled"`
}

// Settings holds non-secret app settings (API Keys are in Keychain, per provider).
type Settings struct {
	Providers       []ProviderConfig `json:"providers,omitempty"`
	SourceLanguage  string           `json:"sourceLanguage,omitempty"`  // e.g. "zh"
	TargetLanguages []string         `json:"targetLanguages,omitempty"` // e.g. ["en", "ja"]
	TimeoutSeconds  int              `json:"timeoutSeconds"`
	// Temperature is a pointer so a deliberate 0 (deterministic output) can be
	// distinguished from "not set" (nil). Use EffectiveTemperature to read it.
	Temperature   *float64 `json:"temperature,omitempty"`
	MaxInputChars int      `json:"maxInputChars"`
	AutoCopy      bool     `json:"autoCopy"`
	ClearOnOpen   bool     `json:"clearOnOpen"`

	// Legacy single-provider fields, kept only for backward-compatible migration
	// (see storage.Load). New configs persist everything under Providers.
	BaseURL string   `json:"baseURL,omitempty"`
	Model   string   `json:"model,omitempty"`
	Models  []string `json:"models,omitempty"`
}

// ProviderModel pairs an enabled provider with one of its model names.
type ProviderModel struct {
	Provider ProviderConfig
	Model    string
}

// Default returns default settings with a single Kimi (Moonshot) provider.
func Default() Settings {
	temp := DefaultTemperature
	return Settings{
		Providers: []ProviderConfig{
			{
				ID:      DefaultProviderID,
				Type:    "kimi",
				BaseURL: "https://api.moonshot.cn/v1",
				Models:  []string{"kimi-k2.6"},
				Enabled: true,
			},
		},
		SourceLanguage:  "zh",
		TargetLanguages: []string{"en"},
		TimeoutSeconds:  30,
		Temperature:     &temp,
		MaxInputChars:   5000,
		AutoCopy:        false,
		ClearOnOpen:     false,
	}
}

// ProviderTypeFromBaseURL infers the provider type from a base URL, used when
// migrating legacy configs and as a UI hint.
func ProviderTypeFromBaseURL(baseURL string) string {
	u := strings.ToLower(strings.TrimSpace(baseURL))
	switch {
	case strings.Contains(u, "moonshot"):
		return "kimi"
	case strings.Contains(u, "deepseek"):
		return "deepseek"
	case strings.Contains(u, "openai"):
		return "openai"
	default:
		return "custom"
	}
}

// EnabledProviders returns the subset of providers that are enabled and have a
// non-empty base URL.
func (s Settings) EnabledProviders() []ProviderConfig {
	var out []ProviderConfig
	for _, p := range s.Providers {
		if p.Enabled && strings.TrimSpace(p.BaseURL) != "" {
			out = append(out, p)
		}
	}
	return out
}

// EffectiveProviderModels expands all enabled providers into (provider, model)
// pairs to be run in parallel during translation.
func (s Settings) EffectiveProviderModels() []ProviderModel {
	var out []ProviderModel
	for _, p := range s.EnabledProviders() {
		for _, m := range p.Models {
			m = strings.TrimSpace(m)
			if m == "" {
				continue
			}
			out = append(out, ProviderModel{Provider: p, Model: m})
		}
	}
	return out
}

// FirstProviderModel returns the first enabled (provider, model) pair, if any.
func (s Settings) FirstProviderModel() (ProviderModel, bool) {
	pms := s.EffectiveProviderModels()
	if len(pms) == 0 {
		return ProviderModel{}, false
	}
	return pms[0], true
}

// EffectiveTemperature returns the configured temperature, or DefaultTemperature
// when unset (nil). A configured 0 is preserved (deterministic output).
func (s Settings) EffectiveTemperature() float64 {
	if s.Temperature == nil {
		return DefaultTemperature
	}
	return *s.Temperature
}

// EffectiveSourceLanguage returns source language, default "zh".
func (s Settings) EffectiveSourceLanguage() string {
	if s.SourceLanguage != "" {
		return s.SourceLanguage
	}
	return "zh"
}

// EffectiveTargetLanguages returns target languages, default ["en"].
func (s Settings) EffectiveTargetLanguages() []string {
	if len(s.TargetLanguages) > 0 {
		return s.TargetLanguages
	}
	return []string{"en"}
}
