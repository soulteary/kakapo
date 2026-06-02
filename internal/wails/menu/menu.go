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
)

// Options 用于配置应用菜单，可选回调由调用方提供。
type Options struct {
	OnAbout    func(*application.Context) // 点击 About 时调用，可为 nil
	OnSettings func(*application.Context) // 点击 Settings 时调用，可为 nil
}

// Set 为 app 创建并设置应用菜单，Quit 固定为 app.Quit()。
// 加入 Edit 菜单角色以启用系统复制/粘贴/全选快捷键（Cmd+C、Cmd+V、Cmd+A 等），
// 否则在无框窗口的 WebView 中这些快捷键可能不生效。
func Set(app *application.App, opts Options) {
	m := app.NewMenu()
	appMenu := m.AddSubmenu("App")

	appMenu.Add("About").OnClick(func(ctx *application.Context) {
		if opts.OnAbout != nil {
			opts.OnAbout(ctx)
		}
	})
	appMenu.Add("Settings").OnClick(func(ctx *application.Context) {
		if opts.OnSettings != nil {
			opts.OnSettings(ctx)
		}
	})
	appMenu.AddSeparator()
	appMenu.Add("Quit").OnClick(func(ctx *application.Context) {
		app.Quit()
	})

	// 启用系统编辑快捷键（复制、粘贴、剪切、全选等），解决无框窗口内 Cmd+C/Cmd+V 失效问题
	m.AddRole(application.EditMenu)

	app.Menu.Set(m)
}
