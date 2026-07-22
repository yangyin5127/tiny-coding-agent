package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"tiny-coding-agent/pkg/utils"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/google/uuid"
)

func isOutsideWorkspace(path string, workspace string) bool {
	workspaceAbs, err := filepath.Abs(workspace)
	if err != nil {
		return true
	}

	var target string

	if filepath.IsAbs(path) {
		target = path
	} else {
		target = filepath.Join(workspaceAbs, path)
	}

	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return true
	}

	targetAbs = filepath.Clean(targetAbs)

	rel, err := filepath.Rel(workspaceAbs, targetAbs)
	if err != nil {
		return true
	}

	return rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func extractPaths(command string) []string {
	re := regexp.MustCompile(`(?:^|\s)(/[^\s"'<>|;&]+|\.{1,2}/[^\s"'<>|;&]+)`)

	matches := re.FindAllStringSubmatch(command, -1)

	var paths []string

	for _, m := range matches {
		if len(m) > 1 {
			paths = append(paths, m[1])
		}
	}

	return paths
}

func containsOutsideWorkspace(command string, workspace string) bool {
	paths := extractPaths(command)

	for _, p := range paths {
		if isOutsideWorkspace(p, workspace) {
			return true
		}
	}

	return false
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
	Execute: func(input json.RawMessage) (string, error) {
		var params struct {
			Command string `json:"command"`
		}
		err := json.Unmarshal(input, &params)
		if err != nil {
			return "", err
		}

		var denyCommands = []string{
			// delete root / and critical system directories
			"rm -rf /",
			"rm -rf /*",
			"rm -rf --no-preserve-root /",

			// disk formatting
			"mkfs",
			"mkfs.ext",
			"mkfs.xfs",
			"mkfs.btrfs",

			// write to disk
			"dd if=/dev/zero of=/dev/",
			"dd if=/dev/random of=/dev/",
			"dd if=/dev/urandom of=/dev/",

			// fork bomb
			":(){ :|:& };:",

			"rm -rf /etc",
			"rm -rf /usr",
			"rm -rf /bin",
			"rm -rf /boot",

			"passwd",

			"shutdown",
			"poweroff",
			"halt",
			"reboot",
		}

		for _, denCommand := range denyCommands {
			if strings.Contains(params.Command, denCommand) {
				return "", fmt.Errorf("dangerous command detected: %s", denCommand)
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
	CanExecute: func(input json.RawMessage) (*ExecutionDecision, error) {
		executionDecision := &ExecutionDecision{
			Allowed: true,
		}
		var params struct {
			Command string `json:"command"`
		}
		err := json.Unmarshal(input, &params)
		if err != nil {
			return nil, err
		}

		var dangerousCommands = []string{
			// permission modification
			"chmod 777",
			"chmod -R 777",
			"chown",
			"chgrp",

			// sudo
			"sudo",

			// delete files
			"rm ",
			"rm -r",
			"rm -rf",

			// move and overwrite system files
			"mv /",
			"cp /",

			// modify disk mounts
			"mount",
			"umount",

			// service control
			"systemctl stop",
			"systemctl disable",
			"service stop",

			// kill processes
			"kill ",
			"killall",
			"pkill",

			// network modification
			"iptables",
			"ufw disable",

			// user management
			"useradd",
			"userdel",
			"usermod",

			// package management
			"apt install",
			"apt remove",
			"yum install",
			"brew install",

			// modify environment
			"export PATH=",
		}

		for _, dangerousCommand := range dangerousCommands {
			if strings.Contains(params.Command, dangerousCommand) {
				executionDecision.Allowed = false

				executionDecision.Request = &AgentInteractionRequest{
					ID:    uuid.New().String(),
					Title: fmt.Sprintf("dangerous command detected: %s, do you want to execute it?", dangerousCommand),
					Options: []*AgentInteractionOption{
						{
							ID:    "approve",
							Title: "Approve",
						},
						{
							ID:    "deny",
							Title: "Deny",
						},
					},
				}

				break
			}
		}

		if containsOutsideWorkspace(params.Command, utils.Getwd()) {
			executionDecision.Allowed = false
			executionDecision.Request = &AgentInteractionRequest{
				ID:    uuid.New().String(),
				Title: fmt.Sprintf("Command \"%s\" attempts to access files outside the workspace. Do you want to execute it?", params.Command),
				Options: []*AgentInteractionOption{
					{
						ID:    "approve",
						Title: "Approve",
					},
					{
						ID:    "deny",
						Title: "Deny",
					},
				},
			}
		}

		return executionDecision, nil
	},
}
