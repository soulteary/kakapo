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
	"net/http"

	"github.com/labstack/echo/v4"
)

// Handler 将 *echo.Echo 包装为 http.Handler，便于作为 Wails Service 挂载到指定 Route。
// 用法：在实现 application.Service 的结构体中嵌入 Handler，并在 ServeHTTP 中调用 h.ServeHTTP(w, r)。
type Handler struct {
	Echo *echo.Echo
}

// ServeHTTP 实现 http.Handler，将请求委托给内嵌的 Echo 实例
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.Echo != nil {
		h.Echo.ServeHTTP(w, r)
	}
}
