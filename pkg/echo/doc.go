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

// Package echo 提供基于 Echo 的 Wails 服务能力抽象，参考 Wails v3 Gin 集成模式：
//   - 使用 Echo 作为 HTTP 处理器，实现 http.Handler 以便挂载到指定路由
//   - 通过 application.NewServiceWithOptions(service, application.ServiceOptions{Route: "/api"}) 注册
//
// 典型用法：
//
//	engine := echo.NewEngine(echo.WithLogPrefix("[ECHO]"))
//	engine.GET("/info", ...)
//	svc := &MyService{Handler: echo.Handler{Echo: engine}}
//
// 实现 ServiceName、ServiceStartup、ServiceShutdown，ServeHTTP 委托给 svc.Handler.ServeHTTP
package echo
