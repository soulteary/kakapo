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
	"path/filepath"
	"testing"
)

func TestSaveLoad_roundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	d := Default()
	if len(d.Providers) == 0 || d.Providers[0].BaseURL == "" || len(d.Providers[0].Models) == 0 {
		t.Error("Default() should have at least one provider with BaseURL and models")
	}
	if d.MaxInputChars <= 0 {
		t.Error("Default() MaxInputChars should be positive")
	}

	if err := SaveToPath(d, path); err != nil {
		t.Fatalf("SaveToPath: %v", err)
	}
	loaded, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("LoadFromPath: %v", err)
	}
	if len(loaded.Providers) != len(d.Providers) || loaded.MaxInputChars != d.MaxInputChars {
		t.Errorf("roundtrip mismatch: saved %+v loaded %+v", d, loaded)
	}
	if loaded.EffectiveTemperature() != d.EffectiveTemperature() {
		t.Errorf("temperature roundtrip mismatch: saved %v loaded %v", d.EffectiveTemperature(), loaded.EffectiveTemperature())
	}
}

// A deliberately configured temperature of 0 must survive a save/load round-trip
// and not be silently reset to the default.
func TestSaveLoad_preserves_zero_temperature(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")

	s := Default()
	zero := 0.0
	s.Temperature = &zero
	if err := SaveToPath(s, path); err != nil {
		t.Fatalf("SaveToPath: %v", err)
	}
	loaded, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("LoadFromPath: %v", err)
	}
	if loaded.Temperature == nil {
		t.Fatal("loaded temperature is nil, expected explicit 0")
	}
	if got := loaded.EffectiveTemperature(); got != 0 {
		t.Errorf("expected temperature 0 to be preserved, got %v", got)
	}
}

// A missing config file yields defaults (with a non-nil temperature).
func TestLoadFromPath_missing_file_returns_default(t *testing.T) {
	dir := t.TempDir()
	loaded, err := LoadFromPath(filepath.Join(dir, "does-not-exist.json"))
	if err != nil {
		t.Fatalf("LoadFromPath on missing file should not error: %v", err)
	}
	if loaded.Temperature == nil || loaded.EffectiveTemperature() != DefaultTemperature {
		t.Errorf("missing file should yield default temperature %v, got %v", DefaultTemperature, loaded.EffectiveTemperature())
	}
}
