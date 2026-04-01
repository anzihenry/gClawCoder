# gClawCoder

Go 语言实现的 Claw Code 项目镜像系统。

## 项目概述

本项目是 Python 版本 Claw Code 的 Go 语言完整实现，提供了命令和工具的镜像系统、查询引擎、运行时管理等核心功能。

## 快速开始

### 构建

```bash
cd gClawCoder
go build -o gclaw ./cmd/gclaw
```

### 运行

```bash
# TUI 界面（推荐）
./gclaw tui

# REPL 模式
./gclaw repl

# 查看帮助
./gclaw help

# 查看摘要
./gclaw summary

# 查看命令列表
./gclaw commands --limit 10

# 查看工具列表
./gclaw tools --limit 10

# 路由提示
./gclaw route "review MCP tool"

# 引导会话
./gclaw bootstrap "review MCP tool" --limit 5

# 运行轮次循环
./gclaw turn-loop "review MCP tool" --max-turns 3
```

## 项目结构

```
gClawCoder/
├── cmd/
│   └── gclaw/          # CLI 主入口
├── internal/
│   ├── commands/       # 命令镜像系统
│   ├── tools/          # 工具镜像系统
│   ├── models/         # 数据模型
│   ├── query/          # 查询引擎
│   ├── runtime/        # 运行时管理
│   ├── session/        # 会话存储
│   └── transcript/     # 转录存储
├── data/               # 参考数据 JSON
├── tests/              # 测试文件
├── go.mod              # Go 模块定义
└── README.md           # 项目文档
```

## 功能特性

### 命令系统
- 从 JSON 快照加载命令元数据
- 支持命令搜索和过滤
- 支持插件命令和技能命令过滤
- 命令执行模拟

### 工具系统
- 从 JSON 快照加载工具元数据
- 支持工具搜索和过滤
- 支持 MCP 工具过滤
- 支持权限上下文过滤
- 工具执行模拟

### 查询引擎
- 支持消息提交和流式输出
- 支持 Token 使用统计
- 支持会话持久化
- 支持结构化输出
- 支持消息压缩

### 运行时
- 提示路由匹配
- 会话引导
- 轮次循环执行
- 权限拒绝推断

## CLI 命令

| 命令 | 描述 |
|------|------|
| `summary` | 渲染 Go 移植工作区摘要 |
| `manifest` | 打印当前工作区清单 |
| `subsystems` | 列出工作区模块 |
| `commands` | 列出镜像命令 |
| `tools` | 列出镜像工具 |
| `route` | 路由提示到命令/工具 |
| `bootstrap` | 构建运行时会话报告 |
| `turn-loop` | 运行轮次循环 |
| `show-command` | 显示单个命令详情 |
| `show-tool` | 显示单个工具详情 |
| `exec-command` | 执行命令模拟 |
| `exec-tool` | 执行工具模拟 |
| `load-session` | 加载已保存的会话 |

## 测试

```bash
# 运行所有测试
go test ./...

# 运行特定包测试
go test ./internal/commands
go test ./internal/tools
go test ./internal/query
```

## 与 Python 版本对比

| 功能 | Python | Go |
|------|--------|-----|
| 命令镜像 | ✅ | ✅ |
| 工具镜像 | ✅ | ✅ |
| 查询引擎 | ✅ | ✅ |
| 会话存储 | ✅ | ✅ |
| 转录存储 | ✅ | ✅ |
| 运行时管理 | ✅ | ✅ |
| 流式输出 | ✅ | ✅ |
| 结构化输出 | ✅ | ✅ |

## 开发

### 添加新命令

1. 在 `data/commands_snapshot.json` 中添加命令元数据
2. 命令会自动被 `commands.PortedCommands()` 加载

### 添加新工具

1. 在 `data/tools_snapshot.json` 中添加工具元数据
2. 工具会自动被 `tools.PortedTools()` 加载

## 许可证

MIT License
