package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sipeed/picoclaw/pkg/routing"
	"github.com/sipeed/picoclaw/pkg/workflow"
)

// Model is the main TUI application model
type Model struct {
	width  int
	height int

	// Sub-components
	statusBar   *StatusBar
	chatView    *ChatView
	missionView *MissionView
	inputBar    *InputBar

	// Current state
	currentModel    string
	currentTier     string
	sessionCost     float64
	workflowEngine  *workflow.Engine
	tierRouter      *routing.TierRouter

	// Layout
	showMissionPanel bool
	focusedView      string // "chat" or "input"
}

// NewModel creates a new TUI model
func NewModel() *Model {
	return &Model{
		statusBar:        NewStatusBar(),
		chatView:         NewChatView(),
		missionView:      NewMissionView(),
		inputBar:         NewInputBar(),
		showMissionPanel: false,
		focusedView:      "input",
	}
}

// Init initializes the TUI
func (m *Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "ctrl+m":
			m.showMissionPanel = !m.showMissionPanel
			m.updateLayout()
		case "tab":
			if m.focusedView == "chat" {
				m.focusedView = "input"
			} else {
				m.focusedView = "chat"
			}
		}

	case ModelSwitchMsg:
		m.currentModel = msg.Model
		m.currentTier = msg.Tier
		m.statusBar.SetModel(msg.Model, msg.Tier)

	case CostUpdateMsg:
		m.sessionCost = msg.Total
		m.statusBar.SetCost(msg.Total)

	case ChatMessageMsg:
		m.chatView.AddMessage(msg)

	case WorkflowUpdateMsg:
		if m.workflowEngine != nil {
			m.missionView.Update(m.workflowEngine)
		}
	}

	// Update sub-components
	var cmd tea.Cmd
	if m.focusedView == "input" {
		_, cmd = m.inputBar.Update(msg)
		cmds = append(cmds, cmd)
	} else {
		_, cmd = m.chatView.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the TUI
func (m *Model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	var sections []string

	// Status bar at top
	sections = append(sections, m.statusBar.View(m.width))

	// Main content area
	contentHeight := m.height - 3 // Reserve space for status bar and input bar

	if m.showMissionPanel {
		// Split view: chat on left, mission panel on right
		chatWidth := m.width * 2 / 3
		missionWidth := m.width - chatWidth - 1

		chatContent := m.chatView.View(chatWidth, contentHeight-2)
		missionContent := m.missionView.View(missionWidth, contentHeight-2)

		// Combine horizontally
		chatLines := strings.Split(chatContent, "\n")
		missionLines := strings.Split(missionContent, "\n")

		maxLines := len(chatLines)
		if len(missionLines) > maxLines {
			maxLines = len(missionLines)
		}

		for i := 0; i < maxLines; i++ {
			var chatLine, missionLine string
			if i < len(chatLines) {
				chatLine = chatLines[i]
			}
			if i < len(missionLines) {
				missionLine = missionLines[i]
			}

			// Pad chat line to full width
			chatLine = chatLine + strings.Repeat(" ", chatWidth-lipgloss.Width(chatLine))

			sections = append(sections, chatLine+"│"+missionLine)
		}
	} else {
		// Full width chat view
		sections = append(sections, m.chatView.View(m.width, contentHeight-2))
	}

	// Input bar at bottom
	sections = append(sections, m.inputBar.View(m.width))

	return strings.Join(sections, "\n")
}

// updateLayout recalculates component sizes based on window size
func (m *Model) updateLayout() {
	// Components will use sizes passed in View() calls
}

// SetWorkflowEngine sets the workflow engine for mission tracking
func (m *Model) SetWorkflowEngine(engine *workflow.Engine) {
	m.workflowEngine = engine
	if engine != nil {
		m.showMissionPanel = true
		m.missionView.Update(engine)
	}
}

// SetTierRouter sets the tier router for cost tracking
func (m *Model) SetTierRouter(router *routing.TierRouter) {
	m.tierRouter = router
}

// Custom message types

// ModelSwitchMsg indicates the LLM model changed
type ModelSwitchMsg struct {
	Model string
	Tier  string
}

// CostUpdateMsg indicates session cost updated
type CostUpdateMsg struct {
	Total float64
}

// ChatMessageMsg represents a chat message to display
type ChatMessageMsg struct {
	Role      string // "user", "assistant", "tool"
	Content   string
	Timestamp time.Time
	ToolName  string // For tool messages
}

// WorkflowUpdateMsg indicates workflow state changed
type WorkflowUpdateMsg struct{}

// Helper to send messages to the TUI
func SendModelSwitch(model, tier string) tea.Msg {
	return ModelSwitchMsg{Model: model, Tier: tier}
}

func SendCostUpdate(total float64) tea.Msg {
	return CostUpdateMsg{Total: total}
}

func SendChatMessage(role, content, toolName string) tea.Msg {
	return ChatMessageMsg{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
		ToolName:  toolName,
	}
}

func SendWorkflowUpdate() tea.Msg {
	return WorkflowUpdateMsg{}
}

// Program wraps the tea.Program for easy integration
type Program struct {
	program *tea.Program
	model   *Model
}

// NewProgram creates a new TUI program
func NewProgram() *Program {
	model := NewModel()
	program := tea.NewProgram(model, tea.WithAltScreen())

	return &Program{
		program: program,
		model:   model,
	}
}

// Run starts the TUI
func (p *Program) Run() error {
	_, err := p.program.Run()
	return err
}

// Send sends a message to the TUI
func (p *Program) Send(msg tea.Msg) {
	p.program.Send(msg)
}

// SetWorkflowEngine sets the workflow engine
func (p *Program) SetWorkflowEngine(engine *workflow.Engine) {
	p.model.SetWorkflowEngine(engine)
}

// SetTierRouter sets the tier router
func (p *Program) SetTierRouter(router *routing.TierRouter) {
	p.model.SetTierRouter(router)
}

// Quit quits the TUI
func (p *Program) Quit() {
	p.program.Quit()
}

// Printf sends a formatted message to the chat
func (p *Program) Printf(format string, args ...interface{}) {
	content := fmt.Sprintf(format, args...)
	p.Send(SendChatMessage("system", content, ""))
}
