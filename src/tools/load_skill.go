package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"tiny-coding-agent/pkg/utils"

	"github.com/anthropics/anthropic-sdk-go"
	"go.yaml.in/yaml/v4"
)

type Skill struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Path        string `json:"-"`
	Source      string `json:"-"`
}

var GlobalDir = "~/.tiny-coding-agent"
var (
	skillsMap = make(map[string]*Skill)
)

func LoadSkills(workDir string) (skillsPromptPart string, err error) {
	dirs := []struct {
		path     string
		source   string
		override bool
	}{
		{
			filepath.Join(workDir, ".claude/skills"),
			"project-claude",
			true,
		},

		{
			filepath.Join(workDir, ".tiny-coding-agent/skills"),
			"project-agent",
			true,
		},
		{
			utils.ExpandHome("~/.tiny-coding-agent/skills"),
			"global-agent",
			false,
		},
	}

	for _, dir := range dirs {
		if err := loadSkillDir(dir.path, dir.source, dir.override); err != nil {
			continue
		}
	}

	if len(skillsMap) == 0 {
		return "", nil
	}

	var builder strings.Builder
	builder.WriteString(
		"\n\nAvailable skills:\n",
	)
	for _, skill := range skillsMap {
		builder.WriteString(fmt.Sprintf("- **%s**: %s\n", skill.Name, skill.Description))
	}

	builder.WriteString("Use load_skill to get full details when needed.")

	return builder.String(), nil
}

func parseSkillFile(path string) (*Skill, error) {

	data, err := os.ReadFile(path)

	if err != nil {
		return nil, err
	}

	parts := strings.SplitN(string(data), "---", 3)

	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid SKILL.md: %s", path)
	}

	var skill Skill

	err = yaml.Unmarshal([]byte(parts[1]), &skill)

	if err != nil {
		return nil, err
	}

	if skill.Name == "" {
		return nil, fmt.Errorf("skill name missing: %s", path)
	}

	skill.Path = path
	return &skill, nil
}

func loadSkillDir(dir string, source string, override bool) error {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return nil
	}

	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		if info.Name() != "SKILL.md" {
			return nil
		}

		skill, err := parseSkillFile(path)
		if err != nil {
			return err
		}

		skill.Source = source

		if override || skillsMap[skill.Name] == nil {
			skillsMap[skill.Name] = skill
		}

		return nil
	})

}

var LoadSkill = ToolDefinition{
	Name:        "load_skill",
	Description: "Load the full content of a skill by name.",
	InputSchema: anthropic.ToolInputSchemaParam{
		Type: "object",
		Properties: map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "The name of the skill to load",
			},
		},
		Required: []string{"name"},
	},
	Execute: func(input json.RawMessage, rt *ToolRuntime) (string, error) {
		var params struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(input, &params); err != nil {
			return "", err
		}

		skill, ok := skillsMap[params.Name]
		if !ok {
			return "", fmt.Errorf("skill not found: %s", params.Name)
		}

		rt.Emit(ToolEvent{
			Type:    "info",
			Message: fmt.Sprintf("Loading skill: %s", skill.Name),
			Data:    nil,
		})

		data, err := os.ReadFile(skill.Path)
		if err != nil {
			return "", err
		}

		return string(data), nil
	},
}
