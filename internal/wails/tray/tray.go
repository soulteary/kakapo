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
	"github.com/wailsapp/wails/v3/pkg/events"
)

type Options struct {
	Name            string
	AssetsPageEntry string
	Icon            []byte
	IconActive      []byte
	// OnShow is invoked whenever the tray window becomes visible (optional).
	// Useful for "mark as seen" side effects such as clearing a Dock badge.
	OnShow func()
}

type Tray struct {
	opts     Options
	app      *application.App
	systray  *application.SystemTray
	window   *application.WebviewWindow
	isActive bool
}

func New(app *application.App, opts Options) *Tray {
	t := &Tray{
		opts: opts,
		app:  app,
	}

	t.systray = app.SystemTray.New()
	t.systray.SetIcon(opts.Icon)
	t.systray.SetLabel(opts.Name)

	t.window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:           opts.Name,
		Width:           600,
		Height:          450,
		Hidden:          true,
		Frameless:       true,
		AlwaysOnTop:     true,
		BackgroundType:  application.BackgroundTypeTransparent,
		URL:             opts.AssetsPageEntry,
		HideOnFocusLost: true, // 点击窗口外自动隐藏
		HideOnEscape:    true, // 按 Escape 键也可隐藏
	})

	// 窗口被隐藏时（含点击外部、按 Escape）同步托盘图标为非激活状态
	t.window.OnWindowEvent(events.Common.WindowHide, func(_ *application.WindowEvent) {
		t.isActive = false
		t.updateTray()
	})

	t.systray.OnClick(t.toggleWindow())
	t.systray.OnRightClick(t.toggleWindow())
	t.systray.OnDoubleClick(t.toggleWindow())

	return t
}

func (t *Tray) Destroy() {
	if t.systray != nil {
		t.systray.Destroy()
	}
}

func (t *Tray) toggleWindow() func() {
	return func() {
		if t.window.IsVisible() {
			t.window.Hide()
			t.toggleActive()
		} else {
			t.systray.PositionWindow(t.window, 2)
			t.window.Show().Focus()
			t.toggleActive()
			if t.opts.OnShow != nil {
				t.opts.OnShow()
			}
		}
	}
}

func (t *Tray) toggleActive() {
	t.isActive = !t.isActive
	t.updateTray()
}

func (t *Tray) updateTray() {
	if t.isActive {
		t.systray.SetIcon(t.opts.IconActive)
	} else {
		t.systray.SetIcon(t.opts.Icon)
	}
}
