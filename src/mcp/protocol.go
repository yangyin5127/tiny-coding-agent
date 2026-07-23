package mcp

import "github.com/anthropics/anthropic-sdk-go"

type Request struct {
	Jsonrpc string `json:"jsonrpc"`
	ID      *int   `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type Response struct {
	Jsonrpc string    `json:"jsonrpc"`
	ID      any       `json:"id"`
	Result  any       `json:"result,omitempty"`
	Error   *RPCError `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type ToolInfo struct {
	Name        string                         `json:"name"`
	Description string                         `json:"description"`
	InputSchema anthropic.ToolInputSchemaParam `json:"inputSchema"`
}

type ToolListResult struct {
	Tools []ToolInfo `json:"tools"`
}

type CallToolParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type CallToolResult struct {
	Content []Content `json:"content"`
}

type Content struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type MCPConfig struct {
	Servers map[string]MCPServerConfig `json:"mcpServers" yaml:"mcpServers"`
}

type MCPServerConfig struct {

	// stdio / http
	Type string `json:"type" yaml:"type"`

	// stdio
	Command string `json:"command,omitempty" yaml:"command,omitempty"`

	Args []string `json:"args,omitempty" yaml:"args,omitempty"`

	// http
	URL string `json:"url,omitempty" yaml:"url,omitempty"`

	// optional
	Env map[string]string `json:"env,omitempty" yaml:"env,omitempty"`

	// optional HTTP header
	Headers map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
}
