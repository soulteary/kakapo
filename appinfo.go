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
	"runtime"
	"time"
)

// 以下构建元数据可在构建时通过 -ldflags "-X main.AppVersion=..." 等方式注入，
// 由 Taskfile 的构建任务自动填充（版本号、Git commit、构建时间）。
// 默认值用于 `go run`/`go build` 等未注入的场景。
var (
	// AppVersion 为应用的语义化版本号，发布时通过构建参数注入。
	AppVersion = "dev"
	// AppCommit 为构建所基于的 Git 提交短哈希。
	AppCommit = "unknown"
	// AppBuildTime 为构建时间（UTC，RFC3339）。
	AppBuildTime = "unknown"
)

// AppInfo 描述应用的运行/构建元数据，通过 AppInfoService 暴露给前端
// （例如「关于」窗口展示版本、平台等信息）。
type AppInfo struct {
	Name      string `json:"name"`
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"buildTime"`
	GoVersion string `json:"goVersion"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	StartedAt int64  `json:"startedAt"`
}

// AppInfoService 是一个 Wails Service，用于向前端提供应用级元数据。
// 它取代了脚手架自带的 GreetService，并作为后续应用级（非翻译）能力的扩展入口。
type AppInfoService struct {
	startedAt time.Time
}

// NewAppInfoService 创建 AppInfoService，并记录进程启动时间。
func NewAppInfoService() *AppInfoService {
	return &AppInfoService{startedAt: time.Now()}
}

// Info 返回当前应用元数据，供前端 UI 展示。
func (s *AppInfoService) Info() AppInfo {
	return AppInfo{
		Name:      "Kakapo",
		Version:   AppVersion,
		Commit:    AppCommit,
		BuildTime: AppBuildTime,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
		StartedAt: s.startedAt.Unix(),
	}
}

// UptimeSeconds 返回应用已运行的秒数，便于前端展示运行时长等信息。
func (s *AppInfoService) UptimeSeconds() int64 {
	return int64(time.Since(s.startedAt).Seconds())
}
