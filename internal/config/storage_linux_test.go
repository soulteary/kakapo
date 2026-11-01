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
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefaultLanguageFallbacks(t *testing.T) {
	var s Settings
	if s.EffectiveSourceLanguage() != "zh" || !reflect.DeepEqual(s.EffectiveTargetLanguages(), []string{"en"}) {
		t.Fatal("missing defaults")
	}
	s.SourceLanguage = "ja"
	s.TargetLanguages = []string{"fr"}
	if s.EffectiveSourceLanguage() != "ja" || s.EffectiveTargetLanguages()[0] != "fr" {
		t.Fatal("explicit languages changed")
	}
}
func TestConfigPathsAndIOFailures(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path, err := ConfigPath()
	if err != nil || path != filepath.Join(dir, "Kakapo", ConfigFilename) {
		t.Fatalf("path=%q err=%v", path, err)
	}
	if err := Save(Default()); err != nil {
		t.Fatal(err)
	}
	got, err := Load()
	if err != nil || got.TimeoutSeconds != 30 {
		t.Fatalf("round trip: %+v %v", got, err)
	}
	if _, err := LoadFromPath(dir); err == nil {
		t.Fatal("directory read accepted")
	}
	if err := os.WriteFile(path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFromPath(path); err == nil {
		t.Fatal("corrupt config accepted")
	}
	s := Default()
	v := math.NaN()
	s.Temperature = &v
	if err := SaveToPath(s, path); err == nil {
		t.Fatal("invalid JSON temperature saved")
	}
	if err := SaveToPath(Default(), dir); err == nil {
		t.Fatal("write to directory accepted")
	}
	blocker := filepath.Join(dir, "file")
	if err := os.WriteFile(blocker, nil, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", blocker)
	if _, err := ConfigPath(); err == nil {
		t.Fatal("invalid config parent accepted")
	}
	if _, err := Load(); err == nil {
		t.Fatal("load ignored path failure")
	}
	if err := Save(Default()); err == nil {
		t.Fatal("save ignored path failure")
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	if _, err := ConfigPath(); err == nil {
		t.Fatal("unavailable config home accepted")
	}
}
func TestLoadLegacyDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	for _, body := range []string{`{}`, `{"baseURL":"https://example.invalid"}`, `{"baseURL":"https://example.invalid","models":["custom"]}`} {
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
		s, err := LoadFromPath(path)
		if err != nil {
			t.Fatal(err)
		}
		if s.TimeoutSeconds != 30 || s.MaxInputChars != 5000 || s.SourceLanguage != "zh" || len(s.TargetLanguages) != 1 || s.Temperature == nil || len(s.Providers) != 1 || s.BaseURL != "" || s.Model != "" || s.Models != nil {
			t.Fatalf("migration: %+v", s)
		}
	}
}
