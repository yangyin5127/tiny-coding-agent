# Tiny Coding Agent

A tiny coding agent written in Go 

用Go语言实现的tiny coding agent

---

## Features

 
- [x] coding / 编码
- [x] skills
- [x] MCP
- [ ] context compact / 上下文压缩
- [ ] hooks / 钩子

---

## Quick Start

### Prerequisites

- Go 1.22+
- An Anthropic-compatible API key / 兼容 Anthropic 的 API和密钥即可

### 1. Setup

```bash
# Clone the repository 
git clone <repo-url>
cd tiny-coding-agent

# Install dependencies
go mod tidy
```

### 2. Configuration

```bash
mkdir -p ~/.tiny-coding-agent
cp conf.example ~/.tiny-coding-agent/agent.conf
```

vim `~/.tiny-coding-agent/agent.conf`:

```
ANTHROPIC_BASE_URL=
ANTHROPIC_API_KEY=
MODEL=
```

| Variable | Description  | Example   |
|----------------|--------------------|----------------|
| `ANTHROPIC_BASE_URL` | Anthropic API base URL / Anthropic API兼容协议地址 | `https://api.deepseek.com/anthropic` |
| `ANTHROPIC_API_KEY` | Your Anthropic API key / API密钥 | `sk-ant-...` |
| `MODEL` | The model to use / 模型 | `deepseek-v4-flash` |


### 3. Run

```bash
# debug
go run ./cmd/tiny-coding-agent/main.go
```

install
```bash

# install
go install ./cmd/tiny-coding-agent

# project dir
tiny-coding-agent
```

---
 
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
| `.tiny-coding-agent/skills/` | Project agent skills / 项目级 |
| `~/.tiny-coding-agent/skills/` | Global skills (user-level) / 全局级  |

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
