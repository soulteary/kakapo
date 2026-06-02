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
	"embed"
	"image/color"
	"log"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/soulteary/kakapo/internal/wails/events"
	"github.com/soulteary/kakapo/internal/wails/menu"
	"github.com/soulteary/kakapo/internal/wails/splash"
	"github.com/soulteary/kakapo/internal/wails/tray"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/services/dock"
)

//go:embed assets/icon.png
var icon []byte

//go:embed assets/icon-active.png
var iconActive []byte

//go:embed all:frontend/dist
var assets embed.FS

// EventShowSplash is emitted (e.g. from the translate web service when the UI
// requests "关于") to ask the main process to show the splash/about window.
const EventShowSplash = "show-splash"

// EventSplashAbout is emitted to the splash page so it switches from the startup
// "loading" view to the "about" view (project links + intro article).
const EventSplashAbout = "splash:about"

// EventTranslateSaved is emitted by the translate web service whenever a
// translation is persisted to history. The main process uses it to refresh the
// Dock badge with the number of results saved since the tray window was last
// opened (an "unread translations" indicator).
const EventTranslateSaved = "translate:saved"

// 默认启动一个每秒发送基于时间的事件的 goroutine。随后运行应用，
// 并记录可能发生的任何错误。
func main() {

	// 先注册事件及关联的 Services，再创建 app 时合并进 Options.Services
	events.AddEmitter(events.EventTime, time.Second, func() string {
		return time.Now().Format(time.RFC1123)
	}, events.WithServices(application.NewService(NewAppInfoService())))

	options := dock.BadgeOptions{
		BackgroundColour: color.RGBA{0, 255, 255, 255},
		FontName:         "arialb.ttf", // System font
		FontSize:         16,
		SmallFontSize:    10,
		TextColour:       color.RGBA{0, 0, 0, 255},
	}
	dockService := dock.NewWithOptions(options)

	var trayInstance *tray.Tray
	var splashWindow *application.WebviewWindow

	// unreadTranslations counts translations saved to history since the tray
	// window was last opened; mirrored onto the Dock badge. Accessed from the
	// Wails event callback goroutine and the tray show callback, so use atomic.
	var unreadTranslations atomic.Int64

	app := application.New(application.Options{
		Name:        "Kakapo",
		Description: "Translator, but smart",
		Services: append(
			events.Services(),
			application.NewService(dockService),
			application.NewServiceWithOptions(NewEchoService(), application.ServiceOptions{
				Route: "/api",
			}),
			application.NewServiceWithOptions(NewTranslateWebService(), application.ServiceOptions{
				Route: "/translate",
			}),
		),
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: false,
			ActivationPolicy: application.ActivationPolicyAccessory,
		},
		OnShutdown: func() {
			events.Stop()
			if trayInstance != nil {
				trayInstance.Destroy()
			}
		},
	})

	// 每保存一条翻译历史，递增未读计数并刷新 Dock 角标。
	app.Event.On(EventTranslateSaved, func(_ *application.CustomEvent) {
		n := unreadTranslations.Add(1)
		_ = dockService.SetBadge(strconv.FormatInt(n, 10))
	})

	trayInstance = tray.New(app, tray.Options{
		Name:            "",
		AssetsPageEntry: "/translate/index.html",
		Icon:            icon,
		IconActive:      iconActive,
		// 打开托盘窗口视为用户已查看结果：清零未读计数并移除 Dock 角标。
		OnShow: func() {
			unreadTranslations.Store(0)
			_ = dockService.RemoveBadge()
		},
	})

	// 启动时展示 splash（欢迎页），数秒后自动隐藏。
	// app 页的加载地址在开发/生产下不同：
	//   - 生产（wails build）：主资源服务器（AssetFileServerFS）会自动 fs.Sub 到
	//     包含 index.html 的最短路径目录（即 frontend/dist/app），故应以根路径 "/"
	//     加载；用 "/app/index.html" 会被解析为 frontend/dist/app/app/index.html 而 404。
	//   - 开发（wails dev）：由 vite 多页 devserver 从 src/projects 根目录提供，
	//     页面位于 "/app/index.html"。
	splashURL := "/"
	if os.Getenv("FRONTEND_DEVSERVER_URL") != "" {
		splashURL = "/app/index.html"
	}
	splashWindow = splash.New(app, splash.Options{
		Title:         "Kakapo",
		URL:           splashURL,
		AutoHideAfter: 3 * time.Second,
	})

	showSplash := func() {
		if splashWindow != nil {
			splashWindow.Show().Focus()
			// 以"关于"模式展示：通知页面切换到项目地址/介绍文章视图。
			app.Event.Emit(EventSplashAbout, nil)
		}
	}

	// 前端「关于」按钮通过 Wails 事件请求显示 splash 窗口。
	app.Event.On(EventShowSplash, func(_ *application.CustomEvent) {
		showSplash()
	})

	// 点击菜单「About」时再次显示 splash 作为关于页。
	menu.Set(app, menu.Options{
		OnAbout: func(_ *application.Context) {
			showSplash()
		},
	})

	events.Start(app)

	err := app.Run()

	if err != nil {
		log.Fatal(err)
	}
}
