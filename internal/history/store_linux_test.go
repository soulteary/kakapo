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

package history

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewStoreConfigDirectory(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	s, err := NewStore()
	if err != nil {
		t.Fatal(err)
	}
	if s.Path() != filepath.Join(dir, "Kakapo", historyFilename) {
		t.Fatalf("path %q", s.Path())
	}
	blocked := filepath.Join(dir, "blocked")
	if err := os.WriteFile(blocked, nil, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", blocked)
	if _, err := NewStore(); err == nil {
		t.Fatal("expected config directory creation failure")
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	if _, err := NewStore(); err == nil || !strings.Contains(err.Error(), "HOME") {
		t.Fatalf("expected unavailable config home: %v", err)
	}
}
