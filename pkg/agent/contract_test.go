package agent

import (
	"fmt"
	"testing"
	"time"

	"github.com/ResistanceIsUseless/picoclaw/pkg/artifacts"
	"github.com/ResistanceIsUseless/picoclaw/pkg/blackboard"
	"github.com/stretchr/testify/assert"
)

func TestNewPhaseContract(t *testing.T) {
	contract := NewPhaseContract("test_phase")

	assert.Equal(t, "test_phase", contract.PhaseName)
	assert.Equal(t, 0, len(contract.RequiredTools))
	assert.Equal(t, 0, len(contract.RequiredArtifacts))
	assert.Equal(t, 1, contract.MinIterations)
	assert.Equal(t, 10, contract.MaxIterations)
}

func TestPhaseContractBuilders(t *testing.T) {
	contract := NewPhaseContract("test").
		AddRequiredTool("tool1").
		AddRequiredTool("tool2").
		AddRequiredArtifact("artifact1").
		AddOptionalTool("tool3").
		SetIterationLimits(2, 8)

	assert.Equal(t, 2, len(contract.RequiredTools))
	assert.Contains(t, contract.RequiredTools, "tool1")
	assert.Contains(t, contract.RequiredTools, "tool2")
	assert.Equal(t, 1, len(contract.RequiredArtifacts))
	assert.Equal(t, 1, len(contract.OptionalTools))
	assert.Equal(t, 2, contract.MinIterations)
	assert.Equal(t, 8, contract.MaxIterations)
}

func TestValidateRequiredTools(t *testing.T) {
	contract := NewPhaseContract("test").
		AddRequiredTool("subfinder").
		AddRequiredTool("nmap")

	state := NewDAGState("test", []string{"subfinder", "nmap"}, nil)

	// Initially, validation should fail
	ctx := &PhaseContext{
		Phase:     "test",
		State:     state,
		Iteration: 1,
	}
	err := contract.Validate(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "subfinder")
	assert.Contains(t, err.Error(), "nmap")

	// Add one tool
	state.AddToolCall(&ToolCall{
		ID:       "1",
		ToolName: "subfinder",
		Status:   StatusCompleted,
	})

	err = contract.Validate(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nmap")
	assert.NotContains(t, err.Error(), "subfinder")

	// Add second tool
	state.AddToolCall(&ToolCall{
		ID:       "2",
		ToolName: "nmap",
		Status:   StatusCompleted,
	})

	err = contract.Validate(ctx)
	assert.NoError(t, err)
}

func TestValidateRequiredArtifacts(t *testing.T) {
	contract := NewPhaseContract("test").
		AddRequiredArtifact(artifacts.ArtifactSubdomainList).
		AddRequiredArtifact(artifacts.ArtifactPortScanResult)

	state := NewDAGState("test", []string{}, nil)

	// Initially, validation should fail
	ctx := &PhaseContext{
		Phase:     "test",
		State:     state,
		Artifacts: []blackboard.ArtifactEnvelope{},
		Iteration: 1,
	}
	err := contract.Validate(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), artifacts.ArtifactSubdomainList)

	// Add one artifact
	ctx.Artifacts = append(ctx.Artifacts, blackboard.ArtifactEnvelope{
		Metadata: blackboard.ArtifactMetadata{
			Type:      artifacts.ArtifactSubdomainList,
			CreatedAt: time.Now(),
		},
	})

	err = contract.Validate(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), artifacts.ArtifactPortScanResult)

	// Add second artifact
	ctx.Artifacts = append(ctx.Artifacts, blackboard.ArtifactEnvelope{
		Metadata: blackboard.ArtifactMetadata{
			Type:      artifacts.ArtifactPortScanResult,
			CreatedAt: time.Now(),
		},
	})

	err = contract.Validate(ctx)
	assert.NoError(t, err)
}

func TestValidateIterations(t *testing.T) {
	contract := NewPhaseContract("test").
		SetIterationLimits(2, 5)

	state := NewDAGState("test", []string{}, nil)

	// Below minimum
	ctx := &PhaseContext{
		Phase:     "test",
		State:     state,
		Iteration: 1,
	}
	err := contract.Validate(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient iterations")

	// At minimum
	ctx.Iteration = 2
	err = contract.Validate(ctx)
	assert.NoError(t, err)

	// Within range
	ctx.Iteration = 3
	err = contract.Validate(ctx)
	assert.NoError(t, err)

	// At maximum
	ctx.Iteration = 5
	err = contract.Validate(ctx)
	assert.NoError(t, err)

	// Above maximum
	ctx.Iteration = 6
	err = contract.Validate(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeded iteration limit")
}

func TestCustomValidationRules(t *testing.T) {
	contract := NewPhaseContract("test").
		AddValidationRule(ValidationRule{
			Name:        "custom_check",
			Description: "Custom validation",
			Validator: func(ctx *PhaseContext) error {
				if ctx.Iteration < 3 {
					return fmt.Errorf("need at least 3 iterations")
				}
				return nil
			},
		})

	state := NewDAGState("test", []string{}, nil)

	// Should fail
	ctx := &PhaseContext{
		Phase:     "test",
		State:     state,
		Iteration: 2,
	}
	err := contract.Validate(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "custom_check")

	// Should pass
	ctx.Iteration = 3
	err = contract.Validate(ctx)
	assert.NoError(t, err)
}

func TestCanComplete(t *testing.T) {
	contract := NewPhaseContract("test").
		AddRequiredTool("tool1")

	state := NewDAGState("test", []string{"tool1"}, nil)
	ctx := &PhaseContext{
		Phase:     "test",
		State:     state,
		Iteration: 1,
	}

	// Not complete
	assert.False(t, contract.CanComplete(ctx))

	// Complete
	state.AddToolCall(&ToolCall{
		ID:       "1",
		ToolName: "tool1",
		Status:   StatusCompleted,
	})
	assert.True(t, contract.CanComplete(ctx))
}

func TestGetCompletionStatus(t *testing.T) {
	contract := NewPhaseContract("test").
		AddRequiredTool("subfinder").
		AddRequiredTool("nmap").
		AddRequiredArtifact(artifacts.ArtifactSubdomainList).
		SetIterationLimits(1, 5)

	state := NewDAGState("test", []string{"subfinder", "nmap"}, nil)
	state.AddToolCall(&ToolCall{
		ID:       "1",
		ToolName: "subfinder",
		Status:   StatusCompleted,
	})

	ctx := &PhaseContext{
		Phase: "test",
		State: state,
		Artifacts: []blackboard.ArtifactEnvelope{
			{
				Metadata: blackboard.ArtifactMetadata{
					Type:      artifacts.ArtifactSubdomainList,
					CreatedAt: time.Now(),
				},
			},
		},
		Iteration: 2,
	}

	status := contract.GetCompletionStatus(ctx)

	assert.Contains(t, status, "Phase Completion Status: test")
	assert.Contains(t, status, "Required Tools")
	assert.Contains(t, status, "✓ subfinder")
	assert.Contains(t, status, "✗ nmap")
	assert.Contains(t, status, "Required Artifacts")
	assert.Contains(t, status, "✓ "+artifacts.ArtifactSubdomainList)
	assert.Contains(t, status, "Iterations: 2 / 5")
	assert.Contains(t, status, "Phase cannot be completed yet")
}

func TestPredefinedContracts(t *testing.T) {
	// Test recon contract
	reconContract, err := GetPredefinedContract("recon")
	assert.NoError(t, err)
	assert.Equal(t, "recon", reconContract.PhaseName)
	assert.Contains(t, reconContract.RequiredTools, "subfinder")
	assert.Contains(t, reconContract.RequiredArtifacts, artifacts.ArtifactSubdomainList)
	assert.Equal(t, 1, len(reconContract.SuccessCriteria))

	// Test port_scan contract
	portContract, err := GetPredefinedContract("port_scan")
	assert.NoError(t, err)
	assert.Equal(t, "port_scan", portContract.PhaseName)
	assert.Contains(t, portContract.RequiredTools, "nmap")

	// Test non-existent contract
	_, err = GetPredefinedContract("nonexistent")
	assert.Error(t, err)
}

func TestReconContractValidation(t *testing.T) {
	contract, err := GetPredefinedContract("recon")
	assert.NoError(t, err)

	state := NewDAGState("recon", []string{"subfinder"}, nil)
	state.AddToolCall(&ToolCall{
		ID:       "1",
		ToolName: "subfinder",
		Status:   StatusCompleted,
	})

	// With SubdomainList artifact - should pass
	ctx := &PhaseContext{
		Phase: "recon",
		State: state,
		Artifacts: []blackboard.ArtifactEnvelope{
			{
				Metadata: blackboard.ArtifactMetadata{
					Type:      artifacts.ArtifactSubdomainList,
					CreatedAt: time.Now(),
				},
			},
		},
		Iteration: 1,
	}

	err = contract.Validate(ctx)
	assert.NoError(t, err)
}

func TestPortScanContractValidation(t *testing.T) {
	contract, err := GetPredefinedContract("port_scan")
	assert.NoError(t, err)

	state := NewDAGState("port_scan", []string{"nmap"}, nil)
	state.AddToolCall(&ToolCall{
		ID:       "1",
		ToolName: "nmap",
		Status:   StatusCompleted,
	})

	// With PortScanResult artifact - should pass
	ctx := &PhaseContext{
		Phase: "port_scan",
		State: state,
		Artifacts: []blackboard.ArtifactEnvelope{
			{
				Metadata: blackboard.ArtifactMetadata{
					Type:      artifacts.ArtifactPortScanResult,
					CreatedAt: time.Now(),
				},
			},
		},
		Iteration: 1,
	}

	err = contract.Validate(ctx)
	assert.NoError(t, err)
}

func TestNewCustomContract(t *testing.T) {
	// Create custom contract based on recon
	custom, err := NewCustomContract("recon", func(c *PhaseContract) {
		c.AddRequiredTool("amass")
		c.SetIterationLimits(2, 8)
	})

	assert.NoError(t, err)
	assert.Equal(t, "recon", custom.PhaseName)
	assert.Contains(t, custom.RequiredTools, "subfinder") // From base
	assert.Contains(t, custom.RequiredTools, "amass")     // Added
	assert.Equal(t, 2, custom.MinIterations)
	assert.Equal(t, 8, custom.MaxIterations)
}

func TestMultipleValidationErrors(t *testing.T) {
	contract := NewPhaseContract("test").
		AddRequiredTool("tool1").
		AddRequiredTool("tool2").
		AddRequiredArtifact("artifact1").
		SetIterationLimits(2, 5).
		AddValidationRule(ValidationRule{
			Name: "custom_rule",
			Validator: func(ctx *PhaseContext) error {
				return fmt.Errorf("custom error")
			},
		})

	state := NewDAGState("test", []string{}, nil)
	ctx := &PhaseContext{
		Phase:     "test",
		State:     state,
		Artifacts: []blackboard.ArtifactEnvelope{},
		Iteration: 1,
	}

	err := contract.Validate(ctx)
	assert.Error(t, err)

	// Should contain all error types
	assert.Contains(t, err.Error(), "tool1")
	assert.Contains(t, err.Error(), "tool2")
	assert.Contains(t, err.Error(), "artifact1")
	assert.Contains(t, err.Error(), "insufficient iterations")
	assert.Contains(t, err.Error(), "custom_rule")
}

func TestCompletionStatusWithAllPassed(t *testing.T) {
	contract := NewPhaseContract("test").
		AddRequiredTool("tool1").
		AddRequiredArtifact("artifact1")

	state := NewDAGState("test", []string{"tool1"}, nil)
	state.AddToolCall(&ToolCall{
		ID:       "1",
		ToolName: "tool1",
		Status:   StatusCompleted,
	})

	ctx := &PhaseContext{
		Phase: "test",
		State: state,
		Artifacts: []blackboard.ArtifactEnvelope{
			{
				Metadata: blackboard.ArtifactMetadata{
					Type:      "artifact1",
					CreatedAt: time.Now(),
				},
			},
		},
		Iteration: 1,
	}

	status := contract.GetCompletionStatus(ctx)

	assert.Contains(t, status, "✓ tool1")
	assert.Contains(t, status, "✓ artifact1")
	assert.Contains(t, status, "Phase can be completed")
}
