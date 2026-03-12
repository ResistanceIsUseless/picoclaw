package tui

import (
	"fmt"
	"strings"

	"github.com/ResistanceIsUseless/picoclaw/pkg/workflow"
	"github.com/charmbracelet/lipgloss"
)

// MissionView displays workflow/mission state
type MissionView struct {
	engine *workflow.Engine
}

// NewMissionView creates a new mission view
func NewMissionView() *MissionView {
	return &MissionView{}
}

// Update updates the mission view with new workflow state
func (m *MissionView) Update(engine *workflow.Engine) {
	m.engine = engine
}

// View renders the mission view
func (m *MissionView) View(width, height int) string {
	if m.engine == nil {
		emptyStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(1, 1)
		return emptyStyle.Render("No active mission")
	}

	wf := m.engine.GetWorkflow()
	state := m.engine.GetState()

	// Style definitions
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("170")).
		Bold(true).
		Underline(true)

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true)

	pendingStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	criticalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	highStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("208"))

	mediumStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("226"))

	lowStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244"))

	var lines []string

	// Mission header
	lines = append(lines, titleStyle.Render("┏━ MISSION ━━━━━━━━━━━━━━━"))
	lines = append(lines, fmt.Sprintf("┃ %s", wf.Name))
	lines = append(lines, fmt.Sprintf("┃ Target: %s", state.Target))
	lines = append(lines, fmt.Sprintf("┃ Started: %s", state.StartTime.Format("15:04:05")))
	lines = append(lines, "┗━━━━━━━━━━━━━━━━━━━━━━━━")
	lines = append(lines, "")

	// Current phase
	if state.CurrentPhase < len(wf.Phases) {
		phase := wf.Phases[state.CurrentPhase]
		lines = append(lines, headerStyle.Render(fmt.Sprintf("Phase %d/%d: %s", state.CurrentPhase+1, len(wf.Phases), phase.Name)))
		lines = append(lines, "")

		// Get current phase execution
		var exec *workflow.PhaseExecution
		if len(state.PhaseHistory) > 0 {
			exec = &state.PhaseHistory[len(state.PhaseHistory)-1]
		}

		if next := nextActionableStep(phase, exec); next != nil {
			required := ""
			if next.Required {
				required = " *"
			}
			lines = append(lines, "Next Action:")
			lines = append(lines, pendingStyle.Render(fmt.Sprintf("  → %s%s", next.Name, required)))
			if next.Description != "" {
				desc := next.Description
				if len(desc) > 70 {
					desc = desc[:67] + "..."
				}
				lines = append(lines, fmt.Sprintf("    %s", desc))
			}
		}
		lines = append(lines, "")

		remainingRequired, remainingOptional := remainingStepCounts(phase, exec)
		lines = append(lines, "Progress:")
		lines = append(lines, fmt.Sprintf("  Required remaining: %d", remainingRequired))
		lines = append(lines, fmt.Sprintf("  Optional remaining: %d", remainingOptional))
		lines = append(lines, "")

		// Completion criteria
		lines = append(lines, "Completion:")
		lines = append(lines, fmt.Sprintf("  %s", phase.Completion.Description))
		lines = append(lines, "")

		// Branches
		if len(phase.Branches) > 0 {
			lines = append(lines, "Possible Branches:")
			for _, branch := range phase.Branches {
				lines = append(lines, fmt.Sprintf("  • %s", branch.Condition))
				if len(branch.Description) > 0 && len(branch.Description) < 40 {
					lines = append(lines, fmt.Sprintf("    %s", branch.Description))
				}
			}
			lines = append(lines, "")
		}
	}

	// Active branches
	if len(state.ActiveBranches) > 0 {
		lines = append(lines, headerStyle.Render("Active Branches:"))
		for _, branch := range state.ActiveBranches {
			status := "🔍"
			if branch.CompletedAt != nil {
				status = "✓"
			}
			line := fmt.Sprintf("  %s %s", status, branch.Condition)
			lines = append(lines, line)
		}
		lines = append(lines, "")
	}

	// Findings summary
	if len(state.Findings) > 0 {
		lines = append(lines, headerStyle.Render(fmt.Sprintf("Findings: %d", len(state.Findings))))

		// Count by severity
		criticalCount := 0
		highCount := 0
		mediumCount := 0
		lowCount := 0
		infoCount := 0

		for _, finding := range state.Findings {
			switch finding.Severity {
			case workflow.SeverityCritical:
				criticalCount++
			case workflow.SeverityHigh:
				highCount++
			case workflow.SeverityMedium:
				mediumCount++
			case workflow.SeverityLow:
				lowCount++
			case workflow.SeverityInformational:
				infoCount++
			}
		}

		if criticalCount > 0 {
			lines = append(lines, criticalStyle.Render(fmt.Sprintf("  ● Critical: %d", criticalCount)))
		}
		if highCount > 0 {
			lines = append(lines, highStyle.Render(fmt.Sprintf("  ● High: %d", highCount)))
		}
		if mediumCount > 0 {
			lines = append(lines, mediumStyle.Render(fmt.Sprintf("  ● Medium: %d", mediumCount)))
		}
		if lowCount > 0 {
			lines = append(lines, lowStyle.Render(fmt.Sprintf("  ● Low: %d", lowCount)))
		}
		if infoCount > 0 {
			lines = append(lines, fmt.Sprintf("  ● Info: %d", infoCount))
		}

		// Show last 3 findings
		lines = append(lines, "")
		lines = append(lines, "Recent:")
		start := max(0, len(state.Findings)-3)
		for i := start; i < len(state.Findings); i++ {
			f := state.Findings[i]
			var severityLabel string
			switch f.Severity {
			case workflow.SeverityCritical:
				severityLabel = criticalStyle.Render(fmt.Sprintf("[%s]", f.Severity))
			case workflow.SeverityHigh:
				severityLabel = highStyle.Render(fmt.Sprintf("[%s]", f.Severity))
			case workflow.SeverityMedium:
				severityLabel = mediumStyle.Render(fmt.Sprintf("[%s]", f.Severity))
			case workflow.SeverityLow:
				severityLabel = lowStyle.Render(fmt.Sprintf("[%s]", f.Severity))
			default:
				severityLabel = fmt.Sprintf("[%s]", f.Severity)
			}

			title := f.Title
			if len(title) > 30 {
				title = title[:27] + "..."
			}

			lines = append(lines, fmt.Sprintf("  %s %s", severityLabel, title))
		}
	}

	// Truncate to height
	if len(lines) > height {
		lines = lines[:height-1]
		lines = append(lines, "...")
	}

	return strings.Join(lines, "\n")
}

func nextActionableStep(phase workflow.Phase, exec *workflow.PhaseExecution) *workflow.Step {
	for i := range phase.Steps {
		step := &phase.Steps[i]
		if step.Required && !isStepComplete(step.ID, exec) {
			return step
		}
	}
	for i := range phase.Steps {
		step := &phase.Steps[i]
		if !isStepComplete(step.ID, exec) {
			return step
		}
	}
	return nil
}

func remainingStepCounts(phase workflow.Phase, exec *workflow.PhaseExecution) (int, int) {
	remainingRequired := 0
	remainingOptional := 0
	for _, step := range phase.Steps {
		if isStepComplete(step.ID, exec) {
			continue
		}
		if step.Required {
			remainingRequired++
		} else {
			remainingOptional++
		}
	}
	return remainingRequired, remainingOptional
}

func isStepComplete(stepID string, exec *workflow.PhaseExecution) bool {
	if exec == nil {
		return false
	}
	for _, completedID := range exec.StepsComplete {
		if completedID == stepID {
			return true
		}
	}
	return false
}
