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

package tray

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"testing"
)

func TestTrayConstructionAndShowCallback(t *testing.T) {
	app := application.New(application.Options{Name: "Tray unit test"})
	calls := 0
	tui := New(app, Options{Name: "Synthetic tray", AssetsPageEntry: "/translate/index.html", Icon: []byte("normal"), IconActive: []byte("active"), OnShow: func() { calls++ }})
	if tui.window == nil || tui.systray == nil || tui.isActive {
		t.Fatal("invalid initial tray state")
	}
	tui.toggleWindow()()
	if calls != 1 || !tui.isActive {
		t.Fatalf("show callback=%d active=%v", calls, tui.isActive)
	}
	tui.toggleActive()
	if tui.isActive {
		t.Fatal("active state not reset")
	}
	tui.Destroy()
	(&Tray{}).Destroy()
}
