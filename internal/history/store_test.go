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
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
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

func TestEntryDecodeCompatibility(t *testing.T) {
	want := Entry{Input: "你好", Output: "hello", CreatedAt: 12, SourceLanguage: "zh", TargetLanguages: []string{"en"}, Models: []string{"m"}, LatencyMs: 7, Results: []ResultItem{{Provider: "p", Model: "m", TargetLanguage: "en", Output: "hello", LatencyMs: 7, Error: "failure"}}}
	for _, data := range []string{
		`{"input":"你好","output":"hello","createdAt":12,"sourceLanguage":"zh","targetLanguages":["en"],"models":["m"],"latencyMs":7,"results":[{"provider":"p","model":"m","targetLanguage":"en","output":"hello","latencyMs":7,"error":"failure"}]}`,
		`{"Input":"你好","Output":"hello","CreatedAt":12,"SourceLanguage":"zh","TargetLanguages":["en"],"Models":["m"],"LatencyMs":7,"Results":[{"Provider":"p","Model":"m","TargetLanguage":"en","Output":"hello","LatencyMs":7,"Error":"failure"}]}`,
		`{"INPUT":"你好","Output":"hello","createdAt":12,"SourceLanguage":"zh","targetLanguages":["en"],"Models":["m"],"LATENCYMS":7,"Results":[{"PROVIDER":"p","Model":"m","targetLanguage":"en","Output":"hello","LatencyMs":7,"ERROR":"failure"}],"unknown":true}`,
	} {
		var got Entry
		if err := json.Unmarshal([]byte(data), &got); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	}
}

func TestDecodeReplacementAndFailure(t *testing.T) {
	entry := Entry{Input: "old", Results: []ResultItem{{Model: "old"}}}
	if err := json.Unmarshal([]byte(`{"output":"new"}`), &entry); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(entry, Entry{Output: "new"}) {
		t.Fatalf("stale fields: %+v", entry)
	}
	if err := json.Unmarshal([]byte(`{"input":"changed","createdAt":"invalid"}`), &entry); err == nil {
		t.Fatal("expected type error")
	}
	if !reflect.DeepEqual(entry, Entry{Output: "new"}) {
		t.Fatalf("modified after error: %+v", entry)
	}
	result := ResultItem{Model: "old"}
	if err := json.Unmarshal([]byte(`{"Output":"new"}`), &result); err != nil {
		t.Fatal(err)
	}
	if result != (ResultItem{Output: "new"}) {
		t.Fatalf("stale result fields: %+v", result)
	}
	if err := json.Unmarshal([]byte(`{"Model":"changed","latencyMs":false}`), &result); err == nil {
		t.Fatal("expected result type error")
	}
	if result != (ResultItem{Output: "new"}) {
		t.Fatalf("result modified after error: %+v", result)
	}
}

func TestDecodeDuplicateKeysAndNull(t *testing.T) {
	for _, data := range []string{`{"Input":"legacy","input":""}`, `{"input":"legacy","Input":""}`} {
		var got Entry
		if err := json.Unmarshal([]byte(data), &got); err != nil {
			t.Fatal(err)
		}
		if got.Input != "" {
			t.Fatalf("last key must win: %+v", got)
		}
	}
	e := Entry{Input: "old"}
	if err := json.Unmarshal([]byte(`null`), &e); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(e, Entry{}) {
		t.Fatalf("null must reset entry: %+v", e)
	}
	r := ResultItem{Model: "old"}
	if err := json.Unmarshal([]byte(`null`), &r); err != nil {
		t.Fatal(err)
	}
	if r != (ResultItem{}) {
		t.Fatalf("null must reset result: %+v", r)
	}
}

func TestStoreLimitsSearchAndConcurrentAdds(t *testing.T) {
	s := &Store{path: filepath.Join(t.TempDir(), "history.json")}
	var wg sync.WaitGroup
	for i := 0; i < maxEntries+10; i++ {
		wg.Go(func() {
			if err := s.Add(Entry{Input: fmt.Sprintf("input-%d", i), Output: "HELLO", CreatedAt: int64(i)}); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	list, err := s.List()
	if err != nil || len(list) != maxEntries {
		t.Fatalf("list count=%d err=%v", len(list), err)
	}
	for i := 1; i < len(list); i++ {
		if list[i-1].CreatedAt < list[i].CreatedAt {
			t.Fatal("unsorted history")
		}
	}
	for query, want := range map[string]int{"  ": maxEntries, "hello": maxEntries, "INPUT-": maxEntries, "absent": 0} {
		got, err := s.Search(query)
		if err != nil || len(got) != want {
			t.Fatalf("search %q count=%d want=%d err=%v", query, len(got), want, err)
		}
	}
	if s.Path() != s.path {
		t.Fatal("wrong history path")
	}
	info, err := os.Stat(s.path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("history permissions: %v %v", info, err)
	}
}

func TestStoreReadAndWriteFailures(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	s := &Store{path: path}
	for _, content := range []string{"", " \n", "null", `[{}]`} {
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		got, err := s.List()
		if err != nil || len(got) != 0 {
			t.Fatalf("empty history: %v %v", got, err)
		}
	}
	for _, content := range []string{"broken", `[{"input":"valid","createdAt":false}]`} {
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := s.List(); err == nil {
			t.Fatal("expected decode error")
		}
		if _, err := s.Search("x"); err == nil {
			t.Fatal("search hid decode error")
		}
		if err := s.Add(Entry{Input: "new"}); err == nil {
			t.Fatal("append overwrote corrupt history")
		}
		got, err := os.ReadFile(path)
		if err != nil || string(got) != content {
			t.Fatal("corrupt history changed")
		}
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := s.List(); err == nil {
		t.Fatal("expected read error for directory")
	}
	s.path = filepath.Join(path, "missing", "history.json")
	if err := s.Add(Entry{Input: "new"}); err == nil {
		t.Fatal("expected missing parent write error")
	}
}
