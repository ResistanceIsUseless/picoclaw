package tools

import (
	"context"
	"fmt"

	"github.com/sipeed/picoclaw/pkg/workflow"
)

// WorkflowStepCompleteTool allows marking workflow steps as complete
type WorkflowStepCompleteTool struct {
	getEngine func() *workflow.Engine
}

func NewWorkflowStepCompleteTool(getEngine func() *workflow.Engine) *WorkflowStepCompleteTool {
	return &WorkflowStepCompleteTool{getEngine: getEngine}
}

func (t *WorkflowStepCompleteTool) Name() string {
	return "workflow_step_complete"
}

func (t *WorkflowStepCompleteTool) Description() string {
	return "Mark a workflow step as complete. Use this when you have finished a step in the current mission phase."
}

func (t *WorkflowStepCompleteTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"step_id": map[string]any{
				"type":        "string",
				"description": "The ID of the step to mark complete (from the workflow context)",
			},
		},
		"required": []string{"step_id"},
	}
}

func (t *WorkflowStepCompleteTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	engine := t.getEngine()
	if engine == nil {
		return NewToolResult("No active mission/workflow")
	}

	stepID, ok := args["step_id"].(string)
	if !ok {
		return NewToolResult("Missing or invalid step_id parameter")
	}

	if err := engine.MarkStepComplete(stepID); err != nil {
		return NewToolResult(fmt.Sprintf("Failed to mark step complete: %v", err))
	}

	return NewToolResult(fmt.Sprintf("Step '%s' marked complete", stepID))
}

// WorkflowCreateBranchTool allows creating investigation branches
type WorkflowCreateBranchTool struct {
	getEngine func() *workflow.Engine
}

func NewWorkflowCreateBranchTool(getEngine func() *workflow.Engine) *WorkflowCreateBranchTool {
	return &WorkflowCreateBranchTool{getEngine: getEngine}
}

func (t *WorkflowCreateBranchTool) Name() string {
	return "workflow_create_branch"
}

func (t *WorkflowCreateBranchTool) Description() string {
	return "Create a new investigation branch when you discover something that requires deeper exploration (e.g., found web service, discovered vulnerability, etc.)"
}

func (t *WorkflowCreateBranchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"condition": map[string]any{
				"type":        "string",
				"description": "The condition/trigger for this branch (e.g., 'web_service_found', 'smb_discovered')",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Description of what this branch will investigate",
			},
		},
		"required": []string{"condition", "description"},
	}
}

func (t *WorkflowCreateBranchTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	engine := t.getEngine()
	if engine == nil {
		return NewToolResult("No active mission/workflow")
	}

	condition, ok := args["condition"].(string)
	if !ok {
		return NewToolResult("Missing or invalid condition parameter")
	}

	description, ok := args["description"].(string)
	if !ok {
		return NewToolResult("Missing or invalid description parameter")
	}

	if err := engine.CreateBranch(condition, description); err != nil {
		return NewToolResult(fmt.Sprintf("Failed to create branch: %v", err))
	}

	return NewToolResult(fmt.Sprintf("Created branch: %s - %s", condition, description))
}

// WorkflowCompleteBranchTool allows marking branches as complete
type WorkflowCompleteBranchTool struct {
	getEngine func() *workflow.Engine
}

func NewWorkflowCompleteBranchTool(getEngine func() *workflow.Engine) *WorkflowCompleteBranchTool {
	return &WorkflowCompleteBranchTool{getEngine: getEngine}
}

func (t *WorkflowCompleteBranchTool) Name() string {
	return "workflow_complete_branch"
}

func (t *WorkflowCompleteBranchTool) Description() string {
	return "Mark an investigation branch as complete when you have finished exploring it."
}

func (t *WorkflowCompleteBranchTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"condition": map[string]any{
				"type":        "string",
				"description": "The condition of the branch to complete",
			},
		},
		"required": []string{"condition"},
	}
}

func (t *WorkflowCompleteBranchTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	engine := t.getEngine()
	if engine == nil {
		return NewToolResult("No active mission/workflow")
	}

	condition, ok := args["condition"].(string)
	if !ok {
		return NewToolResult("Missing or invalid condition parameter")
	}

	if err := engine.CompleteBranch(condition); err != nil {
		return NewToolResult(fmt.Sprintf("Failed to complete branch: %v", err))
	}

	return NewToolResult(fmt.Sprintf("Branch '%s' marked complete", condition))
}

// WorkflowAddFindingTool allows recording findings
type WorkflowAddFindingTool struct {
	getEngine func() *workflow.Engine
}

func NewWorkflowAddFindingTool(getEngine func() *workflow.Engine) *WorkflowAddFindingTool {
	return &WorkflowAddFindingTool{getEngine: getEngine}
}

func (t *WorkflowAddFindingTool) Name() string {
	return "workflow_add_finding"
}

func (t *WorkflowAddFindingTool) Description() string {
	return "Record a security finding or discovery in the mission report. Use this when you find vulnerabilities, misconfigurations, or other notable security issues."
}

func (t *WorkflowAddFindingTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title": map[string]any{
				"type":        "string",
				"description": "Title of the finding",
			},
			"description": map[string]any{
				"type":        "string",
				"description": "Detailed description of the finding",
			},
			"severity": map[string]any{
				"type":        "string",
				"description": "Severity level: critical, high, medium, low, or info",
				"enum":        []string{"critical", "high", "medium", "low", "info"},
			},
			"evidence": map[string]any{
				"type":        "string",
				"description": "Evidence or proof (tool output, logs, etc.)",
			},
		},
		"required": []string{"title", "description", "severity", "evidence"},
	}
}

func (t *WorkflowAddFindingTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	engine := t.getEngine()
	if engine == nil {
		return NewToolResult("No active mission/workflow")
	}

	title, ok := args["title"].(string)
	if !ok {
		return NewToolResult("Missing or invalid title parameter")
	}

	description, ok := args["description"].(string)
	if !ok {
		return NewToolResult("Missing or invalid description parameter")
	}

	severityStr, ok := args["severity"].(string)
	if !ok {
		return NewToolResult("Missing or invalid severity parameter")
	}

	evidence, ok := args["evidence"].(string)
	if !ok {
		return NewToolResult("Missing or invalid evidence parameter")
	}

	// Convert severity string to enum
	var severity workflow.Severity
	switch severityStr {
	case "critical":
		severity = workflow.SeverityCritical
	case "high":
		severity = workflow.SeverityHigh
	case "medium":
		severity = workflow.SeverityMedium
	case "low":
		severity = workflow.SeverityLow
	case "info", "informational":
		severity = workflow.SeverityInformational
	default:
		return NewToolResult(fmt.Sprintf("Invalid severity: %s", severityStr))
	}

	if err := engine.AddFinding(title, description, severity, evidence); err != nil {
		return NewToolResult(fmt.Sprintf("Failed to add finding: %v", err))
	}

	return NewToolResult(fmt.Sprintf("Added %s finding: %s", severityStr, title))
}

// WorkflowAdvancePhaseTool allows advancing to the next phase
type WorkflowAdvancePhaseTool struct {
	getEngine func() *workflow.Engine
}

func NewWorkflowAdvancePhaseTool(getEngine func() *workflow.Engine) *WorkflowAdvancePhaseTool {
	return &WorkflowAdvancePhaseTool{getEngine: getEngine}
}

func (t *WorkflowAdvancePhaseTool) Name() string {
	return "workflow_advance_phase"
}

func (t *WorkflowAdvancePhaseTool) Description() string {
	return "Advance to the next phase of the mission workflow. Only use this when the current phase completion criteria are met."
}

func (t *WorkflowAdvancePhaseTool) Parameters() map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

func (t *WorkflowAdvancePhaseTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	engine := t.getEngine()
	if engine == nil {
		return NewToolResult("No active mission/workflow")
	}

	// Check if phase is complete
	if !engine.IsPhaseComplete() {
		wf := engine.GetWorkflow()
		state := engine.GetState()
		if state.CurrentPhase < len(wf.Phases) {
			phase := wf.Phases[state.CurrentPhase]
			return NewToolResult(fmt.Sprintf("Phase '%s' completion criteria not yet met. Review the phase steps and completion requirements.", phase.Name))
		}
	}

	if err := engine.AdvancePhase(); err != nil {
		return NewToolResult(fmt.Sprintf("Failed to advance phase: %v", err))
	}

	wf := engine.GetWorkflow()
	state := engine.GetState()
	newPhaseName := wf.Phases[state.CurrentPhase].Name

	return NewToolResult(fmt.Sprintf("Advanced to phase: %s", newPhaseName))
}
