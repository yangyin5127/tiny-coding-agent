package hooks

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"tiny-coding-agent/src/tools"
)

type CommandHook struct {
	Command string
}

func (c *CommandHook) Run(ctx *HookToolContext, rt *tools.ToolRuntime) (*Result, error) {

	commandData := map[string]any{
		"tool_name":   ctx.ToolName,
		"tool_use_id": ctx.ToolUseId,
		"input":       ctx.Input,
		"output":      ctx.Output,
	}
	cmd := exec.Command("sh", "-c", c.Command)

	data, _ := json.Marshal(commandData)

	cmd.Stdin = bytes.NewReader(data)

	rt.Emit(tools.ToolEvent{Message: "Executing command hook: " + c.Command})
	out, err := cmd.Output()
	if err != nil {
		rt.Emit(tools.ToolEvent{Message: "Command hook error: " + err.Error()})
		return nil, err
	}
	rt.Emit(tools.ToolEvent{Message: "Command hook output: " + string(out)})
	if len(out) == 0 {
		return nil, nil
	}

	var result Result
	err = json.Unmarshal(out, &result)
	if err != nil {
		return nil, nil
	}

	return &result, nil
}
