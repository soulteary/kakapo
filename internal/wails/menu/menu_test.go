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

package menu

import (
	"github.com/wailsapp/wails/v3/pkg/application"
	"testing"
)

func TestApplicationMenuRegistration(t *testing.T) {
	app := application.New(application.Options{Name: "Menu unit test"})
	Set(app, Options{})
	menu := app.Menu.GetApplicationMenu()
	if menu == nil {
		t.Fatal("application menu missing")
	}
	for _, label := range []string{"App", "About", "Settings", "Quit"} {
		if menu.FindByLabel(label) == nil {
			t.Fatalf("menu item %q missing", label)
		}
	}
}
