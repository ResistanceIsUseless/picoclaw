package agent

import (
	"fmt"
	"strings"

	"github.com/ResistanceIsUseless/picoclaw/pkg/artifacts"
	"github.com/ResistanceIsUseless/picoclaw/pkg/blackboard"
	"github.com/ResistanceIsUseless/picoclaw/pkg/logger"
)

// PhaseContract defines the requirements for a phase to be considered complete
type PhaseContract struct {
	PhaseName        string
	RequiredTools    []string          // Tools that MUST be executed
	RequiredArtifacts []string         // Artifact types that MUST be produced
	OptionalTools    []string          // Tools that MAY be executed
	MinIterations    int               // Minimum number of iterations
	MaxIterations    int               // Maximum number of iterations
	SuccessCriteria  []ValidationRule  // Custom validation rules
}

// ValidationRule defines a custom validation check for phase completion
type ValidationRule struct {
	Name        string
	Description string
	Validator   func(*PhaseContext) error
}

// PhaseContext provides context for validation rules
type PhaseContext struct {
	Phase         string
	State         *DAGState
	Blackboard    *blackboard.Blackboard
	Artifacts     []blackboard.ArtifactEnvelope
	Iteration     int
}

// NewPhaseContract creates a new phase contract
func NewPhaseContract(phaseName string) *PhaseContract {
	return &PhaseContract{
		PhaseName:         phaseName,
		RequiredTools:     make([]string, 0),
		RequiredArtifacts: make([]string, 0),
		OptionalTools:     make([]string, 0),
		MinIterations:     1,
		MaxIterations:     10,
		SuccessCriteria:   make([]ValidationRule, 0),
	}
}

// AddRequiredTool adds a tool that must be executed
func (c *PhaseContract) AddRequiredTool(toolName string) *PhaseContract {
	c.RequiredTools = append(c.RequiredTools, toolName)
	return c
}

// AddRequiredArtifact adds an artifact type that must be produced
func (c *PhaseContract) AddRequiredArtifact(artifactType string) *PhaseContract {
	c.RequiredArtifacts = append(c.RequiredArtifacts, artifactType)
	return c
}

// AddOptionalTool adds a tool that may be executed
func (c *PhaseContract) AddOptionalTool(toolName string) *PhaseContract {
	c.OptionalTools = append(c.OptionalTools, toolName)
	return c
}

// AddValidationRule adds a custom validation rule
func (c *PhaseContract) AddValidationRule(rule ValidationRule) *PhaseContract {
	c.SuccessCriteria = append(c.SuccessCriteria, rule)
	return c
}

// SetIterationLimits sets min/max iteration bounds
func (c *PhaseContract) SetIterationLimits(min, max int) *PhaseContract {
	c.MinIterations = min
	c.MaxIterations = max
	return c
}

// Validate checks if the phase contract is satisfied
func (c *PhaseContract) Validate(ctx *PhaseContext) error {
	errors := make([]string, 0)

	// Check required tools
	if err := c.validateRequiredTools(ctx.State); err != nil {
		errors = append(errors, err.Error())
	}

	// Check required artifacts
	if err := c.validateRequiredArtifacts(ctx.Artifacts); err != nil {
		errors = append(errors, err.Error())
	}

	// Check iteration bounds
	if err := c.validateIterations(ctx.Iteration); err != nil {
		errors = append(errors, err.Error())
	}

	// Run custom validation rules
	for _, rule := range c.SuccessCriteria {
		if err := rule.Validator(ctx); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %s", rule.Name, err.Error()))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("phase contract validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	logger.InfoCF("contract", "Phase contract validated successfully",
		map[string]any{
			"phase": c.PhaseName,
		})

	return nil
}

// validateRequiredTools ensures all required tools have been executed successfully
func (c *PhaseContract) validateRequiredTools(state *DAGState) error {
	if len(c.RequiredTools) == 0 {
		return nil
	}

	missingTools := make([]string, 0)
	completedTools := make(map[string]bool)

	for _, call := range state.GetCompletedTools() {
		completedTools[call.ToolName] = true
	}

	for _, requiredTool := range c.RequiredTools {
		if !completedTools[requiredTool] {
			missingTools = append(missingTools, requiredTool)
		}
	}

	if len(missingTools) > 0 {
		return fmt.Errorf("required tools not executed: %s", strings.Join(missingTools, ", "))
	}

	return nil
}

// validateRequiredArtifacts ensures all required artifact types have been produced
func (c *PhaseContract) validateRequiredArtifacts(artifacts []blackboard.ArtifactEnvelope) error {
	if len(c.RequiredArtifacts) == 0 {
		return nil
	}

	artifactTypes := make(map[string]bool)
	for _, envelope := range artifacts {
		artifactTypes[envelope.Metadata.Type] = true
	}

	missingArtifacts := make([]string, 0)
	for _, requiredType := range c.RequiredArtifacts {
		if !artifactTypes[requiredType] {
			missingArtifacts = append(missingArtifacts, requiredType)
		}
	}

	if len(missingArtifacts) > 0 {
		return fmt.Errorf("required artifacts not produced: %s", strings.Join(missingArtifacts, ", "))
	}

	return nil
}

// validateIterations checks if iteration count is within bounds
func (c *PhaseContract) validateIterations(iteration int) error {
	if iteration < c.MinIterations {
		return fmt.Errorf("insufficient iterations: %d < %d (minimum)", iteration, c.MinIterations)
	}

	if iteration > c.MaxIterations {
		return fmt.Errorf("exceeded iteration limit: %d > %d (maximum)", iteration, c.MaxIterations)
	}

	return nil
}

// CanComplete checks if the phase can be marked as complete
func (c *PhaseContract) CanComplete(ctx *PhaseContext) bool {
	return c.Validate(ctx) == nil
}

// GetCompletionStatus returns a human-readable completion status
func (c *PhaseContract) GetCompletionStatus(ctx *PhaseContext) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## Phase Completion Status: %s\n\n", c.PhaseName))

	// Required tools status
	if len(c.RequiredTools) > 0 {
		sb.WriteString("### Required Tools\n\n")
		completedTools := make(map[string]bool)
		for _, call := range ctx.State.GetCompletedTools() {
			completedTools[call.ToolName] = true
		}

		for _, tool := range c.RequiredTools {
			if completedTools[tool] {
				sb.WriteString(fmt.Sprintf("  ✓ %s\n", tool))
			} else {
				sb.WriteString(fmt.Sprintf("  ✗ %s (not executed)\n", tool))
			}
		}
		sb.WriteString("\n")
	}

	// Required artifacts status
	if len(c.RequiredArtifacts) > 0 {
		sb.WriteString("### Required Artifacts\n\n")
		artifactTypes := make(map[string]bool)
		for _, envelope := range ctx.Artifacts {
			artifactTypes[envelope.Metadata.Type] = true
		}

		for _, artifactType := range c.RequiredArtifacts {
			if artifactTypes[artifactType] {
				sb.WriteString(fmt.Sprintf("  ✓ %s\n", artifactType))
			} else {
				sb.WriteString(fmt.Sprintf("  ✗ %s (not produced)\n", artifactType))
			}
		}
		sb.WriteString("\n")
	}

	// Iteration status
	sb.WriteString(fmt.Sprintf("### Iterations: %d / %d (min: %d, max: %d)\n\n",
		ctx.Iteration, c.MaxIterations, c.MinIterations, c.MaxIterations))

	// Custom validation rules
	if len(c.SuccessCriteria) > 0 {
		sb.WriteString("### Custom Validation Rules\n\n")
		for _, rule := range c.SuccessCriteria {
			err := rule.Validator(ctx)
			if err == nil {
				sb.WriteString(fmt.Sprintf("  ✓ %s\n", rule.Name))
			} else {
				sb.WriteString(fmt.Sprintf("  ✗ %s: %s\n", rule.Name, err.Error()))
			}
		}
		sb.WriteString("\n")
	}

	// Overall status
	if c.CanComplete(ctx) {
		sb.WriteString("**Status**: ✓ Phase can be completed\n")
	} else {
		sb.WriteString("**Status**: ✗ Phase cannot be completed yet\n")
	}

	return sb.String()
}

// PredefinedContracts contains standard phase contracts
var PredefinedContracts = map[string]*PhaseContract{
	"recon": NewPhaseContract("recon").
		AddRequiredTool("subfinder").
		AddRequiredArtifact(artifacts.ArtifactSubdomainList).
		SetIterationLimits(1, 5).
		AddValidationRule(ValidationRule{
			Name:        "subdomain_threshold",
			Description: "At least 1 subdomain must be discovered",
			Validator: func(ctx *PhaseContext) error {
				// Check if SubdomainList artifact exists
				for _, envelope := range ctx.Artifacts {
					if envelope.Metadata.Type == artifacts.ArtifactSubdomainList {
						return nil // Artifact exists, good enough for now
					}
				}
				return fmt.Errorf("no SubdomainList artifact found")
			},
		}),

	"port_scan": NewPhaseContract("port_scan").
		AddRequiredTool("nmap").
		AddRequiredArtifact(artifacts.ArtifactPortScanResult).
		SetIterationLimits(1, 3).
		AddValidationRule(ValidationRule{
			Name:        "hosts_scanned",
			Description: "At least 1 host must be scanned",
			Validator: func(ctx *PhaseContext) error {
				// Check if PortScanResult artifact exists
				for _, envelope := range ctx.Artifacts {
					if envelope.Metadata.Type == artifacts.ArtifactPortScanResult {
						return nil // Artifact exists, good enough for now
					}
				}
				return fmt.Errorf("no PortScanResult artifact found")
			},
		}),

	"service_discovery": NewPhaseContract("service_discovery").
		AddRequiredTool("httpx").
		AddRequiredArtifact(artifacts.ArtifactServiceFingerprint).
		SetIterationLimits(1, 5),

	"vulnerability_scan": NewPhaseContract("vulnerability_scan").
		AddRequiredTool("nuclei").
		AddRequiredArtifact(artifacts.ArtifactVulnerabilityList).
		SetIterationLimits(1, 10).
		AddValidationRule(ValidationRule{
			Name:        "scan_coverage",
			Description: "Must scan all discovered services",
			Validator: func(ctx *PhaseContext) error {
				// This is a placeholder - in real implementation,
				// would check that all services from service_discovery
				// were tested
				return nil
			},
		}),

	"exploitation": NewPhaseContract("exploitation").
		AddRequiredTool("exploit").
		AddRequiredArtifact(artifacts.ArtifactExploitResult).
		SetIterationLimits(1, 20).
		AddValidationRule(ValidationRule{
			Name:        "exploitation_attempted",
			Description: "At least one exploitation attempt must be made",
			Validator: func(ctx *PhaseContext) error {
				// Check if ExploitResult artifact exists
				for _, envelope := range ctx.Artifacts {
					if envelope.Metadata.Type == artifacts.ArtifactExploitResult {
						return nil
					}
				}
				return fmt.Errorf("no exploitation attempts made")
			},
		}),
}

// GetPredefinedContract retrieves a predefined contract by phase name
func GetPredefinedContract(phaseName string) (*PhaseContract, error) {
	contract, exists := PredefinedContracts[phaseName]
	if !exists {
		return nil, fmt.Errorf("no predefined contract for phase %q", phaseName)
	}
	return contract, nil
}

// NewCustomContract creates a new contract based on a predefined one with modifications
func NewCustomContract(baseName string, modifications func(*PhaseContract)) (*PhaseContract, error) {
	base, err := GetPredefinedContract(baseName)
	if err != nil {
		return nil, err
	}

	// Clone the base contract
	custom := &PhaseContract{
		PhaseName:         base.PhaseName,
		RequiredTools:     append([]string{}, base.RequiredTools...),
		RequiredArtifacts: append([]string{}, base.RequiredArtifacts...),
		OptionalTools:     append([]string{}, base.OptionalTools...),
		MinIterations:     base.MinIterations,
		MaxIterations:     base.MaxIterations,
		SuccessCriteria:   append([]ValidationRule{}, base.SuccessCriteria...),
	}

	// Apply modifications
	modifications(custom)

	return custom, nil
}
