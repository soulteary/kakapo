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

package echo

import (
	"os"
	"strings"

	"github.com/labstack/echo/v4"
)

// EngineOption 用于配置 NewEngine 的可选参数
type EngineOption func(*engineConfig)

type engineConfig struct {
	logPrefix string
}

// WithLogPrefix 设置日志中间件的前缀，空字符串表示不添加 Logging 中间件
func WithLogPrefix(prefix string) EngineOption {
	return func(c *engineConfig) {
		c.logPrefix = prefix
	}
}

// httpLogEnabled 返回是否启用每请求访问日志中间件。
// 默认启用（便于开发观测）；将环境变量 KAKAPO_HTTP_LOG 设为
// 0/false/off/no（不区分大小写）可在生产环境关闭，避免静态资源与轮询接口刷屏。
// 该开关只影响访问日志，不影响错误/启动等业务日志。
func httpLogEnabled() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("KAKAPO_HTTP_LOG"))) {
	case "0", "false", "off", "no":
		return false
	default:
		return true
	}
}

// NewEngine 创建已配置好中间件的 Echo 实例，适用于作为 Wails Service 的 HTTP 处理器。
// 默认使用 Recovery；通过 WithLogPrefix 可添加带前缀的 Logging 中间件
// （还需 KAKAPO_HTTP_LOG 未被关闭，详见 httpLogEnabled）。
func NewEngine(opts ...EngineOption) *echo.Echo {
	cfg := &engineConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	e := echo.New()
	e.HideBanner = true
	e.Use(Recovery())
	if cfg.logPrefix != "" && httpLogEnabled() {
		e.Use(Logging(cfg.logPrefix))
	}
	return e
}
