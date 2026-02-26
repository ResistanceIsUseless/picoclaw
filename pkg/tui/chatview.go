package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/glamour"
)

// ChatView displays the conversation history
type ChatView struct {
	messages []ChatMessageMsg
	scroll   int
	renderer *glamour.TermRenderer
}

// NewChatView creates a new chat view
func NewChatView() *ChatView {
	// Create markdown renderer
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(80),
	)

	return &ChatView{
		messages: make([]ChatMessageMsg, 0),
		scroll:   0,
		renderer: renderer,
	}
}

// AddMessage adds a message to the chat
func (c *ChatView) AddMessage(msg ChatMessageMsg) {
	c.messages = append(c.messages, msg)
	// Auto-scroll to bottom
	c.scroll = len(c.messages)
}

// Update handles messages
func (c *ChatView) Update(msg tea.Msg) (*ChatView, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if c.scroll > 0 {
				c.scroll--
			}
		case "down", "j":
			if c.scroll < len(c.messages) {
				c.scroll++
			}
		case "pgup":
			c.scroll = max(0, c.scroll-10)
		case "pgdn":
			c.scroll = min(len(c.messages), c.scroll+10)
		case "home":
			c.scroll = 0
		case "end":
			c.scroll = len(c.messages)
		}
	}
	return c, nil
}

// View renders the chat view
func (c *ChatView) View(width, height int) string {
	if len(c.messages) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(1, 2)
		return emptyStyle.Render("No messages yet. Start chatting!")
	}

	// Style definitions
	userStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true)

	assistantStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("170")).
		Bold(true)

	toolStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)

	systemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true)

	timestampStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	// Render messages
	var lines []string

	// Determine visible range
	visibleStart := max(0, c.scroll-height)
	visibleEnd := min(len(c.messages), visibleStart+height)

	for i := visibleStart; i < visibleEnd; i++ {
		msg := c.messages[i]

		// Role label
		var roleLabel string
		var roleStyle lipgloss.Style

		switch msg.Role {
		case "user":
			roleLabel = "You"
			roleStyle = userStyle
		case "assistant":
			roleLabel = "Assistant"
			roleStyle = assistantStyle
		case "tool":
			roleLabel = fmt.Sprintf("Tool: %s", msg.ToolName)
			roleStyle = toolStyle
		case "system":
			roleLabel = "System"
			roleStyle = systemStyle
		default:
			roleLabel = msg.Role
			roleStyle = systemStyle
		}

		// Timestamp
		timestamp := msg.Timestamp.Format("15:04:05")
		timestampStr := timestampStyle.Render(timestamp)

		// Header line
		header := fmt.Sprintf("%s %s", roleStyle.Render(roleLabel), timestampStr)
		lines = append(lines, header)

		// Message content
		// Try to render markdown for assistant messages
		if msg.Role == "assistant" && c.renderer != nil {
			rendered, err := c.renderer.Render(msg.Content)
			if err == nil {
				lines = append(lines, strings.TrimSpace(rendered))
			} else {
				lines = append(lines, msg.Content)
			}
		} else {
			// Plain text for other messages
			contentLines := strings.Split(msg.Content, "\n")
			for _, line := range contentLines {
				if len(line) > width-4 {
					// Word wrap
					wrapped := wordWrap(line, width-4)
					lines = append(lines, wrapped...)
				} else {
					lines = append(lines, line)
				}
			}
		}

		// Spacing between messages
		lines = append(lines, "")
	}

	// Scroll indicator
	if c.scroll < len(c.messages) {
		scrollText := fmt.Sprintf("▼ %d more messages", len(c.messages)-c.scroll)
		scrollStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Align(lipgloss.Right)
		lines = append(lines, scrollStyle.Width(width).Render(scrollText))
	}

	return strings.Join(lines, "\n")
}

// wordWrap wraps text to the specified width
func wordWrap(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	var lines []string
	var currentLine strings.Builder

	words := strings.Fields(text)
	for i, word := range words {
		if i > 0 {
			// Check if adding this word would exceed width
			if currentLine.Len()+1+len(word) > width {
				lines = append(lines, currentLine.String())
				currentLine.Reset()
				currentLine.WriteString(word)
			} else {
				currentLine.WriteString(" ")
				currentLine.WriteString(word)
			}
		} else {
			currentLine.WriteString(word)
		}
	}

	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return lines
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
