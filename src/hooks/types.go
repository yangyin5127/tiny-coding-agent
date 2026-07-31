package hooks

import (
	"context"
	"encoding/json"
	"tiny-coding-agent/src/tools"
)

type HookEvent string
type Decision string

const (
	PreToolUse         HookEvent = "PreToolUse"
	PostToolUse        HookEvent = "PostToolUse"
	PostToolUseFailure HookEvent = "PostToolUseFailure"

	Allow Decision = "allow"
	Deny  Decision = "deny"
)

type HookToolContext struct {
	Context   context.Context
	ToolName  string
	ToolUseId string
	Input     json.RawMessage
	Output    string
	Error     error
	Metadata  map[string]any
}

type Result struct {
	Decision Decision
	Input    json.RawMessage
	Output   string
	Message  string
}

type Hook interface {
	Run(ctx *HookToolContext, rt *tools.ToolRuntime) (*Result, error)
}
