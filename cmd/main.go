package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	"tiny-coding-agent/src/agent"
	"tiny-coding-agent/src/tools"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/joho/godotenv"
)

type model struct {
	input         textarea.Model
	messages      []chatMessage
	selection     *tools.AgentInteractionOption
	selectionList list.Model
	width         int
	height        int
	inputLines    int
	statusLine    string
	waitingAI     bool
	lastCtrlC     time.Time
	ctrlCHits     int
	quitting      bool
}

type chatMessage struct {
	role    string // "user" or "assistant"
	content string
}

type agentResponseClosedMsg struct{}

type selectionItem struct {
	title     string
	value     string
	requestId string
}

func (i selectionItem) FilterValue() string {
	return i.title
}

func (i selectionItem) Title() string {
	return i.title
}

func (i selectionItem) Description() string {
	return ""
}

func initialModel() model {
	ti := textarea.New()
	ti.Placeholder = "Type a message... Enter to send, Shift+Enter/Ctrl+J for newline"
	ti.Prompt = "> "
	ti.Focus()
	ti.ShowLineNumbers = false
	ti.SetWidth(80)
	ti.SetHeight(1)
	ti.CharLimit = 2000

	return model{
		input:      ti,
		inputLines: 1,
		statusLine: "Enter to send · Shift+Enter/Ctrl+J for newline · Ctrl+C to quit",
	}
}

func waitAgentResponse() tea.Cmd {
	return func() tea.Msg {

		select {
		case event := <-codingAgent.InteractionRequest:
			return event
		case output, ok := <-codingAgent.Output:
			if !ok {
				return agentResponseClosedMsg{}
			}
			return output
		}

	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, waitAgentResponse())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeInput()

	case *agent.AgentOutput:

		if msg.Type != agent.AgentOutputTypeDebug {
			m.messages = append(m.messages, chatMessage{role: "assistant", content: msg.Message})
		} else {
			m.messages = append(m.messages, chatMessage{role: "DEBUG", content: msg.Message})
		}

		m.waitingAI = false
		m.statusLine = "Enter to send · Shift+Enter/Ctrl+J for newline · Ctrl+C to quit"
		return m, waitAgentResponse()

	case *tools.AgentInteractionRequest:
		m.selection = msg.Options[0] // Assuming the first option is the default selection
		m.selectionList = buildSelectionList(msg, m.width)
		m.waitingAI = false
		m.statusLine = "Use Up/Down to choose, Enter to submit"
		return m, waitAgentResponse()

	case agentResponseClosedMsg:
		m.statusLine = "AI response channel is closed"
		return m, nil

	case tea.KeyPressMsg:
		if m.selection != nil {
			switch msg.String() {
			case "ctrl+c":
				now := time.Now()
				if now.Sub(m.lastCtrlC) <= 900*time.Millisecond {
					m.ctrlCHits++
				} else {
					m.ctrlCHits = 1
				}
				m.lastCtrlC = now

				if m.ctrlCHits >= 2 {
					m.quitting = true
					return m, tea.Quit
				}
				m.statusLine = "Press Ctrl+C again to quit"
				return m, nil
			case "enter":
				selectedItem, ok := m.selectionList.SelectedItem().(selectionItem)
				if !ok {
					m.statusLine = "No available options"
					return m, nil
				}

				answer := selectedItem.value
				if strings.TrimSpace(answer) == "" {
					answer = selectedItem.title
				}

				codingAgent.InteractionResponse <- &tools.AgentInteractionResponse{
					OptionID:  answer,
					RequestID: selectedItem.requestId,
				}
				m.statusLine = "user selected " + compactPreview(answer)
				m.selection = nil
				m.selectionList = list.Model{}
				m.waitingAI = true
				m.ctrlCHits = 0
				return m, nil
			}

			m.ctrlCHits = 0
			m.selectionList, cmd = m.selectionList.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		// newline
		case "shift+enter", "alt+enter", "ctrl+j":
			m.input.InsertString("\n")
			m.resizeInput()
			m.ctrlCHits = 0
			return m, nil
		case "ctrl+c":
			now := time.Now()
			if now.Sub(m.lastCtrlC) <= 900*time.Millisecond {
				m.ctrlCHits++
			} else {
				m.ctrlCHits = 1
			}
			m.lastCtrlC = now

			if m.ctrlCHits >= 2 {
				m.quitting = true
				return m, tea.Quit
			}
			m.statusLine = "Press Ctrl+C again to quit"
			return m, nil

		case "enter":
			// send message
			value := strings.TrimSpace(m.input.Value())
			if value == "" {
				m.statusLine = "Input is empty, please type a message"
				return m, nil
			}

			um := anthropic.NewUserMessage(anthropic.NewTextBlock(value))
			codingAgent.UserMessage <- &um
			m.messages = append(m.messages, chatMessage{role: "user", content: value})
			m.input.SetValue("")
			m.input.SetHeight(1)
			m.resizeInput()
			m.waitingAI = true
			m.statusLine = "waiting for AI response..."
			m.ctrlCHits = 0
			return m, nil

		default:
			m.ctrlCHits = 0
		}
	}

	m.input, cmd = m.input.Update(msg)
	m.resizeInput()

	return m, cmd
}

func (m model) View() tea.View {
	v := tea.NewView("")
	v.AltScreen = true

	if m.quitting {
		v.SetContent("\nBye!\n")
		return v
	}

	inputStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#888888")).
		Padding(0, 1)

	contentStyle := lipgloss.NewStyle().Padding(0, 1)

	composerInnerHeight := m.inputLines
	if m.selection != nil {

		composerInnerHeight = selectionListHeightForWindow(m.height)
	}

	contentHeight := max(3, m.height-composerInnerHeight-6)
	body := ""
	if len(m.messages) == 0 {
		body = contentStyle.Height(contentHeight).Render(
			lipgloss.NewStyle().MarginLeft(1).Render(
				"Welcome to tiny coding agent\n\n" +
					"Type a message and press Enter to send.\n" +
					"Shift+Enter for a new line.",
			),
		)
	} else {
		body = contentStyle.Height(contentHeight).Render(renderChat(m.messages))
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Align(lipgloss.Center).
		MarginBottom(1)

	composer := inputStyle.Render(m.input.View())
	if m.selection != nil {
		composer = inputStyle.Render(m.selectionList.View())
	}

	v.SetContent(lipgloss.JoinVertical(
		lipgloss.Left,
		headerStyle.Render("tiny coding agent"),
		body,
		lipgloss.NewStyle().MarginLeft(0).Render(composer),
		m.statusLine,
	))
	return v
}

func (m *model) resizeInput() {
	if m.width > 0 {
		m.input.SetWidth(max(8, m.width-4))
		if m.selection != nil {
			m.selectionList.SetSize(max(8, m.width-8), selectionListHeightForWindow(m.height))
		}
	}
	lines := strings.Count(m.input.Value(), "\n") + 1
	if lines < 1 {
		lines = 1
	}
	m.inputLines = min(lines, 12)
	m.input.SetHeight(m.inputLines)
}

func compactPreview(value string) string {
	oneLine := strings.ReplaceAll(value, "\n", " · ")
	if len(oneLine) <= 80 {
		return oneLine
	}
	return oneLine[:80] + "..."
}

func defaultSelectedIndex(req *tools.AgentInteractionRequest) int {
	for idx, opt := range req.Options {
		if opt != nil && opt.Selected {
			return idx
		}
	}
	return 0
}

func buildSelectionList(req *tools.AgentInteractionRequest, width int) list.Model {
	items := make([]list.Item, 0, len(req.Options))
	for _, opt := range req.Options {
		if opt == nil {
			continue
		}
		items = append(items, selectionItem{title: opt.Title, value: opt.ID, requestId: req.ID})
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = true
	selectionList := list.New(items, delegate, max(8, width-8), selectionListHeightForWindow(0))
	selectionList.Title = req.Title
	selectionList.SetShowHelp(false)
	selectionList.SetShowStatusBar(false)
	selectionList.SetFilteringEnabled(false)
	selectionList.DisableQuitKeybindings()
	selectionList.Select(defaultSelectedIndex(req))
	return selectionList
}

func selectionListHeightForWindow(windowHeight int) int {
	const defaultHeight = 10
	if windowHeight <= 0 {
		return defaultHeight
	}

	// Reserve room for: header, status line and at least 3 lines of chat content.
	maxHeight := windowHeight - 9
	if maxHeight < 3 {
		return 3
	}
	return min(defaultHeight, maxHeight)
}

func renderChat(messages []chatMessage) string {
	var out []string
	for _, msg := range messages {
		label := "You"

		switch msg.role {
		case "user":
			label = "You"
		case "assistant":
			label = "AI"
		case "DEBUG":
			label = "DEBUG"
		}

		lines := strings.Split(msg.content, "\n")
		for i, line := range lines {
			if i == 0 {
				out = append(out, fmt.Sprintf("[%s] %s", label, line))
			} else {
				out = append(out, strings.Repeat(" ", len(label)+3)+line)
			}
		}
	}
	if len(out) > 30 {
		out = out[len(out)-30:]
	}
	return strings.Join(out, "\n")
}

var codingAgent *agent.Agent

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	anthropicBaseUrl := os.Getenv("ANTHROPIC_BASE_URL")
	if anthropicBaseUrl == "" {
		fmt.Fprintf(os.Stderr, "ANTHROPIC_BASE_URL is not set in .env\n")
		os.Exit(1)
	}

	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Fprintf(os.Stderr, "ANTHROPIC_API_KEY is not set in .env\n")
		os.Exit(1)
	}
	codingAgent = agent.NewAgent(apiKey, anthropicBaseUrl)

	go func() {
		err := codingAgent.Run(context.Background())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Agent run failed: %v\n", err)
			return
		}

	}()

	model := initialModel()

	p := tea.NewProgram(model)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Program failed to start: %v\n", err)
		os.Exit(1)
	}
}
