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

package events

import (
	"context"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type emitterEntry struct {
	name     string
	interval time.Duration
	fn       func() string
}

// EmitterOption 为 AddEmitter 的可选参数。
type EmitterOption func(*emitterOpts)

type emitterOpts struct {
	services []application.Service
}

// WithServices 在创建该事件时一并注册的 Services，由调用方在 application.New 前通过 Services() 取回并注入 app。
func WithServices(s ...application.Service) EmitterOption {
	return func(o *emitterOpts) {
		o.services = append(o.services, s...)
	}
}

var (
	emitters []emitterEntry
	services []application.Service
	mu       sync.Mutex

	// cancel stops all running emitter goroutines started by Start; set on
	// Start and invoked by Stop.
	cancel context.CancelFunc
	// wg tracks running emitter goroutines so Stop can wait for them to exit.
	wg sync.WaitGroup
)

// 事件名常量，供 binding 生成器发现；新增事件时在此添加常量并在 init 中注册。
const (
	EventTime = "time"
)

func init() {
	application.RegisterEvent[string](EventTime)
}

// AddEmitter 动态注册一个 string 类型事件：绑定事件名、发送间隔与生成函数，并可选的绑定若干 Service。
// 生成函数 fn 在每次发送时被调用，返回值通过 app.Event.Emit(name, v) 发给前端。
// 通过 WithServices(...) 传入的 Service 会加入模块维护的列表，需在 application.New 时用 Services() 取回并传入 Options.Services。
// 需在调用 Start 之后才会开始按 interval 周期发送。
// 事件名应使用本包定义的常量（如 EventTime），以便 binding 生成器发现。
func AddEmitter(name string, interval time.Duration, fn func() string, opts ...EmitterOption) {
	mu.Lock()
	defer mu.Unlock()
	for _, opt := range opts {
		var o emitterOpts
		opt(&o)
		services = append(services, o.services...)
	}
	emitters = append(emitters, emitterEntry{name: name, interval: interval, fn: fn})
}

// AddService 将单个 Service 加入模块列表，用于与事件一起由 main 在 application.New 时注入 app。
func AddService(s application.Service) {
	mu.Lock()
	defer mu.Unlock()
	services = append(services, s)
}

// Services 返回当前已通过 AddEmitter(..., WithServices(...)) 与 AddService 注册的所有 Service，供 application.New(Options{Services: append(events.Services(), ...) }) 使用。
func Services() []application.Service {
	mu.Lock()
	defer mu.Unlock()
	out := make([]application.Service, len(services))
	copy(out, services)
	return out
}

// Start 为所有已通过 AddEmitter 注册的事件启动后台 goroutine，向 app 周期发送事件。
// 通过返回的 context 取消（见 Stop）可在应用关闭时优雅停止所有 emitter。
func Start(app *application.App) {
	mu.Lock()
	list := make([]emitterEntry, len(emitters))
	copy(list, emitters)
	ctx, c := context.WithCancel(context.Background())
	cancel = c
	mu.Unlock()

	for _, e := range list {
		wg.Add(1)
		go runEmitter(ctx, app, e)
	}
}

// Stop 取消所有由 Start 启动的 emitter goroutine 并等待其退出。可安全多次调用。
// 建议在 application.Options.OnShutdown 中调用。
func Stop() {
	mu.Lock()
	c := cancel
	cancel = nil
	mu.Unlock()
	if c != nil {
		c()
	}
	wg.Wait()
}

func runEmitter(ctx context.Context, app *application.App, e emitterEntry) {
	defer wg.Done()
	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()
	// 立即发送一次，随后按 interval 周期发送，直到 ctx 取消。
	app.Event.Emit(e.name, e.fn())
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			app.Event.Emit(e.name, e.fn())
		}
	}
}
