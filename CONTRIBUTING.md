# 贡献指南

感谢你对 Kakapo 的关注！本文档说明如何在本地搭建环境、提交改动以及代码规范。提交前请通读一遍，能让评审更顺畅。

## 开始之前

- 请先阅读 [README.md](README.md) 了解项目定位、功能与目录结构。
- 较大的功能或破坏性改动，建议先开 Issue 讨论方案，再动手实现，避免返工。
- Bug 报告请尽量附上复现步骤、期望行为与实际行为，以及运行环境（OS 版本、Go 版本）。

## 环境要求

- Go `1.26`
- [Bun](https://bun.sh/)（前端依赖安装与构建）
- Wails CLI：`wails3`
- [Task](https://taskfile.dev/)：`task`
- 推荐在 macOS 开发：项目包含 Keychain、`say` 朗读与 darwin 打包流程，部分能力在其他平台会降级或不可用。

## 本地开发

```bash
# 1. 安装前端依赖
task common:install:frontend:deps

# 2. 开发模式运行（含前端 devserver 与应用启动）
task dev
```

构建与运行：

```bash
task build
task run
```

打包 macOS App：

```bash
task package
```

当修改了导出给前端的 Go 方法（如 `AppInfoService`）后，需要重新生成绑定：

```bash
task common:generate:bindings
```

当更新 `build/config.yml` 的 `info` 或 `fileAssociations` 后：

```bash
task common:update:build-assets
```

## 代码规范

### Go

- 提交前务必运行格式化与静态检查，确保无报错：

```bash
gofmt -w .
go vet ./...
go build ./...
```

- 遵循标准库与现有代码风格；导出的类型/函数请补充文档注释。
- 注释用于解释“为什么”（意图、权衡、约束），不要逐行复述代码在做什么。
- 跨 goroutine 的共享状态需用 `sync` 原语或 `sync/atomic` 保护（可参考 `internal/wails/events`、`internal/speech`）。
- 后台任务请复用可取消的生命周期（`context.Context` + `OnShutdown`），不要遗留无法退出的裸 goroutine。
- 平台相关实现使用构建标签拆分（参考 `internal/speech/speech_darwin.go` 与 `speech_stub.go`），保证非 macOS 平台可编译。

### 前端

- 多页面定义见 `frontend/scripts/multiPages.json`，新增页面时同步更新。
- 保持原生 JS + Vite 的现有风格，避免无必要地引入大型框架依赖。

### 许可证头部

- 项目以 Apache License 2.0 开源，所有 Go 与前端 JS 源文件需在顶部包含许可证头（基于 [google/addlicense](https://github.com/google/addlicense) 校验）。
- 自动生成的 `frontend/bindings`、CSS、HTML 不在校验范围内。
- 新增文件后可自动补全，并在提交前校验（CI 会执行同样的校验）：

```bash
task common:license:add     # 为缺失头部的文件补全
task common:license:check   # 校验所有目标源文件均有头部
```

## 测试

涉及后端逻辑的改动请补充或更新测试，并确保通过：

```bash
go test ./...
```

现有测试可作为范例：`internal/config`、`internal/history`、`internal/translate` 下的 `*_test.go`。

## 提交与 Pull Request

- 提交信息建议使用[约定式提交](https://www.conventionalcommits.org/zh-hans/)风格，例如：
  - `feat: 新增 Dock 角标未读计数`
  - `fix: 修复历史写入失败未提示的问题`
  - `docs: 补充贡献指南`
- 一个 PR 聚焦一件事，描述清楚动机（why）与改动点（what）。
- PR 提交前请自查：`gofmt`/`go vet`/`go build`/`go test` 均通过，且本地能正常 `task dev` 运行。
- 不要提交密钥、`settings.json` 等含敏感信息的文件；API Key 由系统 Keychain 管理，不应写入仓库。

## 安全与隐私

- API Key 仅存于系统 Keychain（以服务商 ID 为账号），切勿硬编码或提交到版本库。
- 涉及上游接口报错透传时，注意不要在日志或界面中泄露敏感信息。

## 贡献授权

- 本项目以 [Apache License 2.0](LICENSE) 开源。除非你另行明确声明，你提交的贡献将默认以同一许可证授权并纳入本项目（参见许可证第 5 条）。
- 请勿提交你无权授权的第三方代码；如引入第三方代码或资源，需确认其许可证兼容，并在 [NOTICE](NOTICE) 中补充相应署名。

## 行为准则

请保持友善、专业与建设性的沟通。欢迎提问与讨论，我们乐于帮助新贡献者上手。
