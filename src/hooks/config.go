package hooks

type Config struct {
	Hooks map[HookEvent][]HooksGroup `json:"hooks"`
}

type HooksGroup struct {
	Matcher string       `json:"matcher"`
	Hooks   []HookConfig `json:"hooks"`
}

type HookConfig struct {
	Type    string `json:"type"`
	Command string `json:"command,omitempty"`
}
