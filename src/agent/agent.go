package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"tiny-coding-agent/pkg/utils"
	"tiny-coding-agent/src/hooks"
	"tiny-coding-agent/src/mcp"
	"tiny-coding-agent/src/prompt"
	"tiny-coding-agent/src/tools"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

var agentTools = []tools.ToolDefinition{
	tools.BashTool,
	tools.ReadFileTool,
	tools.WriteFileTool,
	tools.EditFileTool,
	tools.GlobTool,
	tools.LoadSkill,
	CompactTool,
}

const (
	AgentOutputTypeDebug     = "debug"
	AgentOutputTypeError     = "error"
	AgentOutputTypeText      = "text"
	AgentOutputTypeThinking  = "thinking"
	AgentOutputTypeToolUse   = "tool_use"
	AgentOutputTypeToolHooks = "tool_hooks"
)

type AgentOutput struct {
	Type    string
	Message string
	Params  map[string]any
}

func NewAgentOutput(outputType, message string, params map[string]any) *AgentOutput {
	return &AgentOutput{
		Type:    outputType,
		Message: message,
		Params:  params,
	}
}

func NewAgentOutputDebug(message string) *AgentOutput {
	return &AgentOutput{
		Type:    AgentOutputTypeDebug,
		Message: message,
	}
}

func NewAgentOutputError(message string) *AgentOutput {
	return &AgentOutput{
		Type:    AgentOutputTypeError,
		Message: message,
	}
}

type Agent struct {
	client              *anthropic.Client
	UserMessage         chan *anthropic.MessageParam
	Output              chan *AgentOutput
	InteractionRequest  chan *tools.AgentInteractionRequest
	InteractionResponse chan *tools.AgentInteractionResponse
	toolDefinitions     []tools.ToolDefinition
	hookManager         *hooks.HookManager
}

func NewAgent(apiKey string, anthropicBaseUrl string) *Agent {
	client := anthropic.NewClient(option.WithAPIKey(apiKey), option.WithBaseURL(anthropicBaseUrl))
	return &Agent{
		client:              &client,
		UserMessage:         make(chan *anthropic.MessageParam),
		Output:              make(chan *AgentOutput),
		InteractionRequest:  make(chan *tools.AgentInteractionRequest),
		InteractionResponse: make(chan *tools.AgentInteractionResponse),
	}
}

func (a *Agent) Run(ctx context.Context) error {

	conversation := []anthropic.MessageParam{}
	systemPrompt := fmt.Sprintf(prompt.SystemPrompt, utils.Getwd())

	if skillsPromptPart, err := tools.LoadSkills(utils.Getwd()); err == nil {
		if len(skillsPromptPart) > 0 {
			systemPrompt += skillsPromptPart
			a.Output <- NewAgentOutput(AgentOutputTypeDebug, "Loaded skills success", nil)
		}

	}
	// a.Output <- NewAgentOutput(AgentOutputTypeDebug, "Loaded system prompt:\n"+systemPrompt, nil)

	MODEL := os.Getenv("MODEL")
	toolParams := []anthropic.ToolUnionParam{}
	a.toolDefinitions = append([]tools.ToolDefinition{}, agentTools...)
	for _, tool := range agentTools {
		toolParams = append(toolParams, anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        tool.Name,
				Description: anthropic.String(tool.Description),
				InputSchema: tool.InputSchema,
			},
		})
	}

	// load mcp
	mcpManager := mcp.NewManager()
	mcpServers, mcpTools, err := mcpManager.LoadAllTools(ctx, utils.Getwd())
	if err != nil {
		a.Output <- NewAgentOutputError("Failed to load MCP tools: " + err.Error())
		return err
	}

	for mcpServerName := range mcpServers {
		a.Output <- NewAgentOutput(AgentOutputTypeDebug, fmt.Sprintf("Loaded MCP server '%s' success", mcpServerName), nil)
	}

	for _, tool := range mcpTools {
		a.toolDefinitions = append(a.toolDefinitions, tool)
		toolParams = append(toolParams, anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        tool.Name,
				Description: anthropic.String(tool.Description),
				InputSchema: tool.InputSchema,
			},
		})
		// a.Output <- NewAgentOutput(AgentOutputTypeDebug, "Loaded MCP tool: "+tool.Name, nil)
	}

	a.Output <- NewAgentOutput(AgentOutputTypeDebug, fmt.Sprintf("\tLoaded %d MCP tools in total.", len(mcpTools)), nil)

	// init hook manager
	hookManager := hooks.NewHookManager(&tools.ToolRuntime{
		Emit: func(event tools.ToolEvent) {
			a.Output <- NewAgentOutput(AgentOutputTypeToolHooks, "\t"+event.Message, nil)
		},
	})
	err = hookManager.LoadHooks(ctx, utils.Getwd())
	if err != nil {
		a.Output <- NewAgentOutputError("Failed to load hooks: " + err.Error())
		return err
	}

	a.hookManager = hookManager
	a.Output <- NewAgentOutput(AgentOutputTypeDebug, "Loaded hooks successfully", nil)

	for {
		userMessage := <-a.UserMessage
		conversation = append(conversation, *userMessage)

		for {

			conversation, err = ContextCompact(ctx, a, conversation)
			if err != nil {
				a.Output <- NewAgentOutputError("Failed to compact context: " + err.Error())
				continue
			}

			// agent loop
			message, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
				MaxTokens: 8000,
				Messages:  conversation,
				System:    []anthropic.TextBlockParam{{Text: systemPrompt}},
				Model:     MODEL,
				Tools:     toolParams,
			})

			if err != nil {
				a.Output <- NewAgentOutputError("Failed to get model response: " + err.Error())
				continue
			}

			conversation = append(conversation, message.ToParam())

			toolResults := []anthropic.ContentBlockParamUnion{}

			manualCompact := false
			for _, content := range message.Content {
				if content.Type == "tool_use" {

					if content.Name == CompactTool.Name {

						conversation, err = CompactHistory(ctx, a, conversation[:len(conversation)-2])
						if err != nil {
							a.Output <- NewAgentOutputError("Failed to auto compact history: " + err.Error())
						} else {
							a.Output <- NewAgentOutput(AgentOutputTypeDebug, "Compacted.", nil)
						}
						manualCompact = true
						continue
					}
					result := a.executeTool(content.ID, content.Name, content.Input)
					toolResults = append(toolResults, result)
				} else {

					if content.Type == "text" {
						a.Output <- NewAgentOutput(AgentOutputTypeText, content.Text, nil)
					}

					if content.Type == "thinking" {
						a.Output <- NewAgentOutput(AgentOutputTypeThinking, "Thinking... "+content.Thinking, nil)
					}
				}

			}

			if len(toolResults) > 0 {
				conversation = append(conversation, anthropic.NewUserMessage(toolResults...))
			}
			if manualCompact {

				break
			}

			if message.StopReason != "tool_use" {
				// break agent loop if the model is not requesting a tool use
				break
			}
		}

	}
}

func (a *Agent) executeTool(id, name string, input json.RawMessage) anthropic.ContentBlockParamUnion {
	var toolDefinition *tools.ToolDefinition
	for _, tool := range a.toolDefinitions {
		if tool.Name == name {
			toolDefinition = &tool
			break
		}
	}
	if toolDefinition == nil {
		return anthropic.NewToolResultBlock(id, "tool not found", true)
	}

	if toolDefinition.CanExecute != nil {
		executionDecision, err := toolDefinition.CanExecute(input)
		if err != nil {
			return anthropic.NewToolResultBlock(id, "Failed to determine if tool can be executed: "+err.Error(), true)
		}
		if !executionDecision.Allowed {

			// debug info
			// a.Output <- NewAgentOutput(AgentOutputTypeDebug, "Waiting for user permission to execute tool "+name, nil)
			// ask user to grant permission for tool execution
			a.InteractionRequest <- executionDecision.Request

			// wait for user permission response
			approveResult := <-a.InteractionResponse

			if approveResult.OptionID != tools.ToolApprovalAllow {
				a.Output <- NewAgentOutputError("Execution of tool " + name + " was not allowed")
				return anthropic.NewToolResultBlock(id, "Tool execution denied by user approval policy. The user did not grant permission to run this tool.", false)
			}

		}
	}

	hookCtx := &hooks.HookToolContext{
		Context:   context.Background(),
		ToolName:  name,
		ToolUseId: id,
		Input:     input,
	}

	if a.hookManager != nil {
		result, err := a.hookManager.Execute(hooks.PreToolUse, hookCtx)
		if err != nil {
			a.Output <- NewAgentOutputError("Hook execution failed: " + err.Error())
			return anthropic.NewToolResultBlock(
				id,
				err.Error(),
				true,
			)
		}

		if result != nil && result.Decision == hooks.Deny {
			a.Output <- NewAgentOutputError("Hook denied tool execution: " + result.Message)
			return anthropic.NewToolResultBlock(
				id,
				result.Message,
				false,
			)
		}

		if result != nil {
			input = result.Input
		}

	}

	a.Output <- NewAgentOutput(AgentOutputTypeToolUse, "Ran "+name+"", nil)
	result, err := toolDefinition.Execute(input, &tools.ToolRuntime{
		Emit: func(event tools.ToolEvent) {
			a.Output <- NewAgentOutput(AgentOutputTypeToolUse, "\t"+event.Message, nil)
		},
	})

	if err != nil {
		if a.hookManager != nil {
			_, _ = a.hookManager.Execute(hooks.PostToolUseFailure, hookCtx)
		}

		return anthropic.NewToolResultBlock(id, err.Error(), true)
	}

	hookCtx.Output = result
	if a.hookManager != nil {
		postResult, err := a.hookManager.Execute(hooks.PostToolUse, hookCtx)
		if err != nil {
			a.Output <- NewAgentOutputError("Post tool use hook execution failed: " + err.Error())
		}

		if postResult != nil && postResult.Output != "" {
			result = postResult.Output
		}
	}

	return anthropic.NewToolResultBlock(id, result, false)

}
