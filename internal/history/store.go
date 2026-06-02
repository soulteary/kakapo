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
	"bytes"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const (
	historyFilename = "history.json"
	maxEntries      = 50
)

// Entry is a single history record.
type ResultItem struct {
	Provider       string `json:"provider,omitempty"`
	Model          string `json:"model"`
	TargetLanguage string `json:"targetLanguage"`
	Output         string `json:"output"`
	LatencyMs      int64  `json:"latencyMs"`
	Error          string `json:"error,omitempty"`
}

type Entry struct {
	Input           string       `json:"input"`
	Output          string       `json:"output"`
	CreatedAt       int64        `json:"createdAt"`
	SourceLanguage  string       `json:"sourceLanguage,omitempty"`
	TargetLanguages []string     `json:"targetLanguages,omitempty"`
	Models          []string     `json:"models,omitempty"`
	LatencyMs       int64        `json:"latencyMs,omitempty"`
	Results         []ResultItem `json:"results,omitempty"`
}

// UnmarshalJSON decodes an Entry from the canonical lowercase keys, falling back
// to capitalized keys for backward compatibility with older history files that
// were written using Go's default (exported field) names.
func (e *Entry) UnmarshalJSON(data []byte) error {
	type alias Entry // alias drops the custom method to avoid recursion
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*e = Entry(a)

	var legacy struct {
		Input           string       `json:"Input"`
		Output          string       `json:"Output"`
		CreatedAt       int64        `json:"CreatedAt"`
		SourceLanguage  string       `json:"SourceLanguage"`
		TargetLanguages []string     `json:"TargetLanguages"`
		Models          []string     `json:"Models"`
		LatencyMs       int64        `json:"LatencyMs"`
		Results         []ResultItem `json:"Results"`
	}
	if json.Unmarshal(data, &legacy) == nil {
		if e.Input == "" {
			e.Input = legacy.Input
		}
		if e.Output == "" {
			e.Output = legacy.Output
		}
		if e.CreatedAt == 0 {
			e.CreatedAt = legacy.CreatedAt
		}
		if e.SourceLanguage == "" {
			e.SourceLanguage = legacy.SourceLanguage
		}
		if len(e.TargetLanguages) == 0 {
			e.TargetLanguages = legacy.TargetLanguages
		}
		if len(e.Models) == 0 {
			e.Models = legacy.Models
		}
		if e.LatencyMs == 0 {
			e.LatencyMs = legacy.LatencyMs
		}
		if len(e.Results) == 0 {
			e.Results = legacy.Results
		}
	}
	return nil
}

// UnmarshalJSON decodes a ResultItem, accepting both lowercase and capitalized
// keys (older files used capitalized names).
func (r *ResultItem) UnmarshalJSON(data []byte) error {
	type alias ResultItem
	var a alias
	if err := json.Unmarshal(data, &a); err != nil {
		return err
	}
	*r = ResultItem(a)

	var legacy struct {
		Provider       string `json:"Provider"`
		Model          string `json:"Model"`
		TargetLanguage string `json:"TargetLanguage"`
		Output         string `json:"Output"`
		LatencyMs      int64  `json:"LatencyMs"`
		Error          string `json:"Error"`
	}
	if json.Unmarshal(data, &legacy) == nil {
		if r.Provider == "" {
			r.Provider = legacy.Provider
		}
		if r.Model == "" {
			r.Model = legacy.Model
		}
		if r.TargetLanguage == "" {
			r.TargetLanguage = legacy.TargetLanguage
		}
		if r.Output == "" {
			r.Output = legacy.Output
		}
		if r.LatencyMs == 0 {
			r.LatencyMs = legacy.LatencyMs
		}
		if r.Error == "" {
			r.Error = legacy.Error
		}
	}
	return nil
}

// Store persists and retrieves history (file in user config dir).
type Store struct {
	mu   sync.Mutex
	path string
}

// NewStore returns a store using the Kakapo config directory.
func NewStore() (*Store, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	appDir := filepath.Join(dir, "Kakapo")
	if err := os.MkdirAll(appDir, 0700); err != nil {
		return nil, err
	}
	return &Store{path: filepath.Join(appDir, historyFilename)}, nil
}

// Path returns the absolute path of the history file (for logging/debug).
func (s *Store) Path() string {
	return s.path
}

func (s *Store) load() ([]Entry, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		_ = os.Remove(s.path)
		return nil, nil
	}
	// Entry/ResultItem implement UnmarshalJSON to accept both lowercase and
	// capitalized keys, so a single decode covers current and legacy files.
	var list []Entry
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	// Drop stale/malformed entries (no input and no output).
	out := list[:0]
	for _, e := range list {
		if strings.TrimSpace(e.Input) != "" || strings.TrimSpace(e.Output) != "" {
			out = append(out, e)
		}
	}
	if len(out) == 0 {
		_ = os.Remove(s.path)
		return nil, nil
	}
	return out, nil
}

// save persists entries. If list is nil or empty, the history file is removed so that
// "file exists" means "has records". Never writes "null", only removes the file.
func (s *Store) save(list []Entry) error {
	if len(list) == 0 {
		_ = os.Remove(s.path)
		return nil
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(s.path, data, 0600); err != nil {
		log.Printf("[Kakapo] 历史写入失败 %s: %v", s.path, err)
		return err
	}
	return nil
}

// Add appends an entry and keeps at most maxEntries (newest first).
func (s *Store) Add(e Entry) error {
	if strings.TrimSpace(e.Input) == "" && strings.TrimSpace(e.Output) == "" {
		return nil // 不保存空记录，避免反复写入又清空
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	list, err := s.load()
	if err != nil {
		return err
	}
	list = append([]Entry{e}, list...)
	if len(list) > maxEntries {
		list = list[:maxEntries]
	}
	return s.save(list)
}

// List returns all entries, newest first.
func (s *Store) List() ([]Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	list, err := s.load()
	if err != nil {
		return nil, err
	}
	sort.Slice(list, func(i, j int) bool { return list[i].CreatedAt > list[j].CreatedAt })
	return list, nil
}

// Search returns entries whose input or output contains the query (case-insensitive).
func (s *Store) Search(query string) ([]Entry, error) {
	list, err := s.List()
	if err != nil {
		return nil, err
	}
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return list, nil
	}
	var out []Entry
	for _, e := range list {
		if strings.Contains(strings.ToLower(e.Input), query) || strings.Contains(strings.ToLower(e.Output), query) {
			out = append(out, e)
		}
	}
	return out, nil
}

// Clear removes all history.
func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.save(nil)
}
