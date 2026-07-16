package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	"tiny-coding-agent/src/agent"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/joho/godotenv"
)

type model struct {
	input      textarea.Model
	messages   []chatMessage
	width      int
	height     int
	inputLines int
	statusLine string
	waitingAI  bool
	lastCtrlC  time.Time
	ctrlCHits  int
	quitting   bool
}

type chatMessage struct {
	role    string // "user" or "assistant"
	content string
}

type agentResponseMsg string

type agentResponseClosedMsg struct{}

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
		response, ok := <-codingAgent.Response
		if !ok {
			return agentResponseClosedMsg{}
		}
		return agentResponseMsg(response)
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

	case agentResponseMsg:
		m.messages = append(m.messages, chatMessage{role: "assistant", content: string(msg)})
		m.waitingAI = false
		m.statusLine = "Enter to send · Shift+Enter/Ctrl+J for newline · Ctrl+C to quit"
		return m, waitAgentResponse()

	case agentResponseClosedMsg:
		m.statusLine = "AI response channel is closed"
		return m, nil

	case tea.KeyPressMsg:

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

	contentHeight := max(3, m.height-m.inputLines-6)
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

func renderChat(messages []chatMessage) string {
	var out []string
	for _, msg := range messages {
		label := "You"
		if msg.role == "assistant" {
			label = "AI"
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
	if len(out) > 40 {
		out = out[len(out)-40:]
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
