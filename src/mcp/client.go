package mcp

import (
	"context"
	"encoding/json"
	"sync"
)

// https://modelcontextprotocol.io/
type Client struct {
	transport Transport
	id        int
	mu        sync.Mutex
}

func NewClient(transport Transport, id int) *Client {
	return &Client{
		transport: transport,
		id:        id,
	}
}

func (c *Client) nextID() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.id++
	return c.id
}

func (c *Client) Call(ctx context.Context, method string, params any) (any, error) {
	id := c.nextID()
	reqBody := &Request{
		Jsonrpc: "2.0",
		ID:      &id,
		Method:  method,
		Params:  params,
	}
	resp, err := c.transport.Send(ctx, reqBody)
	if err != nil {
		return nil, err
	}
	return resp.Result, nil
}

func (c *Client) Initialize(ctx context.Context) (any, error) {
	// https://modelcontextprotocol.io/specification/2025-11-25/basic/lifecycle
	return c.Call(ctx, "initialize", map[string]any{
		"protocolVersion": "2025-11-25",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]string{
			"name":    "tiny-coding-agent",
			"version": "0.1.0",
		},
	})
}

func (c *Client) Initialized(ctx context.Context) error {

	return c.notify(ctx, "notifications/initialized")
}

func (c *Client) notify(ctx context.Context, method string) error {

	req := &Request{
		Jsonrpc: "2.0",
		Method:  method,
	}

	return c.transport.Notify(ctx, req)
}

func (c *Client) ListTools(ctx context.Context) ([]ToolInfo, error) {
	result, err := c.Call(ctx, "tools/list", nil)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	var ret ToolListResult
	if err := json.Unmarshal(data, &ret); err != nil {
		return nil, err
	}
	return ret.Tools, nil
}

func (c *Client) CallTool(ctx context.Context, toolName string, params map[string]any) ([]string, error) {
	result, err := c.Call(ctx, "tools/call", CallToolParams{
		Name:      toolName,
		Arguments: params,
	})
	if err != nil {
		return nil, err
	}

	data, _ := json.Marshal(result)
	var ret CallToolResult
	if err := json.Unmarshal(data, &ret); err != nil {
		return nil, err
	}

	var texts []string
	for _, item := range ret.Content {
		if item.Type == "text" {
			texts = append(texts, item.Text)
		}
	}
	return texts, nil
}

func (c *Client) Close() {
	c.transport.Close()
}
