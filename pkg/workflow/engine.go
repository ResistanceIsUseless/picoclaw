package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/ResistanceIsUseless/picoclaw/pkg/logger"
)

// Engine manages workflow execution and state
type Engine struct {
	workflow  *Workflow
	state     *MissionState
	workspace string
	component string
}

// NewEngine creates a new workflow engine
func NewEngine(workflow *Workflow, target string, workspace string) *Engine {
	state := &MissionState{
		WorkflowName:   workflow.Name,
		Target:         target,
		StartTime:      time.Now(),
		CurrentPhase:   0,
		PhaseHistory:   make([]PhaseExecution, 0),
		ActiveBranches: make([]ActiveBranch, 0),
		Findings:       make([]Finding, 0),
		Metadata:       make(map[string]interface{}),
	}

	return &Engine{
		workflow:  workflow,
		state:     state,
		workspace: workspace,
		component: "workflow",
	}
}

// LoadEngine loads an existing workflow engine from state
func LoadEngine(workflow *Workflow, stateFile string, workspace string) (*Engine, error) {
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state MissionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state: %w", err)
	}

	return &Engine{
		workflow:  workflow,
		state:     &state,
		workspace: workspace,
		component: "workflow",
	}, nil
}

// GetContextPrompt returns markdown context to inject into system prompt
func (e *Engine) GetContextPrompt() string {
	if e.workflow == nil || e.state == nil {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("# Active Mission Context\n\n")
	sb.WriteString(fmt.Sprintf("**Workflow**: %s\n", e.workflow.Name))
	sb.WriteString(fmt.Sprintf("**Target**: %s\n", e.state.Target))
	sb.WriteString(fmt.Sprintf("**Started**: %s\n\n", e.state.StartTime.Format("2006-01-02 15:04:05")))

	// Current phase
	if e.state.CurrentPhase < len(e.workflow.Phases) {
		phase := e.workflow.Phases[e.state.CurrentPhase]
		sb.WriteString(fmt.Sprintf("## Current Phase: %s\n\n", phase.Name))

		// Steps
		exec := e.getCurrentPhaseExecution()
		if exec != nil {
			sb.WriteString("### Steps:\n")
			for _, step := range phase.Steps {
				status := "○"
				if e.isStepComplete(step.ID, exec) {
					status = "✓"
				}
				required := ""
				if step.Required {
					required = " (required)"
				}
				sb.WriteString(fmt.Sprintf("- %s %s%s\n", status, step.Name, required))
				if step.Description != "" {
					sb.WriteString(fmt.Sprintf("  %s\n", step.Description))
				}
			}
			sb.WriteString("\n")
		}

		// Completion criteria
		sb.WriteString(fmt.Sprintf("### Completion: %s\n", phase.Completion.Description))
		sb.WriteString("\n")

		// Possible branches
		if len(phase.Branches) > 0 {
			sb.WriteString("### Possible Branches:\n")
			for _, branch := range phase.Branches {
				sb.WriteString(fmt.Sprintf("- **%s**: %s\n", branch.Condition, branch.Description))
			}
			sb.WriteString("\n")
		}
	}

	// Active branches
	if len(e.state.ActiveBranches) > 0 {
		sb.WriteString("## Active Investigation Branches:\n")
		for _, branch := range e.state.ActiveBranches {
			status := "🔍 Active"
			if branch.CompletedAt != nil {
				status = "✓ Complete"
			}
			sb.WriteString(fmt.Sprintf("- **%s**: %s - %s\n", branch.Condition, branch.Description, status))
		}
		sb.WriteString("\n")
	}

	// Recent findings
	if len(e.state.Findings) > 0 {
		sb.WriteString(fmt.Sprintf("## Findings: %d total\n", len(e.state.Findings)))
		// Show last 3 findings
		count := len(e.state.Findings)
		start := 0
		if count > 3 {
			start = count - 3
		}
		for i := start; i < count; i++ {
			f := e.state.Findings[i]
			sb.WriteString(fmt.Sprintf("- [%s] %s\n", f.Severity, f.Title))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// MarkStepComplete marks a step as complete in the current phase
func (e *Engine) MarkStepComplete(stepID string) error {
	exec := e.getCurrentPhaseExecution()
	if exec == nil {
		return fmt.Errorf("no active phase execution")
	}

	// Check if already complete
	for _, id := range exec.StepsComplete {
		if id == stepID {
			return nil // Already complete
		}
	}

	exec.StepsComplete = append(exec.StepsComplete, stepID)

	logger.InfoCF(e.component, "Step complete", map[string]any{
		"phase": exec.PhaseName,
		"step":  stepID,
	})

	return e.SaveState()
}

// CreateBranch creates a new investigation branch
func (e *Engine) CreateBranch(condition, description string) error {
	branch := ActiveBranch{
		Condition:   condition,
		Description: description,
		CreatedAt:   time.Now(),
		Findings:    make([]Finding, 0),
	}

	e.state.ActiveBranches = append(e.state.ActiveBranches, branch)

	logger.InfoCF(e.component, "Branch created", map[string]any{
		"condition": condition,
		"description": description,
	})

	return e.SaveState()
}

// CompleteBranch marks a branch as complete
func (e *Engine) CompleteBranch(condition string) error {
	for i := range e.state.ActiveBranches {
		if e.state.ActiveBranches[i].Condition == condition {
			now := time.Now()
			e.state.ActiveBranches[i].CompletedAt = &now

			logger.InfoCF(e.component, "Branch completed", map[string]any{
				"condition": condition,
			})

			return e.SaveState()
		}
	}
	return fmt.Errorf("branch not found: %s", condition)
}

// AddFinding adds a finding to the mission
func (e *Engine) AddFinding(title, description string, severity Severity, evidence string) error {
	finding := Finding{
		ID:          uuid.New().String(),
		Title:       title,
		Description: description,
		Severity:    severity,
		Phase:       e.workflow.Phases[e.state.CurrentPhase].Name,
		CreatedAt:   time.Now(),
		Evidence:    evidence,
		Metadata:    make(map[string]interface{}),
	}

	e.state.Findings = append(e.state.Findings, finding)

	logger.InfoCF(e.component, "Finding added", map[string]any{
		"title":    title,
		"severity": severity,
		"phase":    finding.Phase,
	})

	return e.SaveState()
}

// AdvancePhase moves to the next phase
func (e *Engine) AdvancePhase() error {
	// Close current phase
	exec := e.getCurrentPhaseExecution()
	if exec != nil {
		now := time.Now()
		exec.EndTime = &now
	}

	// Move to next phase
	if e.state.CurrentPhase >= len(e.workflow.Phases)-1 {
		return fmt.Errorf("already at final phase")
	}

	e.state.CurrentPhase++

	// Create new phase execution
	e.startPhaseExecution()

	logger.InfoCF(e.component, "Phase advanced", map[string]any{
		"new_phase": e.workflow.Phases[e.state.CurrentPhase].Name,
		"phase_num": e.state.CurrentPhase,
	})

	return e.SaveState()
}

// IsPhaseComplete checks if current phase completion criteria are met
func (e *Engine) IsPhaseComplete() bool {
	if e.state.CurrentPhase >= len(e.workflow.Phases) {
		return false
	}

	phase := e.workflow.Phases[e.state.CurrentPhase]
	exec := e.getCurrentPhaseExecution()
	if exec == nil {
		return false
	}

	switch phase.Completion.Type {
	case CompletionAllRequired:
		// All required steps must be complete
		for _, step := range phase.Steps {
			if step.Required && !e.isStepComplete(step.ID, exec) {
				return false
			}
		}
		return true

	case CompletionAnyBranch:
		// At least one branch must be created
		return len(e.state.ActiveBranches) > 0

	case CompletionCustom:
		// Cannot auto-determine, return false
		return false

	default:
		return false
	}
}

// SaveState persists mission state to disk
func (e *Engine) SaveState() error {
	stateDir := filepath.Join(e.workspace, "missions")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create missions directory: %w", err)
	}

	// Sanitize target for filename
	safeName := strings.ReplaceAll(e.state.Target, "/", "_")
	safeName = strings.ReplaceAll(safeName, ":", "_")
	stateFile := filepath.Join(stateDir, fmt.Sprintf("%s_state.json", safeName))

	data, err := json.MarshalIndent(e.state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(stateFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// Helper methods

func (e *Engine) getCurrentPhaseExecution() *PhaseExecution {
	if len(e.state.PhaseHistory) == 0 {
		e.startPhaseExecution()
	}
	if len(e.state.PhaseHistory) > 0 {
		return &e.state.PhaseHistory[len(e.state.PhaseHistory)-1]
	}
	return nil
}

func (e *Engine) startPhaseExecution() {
	if e.state.CurrentPhase >= len(e.workflow.Phases) {
		return
	}

	phase := e.workflow.Phases[e.state.CurrentPhase]
	exec := PhaseExecution{
		PhaseName:     phase.Name,
		StartTime:     time.Now(),
		StepsComplete: make([]string, 0),
		Notes:         make([]string, 0),
	}

	e.state.PhaseHistory = append(e.state.PhaseHistory, exec)
}

func (e *Engine) isStepComplete(stepID string, exec *PhaseExecution) bool {
	for _, id := range exec.StepsComplete {
		if id == stepID {
			return true
		}
	}
	return false
}

// GetState returns the current mission state
func (e *Engine) GetState() *MissionState {
	return e.state
}

// GetWorkflow returns the workflow definition
func (e *Engine) GetWorkflow() *Workflow {
	return e.workflow
}
