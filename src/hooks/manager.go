package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"tiny-coding-agent/src/tools"
)

type RegisteredHook struct {
	Matcher *regexp.Regexp
	Hook    Hook
}

type HookManager struct {
	Hooks   map[HookEvent][]RegisteredHook
	Runtime *tools.ToolRuntime
}

func NewHookManager(rt *tools.ToolRuntime) *HookManager {
	return &HookManager{
		Hooks:   make(map[HookEvent][]RegisteredHook),
		Runtime: rt,
	}
}

func compileMatcher(pattern string) (*regexp.Regexp, error) {

	parts := strings.Split(pattern, "|")
	for i, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.ToLower(p)
		parts[i] = p + "(_.*)?"
	}

	regex := "(?i)^(" + strings.Join(parts, "|") + ")$"

	return regexp.Compile(regex)
}

func (m *HookManager) RegisterHook(event HookEvent, matcher string, hook Hook) error {
	re, err := compileMatcher(matcher)
	if err != nil {
		return err
	}
	m.Hooks[event] = append(m.Hooks[event], RegisteredHook{
		Matcher: re,
		Hook:    hook,
	})
	return nil
}

func (m *HookManager) Execute(event HookEvent, ctx *HookToolContext) (*Result, error) {
	final := &Result{
		Decision: Allow,
		Input:    ctx.Input,
		Output:   ctx.Output,
	}
	for _, item := range m.Hooks[event] {
		if !item.Matcher.MatchString(ctx.ToolName) {
			continue
		}

		result, err := item.Hook.Run(ctx, m.Runtime)
		if err != nil {
			return final, err
		}

		if result == nil {
			continue
		}

		if result.Decision != Allow {
			final = result
			break
		}

		if result.Input != nil {
			ctx.Input = result.Input
			final.Input = result.Input
		}

		if result.Output != "" {
			ctx.Output = result.Output
			final.Output = result.Output
		}
	}

	return final, nil
}

func (m *HookManager) LoadHooks(ctx context.Context, workDir string) error {
	configPath := fmt.Sprintf("%s/.tiny-coding-agent/hooks.json", workDir)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return err
	}

	for eventName, groups := range config.Hooks {
		for _, group := range groups {
			for _, item := range group.Hooks {
				// https://code.claude.com/docs/en/hooks#hook-handler-fields
				// command,http,mcp_tool...
				switch item.Type {
				case "command":
					err := m.RegisterHook(eventName, group.Matcher, &CommandHook{
						Command: item.Command,
					})
					if err != nil {
						return err
					}
				}
			}

		}

	}
	return nil
}
