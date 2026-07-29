package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"time"
	"tiny-coding-agent/pkg/utils"
	"tiny-coding-agent/src/prompt"
	"tiny-coding-agent/src/tools"

	"github.com/anthropics/anthropic-sdk-go"
)

const (
	CONTEXT_LIMIT       = 80000
	KEEP_RECENT         = 3
	PERSIST_THRESHOLD   = 30000
	TOOL_RESULT_PREVIEW = 20000
	TOOL_RESULT_BUDGET  = 20000
)

type ContextCompactConfig struct {
	ContextLimit       int
	KeepRecent         int
	PersistThreshold   int
	ToolResultPreview  int
	ToolResultBudget   int
	ToolResultBasePath string
}

var DefaultContextCompactConfig = ContextCompactConfig{
	ContextLimit:       CONTEXT_LIMIT,
	KeepRecent:         KEEP_RECENT,
	PersistThreshold:   PERSIST_THRESHOLD,
	ToolResultPreview:  TOOL_RESULT_PREVIEW,
	ToolResultBudget:   TOOL_RESULT_BUDGET,
	ToolResultBasePath: utils.Getwd(),
}

func normalizeContextCompactConfig(cfg ContextCompactConfig) ContextCompactConfig {
	if cfg.ContextLimit <= 0 {
		cfg.ContextLimit = CONTEXT_LIMIT
	}
	if cfg.KeepRecent < 0 {
		cfg.KeepRecent = KEEP_RECENT
	}
	if cfg.PersistThreshold <= 0 {
		cfg.PersistThreshold = PERSIST_THRESHOLD
	}
	if cfg.ToolResultPreview <= 0 {
		cfg.ToolResultPreview = TOOL_RESULT_PREVIEW
	}
	if cfg.ToolResultBudget <= 0 {
		cfg.ToolResultBudget = TOOL_RESULT_BUDGET
	}
	if cfg.ToolResultBasePath == "" {
		cfg.ToolResultBasePath = utils.Getwd()
	}

	return cfg
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func PersistLargeToolResult(ctx context.Context, toolUseId string, content string) string {
	return PersistLargeToolResultWithConfig(ctx, toolUseId, content, DefaultContextCompactConfig)
}

func PersistLargeToolResultWithConfig(ctx context.Context, toolUseId string, content string, cfg ContextCompactConfig) string {
	cfg = normalizeContextCompactConfig(cfg)

	if len(content) <= cfg.PersistThreshold {
		return content
	}

	toolResultDir := path.Join(cfg.ToolResultBasePath, ".tiny-coding-agent", ".task_outputs", "tool-results")
	// Ensure the tool result directory exists
	if _, err := os.Stat(toolResultDir); os.IsNotExist(err) {
		_ = os.MkdirAll(toolResultDir, os.ModePerm)
	}
	persistedPath := path.Join(toolResultDir, toolUseId+".txt")
	_ = os.WriteFile(persistedPath, []byte(content), os.ModePerm)

	previewEnd := minInt(len(content), cfg.ToolResultPreview)
	return content[:cfg.PersistThreshold] + "\n<persisted-output>\nFull output: " + persistedPath + "\nPreview:\n" + content[:previewEnd] + "\n</persisted-output>"

}
func ToolResultBudget(ctx context.Context, messages []anthropic.MessageParam) ([]anthropic.MessageParam, error) {
	return ToolResultBudgetWithConfig(ctx, messages, DefaultContextCompactConfig)
}

func ToolResultBudgetWithConfig(ctx context.Context, messages []anthropic.MessageParam, cfg ContextCompactConfig) ([]anthropic.MessageParam, error) {
	cfg = normalizeContextCompactConfig(cfg)

	if len(messages) == 0 {
		return messages, nil
	}
	lastMessge := messages[len(messages)-1]
	if lastMessge.Role != "user" {
		return messages, nil
	}

	totalBytes := 0
	for _, content := range lastMessge.Content {
		// fmt.Printf("content1 %s\n", content.OfToolResult.Type)

		if content.OfToolResult != nil {
			for _, block := range content.OfToolResult.Content {
				if block.OfText != nil {
					totalBytes = totalBytes + len(block.OfText.Text)
				}
			}
		}
	}

	if totalBytes <= cfg.ToolResultBudget {
		return messages, nil
	}

	for i := len(lastMessge.Content) - 1; i >= 0; i-- {
		content := lastMessge.Content[i]
		if content.OfToolResult != nil {
			for _, block := range content.OfToolResult.Content {
				if block.OfText != nil {
					blockSize := len(block.OfText.Text)
					if blockSize > cfg.PersistThreshold {
						block.OfText.Text = PersistLargeToolResultWithConfig(ctx, content.OfToolResult.ToolUseID, block.OfText.Text, cfg)
					}
					totalBytes = totalBytes - (blockSize - len(block.OfText.Text))
				}
			}

			if totalBytes <= cfg.ToolResultBudget {
				break
			}
		}
	}
	return messages, nil

}

// replace old tool results with placeholders
func MicroContext(ctx context.Context, messages []anthropic.MessageParam) ([]anthropic.MessageParam, error) {
	toolResults := []*anthropic.ToolResultBlockParam{}

	for _, message := range messages {
		if message.Role == "user" {
			for _, content := range message.Content {
				if content.OfToolResult != nil {
					toolResults = append(toolResults, content.OfToolResult)
				}
			}
		}
	}

	if len(toolResults) > DefaultContextCompactConfig.KeepRecent {
		for i := 0; i < len(toolResults)-DefaultContextCompactConfig.KeepRecent; i++ {
			for j := 0; j < len(toolResults[i].Content); j++ {
				if toolResults[i].Content[j].OfText != nil && len(toolResults[i].Content[j].OfText.Text) > 200 {
					toolResults[i].Content[j].OfText.Text = "[Earlier tool result compacted. Re-run if needed.]"
				}
			}
		}
	}
	return messages, nil
}

func CompactHistory(ctx context.Context, agent *Agent, messages []anthropic.MessageParam) ([]anthropic.MessageParam, error) {

	transcriptsDir := path.Join(utils.ExpandHome("~/.tiny-coding-agent"), ".transcripts")
	if _, err := os.Stat(transcriptsDir); os.IsNotExist(err) {
		if err := os.MkdirAll(transcriptsDir, 0755); err != nil {
			return messages, err
		}
	}

	transcriptPath := path.Join(transcriptsDir, fmt.Sprintf("transcript_%d.jsonl", time.Now().Unix()))

	agent.Output <- NewAgentOutput(AgentOutputTypeDebug, "Creating transcript file at: "+transcriptPath, nil)
	f, err := os.Create(transcriptPath)
	if err != nil {
		return messages, err
	}
	defer f.Close()

	for _, message := range messages {
		data, err := json.Marshal(message)
		if err != nil {
			return messages, err
		}
		if _, err := f.WriteString(string(data) + "\n"); err != nil {
			return messages, err
		}
	}

	newMessages := append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(prompt.SystemCompactPrompt)))

	agent.Output <- NewAgentOutput(AgentOutputTypeDebug, "compacting conversation", nil)

	MODEL := os.Getenv("MODEL")
	result, err := agent.client.Messages.New(ctx, anthropic.MessageNewParams{
		MaxTokens: 8000,
		Messages:  newMessages,
		Model:     MODEL,
	})

	if err != nil {
		agent.Output <- NewAgentOutput(AgentOutputTypeError, "Error compacting conversation: "+err.Error(), nil)
		return messages, err
	}

	compactedText := ""
	for _, msg := range result.Content {
		if msg.Type == "text" {
			compactedText = msg.Text
			break
		}
	}

	compactMessages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(compactedText)),
	}

	// var preservedMessages []anthropic.MessageParam
	// if len(messages) > 3 {
	// 	preservedMessages = messages[len(messages)-3:]
	// } else {
	// 	preservedMessages = messages
	// }

	// compactMessages = append(compactMessages, preservedMessages...)

	return compactMessages, nil

}

func EstimateTokens(messages []anthropic.MessageParam) int {
	// Rough token count: ~4 chars per token.
	allMessages, err := json.Marshal(messages)
	if err != nil {
		return 0
	}
	return len(string(allMessages)) / 4
}

func ContextCompact(ctx context.Context, agent *Agent, messages []anthropic.MessageParam) ([]anthropic.MessageParam, error) {

	newMessages, err := ToolResultBudget(ctx, messages)
	if err != nil {
		return messages, err
	}

	newMessages, err = MicroContext(ctx, newMessages)
	if err != nil {
		return newMessages, err
	}

	if EstimateTokens(messages) > CONTEXT_LIMIT {
		agent.Output <- NewAgentOutput(AgentOutputTypeDebug, "Message token count exceeds limit", nil)
		newMessages, err = CompactHistory(ctx, agent, newMessages)
		if err != nil {
			return newMessages, err
		}
	}

	return newMessages, nil
}

var CompactTool = tools.ToolDefinition{
	Name:        "compact",
	Description: "Summarize earlier conversation to free context space.",
	InputSchema: anthropic.ToolInputSchemaParam{
		Type:       "object",
		Properties: map[string]any{},
	},
	CanExecute: func(input json.RawMessage) (*tools.ExecutionDecision, error) {
		return &tools.ExecutionDecision{
			Allowed: true,
		}, nil
	},
	Execute: func(input json.RawMessage, rt *tools.ToolRuntime) (string, error) {

		return "", nil
	},
}
