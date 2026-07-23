package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"tiny-coding-agent/src/tools"
)

type Manager struct {
	clients map[string]*Client

	tools []tools.ToolDefinition
}

func NewManager() *Manager {
	return &Manager{
		clients: make(map[string]*Client),
		tools:   []tools.ToolDefinition{},
	}
}

func NormalizeMcpName(name string) string {
	// Replace non [a-zA-Z0-9_-] with underscore.
	re := regexp.MustCompile(`[^a-zA-Z0-9_-]`)
	return re.ReplaceAllString(name, "_")
}

func NewTransport(ctx context.Context, config *MCPServerConfig) (Transport, error) {
	if config == nil {
		return nil, nil
	}

	switch config.Type {
	case MCPTypeStdio:
		return NewStdioTransport(ctx, config)
	case MCPTypeHTTP:
		return NewHttpTransport(config)
	}

	if len(config.Type) == 0 && len(config.Command) > 0 {
		return NewStdioTransport(ctx, config)
	}

	if len(config.Type) == 0 && len(config.URL) > 0 {
		return NewHttpTransport(config)
	}

	return nil, nil
}

func ConvertTool(server string, client *Client, mcpTool ToolInfo) tools.ToolDefinition {

	toolName := fmt.Sprintf("mcp_%s_%s", server, NormalizeMcpName(mcpTool.Name))
	return tools.ToolDefinition{
		Name:        toolName,
		Description: mcpTool.Description,
		InputSchema: mcpTool.InputSchema,
		Execute: func(input json.RawMessage, rt *tools.ToolRuntime) (string, error) {
			var args map[string]any

			err := json.Unmarshal(input, &args)

			if err != nil {
				return "", err
			}
			rt.Emit(tools.ToolEvent{
				Type:    "info",
				Message: "Calling mcp tool " + toolName,
			})
			contents, err := client.CallTool(context.Background(), mcpTool.Name, args)
			if err != nil {
				return "", err
			}
			return strings.Join(contents, "\n"), nil

		},
	}
}

func (m *Manager) LoadConfig(workDir string) (*MCPConfig, error) {
	// Load the MCP configuration from the specified work directory.
	configPath := fmt.Sprintf("%s/.tiny-coding-agent/.mcp.json", workDir)

	// check if exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, nil
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config MCPConfig
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (m *Manager) LoadAllTools(ctx context.Context, workDir string) (map[string]MCPServerConfig, []tools.ToolDefinition, error) {

	// project
	config, err := m.LoadConfig(workDir)
	if err != nil {
		return nil, nil, err
	}
	if config == nil {
		return nil, nil, nil
	}

	var allTools []tools.ToolDefinition

	for serverName, config := range config.Servers {
		transport, err := NewTransport(ctx, &config)
		if err != nil {
			return nil, nil, err
		}

		if transport == nil {
			continue
		}

		client := NewClient(transport, 0)

		_, err = client.Initialize(ctx)
		if err != nil {
			client.Close()
			return nil, nil, err
		}
		m.clients[serverName] = client

		_ = client.Initialized(ctx)

		mcpTools, err := client.ListTools(ctx)
		if err != nil {
			client.Close()
			return nil, nil, err
		}

		for _, mcpTool := range mcpTools {
			allTools = append(allTools, ConvertTool(serverName, client, mcpTool))
		}
	}

	return config.Servers, allTools, nil
}
