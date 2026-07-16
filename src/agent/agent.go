package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"tiny-coding-agent/pkg/utils"
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
}

type Agent struct {
	client      *anthropic.Client
	UserMessage chan *anthropic.MessageParam
	Response    chan string
}

func NewAgent(apiKey string, anthropicBaseUrl string) *Agent {
	client := anthropic.NewClient(option.WithAPIKey(apiKey), option.WithBaseURL(anthropicBaseUrl))
	return &Agent{
		client:      &client,
		UserMessage: make(chan *anthropic.MessageParam),
		Response:    make(chan string, 10),
	}
}

func (a *Agent) Run(ctx context.Context) error {

	conversation := []anthropic.MessageParam{}
	systemPrompt := fmt.Sprintf(prompt.SystemPrompt, utils.Getwd())
	MODEL := os.Getenv("MODEL")
	tools := []anthropic.ToolUnionParam{}
	for _, tool := range agentTools {
		tools = append(tools, anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        tool.Name,
				Description: anthropic.String(tool.Description),
				InputSchema: tool.InputSchema,
			},
		})
	}
	for {
		userMessage := <-a.UserMessage
		conversation = append(conversation, *userMessage)

		for {
			// agent loop
			message, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
				MaxTokens: 8000,
				Messages:  conversation,
				System:    []anthropic.TextBlockParam{{Text: systemPrompt}},
				Model:     MODEL,
				Tools:     tools,
			})

			if err != nil {
				a.Response <- "Failed to get model response: " + err.Error()
				continue
			}

			conversation = append(conversation, message.ToParam())

			toolResults := []anthropic.ContentBlockParamUnion{}
			for _, content := range message.Content {
				if content.Type == "tool_use" {
					result := a.executeTool(content.ID, content.Name, content.Input)
					toolResults = append(toolResults, result)

				} else {

					if content.Type == "text" {
						a.Response <- content.Text
					}

					if content.Type == "thinking" {
						a.Response <- "Thinking... " + content.Thinking
					}
				}

			}

			if len(toolResults) > 0 {
				conversation = append(conversation, anthropic.NewUserMessage(toolResults...))
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
	for _, tool := range agentTools {
		if tool.Name == name {
			toolDefinition = &tool
			break
		}
	}
	if toolDefinition == nil {
		return anthropic.NewToolResultBlock(id, "tool not found", true)
	}

	result, err := toolDefinition.Function(input)

	if err != nil {
		return anthropic.NewToolResultBlock(id, err.Error(), true)
	}
	a.Response <- "Ran " + name + ""

	return anthropic.NewToolResultBlock(id, result, false)

}
