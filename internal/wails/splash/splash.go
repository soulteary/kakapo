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
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// defaultMacWindow 返回模块预设的 Mac 窗口样式（毛玻璃、隐藏标题栏等）。
func defaultMacWindow() application.MacWindow {
	return application.MacWindow{
		InvisibleTitleBarHeight: 50,
		Backdrop:                application.MacBackdropTranslucent,
		TitleBar:                application.MacTitleBarHiddenInset,
	}
}

// Options 配置启动/欢迎窗口。
type Options struct {
	Title              string
	URL                string
	BackgroundR, G, B  uint8                  // 未设置时使用默认深色 27,38,54
	Mac                *application.MacWindow // 为 nil 时使用模块预设
	UseApplicationMenu *bool                  // 为 nil 时默认 true
	// Show 创建后是否显示：nil 或 true = 显示，false = 隐藏。默认显示。
	Show *bool
	// AutoHideAfter 显示后经过该时长自动隐藏；0 表示不自动隐藏。
	AutoHideAfter time.Duration
}

// New 创建 splash 窗口，根据 Options.Show 决定是否显示，返回窗口供后续 Hide/Show。
func New(app *application.App, opts Options) *application.WebviewWindow {
	if opts.BackgroundR == 0 && opts.G == 0 && opts.B == 0 {
		opts.BackgroundR, opts.G, opts.B = 27, 38, 54
	}

	mac := defaultMacWindow()
	if opts.Mac != nil {
		mac = *opts.Mac
	}

	show := opts.Show == nil || *opts.Show
	useAppMenu := opts.UseApplicationMenu == nil || *opts.UseApplicationMenu

	w := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:              opts.Title,
		Mac:                mac,
		UseApplicationMenu: useAppMenu,
		BackgroundColour:   application.NewRGB(opts.BackgroundR, opts.G, opts.B),
		URL:                opts.URL,
	})

	if show {
		w.Show()
	}
	if opts.AutoHideAfter > 0 {
		go func() {
			time.Sleep(opts.AutoHideAfter)
			w.Hide()
		}()
	}
	return w
}
