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
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/soulteary/kakapo/internal/speech"
	"github.com/soulteary/kakapo/internal/translate"
	pkgecho "github.com/soulteary/kakapo/pkg/echo"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// translateAssets 嵌入翻译页所需资源（/translate 页面及静态文件）
//
//go:embed all:frontend/dist/translate
var translateAssets embed.FS

// TranslateWebService 提供 /translate 页面及 REST 翻译 API
type TranslateWebService struct {
	pkgecho.Handler
	translateApp *TranslateApp
	assetsFS     fs.FS
	assetPrefix  string // 当 fs.Sub 失败时用完整路径读文件，如 "frontend/dist/translate/"
}

// NewTranslateWebService 创建翻译 Web 服务实例
func NewTranslateWebService() *TranslateWebService {
	e := pkgecho.NewEngine(pkgecho.WithLogPrefix("[TRANSLATE]"))
	// embed 为 all:frontend/dist/translate 时，FS 内路径为 frontend/dist/translate/...
	// 依次尝试两种常见路径，避免因目录未构建或路径差异导致 panic
	var assetsFS fs.FS
	var prefix string
	for _, subPath := range []string{"frontend/dist/translate", "translate"} {
		sub, err := fs.Sub(translateAssets, subPath)
		if err == nil {
			assetsFS = sub
			prefix = ""
			break
		}
	}
	if assetsFS == nil {
		assetsFS = translateAssets
		prefix = "frontend/dist/translate/"
	}
	svc := &TranslateWebService{Handler: pkgecho.Handler{Echo: e}, assetsFS: assetsFS, assetPrefix: prefix}
	svc.setupTranslateRoutes()
	return svc
}

// ServiceName 返回服务名称
func (s *TranslateWebService) ServiceName() string {
	return "Translate Web Service"
}

// ServiceStartup 在服务启动时调用，初始化 TranslateApp
func (s *TranslateWebService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	s.translateApp = NewTranslateApp()
	s.translateApp.startup(ctx)
	return nil
}

// ServiceShutdown 在服务关闭时调用
func (s *TranslateWebService) ServiceShutdown(context.Context) error {
	return nil
}

// setupTranslateRoutes 配置 /translate 页面与 /translate/api/* 翻译接口
func (s *TranslateWebService) setupTranslateRoutes() {
	e := s.Handler.Echo
	e.Use(middleware.BodyLimit("1M"))
	readAsset := func(name string) ([]byte, error) { return fs.ReadFile(s.assetsFS, s.assetPrefix+name) }
	// 首页：翻译页 HTML
	e.GET("/", func(c echo.Context) error {
		data, err := readAsset("index.html")
		if err != nil {
			return c.NoContent(http.StatusNotFound)
		}
		return c.HTMLBlob(http.StatusOK, data)
	})
	e.GET("/index.html", func(c echo.Context) error {
		data, err := readAsset("index.html")
		if err != nil {
			return c.NoContent(http.StatusNotFound)
		}
		return c.HTMLBlob(http.StatusOK, data)
	})

	// 静态资源
	e.GET("/style.css", func(c echo.Context) error {
		data, err := readAsset("style.css")
		if err != nil {
			return c.NoContent(http.StatusNotFound)
		}
		return c.Blob(http.StatusOK, "text/css; charset=utf-8", data)
	})
	e.GET("/main.js", func(c echo.Context) error {
		data, err := readAsset("main.js")
		if err != nil {
			return c.NoContent(http.StatusNotFound)
		}
		return c.Blob(http.StatusOK, "application/javascript; charset=utf-8", data)
	})
	e.GET("/translate-api.js", func(c echo.Context) error {
		data, err := readAsset("translate-api.js")
		if err != nil {
			return c.NoContent(http.StatusNotFound)
		}
		return c.Blob(http.StatusOK, "application/javascript; charset=utf-8", data)
	})

	// 构建产出的 assets/*（index-xxx.js / index-xxx.css 等）
	e.GET("/assets/*", func(c echo.Context) error {
		name := c.Param("*")
		if name == "" {
			return c.NoContent(http.StatusNotFound)
		}
		data, err := readAsset("assets/" + name)
		if err != nil {
			return c.NoContent(http.StatusNotFound)
		}
		ctype := mime.TypeByExtension(filepath.Ext(name))
		switch filepath.Ext(name) {
		case ".js", ".mjs":
			ctype = "application/javascript; charset=utf-8"
		case ".css":
			ctype = "text/css; charset=utf-8"
		}
		if ctype == "" {
			ctype = "application/octet-stream"
		}
		return c.Blob(http.StatusOK, ctype, data)
	})

	// 根目录静态文件（如 logo，来自 public/ 复制到 dist/translate/）
	e.GET("/kakapo.png", func(c echo.Context) error {
		data, err := readAsset("kakapo.png")
		if err != nil {
			return c.NoContent(http.StatusNotFound)
		}
		return c.Blob(http.StatusOK, "image/png", data)
	})

	e.POST("/api/translate", func(c echo.Context) error {
		if s.translateApp == nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "service not ready"})
		}
		var body struct {
			Text            string   `json:"text"`
			SourceLanguage  string   `json:"sourceLanguage"`
			TargetLanguages []string `json:"targetLanguages"`
			Models          []string `json:"models"`
		}
		if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid JSON: " + err.Error()})
		}
		// Legacy: only "text" (or no targetLanguages/models) -> single model, zh->en, return TranslationResult
		useLegacy := body.Text != "" && len(body.TargetLanguages) == 0 && len(body.Models) == 0
		if useLegacy {
			result, err := s.translateApp.TranslateZhToEn(body.Text)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": translate.ErrorMessage(err)})
			}
			return c.JSON(http.StatusOK, result)
		}
		if body.Text == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "请输入要翻译的内容"})
		}
		// Extended: parallel multi-model, multi-target
		result, err := s.translateApp.TranslateParallel(body.Text, body.SourceLanguage, body.TargetLanguages, body.Models)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": translate.ErrorMessage(err)})
		}
		return c.JSON(http.StatusOK, result)
	})

	e.GET("/api/settings", func(c echo.Context) error {
		if s.translateApp == nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "service not ready"})
		}
		settings, err := s.translateApp.GetSettings()
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, settings)
	})

	e.PUT("/api/settings", func(c echo.Context) error {
		if s.translateApp == nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "service not ready"})
		}
		var dto SettingsDTO
		if err := json.NewDecoder(c.Request().Body).Decode(&dto); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid JSON: " + err.Error()})
		}
		if err := s.translateApp.SaveSettings(&dto); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		return c.NoContent(http.StatusOK)
	})

	e.GET("/api/history", func(c echo.Context) error {
		if s.translateApp == nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "service not ready"})
		}
		query := c.QueryParam("q")
		entries, err := s.translateApp.GetHistory(query)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		if entries == nil {
			entries = []HistoryEntry{}
		}
		return c.JSON(http.StatusOK, entries)
	})

	e.POST("/api/history", func(c echo.Context) error {
		if s.translateApp == nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "service not ready"})
		}
		var body HistoryPostBody
		if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
			log.Printf("[TRANSLATE] POST /api/history Decode error: %v", err)
			return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}
		if err := s.translateApp.AddHistory(&body); err != nil {
			log.Printf("[TRANSLATE] POST /api/history AddHistory error: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		// 通知主进程刷新 Dock 角标（未读翻译计数）。
		if app := application.Get(); app != nil {
			app.Event.Emit(EventTranslateSaved, nil)
		}
		return c.NoContent(http.StatusOK)
	})

	// 显示 splash/关于窗口：通过 Wails 事件通知主进程（窗口在 main.go 中持有）。
	e.POST("/api/splash", func(c echo.Context) error {
		if app := application.Get(); app != nil {
			app.Event.Emit(EventShowSplash, nil)
			return c.NoContent(http.StatusNoContent)
		}
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "application not ready"})
	})

	e.DELETE("/api/history", func(c echo.Context) error {
		if s.translateApp == nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "service not ready"})
		}
		if err := s.translateApp.ClearHistory(); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.NoContent(http.StatusOK)
	})

	// 朗读：使用 macOS say，按 language 选语音；非 macOS 返回 501
	const maxSpeakChars = 5000
	e.POST("/api/speak", func(c echo.Context) error {
		var body struct {
			Text     string `json:"text"`
			Language string `json:"language"`
		}
		if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid JSON: " + err.Error()})
		}
		text := body.Text
		if text == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "请输入要朗读的内容"})
		}
		if len([]rune(text)) > maxSpeakChars {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "朗读内容过长，请缩短后重试"})
		}
		err := speech.Speak(c.Request().Context(), text, body.Language)
		if err != nil {
			if err == speech.ErrUnsupported {
				return c.JSON(http.StatusNotImplemented, map[string]string{"error": "当前系统不支持朗读（仅支持 macOS）"})
			}
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return c.NoContent(http.StatusNoContent)
	})
}
