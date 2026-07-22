package tools

import (
	"encoding/json"
	"path"

	"github.com/anthropics/anthropic-sdk-go"
)

const (
	ToolApprovalAllow = "approve"
	ToolApprovalDeny  = "deny"
)

type ToolEvent struct {
	Type    string
	Message string
	Data    any
}

type ToolRuntime struct {
	Emit func(event ToolEvent)
}

type ToolDefinition struct {
	Name        string                         `json:"name"`
	Description string                         `json:"description"`
	InputSchema anthropic.ToolInputSchemaParam `json:"input_schema"`
	Execute     func(input json.RawMessage, rt *ToolRuntime) (string, error)
	CanExecute  func(input json.RawMessage) (*ExecutionDecision, error)
}

type ExecutionDecision struct {
	Allowed bool
	Request *AgentInteractionRequest
}

func AbsPath(workdir, p string) (string, error) {
	// 检查是否已经是绝对路径
	if path.IsAbs(p) {
		return path.Clean(p), nil
	}

	filePath := path.Join(workdir, p)

	absPath := path.Clean(filePath)

	return absPath, nil

}
