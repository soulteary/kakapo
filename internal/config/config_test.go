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

import "testing"

func TestEffectiveProviderModels(t *testing.T) {
	s := Settings{
		Providers: []ProviderConfig{
			{ID: "a", BaseURL: "https://api.moonshot.cn/v1", Models: []string{"kimi-k2.6", " "}, Enabled: true},
			{ID: "b", BaseURL: "https://api.deepseek.com", Models: []string{"deepseek-v4-pro"}, Enabled: true},
			{ID: "c", BaseURL: "https://api.openai.com", Models: []string{"gpt-4o-mini"}, Enabled: false},
			{ID: "d", BaseURL: "", Models: []string{"x"}, Enabled: true},
		},
	}
	pms := s.EffectiveProviderModels()
	if len(pms) != 2 {
		t.Fatalf("expected 2 enabled (provider,model) pairs, got %d: %+v", len(pms), pms)
	}
	if pms[0].Provider.ID != "a" || pms[0].Model != "kimi-k2.6" {
		t.Errorf("unexpected first pair: %+v", pms[0])
	}
	if pms[1].Provider.ID != "b" || pms[1].Model != "deepseek-v4-pro" {
		t.Errorf("unexpected second pair: %+v", pms[1])
	}
}

func TestFirstProviderModel(t *testing.T) {
	if _, ok := (Settings{}).FirstProviderModel(); ok {
		t.Error("empty settings should have no first provider model")
	}
	pm, ok := Default().FirstProviderModel()
	if !ok || pm.Model != "kimi-k2.6" {
		t.Errorf("default first provider model = %+v, ok=%v", pm, ok)
	}
}

func TestEffectiveTemperature(t *testing.T) {
	// nil -> default
	if got := (Settings{}).EffectiveTemperature(); got != DefaultTemperature {
		t.Errorf("nil temperature should default to %v, got %v", DefaultTemperature, got)
	}
	// explicit 0 must be preserved (deterministic output), not overwritten
	zero := 0.0
	if got := (Settings{Temperature: &zero}).EffectiveTemperature(); got != 0 {
		t.Errorf("explicit temperature 0 should be preserved, got %v", got)
	}
	// explicit non-zero preserved
	v := 0.7
	if got := (Settings{Temperature: &v}).EffectiveTemperature(); got != 0.7 {
		t.Errorf("explicit temperature 0.7 should be preserved, got %v", got)
	}
}

func TestProviderTypeFromBaseURL(t *testing.T) {
	cases := map[string]string{
		"https://api.moonshot.cn/v1": "kimi",
		"https://api.deepseek.com":   "deepseek",
		"https://api.openai.com":     "openai",
		"https://example.com/v1":     "custom",
	}
	for url, want := range cases {
		if got := ProviderTypeFromBaseURL(url); got != want {
			t.Errorf("ProviderTypeFromBaseURL(%q) = %q, want %q", url, got, want)
		}
	}
}

func TestMigrateLegacyProviders(t *testing.T) {
	d := Default()

	// Legacy with explicit Model -> single provider, id=default, type inferred.
	legacy := Settings{BaseURL: "https://api.deepseek.com", Model: "deepseek-v4-pro"}
	got := migrateLegacyProviders(legacy, d)
	if len(got) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(got))
	}
	if got[0].ID != DefaultProviderID || got[0].Type != "deepseek" || got[0].BaseURL != "https://api.deepseek.com" {
		t.Errorf("unexpected migrated provider: %+v", got[0])
	}
	if len(got[0].Models) != 1 || got[0].Models[0] != "deepseek-v4-pro" {
		t.Errorf("unexpected migrated models: %+v", got[0].Models)
	}
	if !got[0].Enabled {
		t.Error("migrated provider should be enabled")
	}

	// No legacy data -> falls back to default providers.
	got = migrateLegacyProviders(Settings{}, d)
	if len(got) != len(d.Providers) || got[0].ID != DefaultProviderID {
		t.Errorf("expected default providers fallback, got %+v", got)
	}
}
