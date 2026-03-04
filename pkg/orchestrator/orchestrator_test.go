package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/ResistanceIsUseless/picoclaw/pkg/blackboard"
	"github.com/ResistanceIsUseless/picoclaw/pkg/registry"
	"github.com/stretchr/testify/assert"
)

func TestNewOrchestrator(t *testing.T) {
	pipeline := NewPipeline("test", "Test pipeline", "web")
	bb := blackboard.New(nil)
	toolRegistry := registry.NewToolRegistry()

	orch := NewOrchestrator(pipeline, bb, toolRegistry)

	assert.NotNil(t, orch)
	assert.Equal(t, pipeline, orch.pipeline)
	assert.Equal(t, bb, orch.blackboard)
	assert.NotNil(t, orch.graph)
	assert.NotNil(t, orch.entityRegistry)
}

func TestCheckDependencies(t *testing.T) {
	pipeline := NewPipeline("test", "Test pipeline", "web").
		AddPhase(&PhaseDefinition{
			Name:      "phase1",
			Objective: "First phase",
			Tools:     []string{"tool1"},
			MinIterations: 1,
			MaxIterations: 3,
		}).
		AddPhase(&PhaseDefinition{
			Name:      "phase2",
			Objective: "Second phase",
			Tools:     []string{"tool2"},
			DependsOn: []string{"phase1"},
			MinIterations: 1,
			MaxIterations: 3,
		})

	bb := blackboard.New(nil)
	toolRegistry := registry.NewToolRegistry()
	orch := NewOrchestrator(pipeline, bb, toolRegistry)

	// phase2 should fail dependency check initially
	err := orch.checkDependencies(pipeline.Phases[1])
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "phase1")

	// Mark phase1 as completed
	orch.completedPhases = append(orch.completedPhases, "phase1")

	// phase2 should now pass dependency check
	err = orch.checkDependencies(pipeline.Phases[1])
	assert.NoError(t, err)
}

func TestCreateContract(t *testing.T) {
	pipeline := NewPipeline("test", "Test", "web")
	bb := blackboard.New(nil)
	toolRegistry := registry.NewToolRegistry()
	orch := NewOrchestrator(pipeline, bb, toolRegistry)

	phaseDef := &PhaseDefinition{
		Name:              "test_phase",
		Objective:         "Test objective",
		Tools:             []string{"tool1", "tool2"},
		RequiredTools:     []string{"tool1"},
		RequiredArtifacts: []string{"Artifact1"},
		MinIterations:     2,
		MaxIterations:     5,
	}

	contract := orch.createContract(phaseDef)

	assert.Equal(t, "test_phase", contract.PhaseName)
	assert.Equal(t, 2, contract.MinIterations)
	assert.Equal(t, 5, contract.MaxIterations)
	assert.Contains(t, contract.RequiredTools, "tool1")
	assert.Contains(t, contract.RequiredArtifacts, "Artifact1")
}

func TestGetCurrentPhase(t *testing.T) {
	pipeline := NewPipeline("test", "Test", "web")
	bb := blackboard.New(nil)
	toolRegistry := registry.NewToolRegistry()
	orch := NewOrchestrator(pipeline, bb, toolRegistry)

	// Initially no current phase
	assert.Nil(t, orch.GetCurrentPhase())

	// Set current phase
	phaseExec := &PhaseExecution{
		PhaseName: "test_phase",
		Status:    PhaseRunning,
		StartTime: time.Now(),
	}
	orch.currentPhase = phaseExec

	// Should return current phase
	current := orch.GetCurrentPhase()
	assert.NotNil(t, current)
	assert.Equal(t, "test_phase", current.PhaseName)
}

func TestGetCompletedPhases(t *testing.T) {
	pipeline := NewPipeline("test", "Test", "web")
	bb := blackboard.New(nil)
	toolRegistry := registry.NewToolRegistry()
	orch := NewOrchestrator(pipeline, bb, toolRegistry)

	// Initially empty
	assert.Equal(t, 0, len(orch.GetCompletedPhases()))

	// Add completed phases
	orch.completedPhases = []string{"phase1", "phase2"}

	completed := orch.GetCompletedPhases()
	assert.Equal(t, 2, len(completed))
	assert.Contains(t, completed, "phase1")
	assert.Contains(t, completed, "phase2")
}

func TestEscalate(t *testing.T) {
	pipeline := NewPipeline("test", "Test", "web")
	bb := blackboard.New(nil)
	toolRegistry := registry.NewToolRegistry()
	orch := NewOrchestrator(pipeline, bb, toolRegistry)

	// Escalate with no running phase should fail
	err := orch.Escalate("test_phase", "test reason")
	assert.Error(t, err)

	// Set running phase
	phaseExec := &PhaseExecution{
		PhaseName: "test_phase",
		Status:    PhaseRunning,
	}
	orch.currentPhase = phaseExec

	// Escalate should succeed
	err = orch.Escalate("test_phase", "test reason")
	assert.NoError(t, err)
	assert.Equal(t, PhaseEscalated, phaseExec.Status)
	assert.NotNil(t, phaseExec.Error)
}

func TestSummary(t *testing.T) {
	pipeline := NewPipeline("test", "Test", "web").
		AddPhase(&PhaseDefinition{
			Name:      "phase1",
			Objective: "First",
			Tools:     []string{"tool1"},
			MinIterations: 1,
			MaxIterations: 3,
		})

	bb := blackboard.New(nil)
	toolRegistry := registry.NewToolRegistry()
	orch := NewOrchestrator(pipeline, bb, toolRegistry)

	summary := orch.Summary()

	assert.Contains(t, summary, "Pipeline: test")
	assert.Contains(t, summary, "Completed Phases: 0/1")
}

func TestExecuteValidation(t *testing.T) {
	// Invalid pipeline - no phases
	pipeline := NewPipeline("test", "Test", "web")
	bb := blackboard.New(nil)
	toolRegistry := registry.NewToolRegistry()
	orch := NewOrchestrator(pipeline, bb, toolRegistry)

	ctx := context.Background()
	err := orch.Execute(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pipeline validation")
}

func TestExecuteContextCancellation(t *testing.T) {
	pipeline := NewPipeline("test", "Test", "web").
		AddPhase(&PhaseDefinition{
			Name:          "phase1",
			Objective:     "First",
			Tools:         []string{"tool1"},
			MinIterations: 1,
			MaxIterations: 100, // Many iterations
		})

	bb := blackboard.New(nil)
	toolRegistry := registry.NewToolRegistry()
	orch := NewOrchestrator(pipeline, bb, toolRegistry)

	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel immediately
	cancel()

	// Execute should fail with context error
	err := orch.Execute(ctx)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestGetGraph(t *testing.T) {
	pipeline := NewPipeline("test", "Test", "web")
	bb := blackboard.New(nil)
	toolRegistry := registry.NewToolRegistry()
	orch := NewOrchestrator(pipeline, bb, toolRegistry)

	graph := orch.GetGraph()
	assert.NotNil(t, graph)
}

func TestGetBlackboard(t *testing.T) {
	pipeline := NewPipeline("test", "Test", "web")
	bb := blackboard.New(nil)
	toolRegistry := registry.NewToolRegistry()
	orch := NewOrchestrator(pipeline, bb, toolRegistry)

	board := orch.GetBlackboard()
	assert.Equal(t, bb, board)
}
