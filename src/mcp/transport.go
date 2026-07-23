package mcp

import "context"

const (
	MCPTypeStdio = "stdio"
	MCPTypeHTTP  = "http"
)

type Transport interface {
	Send(ctx context.Context, req *Request) (*Response, error)
	Notify(ctx context.Context, req *Request) error

	Close() error
}
