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

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestHistoryInitializationFailureIsReportedAndRetried(t *testing.T) {
	isolatedConfig(t)
	blocked := filepath.Join(t.TempDir(), "blocked")
	if err := os.WriteFile(blocked, nil, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CONFIG_HOME", blocked)
	app := NewTranslateApp()
	app.startup(context.Background())
	if _, err := app.GetHistory(""); err == nil {
		t.Error("unavailable history was reported as empty")
	}
	if err := app.ClearHistory(); err == nil {
		t.Error("unavailable history was reported as cleared")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := app.AddHistory(&HistoryPostBody{Input: "recovered", Output: "saved"}); err != nil {
		t.Errorf("initialization did not retry after storage recovered: %v", err)
		return
	}
	entries, err := app.GetHistory("")
	if err != nil || len(entries) != 1 || entries[0].Output != "saved" {
		t.Fatalf("recovered history: %+v %v", entries, err)
	}
}
