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
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	pkgecho "github.com/soulteary/kakapo/pkg/echo"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// 基础 Server：提供 Echo API 服务（/api/info, /api/users 等）
// User 表示系统中的用户
type User struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

// EventData 表示事件中传递的数据
type EventData struct {
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// EchoService 实现基于 Echo 的 Wails 服务，用于处理 HTTP 请求
type EchoService struct {
	pkgecho.Handler
	users  []User
	nextID int
	mu     sync.RWMutex
	app    *application.App
}

// NewEchoService 创建新的 EchoService 实例
func NewEchoService() *EchoService {
	e := pkgecho.NewEngine(pkgecho.WithLogPrefix("[ECHO]"))
	svc := &EchoService{
		Handler: pkgecho.Handler{Echo: e},
		users: []User{
			{ID: 1, Name: "Alice", Email: "alice@example.com", CreatedAt: time.Now().Add(-72 * time.Hour)},
			{ID: 2, Name: "Bob", Email: "bob@example.com", CreatedAt: time.Now().Add(-48 * time.Hour)},
			{ID: 3, Name: "Charlie", Email: "charlie@example.com", CreatedAt: time.Now().Add(-24 * time.Hour)},
		},
		nextID: 4,
	}
	svc.setupRoutes()
	return svc
}

// ServiceName 返回服务名称
func (s *EchoService) ServiceName() string {
	return "Echo API Service"
}

// ServiceStartup 在服务启动时调用
func (s *EchoService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	s.app = application.Get()

	// 注册可由前端触发的事件处理
	s.app.Event.On("echo-api-event", func(event *application.CustomEvent) {
		s.app.Logger.Info("Received event from frontend", "data", event.Data)
		s.app.Event.Emit("echo-api-response", map[string]interface{}{
			"message": "Response from Echo API Service",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	return nil
}

// ServiceShutdown 在服务关闭时调用
func (s *EchoService) ServiceShutdown(ctx context.Context) error {
	return nil
}

// setupRoutes 配置 API 路由
func (s *EchoService) setupRoutes() {
	e := s.Handler.Echo
	e.Use(middleware.BodyLimit("1M"))
	// 基础信息接口
	e.GET("/info", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"service": "Echo API Service",
			"version": "1.0.0",
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	// 用户相关路由组
	users := e.Group("/users")
	{
		// 获取所有用户
		users.GET("", func(c echo.Context) error {
			s.mu.RLock()
			defer s.mu.RUnlock()
			return c.JSON(http.StatusOK, s.users)
		})

		// 按 ID 获取用户
		users.GET("/:id", func(c echo.Context) error {
			id, err := strconv.Atoi(c.Param("id"))
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID"})
			}
			s.mu.RLock()
			defer s.mu.RUnlock()
			for _, user := range s.users {
				if user.ID == id {
					return c.JSON(http.StatusOK, user)
				}
			}
			return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
		})

		// 创建用户
		users.POST("", func(c echo.Context) error {
			var newUser User
			if err := c.Bind(&newUser); err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
			}
			s.mu.Lock()
			defer s.mu.Unlock()
			newUser.ID = s.nextID
			newUser.CreatedAt = time.Now()
			s.nextID++
			s.users = append(s.users, newUser)
			if s.app != nil {
				s.app.Event.Emit("user-created", newUser)
			}
			return c.JSON(http.StatusCreated, newUser)
		})

		// 删除用户
		users.DELETE("/:id", func(c echo.Context) error {
			id, err := strconv.Atoi(c.Param("id"))
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid user ID"})
			}
			s.mu.Lock()
			defer s.mu.Unlock()
			for i, user := range s.users {
				if user.ID == id {
					s.users = append(s.users[:i], s.users[i+1:]...)
					return c.JSON(http.StatusOK, map[string]string{"message": "User deleted"})
				}
			}
			return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
		})
	}
}
