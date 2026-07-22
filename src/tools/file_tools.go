package tools

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"os"

	"tiny-coding-agent/pkg/utils"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/google/uuid"
)

var ReadFileTool = ToolDefinition{
	Name:        "read_file",
	Description: "Read file contents",
	InputSchema: anthropic.ToolInputSchemaParam{
		Type: "object",
		Properties: map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "File path to read",
			},
		},
		Required: []string{"path"},
	},
	Execute: func(input json.RawMessage) (string, error) {
		var params struct {
			Path string `json:"path"`
		}
		err := json.Unmarshal(input, &params)
		if err != nil {
			return "", err
		}

		absPath, err := AbsPath(utils.Getwd(), params.Path)
		if err != nil {
			return "", err
		}

		content, err := os.ReadFile(absPath)
		if err != nil {
			return "", err
		}
		return string(content), nil
	},
	CanExecute: func(input json.RawMessage) (*ExecutionDecision, error) {
		var params struct {
			Path string `json:"path"`
		}

		executionDecision := &ExecutionDecision{
			Allowed: true,
		}

		err := json.Unmarshal(input, &params)
		if err != nil {
			return nil, err
		}

		absPath, err := AbsPath(utils.Getwd(), params.Path)
		if err != nil {
			return nil, err
		}

		if !strings.HasPrefix(absPath, utils.Getwd()) {
			executionDecision.Allowed = false
			executionDecision.Request = &AgentInteractionRequest{
				ID:    uuid.New().String(),
				Title: fmt.Sprintf("Attempted to read a file: %s outside the working directory", params.Path),
				Type:  InteractionTypeApproval,
				Options: []*AgentInteractionOption{
					{
						ID:    ToolApprovalAllow,
						Title: "Approve",
					},
					{
						ID:    ToolApprovalDeny,
						Title: "Deny",
					},
				},
			}
		}

		return executionDecision, nil

	},
}

var EditFileTool = ToolDefinition{
	Name:        "edit_file",
	Description: "Make edits to a text file. Replaces 'old_str' with 'new_str' in the given file",
	InputSchema: anthropic.ToolInputSchemaParam{
		Type: "object",
		Properties: map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "File path to edit",
			},
			"old_str": map[string]any{
				"type":        "string",
				"description": "String to be replaced in the file",
			},
			"new_str": map[string]any{
				"type":        "string",
				"description": "String to replace with in the file",
			},
		},
		Required: []string{"path", "old_str", "new_str"},
	},
	Execute: func(input json.RawMessage) (string, error) {
		var params struct {
			Path   string `json:"path"`
			OldStr string `json:"old_str"`
			NewStr string `json:"new_str"`
		}
		err := json.Unmarshal(input, &params)
		if err != nil {
			return "", err
		}

		absPath, err := AbsPath(utils.Getwd(), params.Path)
		if err != nil {
			return "", err
		}

		content, err := os.ReadFile(absPath)
		if err != nil {
			return "", err
		}

		newContent := strings.Replace(string(content), params.OldStr, params.NewStr, -1)

		err = os.WriteFile(absPath, []byte(newContent), 0644)
		if err != nil {
			return "", err
		}

		return "File edited successfully", nil
	},
	CanExecute: func(input json.RawMessage) (*ExecutionDecision, error) {
		var params struct {
			Path   string `json:"path"`
			OldStr string `json:"old_str"`
			NewStr string `json:"new_str"`
		}

		executionDecision := &ExecutionDecision{
			Allowed: true,
		}

		err := json.Unmarshal(input, &params)
		if err != nil {
			return nil, err
		}

		absPath, err := AbsPath(utils.Getwd(), params.Path)
		if err != nil {
			return nil, err
		}

		if !strings.HasPrefix(absPath, utils.Getwd()) {
			executionDecision.Allowed = false
			executionDecision.Request = &AgentInteractionRequest{
				ID:    uuid.New().String(),
				Title: fmt.Sprintf("Attempted to edit a file: %s outside the working directory", params.Path),
				Type:  InteractionTypeApproval,
				Options: []*AgentInteractionOption{
					{
						ID:    ToolApprovalAllow,
						Title: "Approve",
					},
					{
						ID:    ToolApprovalDeny,
						Title: "Deny",
					},
				},
			}
		}

		return executionDecision, nil
	},
}

var WriteFileTool = ToolDefinition{
	Name:        "write_file",
	Description: "Write content to a file.",
	InputSchema: anthropic.ToolInputSchemaParam{
		Type: "object",
		Properties: map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "File path to write to",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Content to write to the file",
			},
		},
		Required: []string{"path", "content"},
	},
	Execute: func(input json.RawMessage) (string, error) {
		var params struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		err := json.Unmarshal(input, &params)
		if err != nil {
			return "", err
		}

		absPath, err := AbsPath(utils.Getwd(), params.Path)
		if err != nil {
			return "", err
		}

		err = os.WriteFile(absPath, []byte(params.Content), 0644)
		if err != nil {
			return "", err
		}

		return "File written successfully", nil
	},
	CanExecute: func(input json.RawMessage) (*ExecutionDecision, error) {
		var params struct {
			Path string `json:"path"`
		}

		executionDecision := &ExecutionDecision{
			Allowed: true,
		}

		err := json.Unmarshal(input, &params)
		if err != nil {
			return nil, err
		}

		absPath, err := AbsPath(utils.Getwd(), params.Path)
		if err != nil {
			return nil, err
		}

		if !strings.HasPrefix(absPath, utils.Getwd()) {
			executionDecision.Allowed = false
			executionDecision.Request = &AgentInteractionRequest{
				ID:    uuid.New().String(),
				Title: fmt.Sprintf("Attempted to write to a file: %s outside the working directory", params.Path),
				Type:  InteractionTypeApproval,
				Options: []*AgentInteractionOption{
					{
						ID:    ToolApprovalAllow,
						Title: "Approve",
					},
					{
						ID:    ToolApprovalDeny,
						Title: "Deny",
					},
				},
			}
		}

		return executionDecision, nil
	},
}

var GlobTool = ToolDefinition{
	Name:        "glob",
	Description: "List files matching a glob pattern",
	InputSchema: anthropic.ToolInputSchemaParam{
		Type: "object",
		Properties: map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "Glob pattern to match files",
			},
		},
		Required: []string{"pattern"},
	},
	Execute: func(input json.RawMessage) (string, error) {
		var params struct {
			Pattern string `json:"pattern"`
		}
		err := json.Unmarshal(input, &params)
		if err != nil {
			return "", err
		}

		matches, err := filepath.Glob(filepath.Join(utils.Getwd(), params.Pattern))
		if err != nil {
			return "", err
		}

		var resutls []string
		for _, match := range matches {
			relPath, err := filepath.Rel(utils.Getwd(), match)
			if err != nil {
				return "", err
			}
			resutls = append(resutls, relPath)
		}

		return strings.Join(resutls, "\n"), nil
	},
	CanExecute: func(input json.RawMessage) (*ExecutionDecision, error) {
		return &ExecutionDecision{
			Allowed: true,
		}, nil
	},
}
