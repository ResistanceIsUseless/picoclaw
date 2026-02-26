package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// StatusBar displays current model, tier, and cost at the top
type StatusBar struct {
	model string
	tier  string
	cost  float64
}

// NewStatusBar creates a new status bar
func NewStatusBar() *StatusBar {
	return &StatusBar{
		model: "initializing...",
		tier:  "",
		cost:  0.0,
	}
}

// SetModel sets the current model and tier
func (s *StatusBar) SetModel(model, tier string) {
	s.model = model
	s.tier = tier
}

// SetCost sets the session cost
func (s *StatusBar) SetCost(cost float64) {
	s.cost = cost
}

// View renders the status bar
func (s *StatusBar) View(width int) string {
	// Style definitions
	statusStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)

	costStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("61")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)

	// Build status text
	modelText := fmt.Sprintf("Model: %s", s.model)
	if s.tier != "" {
		modelText = fmt.Sprintf("Model: %s [%s]", s.model, s.tier)
	}

	costText := fmt.Sprintf("Cost: $%.4f", s.cost)

	// Render components
	modelPart := statusStyle.Render(modelText)
	costPart := costStyle.Render(costText)

	// Calculate spacing
	usedWidth := lipgloss.Width(modelPart) + lipgloss.Width(costPart)
	spacing := strings.Repeat(" ", max(0, width-usedWidth))

	// Combine
	return modelPart + spacing + costPart
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
