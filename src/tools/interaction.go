package tools

type InteractionType string

const (
	InteractionTypeApproval InteractionType = "approval"
	InteractionTypeChoice   InteractionType = "choice"
)

type AgentInteractionRequest struct {
	ID      string
	Type    InteractionType
	Title   string
	Options []*AgentInteractionOption
}

type AgentInteractionOption struct {
	ID       string // option value
	Title    string
	Selected bool
}

type AgentInteractionResponse struct {
	RequestID string
	OptionID  string
	// Cancelled bool
}
