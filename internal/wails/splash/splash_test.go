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

package splash

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"testing"
)

func TestSplashConstruction(t *testing.T) {
	app := application.New(application.Options{Name: "Splash unit test"})
	defaults := defaultMacWindow()
	if defaults.InvisibleTitleBarHeight != 50 || defaults.Backdrop != application.MacBackdropTranslucent || defaults.TitleBar != application.MacTitleBarHiddenInset {
		t.Fatal("default macOS splash style changed")
	}
	first := New(app, Options{Title: "First", URL: "/"})
	if first == nil || first.Name() == "" {
		t.Fatal("default splash not registered")
	}
	no := false
	custom := application.MacWindow{}
	second := New(app, Options{Title: "Second", URL: "/custom", BackgroundR: 1, G: 2, B: 3, Mac: &custom, Show: &no, UseApplicationMenu: &no})
	if second == nil || second.Name() == first.Name() {
		t.Fatal("independent splash not registered")
	}
}
