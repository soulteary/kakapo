# 贡献指南

感谢你对 Kakapo 的关注！本文档说明如何在本地搭建环境、提交改动以及代码规范。提交前请通读一遍，能让评审更顺畅。

## 开始之前

- 请先阅读 [README.md](README.md) 了解项目定位、功能与目录结构。
- 较大的功能或破坏性改动，建议先开 Issue 讨论方案，再动手实现，避免返工。
- Bug 报告请尽量附上复现步骤、期望行为与实际行为，以及运行环境（OS 版本、Go 版本）。

## 环境要求

- Go `1.27.3`
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

Linux CI 对所有生产包执行竞态测试，并要求总体语句覆盖率至少 95%，不按包覆盖率平均值计算，也不排除桌面代码。请在安装 Linux GUI 编译依赖并准备前端资源后运行：

```bash
go test -race -count=1 -coverprofile=coverage.out -covermode=atomic ./...
awk -f .github/scripts/check-coverage.awk coverage.out
```

Linux 专用业务测试使用隔离的配置目录和可注入凭据接口；合成内存凭据只验证业务绑定与失败路径，不等同于真实 macOS Keychain 集成测试。窗口组装测试不启动原生事件循环，不应声称验证了 GUI 交互或系统托盘显示。

现有测试可作为范例：`internal/config`、`internal/history`、`internal/translate` 下的 `*_test.go`。

历史解码基准（请在相同 Linux 环境和 Go 版本下分别测量改动前后）：

```bash
go test ./internal/history -run '^$' -bench BenchmarkHistoryDecode -benchmem -count=8 -cpu=1
```

基准覆盖 50 条历史记录、每条 1 个或 6 个结果，报告耗时、吞吐、每次分配字节数与分配次数。它只测量内存中的 JSON 解码，不代表磁盘读写、网络翻译或界面端到端延迟。避免同时运行构建、依赖安装等任务；共享宿主负载波动时，不应仅凭前后两组耗时宣称加速。

历史文件仍兼容小写、旧版首字母大写及混合大小写 JSON 字段；重复字段保持 `encoding/json` 的后出现值覆盖规则。解码优化不改变文件格式，不需要迁移或额外持久卷，旧版可读取同一文件。现有存储限制仍然适用：应由单个应用实例使用配置目录，多实例之间没有文件锁；本次解码优化没有增加跨进程写入保障。


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
- 上游错误正文、异常成功响应正文和底层网络错误可能包含凭据、URL 查询参数或输入文本，不得透传到诊断或历史错误字段。使用已知错误类型与 HTTP 状态排障；`APIError.Detail` 仅为源码兼容保留，新错误应保持为空。
- 隐私回归请使用合成数据与本地 HTTP 服务，不要使用真实密钥或敏感正文。升级对旧记录、日志、备份和导出副本的处理要求见 [README.md](README.md#错误诊断与升级隐私要求)。

## 贡献授权

- 本项目以 [Apache License 2.0](LICENSE) 开源。除非你另行明确声明，你提交的贡献将默认以同一许可证授权并纳入本项目（参见许可证第 5 条）。
- 请勿提交你无权授权的第三方代码；如引入第三方代码或资源，需确认其许可证兼容，并在 [NOTICE](NOTICE) 中补充相应署名。

## 行为准则

请保持友善、专业与建设性的沟通。欢迎提问与讨论，我们乐于帮助新贡献者上手。

## 凭据一致性回归

`credential_updates_linux_test.go` 使用合成凭据后端与真实配置文件，覆盖提交失败、丢失确认、并发快照与显式恢复；`credential_crash_linux_test.go` 使用真实私有文件作为凭据服务替身，并通过 SIGKILL 和新进程验证待恢复状态。它们不验证原生 macOS Keychain 的授权弹窗、锁定状态或系统实现。不得将模拟后端故障写成真实 Keychain 集成结果。

修改恢复逻辑时，确认单条与并行翻译都读取待处理状态，写入凭据前标记已持久化，且无论失败还是确认丢失，都不会通过更换 Provider ID、删除标记或忽略删除错误恢复请求。前端新建身份重试测试：`bun test src/projects/translate/provider-id.test.js`（在 `frontend` 目录执行）。
