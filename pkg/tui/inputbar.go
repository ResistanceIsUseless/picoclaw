package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// InputBar handles user input at the bottom
type InputBar struct {
	input    string
	cursor   int
	focused  bool
	onSubmit func(string)
}

// NewInputBar creates a new input bar
func NewInputBar() *InputBar {
	return &InputBar{
		input:   "",
		cursor:  0,
		focused: true,
	}
}

// SetOnSubmit sets the callback for when input is submitted
func (i *InputBar) SetOnSubmit(fn func(string)) {
	i.onSubmit = fn
}

// Update handles messages
func (i *InputBar) Update(msg tea.Msg) (*InputBar, tea.Cmd) {
	if !i.focused {
		return i, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if len(strings.TrimSpace(i.input)) > 0 {
				if i.onSubmit != nil {
					i.onSubmit(i.input)
				}
				i.input = ""
				i.cursor = 0
			}

		case "backspace":
			if i.cursor > 0 {
				i.input = i.input[:i.cursor-1] + i.input[i.cursor:]
				i.cursor--
			}

		case "delete":
			if i.cursor < len(i.input) {
				i.input = i.input[:i.cursor] + i.input[i.cursor+1:]
			}

		case "left":
			if i.cursor > 0 {
				i.cursor--
			}

		case "right":
			if i.cursor < len(i.input) {
				i.cursor++
			}

		case "home", "ctrl+a":
			i.cursor = 0

		case "end", "ctrl+e":
			i.cursor = len(i.input)

		case "ctrl+u":
			// Clear line
			i.input = ""
			i.cursor = 0

		default:
			// Regular character input
			if len(msg.String()) == 1 {
				runes := []rune(i.input)
				before := string(runes[:i.cursor])
				after := string(runes[i.cursor:])
				i.input = before + msg.String() + after
				i.cursor++
			}
		}
	}

	return i, nil
}

// View renders the input bar
func (i *InputBar) View(width int) string {
	// Style definitions
	promptStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true)

	inputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15"))

	cursorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("86"))

	// Build prompt
	prompt := promptStyle.Render("› ")

	// Build input with cursor
	var displayInput string
	if i.focused && i.cursor < len(i.input) {
		// Show cursor
		before := i.input[:i.cursor]
		cursor := string(i.input[i.cursor])
		after := i.input[i.cursor+1:]
		displayInput = inputStyle.Render(before) +
			cursorStyle.Render(cursor) +
			inputStyle.Render(after)
	} else if i.focused && i.cursor == len(i.input) {
		// Cursor at end
		displayInput = inputStyle.Render(i.input) + cursorStyle.Render(" ")
	} else {
		// Not focused
		displayInput = inputStyle.Render(i.input)
	}

	// Combine
	line := prompt + displayInput

	// Pad to full width
	lineWidth := lipgloss.Width(prompt) + len(i.input)
	if i.focused {
		lineWidth++ // cursor
	}
	if lineWidth < width {
		line += strings.Repeat(" ", width-lineWidth)
	}

	return line
}

// SetFocused sets the focus state
func (i *InputBar) SetFocused(focused bool) {
	i.focused = focused
}

// Clear clears the input
func (i *InputBar) Clear() {
	i.input = ""
	i.cursor = 0
}
