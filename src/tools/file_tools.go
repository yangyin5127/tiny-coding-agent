package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"os"

	"tiny-coding-agent/pkg/utils"

	"github.com/anthropics/anthropic-sdk-go"
)

var ReadFileTool = ToolDefinition{
	Name:        "read_file",
	Description: "Read file contents",
	InputSchema: anthropic.ToolInputSchemaParam{
		Type: "object",
		Properties: map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "File path relative to the working directory",
			},
		},
		Required: []string{"path"},
	},
	Function: func(input json.RawMessage) (string, error) {
		var params struct {
			Path string `json:"path"`
		}
		err := json.Unmarshal(input, &params)
		if err != nil {
			return "", err
		}

		safePath, err := safePath(utils.Getwd(), params.Path)
		if err != nil {
			return "", err
		}

		content, err := os.ReadFile(safePath)
		if err != nil {
			return "", err
		}
		return string(content), nil
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
				"description": "File path relative to the working directory",
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
	Function: func(input json.RawMessage) (string, error) {
		var params struct {
			Path   string `json:"path"`
			OldStr string `json:"old_str"`
			NewStr string `json:"new_str"`
		}
		err := json.Unmarshal(input, &params)
		if err != nil {
			return "", err
		}

		safePath, err := safePath(utils.Getwd(), params.Path)
		if err != nil {
			return "", err
		}

		content, err := os.ReadFile(safePath)
		if err != nil {
			return "", err
		}

		newContent := strings.Replace(string(content), params.OldStr, params.NewStr, -1)

		err = os.WriteFile(safePath, []byte(newContent), 0644)
		if err != nil {
			return "", err
		}

		return "File edited successfully", nil
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
				"description": "File path relative to the working directory",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Content to write to the file",
			},
		},
		Required: []string{"path", "content"},
	},
	Function: func(input json.RawMessage) (string, error) {
		var params struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		err := json.Unmarshal(input, &params)
		if err != nil {
			return "", err
		}

		safePath, err := safePath(utils.Getwd(), params.Path)
		if err != nil {
			return "", err
		}

		err = os.WriteFile(safePath, []byte(params.Content), 0644)
		if err != nil {
			return "", err
		}

		return "File written successfully", nil
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
	Function: func(input json.RawMessage) (string, error) {
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
}

var BashTool = ToolDefinition{
	Name:        "bash",
	Description: "Execute a bash command and return the output",
	InputSchema: anthropic.ToolInputSchemaParam{
		Type: "object",
		Properties: map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "The bash command to execute",
			},
		},
		Required: []string{"command"},
	},
	Function: func(input json.RawMessage) (string, error) {
		var params struct {
			Command string `json:"command"`
		}
		err := json.Unmarshal(input, &params)
		if err != nil {
			return "", err
		}

		dangerousCommands := []string{
			"rm -rf /",
			"rm -rf ~",
			"sudo",
			"shutdown",
			"reboot",
			"init 0",
			"halt",
			"poweroff",
			"mkfs",
			"dd if=/dev/zero",
			":(){ :|:& };:",
			"> /dev/",
			">/dev/",
			"chmod 777 /",
			"chown -R",
			"passwd",
			"killall",
			"pkill",
			"systemctl stop",
			"service .* stop",
			"kill -9"}

		for _, dangerousCommand := range dangerousCommands {
			if strings.Contains(params.Command, dangerousCommand) {
				return "", fmt.Errorf("dangerous command detected: %s", dangerousCommand)
			}
		}

		// Create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "bash", "-c", params.Command)
		cmd.Dir = utils.Getwd()
		output, err := cmd.CombinedOutput()

		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("command execution timed out")
		}

		// Check for command execution errors
		if err != nil {
			return "", fmt.Errorf("command execution failed: %v, output: %s", err, string(output))
		}

		result := strings.TrimSpace(string(output))
		if result == "" {
			result = "(no output)"
		}

		if len(result) > 5000 {
			result = result[:5000]
		}

		return result, nil
	},
}
