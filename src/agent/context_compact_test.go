package agent

import (
	"context"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
)

func firstToolResultText(message anthropic.MessageParam) string {
	for _, content := range message.Content {
		if content.OfToolResult == nil {
			continue
		}
		for _, block := range content.OfToolResult.Content {
			if block.OfText != nil {
				return block.OfText.Text
			}
		}
	}

	return ""
}

func TestPersistLargeToolResultWithConfig_Boundary(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cfg := ContextCompactConfig{
		PersistThreshold:   5,
		ToolResultPreview:  3,
		ToolResultBudget:   10,
		ToolResultBasePath: tempDir,
	}

	content := "abcde"
	got := PersistLargeToolResultWithConfig(ctx, "tool-1", content, cfg)
	if got != content {
		t.Fatalf("expected content unchanged at threshold, got: %q", got)
	}

	persistedFile := path.Join(tempDir, ".tiny-coding-agent", ".task_outputs", "tool-results", "tool-1.txt")
	if _, err := os.Stat(persistedFile); !os.IsNotExist(err) {
		t.Fatalf("expected no persisted file when len(content) == threshold, stat err=%v", err)
	}
}

func TestPersistLargeToolResultWithConfig_PersistsWhenOverThreshold(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cfg := ContextCompactConfig{
		PersistThreshold:   5,
		ToolResultPreview:  3,
		ToolResultBudget:   10,
		ToolResultBasePath: tempDir,
	}

	content := "abcdef"
	got := PersistLargeToolResultWithConfig(ctx, "tool-2", content, cfg)

	if !strings.Contains(got, "<persisted-output>") {
		t.Fatalf("expected persisted marker in output, got: %q", got)
	}
	if !strings.Contains(got, "Preview:\nabc") {
		t.Fatalf("expected preview content in output, got: %q", got)
	}

	persistedFile := path.Join(tempDir, ".tiny-coding-agent", ".task_outputs", "tool-results", "tool-2.txt")
	persistedContent, err := os.ReadFile(persistedFile)
	if err != nil {
		t.Fatalf("expected persisted file to exist: %v", err)
	}
	if string(persistedContent) != content {
		t.Fatalf("unexpected persisted content: got %q want %q", string(persistedContent), content)
	}
}

func TestToolResultBudgetWithConfig(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cfg := ContextCompactConfig{
		PersistThreshold:   5,
		ToolResultPreview:  3,
		ToolResultBudget:   4,
		ToolResultBasePath: tempDir,
	}

	t.Run("no changes when budget not exceeded", func(t *testing.T) {
		original := "abcd"
		messages := []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewToolResultBlock("id-stable", original, false),
			),
		}

		got, _ := ToolResultBudgetWithConfig(ctx, messages, cfg)
		if text := firstToolResultText(got[len(got)-1]); text != original {
			t.Fatalf("expected text unchanged when budget is not exceeded, got: %q", text)
		}
	})

	t.Run("persists when budget exceeded", func(t *testing.T) {
		original := "abcdef"
		messages := []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewToolResultBlock("id-budget", original, false),
			),
		}

		got, _ := ToolResultBudgetWithConfig(ctx, messages, cfg)
		text := firstToolResultText(got[len(got)-1])
		if !strings.Contains(text, "<persisted-output>") {
			t.Fatalf("expected persisted marker after budget compaction, got: %q", text)
		}
	})

	t.Run("ignores non-user last message", func(t *testing.T) {
		messages := []anthropic.MessageParam{
			anthropic.NewUserMessage(
				anthropic.NewToolResultBlock("id-user", "abcdef", false),
			),
			anthropic.NewAssistantMessage(
				anthropic.NewTextBlock("assistant reply"),
			),
		}

		got, _ := ToolResultBudgetWithConfig(ctx, messages, cfg)
		if len(got) != len(messages) {
			t.Fatalf("expected messages length unchanged, got %d want %d", len(got), len(messages))
		}
	})
}

func TestMicroContext(t *testing.T) {

}
