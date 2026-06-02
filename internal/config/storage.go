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

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// ConfigFilename is the name of the config file in the app config dir.
const ConfigFilename = "settings.json"

// ConfigPath returns the path to the config file (user config dir / Kakapo / settings.json).
func ConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appDir := filepath.Join(dir, "Kakapo")
	if err := os.MkdirAll(appDir, 0700); err != nil {
		return "", err
	}
	return filepath.Join(appDir, ConfigFilename), nil
}

// Load reads settings from the default config file. Returns default settings if
// the file does not exist.
func Load() (Settings, error) {
	path, err := ConfigPath()
	if err != nil {
		return Default(), err
	}
	return LoadFromPath(path)
}

// LoadFromPath reads settings from an explicit path (used by Load and tests).
// Returns default settings if the file does not exist, and applies the same
// migration/default normalization as Load.
func LoadFromPath(path string) (Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return Default(), err
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return Default(), err
	}
	d := Default()
	// Migrate legacy single-provider configs into Providers. The legacy API key
	// stays under Keychain account = DefaultProviderID, so it keeps working.
	if len(s.Providers) == 0 {
		s.Providers = migrateLegacyProviders(s, d)
	}
	// Clear legacy fields so they are not re-persisted on the next Save.
	s.BaseURL = ""
	s.Model = ""
	s.Models = nil
	// Apply defaults for missing/zero global values (backward compatible).
	if s.TimeoutSeconds <= 0 {
		s.TimeoutSeconds = d.TimeoutSeconds
	}
	if s.MaxInputChars <= 0 {
		s.MaxInputChars = d.MaxInputChars
	}
	if s.SourceLanguage == "" {
		s.SourceLanguage = d.SourceLanguage
	}
	if len(s.TargetLanguages) == 0 {
		s.TargetLanguages = d.TargetLanguages
	}
	if s.Temperature == nil {
		s.Temperature = d.Temperature
	}
	return s, nil
}

// migrateLegacyProviders builds a Providers slice from legacy single-provider
// fields (BaseURL/Model/Models). Falls back to the default provider when no
// legacy data is present.
func migrateLegacyProviders(s Settings, d Settings) []ProviderConfig {
	baseURL := s.BaseURL
	if baseURL == "" {
		return d.Providers
	}
	models := s.Models
	if len(models) == 0 && s.Model != "" {
		models = []string{s.Model}
	}
	if len(models) == 0 {
		models = []string{"kimi-k2.6"}
	}
	return []ProviderConfig{
		{
			ID:      DefaultProviderID,
			Type:    ProviderTypeFromBaseURL(baseURL),
			BaseURL: baseURL,
			Models:  models,
			Enabled: true,
		},
	}
}

// Save writes settings to the default config file.
func Save(s Settings) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	return SaveToPath(s, path)
}

// SaveToPath writes settings to an explicit path (used by Save and tests).
func SaveToPath(s Settings, path string) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
