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
	"log"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Recovery 返回 Echo 的 panic 恢复中间件
func Recovery() echo.MiddlewareFunc {
	return middleware.Recover()
}

// Logging 返回带前缀的请求日志中间件，prefix 用于区分不同服务（如 "[ECHO]"、"[TRANSLATE]"）
func Logging(prefix string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			err := next(c)
			latency := time.Since(start)
			log.Printf("%s %s %s %d %s", prefix, c.Request().Method, c.Request().URL.Path, c.Response().Status, latency)
			return err
		}
	}
}
