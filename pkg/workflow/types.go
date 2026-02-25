package workflow

import "time"

// Workflow represents a multi-phase methodology
type Workflow struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Phases      []Phase `json:"phases"`
}

// Phase represents a stage in the workflow
type Phase struct {
	Name       string              `json:"name"`
	Steps      []Step              `json:"steps"`
	Completion CompletionCriteria  `json:"completion"`
	Branches   []Branch            `json:"branches,omitempty"`
}

// Step represents an action within a phase
type Step struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required"`
	Completed   bool   `json:"completed"`
}

// CompletionCriteria defines when a phase is considered complete
type CompletionCriteria struct {
	Type        CompletionType `json:"type"`
	Description string         `json:"description"`
	// For "all_required" type: phase completes when all required steps are done
	// For "any_branch" type: phase completes when any branch is created
	// For "custom" type: use Description for manual evaluation
}

// CompletionType defines how phase completion is determined
type CompletionType string

const (
	CompletionAllRequired CompletionType = "all_required" // All required steps must be complete
	CompletionAnyBranch   CompletionType = "any_branch"   // At least one branch must be created
	CompletionCustom      CompletionType = "custom"       // Custom criteria (evaluated manually)
)

// Branch represents a conditional workflow path based on discoveries
type Branch struct {
	Condition   string `json:"condition"`   // e.g., "web_service_found", "smb_found"
	Description string `json:"description"` // Human-readable description
	TargetPhase string `json:"target_phase,omitempty"` // Phase to jump to (optional)
	Steps       []Step `json:"steps,omitempty"`        // Additional steps for this branch
}

// MissionState tracks the current state of a workflow execution
type MissionState struct {
	WorkflowName  string                 `json:"workflow_name"`
	Target        string                 `json:"target"`
	StartTime     time.Time              `json:"start_time"`
	CurrentPhase  int                    `json:"current_phase"`
	PhaseHistory  []PhaseExecution       `json:"phase_history"`
	ActiveBranches []ActiveBranch        `json:"active_branches"`
	Findings      []Finding              `json:"findings"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// PhaseExecution tracks execution of a phase
type PhaseExecution struct {
	PhaseName    string            `json:"phase_name"`
	StartTime    time.Time         `json:"start_time"`
	EndTime      *time.Time        `json:"end_time,omitempty"`
	StepsComplete []string          `json:"steps_complete"`
	Notes        []string           `json:"notes,omitempty"`
}

// ActiveBranch tracks a branch that has been activated
type ActiveBranch struct {
	Condition   string     `json:"condition"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Findings    []Finding  `json:"findings,omitempty"`
}

// Finding represents a discovery made during workflow execution
type Finding struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Severity    Severity               `json:"severity"`
	Phase       string                 `json:"phase"`
	CreatedAt   time.Time              `json:"created_at"`
	Evidence    string                 `json:"evidence,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Severity levels for findings
type Severity string

const (
	SeverityCritical      Severity = "critical"
	SeverityHigh          Severity = "high"
	SeverityMedium        Severity = "medium"
	SeverityLow           Severity = "low"
	SeverityInformational Severity = "informational"
)
