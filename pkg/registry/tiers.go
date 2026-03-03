package registry

import "fmt"

// ToolTier defines the trust and approval level for tools
type ToolTier int

const (
	// TierOrchestrator (-1) - Orchestrator-injected tools that the model can call
	// but are actually handled by the orchestrator (complete_phase, validate_artifact, escalate)
	TierOrchestrator ToolTier = -1

	// TierHardwired (0) - Fully trusted, invisible to model
	// Output appears as given truth with no attribution
	// Examples: subfinder, amass, crtsh, dns_enum
	// These are the "seed" tools that start phases
	TierHardwired ToolTier = 0

	// TierAutoApprove (1) - Trusted, visible to model, auto-approved
	// Model sees these ran and their results
	// Examples: nmap, httpx, whatweb, nuclei (non-exploiting scanners)
	TierAutoApprove ToolTier = 1

	// TierHuman (2) - Requires explicit human approval per execution
	// Examples: exploitation tools, fuzzing, anything that modifies target state
	TierHuman ToolTier = 2

	// TierBanned (3) - Explicitly forbidden, always rejected
	// Examples: rm -rf, destructive operations, out-of-scope actions
	TierBanned ToolTier = 3
)

// String returns the human-readable tier name
func (t ToolTier) String() string {
	switch t {
	case TierOrchestrator:
		return "TierOrchestrator"
	case TierHardwired:
		return "TierHardwired"
	case TierAutoApprove:
		return "TierAutoApprove"
	case TierHuman:
		return "TierHuman"
	case TierBanned:
		return "TierBanned"
	default:
		return fmt.Sprintf("TierUnknown(%d)", t)
	}
}

// RequiresApproval returns true if this tier requires human approval
func (t ToolTier) RequiresApproval() bool {
	return t == TierHuman
}

// IsVisible returns true if the model should see this tool's existence and output
func (t ToolTier) IsVisible() bool {
	return t != TierHardwired
}

// IsAllowed returns true if this tier can be executed (not banned)
func (t ToolTier) IsAllowed() bool {
	return t != TierBanned
}

// CanModelRequest returns true if the model can request this tool
func (t ToolTier) CanModelRequest() bool {
	// Model can request Tier 1 and Tier 2
	// Model cannot request Tier -1 (but they're injected as available)
	// Model cannot request Tier 0 (invisible)
	// Model cannot request Tier 3 (banned)
	return t == TierAutoApprove || t == TierHuman || t == TierOrchestrator
}

// TierPolicy defines behavior for a tool tier
type TierPolicy struct {
	Tier              ToolTier
	RequiresApproval  bool
	VisibleToModel    bool
	AllowModelRequest bool
	Description       string
}

// GetTierPolicy returns the policy for a given tier
func GetTierPolicy(tier ToolTier) TierPolicy {
	return TierPolicy{
		Tier:              tier,
		RequiresApproval:  tier.RequiresApproval(),
		VisibleToModel:    tier.IsVisible(),
		AllowModelRequest: tier.CanModelRequest(),
		Description:       getTierDescription(tier),
	}
}

func getTierDescription(tier ToolTier) string {
	switch tier {
	case TierOrchestrator:
		return "Orchestrator-injected tools (complete_phase, validate_artifact, escalate)"
	case TierHardwired:
		return "Fully trusted, invisible to model - output appears as given truth"
	case TierAutoApprove:
		return "Trusted, visible, auto-approved - standard reconnaissance tools"
	case TierHuman:
		return "Requires human approval - exploitation, fuzzing, state-changing operations"
	case TierBanned:
		return "Explicitly forbidden - destructive or out-of-scope operations"
	default:
		return "Unknown tier"
	}
}

// ValidationResult indicates whether a tool execution should proceed
type ValidationResult struct {
	Allowed        bool
	RequiresHuman  bool
	Tier           ToolTier
	RejectReason   string
	ApprovalPrompt string // If RequiresHuman, this is the prompt to show operator
}

// ValidateToolExecution checks if a tool can be executed based on its tier
func ValidateToolExecution(tier ToolTier, toolName string, args map[string]any) ValidationResult {
	if tier == TierBanned {
		return ValidationResult{
			Allowed:      false,
			RequiresHuman: false,
			Tier:         tier,
			RejectReason: fmt.Sprintf("Tool %q is banned (Tier 3) and cannot be executed", toolName),
		}
	}

	if tier == TierHuman {
		return ValidationResult{
			Allowed:        true,
			RequiresHuman:  true,
			Tier:           tier,
			ApprovalPrompt: formatApprovalPrompt(toolName, args),
		}
	}

	// Tier -1, 0, 1 auto-approved
	return ValidationResult{
		Allowed:       true,
		RequiresHuman: false,
		Tier:          tier,
	}
}

func formatApprovalPrompt(toolName string, args map[string]any) string {
	prompt := fmt.Sprintf("Tool %q (Tier 2 - Human Approval Required)\n", toolName)
	prompt += "\nArguments:\n"
	for k, v := range args {
		prompt += fmt.Sprintf("  %s: %v\n", k, v)
	}
	prompt += "\nThis tool may modify target state or perform exploitation.\n"
	prompt += "Approve execution? (y/n): "
	return prompt
}

// TierUpgradeRequest represents a request from the model to use a tool
// outside its current phase's allowed tier list
type TierUpgradeRequest struct {
	ToolName       string
	RequestedTier  ToolTier
	CurrentMaxTier ToolTier
	Justification  string
	PhaseContext   string
}

// ValidateTierUpgrade checks if a tier upgrade request should be allowed
func ValidateTierUpgrade(req TierUpgradeRequest) ValidationResult {
	// Model can request Tier 1 tools when in a Tier 0-only phase
	// Model can request Tier 2 tools but they need human approval
	// Model cannot request Tier 3 (banned)
	// Model cannot request Tier -1 directly (those are injected)

	if req.RequestedTier == TierBanned {
		return ValidationResult{
			Allowed:      false,
			RequiresHuman: false,
			Tier:         req.RequestedTier,
			RejectReason: "Cannot upgrade to banned tier",
		}
	}

	if req.RequestedTier == TierHardwired {
		return ValidationResult{
			Allowed:      false,
			RequiresHuman: false,
			Tier:         req.RequestedTier,
			RejectReason: "Cannot request Tier 0 (hardwired) tools - these are invisible to model",
		}
	}

	if req.RequestedTier == TierHuman {
		return ValidationResult{
			Allowed:        true,
			RequiresHuman:  true,
			Tier:           req.RequestedTier,
			ApprovalPrompt: formatUpgradePrompt(req),
		}
	}

	// Tier 1 (AutoApprove) can be upgraded to without approval
	return ValidationResult{
		Allowed:       true,
		RequiresHuman: false,
		Tier:          req.RequestedTier,
	}
}

func formatUpgradePrompt(req TierUpgradeRequest) string {
	prompt := fmt.Sprintf("Tier Upgrade Request: %q (Tier %s → Tier %s)\n",
		req.ToolName, req.CurrentMaxTier, req.RequestedTier)
	prompt += fmt.Sprintf("\nPhase: %s\n", req.PhaseContext)
	if req.Justification != "" {
		prompt += fmt.Sprintf("Model justification: %s\n", req.Justification)
	}
	prompt += "\nThis tool is outside the phase's pre-approved toolset.\n"
	prompt += "Approve tier upgrade and execution? (y/n): "
	return prompt
}
