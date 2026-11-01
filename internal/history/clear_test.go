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
	"testing"
)

func TestClearReportsRemovalFailure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(path, "keep")
	if err := os.WriteFile(sentinel, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	s := &Store{path: path}
	if err := s.Clear(); err == nil {
		t.Fatal("Clear reported success although history could not be removed")
	}
	data, err := os.ReadFile(sentinel)
	if err != nil || string(data) != "preserve" {
		t.Fatalf("existing data changed: %q %v", data, err)
	}
}

func TestClearMissingHistoryIsSuccessful(t *testing.T) {
	s := &Store{path: filepath.Join(t.TempDir(), "history.json")}
	if err := s.Clear(); err != nil {
		t.Fatal(err)
	}
	if err := s.Add(Entry{Input: "saved", CreatedAt: 1}); err != nil {
		t.Fatal(err)
	}
	if err := s.Clear(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.path); !os.IsNotExist(err) {
		t.Fatalf("history still exists: %v", err)
	}
	if err := s.Clear(); err != nil {
		t.Fatal(err)
	}
}
