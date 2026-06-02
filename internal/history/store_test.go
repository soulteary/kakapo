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
	"encoding/json"
	"testing"
)

func TestEntryUnmarshal_lowercase(t *testing.T) {
	data := []byte(`[{"input":"你好","output":"hi","createdAt":123,"models":["m1"],"latencyMs":50,
		"results":[{"provider":"kimi","model":"m1","targetLanguage":"en","output":"hi","latencyMs":50}]}]`)
	var list []Entry
	if err := json.Unmarshal(data, &list); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(list))
	}
	e := list[0]
	if e.Input != "你好" || e.Output != "hi" || e.CreatedAt != 123 || e.LatencyMs != 50 {
		t.Errorf("unexpected entry: %+v", e)
	}
	if len(e.Results) != 1 || e.Results[0].Provider != "kimi" || e.Results[0].TargetLanguage != "en" {
		t.Errorf("unexpected results: %+v", e.Results)
	}
}

// Older files were written with capitalized (exported field) keys; they must
// still decode correctly.
func TestEntryUnmarshal_capitalized_legacy(t *testing.T) {
	data := []byte(`[{"Input":"你好","Output":"hi","CreatedAt":123,"Models":["m1"],"LatencyMs":50,
		"Results":[{"Provider":"kimi","Model":"m1","TargetLanguage":"en","Output":"hi","LatencyMs":50}]}]`)
	var list []Entry
	if err := json.Unmarshal(data, &list); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(list))
	}
	e := list[0]
	if e.Input != "你好" || e.Output != "hi" || e.CreatedAt != 123 || e.LatencyMs != 50 {
		t.Errorf("unexpected entry from legacy keys: %+v", e)
	}
	if len(e.Results) != 1 || e.Results[0].Provider != "kimi" || e.Results[0].TargetLanguage != "en" {
		t.Errorf("unexpected results from legacy keys: %+v", e.Results)
	}
}

func TestStore_AddListClear(t *testing.T) {
	dir := t.TempDir()
	s := &Store{path: dir + "/history.json"}

	if err := s.Add(Entry{Input: "a", Output: "x", CreatedAt: 1}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := s.Add(Entry{Input: "b", Output: "y", CreatedAt: 2}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	list, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 || list[0].CreatedAt != 2 {
		t.Errorf("expected 2 entries newest-first, got %+v", list)
	}
	// Empty entries are not stored.
	if err := s.Add(Entry{Input: "  ", Output: ""}); err != nil {
		t.Fatalf("Add empty: %v", err)
	}
	if list, _ := s.List(); len(list) != 2 {
		t.Errorf("empty entry should not be stored, got %d", len(list))
	}
	if err := s.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if list, _ := s.List(); len(list) != 0 {
		t.Errorf("expected empty after clear, got %d", len(list))
	}
}
