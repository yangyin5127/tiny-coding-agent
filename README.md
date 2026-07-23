# Tiny Coding Agent

A tiny coding agent written in Go 
用Go语言编写tiny coding agent

---

## Features / 功能

### 🛠 Built-in Tools / 内置工具

built-in capabilities:
内置能力：

| Tool / 工具 | Description / 描述 |
|-------------|-------------------|
| `bash` | Execute bash commands with automatic dangerous command detection / 执行 bash 命令 |
| `read_file` | Read file contents / 读取文件内容 |
| `write_file` | Write content to a file / 写入文件内容 |
| `edit_file` | Make edits to a text file by replacing strings / 编辑文本文件 |
| `glob` | List files matching a glob pattern / 列出匹配 glob 模式的文件 |
| `load_skill` | Load content of skill / 加载技能（Skill）内容 |
| `mcp` | Interact with external MCP servers / 与外部MCP服务交互 |

---

## Quick Start / 快速开始

### Prerequisites / 前置要求

- Go 1.22+
- An Anthropic-compatible API key / 一个兼容 Anthropic 的 API和密钥

### 1. Setup / 安装

```bash
# Clone the repository / 克隆仓库
git clone <repo-url>
cd tiny-coding-agent

# Install dependencies / 安装依赖
go mod tidy
```

### 2. Configuration / 配置

```bash
mkdir -p ~/.tiny-coding-agent
cp conf.example ~/.tiny-coding-agent/agent.conf
```

Then edit `~/.tiny-coding-agent/agent.conf`:
然后编辑 `~/.tiny-coding-agent/agent.conf`：

```
ANTHROPIC_BASE_URL=
ANTHROPIC_API_KEY=
MODEL=
```

### 3. Run / 运行

```bash
go run ./cmd/main.go
```

---

## Configuration / 配置说明

The agent loads configuration from `~/.tiny-coding-agent/agent.conf`.

agent从 `~/.tiny-coding-agent/agent.conf` 加载配置。

### Environment Variables / 环境变量

| Variable / 变量 | Description / 描述 | Example / 示例 |
|----------------|--------------------|----------------|
| `ANTHROPIC_BASE_URL` | Anthropic API base URL / Anthropic API兼容协议地址 | `https://api.anthropic.com`, `https://api.deepseek.com/anthropic` |
| `ANTHROPIC_API_KEY` | Your Anthropic API key / API 密钥 | `sk-ant-...` |
| `MODEL` | The model to use / 模型 | `claude-sonnet-4-20250514`, `deepseek-v4-flash` |


A sample template is provided at [`conf.example`](conf.example) in the project root.
项目根目录提供了示例模板 [`conf.example`](conf.example)。

### MCP Servers / MCP 服务器

Configure external MCP servers to extend the agent's capabilities.
配置外部 MCP服务以扩展代理能力。

`.tiny-coding-agent/.mcp.json` 配置至 project 根目录下的示例文件:

```json
{
    "mcpServers": {
        "currency-exchange": {
            "type": "http",
            "url": "https://currency-mcp.wesbos.com/mcp"
        },
        "drawio": {
            "type": "stdio",
            "command": "npx",
            "args": [
                "@drawio/mcp"
            ]
        }
    }
}
```

### Skills / 技能

Place `SKILL.md` files in the following directories:
将 `SKILL.md` 文件放置在以下目录：

| Directory / 目录 | Description / 描述 |
|------------------|--------------------|
| `.tiny-coding-agent/skills/` | Project agent skills / 项目代理技能 |
| `~/.tiny-coding-agent/skills/` | Global skills (user-level) / 全局技能（用户级） |

---

## Project Structure / 项目结构

```
tiny-coding-agent/
├── cmd/
│   └── main.go              # Entry point with TUI / TUI 入口
├── pkg/
│   └── utils/
│       └── utils.go         # Utility functions / 工具函数
├── src/
│   ├── agent/
│   │   └── agent.go         # Core AI agent loop / 核心 AI 代理循环
│   ├── mcp/
│   │   ├── client.go        # MCP client implementation / MCP 客户端实现
│   │   ├── http_transport.go # HTTP transport for MCP / HTTP 传输层
│   │   ├── manager.go       # MCP server & tool manager / MCP 服务器和工具管理器
│   │   ├── protocol.go      # MCP protocol types / MCP 协议类型
│   │   ├── stdio_transport.go # Stdio transport for MCP / Stdio 传输层
│   │   └── transport.go     # Transport interface / 传输接口
│   ├── prompt/
│   │   └── prompt.go        # System prompt template / 系统提示词模板
│   └── tools/
│       ├── bash_tool.go      # Bash execution tool / Bash 执行工具
│       ├── file_tools.go     # File read/write/edit/glob tools / 文件读写编辑工具
│       ├── interaction.go    # User interaction types / 用户交互类型
│       ├── load_skill.go     # Skills loader & tool / 技能加载器
│       └── tools_define.go   # Tool definition framework / 工具定义框架
├── conf.example             # Configuration template / 配置模板
├── go.mod
├── go.sum
└── README.md
```

---

## License / 许可证

[MIT](LICENSE)
