# YyslsPlayer

[![Wails v3](https://img.shields.io/badge/Wails-v3.0.0--alpha-red)](https://v3.wails.io)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)](https://go.dev)
[![React](https://img.shields.io/badge/React-19.2-61DAFB?logo=react)](https://react.dev)
[![MUI](https://img.shields.io/badge/MUI-9.0-007FFF?logo=mui)](https://mui.com)
[![TypeScript](https://img.shields.io/badge/TypeScript-strict-3178C6?logo=typescript)](https://www.typescriptlang.org)

`YyslsPlayer`（软件中文标题：`燕云流音`）是面向《燕云十六声》的 MIDI 演奏工具。项目当前目标是支持 36 键模式：导入 MIDI、生成预览、按用户配置映射到 36 个 lane，并通过后端按键模拟完成自动演奏。

当前项目由 Wails v3 + Go + React + MUI 构建，前端依赖必须使用 `pnpm` 安装，Wails bindings 由 `wails3 generate bindings` 生成。

## 功能范围

- 仅支持《燕云十六声》36 键模式。
- 36 键内部模型固定为 3 个八度、36 个连续半音 lane：`0..35`。
- 默认音域基准为 `C3..B5` / MIDI note `48..83`，通过 `baseNote` / profile 支持用户校准。
- 预览和演奏必须使用同一套 `PlayPlan`。
- 物理键位映射与音调映射分离。
- MIDI 解析、标准化、映射、调度优先放在 Go 后端。

## 环境要求

| 依赖 | 要求 | 验证命令 | 说明 |
|------|------|----------|------|
| Go | `1.25+` | `go version` | `go.mod` 当前为 `go 1.25.0`。如果命令不存在，需要安装 Go 并把 Go 安装目录和 `%USERPROFILE%\go\bin` 加入 PATH。 |
| Node.js | `20+`，建议 `24+` | `node --version` | 当前前端依赖可在 Node `24.13.0` 下安装。 |
| pnpm | `10+` | `pnpm --version` | 本项目使用 `frontend/pnpm-lock.yaml`，不要混用 npm/yarn 安装前端依赖。 |
| task | latest | `task --version` | go-task，用于 `task dev`、`task build`、`task package`。 |
| Wails CLI | v3 alpha | `wails3 doctor` | Go service 改动后需要 `wails3 generate bindings`。 |
| WebView2 Runtime | Windows 必装 | `wails3 doctor` | Windows 10 1803+ 通常已内置，旧系统从 Microsoft WebView2 页面安装。 |

安装常用命令：

```sh
npm i -g pnpm
go install github.com/go-task/task/v3/cmd/task@latest
go install github.com/wailsapp/wails/v3/cmd/wails3@latest
```

Windows 注意事项：安装 Go / task / wails3 后，确保下面路径在当前终端 PATH 中，否则会出现 `go: command not found` 或 `wails3: command not found`：

```text
Go 安装目录，例如 C:\Program Files\Go\bin
Go 用户工具目录，例如 %USERPROFILE%\go\bin
```

## 首次启动

在仓库根目录执行：

```sh
# 1. 安装前端依赖，解决 @mui/material、@mui/icons-material 等第三方包 TS2307
pnpm --dir frontend install

# 2. 整理 Go 依赖，需要 go 在 PATH 中
go mod tidy

# 3. 生成 Wails bindings，解决 @bindings/YyslsPlayer/... TS2307
wails3 generate bindings

# 4. 前端类型检查
pnpm --dir frontend typecheck

# 5. 开发启动
wails3 dev -config ./build/config.yml -port 9245
# 或
task dev
```

如果只想先安装前端依赖并验证第三方包解析：

```sh
pnpm --dir frontend install
pnpm --dir frontend typecheck
```

注意：未生成 `frontend/bindings/` 时，`typecheck` 仍会报 `@bindings/YyslsPlayer/...` 找不到。这不是 MUI 依赖问题，而是 Wails 生成文件缺失。

## 当前依赖

前端关键依赖位于 `frontend/package.json`：

| 依赖 | 当前声明 |
|------|----------|
| React | `^19.2.6` |
| React DOM | `^19.2.6` |
| MUI Material | `^9.0.1` |
| MUI Icons | `^9.0.1` |
| MUI System | `^9.0.1` |
| MUI X Charts | `^9.3.0` |
| Emotion React | `^11.14.0` |
| Emotion Styled | `^11.14.1` |
| Wails runtime | `3.0.0-alpha.79` |
| Vite | `^8.0.14` |
| TypeScript | `^6.0.3` |

`@mui/material` 已在依赖中声明；如果仍然出现 `TS2307: Cannot find module '@mui/material'`，通常是 `frontend/node_modules` 未安装或编辑器没有使用 `frontend/tsconfig.json`。

## 常用命令

| 命令 | 说明 |
|------|------|
| `pnpm --dir frontend install` | 安装前端依赖。 |
| `pnpm --dir frontend typecheck` | 前端 TypeScript 检查。 |
| `pnpm --dir frontend build` | 前端单独构建，输出到 `frontend/dist`。 |
| `go mod tidy` | 整理 Go 依赖。 |
| `go test ./...` | 运行 Go 测试。 |
| `wails3 generate bindings` | 生成 `frontend/bindings/YyslsPlayer/...`。 |
| `wails3 doctor` | 检查 Wails 本机环境。 |
| `wails3 dev -config ./build/config.yml -port 9245` | 运行 Wails 开发模式。 |
| `task dev` | 等价开发启动封装。 |
| `task build` | 当前平台生产构建。 |
| `task package` | 当前平台打包。 |

`main.go` 使用 `//go:embed all:frontend/dist`，所以单独执行 `go build .` 前必须先执行 `pnpm --dir frontend build`。

## TS2307 排障

### `Cannot find module '@mui/material'`

原因通常是前端依赖没有安装到 `frontend/node_modules`，或编辑器打开了错误的 TypeScript 项目。

处理顺序：

```sh
pnpm --dir frontend install
pnpm --dir frontend typecheck
```

如果命令行正常但编辑器仍报错：

- 确认编辑器工作区打开的是仓库根目录或 `frontend/`。
- 确认 TypeScript 使用的是项目内 `frontend/tsconfig.json`。
- 确认没有用 npm/yarn 覆盖 `pnpm-lock.yaml` 的安装结果。
- 重启 TypeScript Server。

### `Cannot find module '@bindings/YyslsPlayer/...'`

原因是 `frontend/bindings/` 是 Wails 生成目录，已被 `.gitignore` 排除，不会随仓库提交。

处理顺序：

```sh
wails3 generate bindings
pnpm --dir frontend typecheck
```

如果 `wails3 generate bindings` 无法执行，先确认：

```sh
go version
wails3 doctor
```

当前 shell 若显示 `go: command not found`，需要修复 Go 的 PATH 后再生成 bindings。

### `tsc is not recognized` 或 `tsc: command not found`

原因是前端依赖尚未安装，项目没有全局依赖 `tsc`。

处理：

```sh
pnpm --dir frontend install
pnpm --dir frontend typecheck
```

不要依赖全局 TypeScript。项目会使用 `frontend/node_modules/.bin/tsc`。

## 项目结构

```text
YyslsPlayer/
├── main.go                         # 仅 embed.FS + app.Run，不放业务逻辑
├── go.mod                          # module YyslsPlayer，Go 1.25
├── Taskfile.yml                    # 顶层 dev/build/package 入口
├── docs/                           # 需求、开发计划、验收记录等文档
├── internal/
│   ├── app/                        # 应用装配、窗口创建、服务注册
│   ├── events/                     # 类型化事件注册
│   ├── services/                   # Wails service，按业务领域拆分
│   ├── storage/                    # JSON 文件持久化
│   └── utils/                      # cryptox/logx/filex 等通用工具
├── frontend/
│   ├── package.json                # React/MUI/Vite/TypeScript 依赖
│   ├── pnpm-lock.yaml              # 前端锁文件
│   ├── tsconfig.json               # strict + @/* + @bindings/* paths
│   ├── vite.config.ts              # Vite + Wails runtime plugin
│   ├── bindings/                   # Wails 生成目录，gitignore
│   └── src/                        # React MVVM 前端
└── build/                          # Windows 构建配置与资源
```

## 数据目录

默认数据文件为 `yyslsplayer.json`：

| 平台 | 默认路径 |
|------|----------|
| Windows | `%AppData%\YyslsPlayer\yyslsplayer.json` |

应用内「设置 -> 数据存储」可修改数据库位置。

## 工程规范摘要

| 类别 | 规则 |
|------|------|
| 文档 | 需求、计划、调研、验收等文档统一放入 `docs/`。 |
| 后端入口 | `main.go` 只负责 embed 和 `app.Run(assets)`，不要追加业务逻辑。 |
| 后端业务 | MIDI、映射、播放调度、按键模拟等业务放入 `internal/services/<domain>/`。 |
| 持久化 | 统一走后端 JSON Store，前端禁止用 `localStorage` / `sessionStorage` 做业务持久化。 |
| 前端分层 | View 不写业务逻辑；ViewModel 不返回 JSX；ViewModel 不直接 import bindings。 |
| 前端调用后端 | 必须经过 `frontend/src/services/<domain>/XxxService.ts`。 |
| i18n | 所有人类可见字符串必须走 `t('key')`。 |
| UI 图标 | 使用 `@mui/icons-material`，优先 `*Rounded` 系列。 |
| 样式颜色 | 只能取 `theme.palette.foundation.*`，不要硬编码十六进制。 |
| 原生对话框 | 文件选择、保存、确认、错误提示必须走 `NativeDialogs`。 |

更多前端规则见 `frontend/CLAUDE.md`，构建规则见 `build/CLAUDE.md`。

## 文档索引

| 文档 | 内容 |
|------|------|
| `CLAUDE.md` | AI 协作入口、项目硬约束、开发命令。 |
| `docs/开发文档-1.0.0.md` | 1.0.0 功能需求、技术架构、模型、依赖选型、验收标准。 |
| `docs/开发计划-1.0.0.md` | 1.0.0 里程碑、任务拆解、测试计划、验收清单。 |
| `docs/验收记录-1.0.0.md` | 1.0.0 样本 MIDI 验收、文档同步状态、工具链验证限制。 |
| `frontend/CLAUDE.md` | 前端 MVVM、路由、i18n、主题、原生对话框、Icon 规范。 |
| `build/CLAUDE.md` | Windows 构建资源与 Taskfile 结构。 |

## License

MIT
