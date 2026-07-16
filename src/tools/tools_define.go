package tools

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

type ToolDefinition struct {
	Name        string                         `json:"name"`
	Description string                         `json:"description"`
	InputSchema anthropic.ToolInputSchemaParam `json:"input_schema"`
	Function    func(input json.RawMessage) (string, error)
}

func safePath(workdir, p string) (string, error) {
	filePath := path.Join(workdir, p)

	absPath := path.Clean(filePath)

	if !strings.HasPrefix(absPath, workdir) {
		return "", fmt.Errorf("path escapes workspace: %s", p)
	}

	return absPath, nil

}
