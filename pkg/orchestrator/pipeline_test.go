package orchestrator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPipeline(t *testing.T) {
	pipeline := NewPipeline("test", "Test pipeline", "web")

	assert.Equal(t, "test", pipeline.Name)
	assert.Equal(t, "Test pipeline", pipeline.Description)
	assert.Equal(t, "web", pipeline.Domain)
	assert.Equal(t, 0, len(pipeline.Phases))
}

func TestAddPhase(t *testing.T) {
	pipeline := NewPipeline("test", "Test", "web")

	phase := &PhaseDefinition{
		Name:      "phase1",
		Objective: "Test objective",
		Tools:     []string{"tool1"},
		MinIterations: 1,
		MaxIterations: 3,
	}

	pipeline.AddPhase(phase)

	assert.Equal(t, 1, len(pipeline.Phases))
	assert.Equal(t, "phase1", pipeline.Phases[0].Name)
}

func TestPipelineValidate(t *testing.T) {
	// Valid pipeline
	pipeline := NewPipeline("test", "Test", "web").
		AddPhase(&PhaseDefinition{
			Name:      "phase1",
			Objective: "First",
			Tools:     []string{"tool1"},
			MinIterations: 1,
			MaxIterations: 3,
		})

	err := pipeline.Validate()
	assert.NoError(t, err)

	// Empty name
	invalidPipeline := NewPipeline("", "Test", "web")
	err = invalidPipeline.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name cannot be empty")

	// No phases
	emptyPipeline := NewPipeline("test", "Test", "web")
	err = emptyPipeline.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one phase")

	// Duplicate phase names
	dupPipeline := NewPipeline("test", "Test", "web").
		AddPhase(&PhaseDefinition{
			Name:      "phase1",
			Objective: "First",
			Tools:     []string{"tool1"},
			MinIterations: 1,
			MaxIterations: 3,
		}).
		AddPhase(&PhaseDefinition{
			Name:      "phase1",
			Objective: "Duplicate",
			Tools:     []string{"tool2"},
			MinIterations: 1,
			MaxIterations: 3,
		})

	err = dupPipeline.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate")
}

func TestPhaseDefinitionValidate(t *testing.T) {
	// Valid phase
	valid := &PhaseDefinition{
		Name:          "phase1",
		Objective:     "Test",
		Tools:         []string{"tool1", "tool2"},
		RequiredTools: []string{"tool1"},
		MinIterations: 1,
		MaxIterations: 3,
	}

	err := valid.Validate()
	assert.NoError(t, err)

	// Empty name
	noName := &PhaseDefinition{
		Objective:     "Test",
		Tools:         []string{"tool1"},
		MinIterations: 1,
		MaxIterations: 3,
	}
	err = noName.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name cannot be empty")

	// Empty objective
	noObjective := &PhaseDefinition{
		Name:          "phase1",
		Tools:         []string{"tool1"},
		MinIterations: 1,
		MaxIterations: 3,
	}
	err = noObjective.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "objective cannot be empty")

	// No tools
	noTools := &PhaseDefinition{
		Name:          "phase1",
		Objective:     "Test",
		MinIterations: 1,
		MaxIterations: 3,
	}
	err = noTools.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one tool")

	// Invalid min iterations
	badMin := &PhaseDefinition{
		Name:          "phase1",
		Objective:     "Test",
		Tools:         []string{"tool1"},
		MinIterations: 0,
		MaxIterations: 3,
	}
	err = badMin.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "min_iterations")

	// Max < min
	badMax := &PhaseDefinition{
		Name:          "phase1",
		Objective:     "Test",
		Tools:         []string{"tool1"},
		MinIterations: 5,
		MaxIterations: 3,
	}
	err = badMax.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_iterations")

	// Required tool not in available tools
	missingTool := &PhaseDefinition{
		Name:          "phase1",
		Objective:     "Test",
		Tools:         []string{"tool1"},
		RequiredTools: []string{"tool2"},
		MinIterations: 1,
		MaxIterations: 3,
	}
	err = missingTool.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in available tools")
}

func TestDependencyValidation(t *testing.T) {
	// Valid dependencies
	pipeline := NewPipeline("test", "Test", "web").
		AddPhase(&PhaseDefinition{
			Name:      "phase1",
			Objective: "First",
			Tools:     []string{"tool1"},
			MinIterations: 1,
			MaxIterations: 3,
		}).
		AddPhase(&PhaseDefinition{
			Name:      "phase2",
			Objective: "Second",
			Tools:     []string{"tool2"},
			DependsOn: []string{"phase1"},
			MinIterations: 1,
			MaxIterations: 3,
		})

	err := pipeline.Validate()
	assert.NoError(t, err)

	// Unknown dependency
	invalidPipeline := NewPipeline("test", "Test", "web").
		AddPhase(&PhaseDefinition{
			Name:      "phase1",
			Objective: "First",
			Tools:     []string{"tool1"},
			DependsOn: []string{"unknown_phase"},
			MinIterations: 1,
			MaxIterations: 3,
		})

	err = invalidPipeline.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown phase")
}

func TestCircularDependencies(t *testing.T) {
	// Direct circular dependency
	circular := NewPipeline("test", "Test", "web").
		AddPhase(&PhaseDefinition{
			Name:      "phase1",
			Objective: "First",
			Tools:     []string{"tool1"},
			DependsOn: []string{"phase2"},
			MinIterations: 1,
			MaxIterations: 3,
		}).
		AddPhase(&PhaseDefinition{
			Name:      "phase2",
			Objective: "Second",
			Tools:     []string{"tool2"},
			DependsOn: []string{"phase1"},
			MinIterations: 1,
			MaxIterations: 3,
		})

	err := circular.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circular")

	// Indirect circular dependency
	indirect := NewPipeline("test", "Test", "web").
		AddPhase(&PhaseDefinition{
			Name:      "phase1",
			Objective: "First",
			Tools:     []string{"tool1"},
			DependsOn: []string{"phase3"},
			MinIterations: 1,
			MaxIterations: 3,
		}).
		AddPhase(&PhaseDefinition{
			Name:      "phase2",
			Objective: "Second",
			Tools:     []string{"tool2"},
			DependsOn: []string{"phase1"},
			MinIterations: 1,
			MaxIterations: 3,
		}).
		AddPhase(&PhaseDefinition{
			Name:      "phase3",
			Objective: "Third",
			Tools:     []string{"tool3"},
			DependsOn: []string{"phase2"},
			MinIterations: 1,
			MaxIterations: 3,
		})

	err = indirect.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circular")
}

func TestGetPhase(t *testing.T) {
	pipeline := NewPipeline("test", "Test", "web").
		AddPhase(&PhaseDefinition{
			Name:      "phase1",
			Objective: "First",
			Tools:     []string{"tool1"},
			MinIterations: 1,
			MaxIterations: 3,
		})

	// Existing phase
	phase, err := pipeline.GetPhase("phase1")
	assert.NoError(t, err)
	assert.Equal(t, "phase1", phase.Name)

	// Non-existent phase
	_, err = pipeline.GetPhase("nonexistent")
	assert.Error(t, err)
}

func TestTopologicalSort(t *testing.T) {
	pipeline := NewPipeline("test", "Test", "web").
		AddPhase(&PhaseDefinition{
			Name:      "phase3",
			Objective: "Third",
			Tools:     []string{"tool3"},
			DependsOn: []string{"phase1", "phase2"},
			MinIterations: 1,
			MaxIterations: 3,
		}).
		AddPhase(&PhaseDefinition{
			Name:      "phase1",
			Objective: "First",
			Tools:     []string{"tool1"},
			MinIterations: 1,
			MaxIterations: 3,
		}).
		AddPhase(&PhaseDefinition{
			Name:      "phase2",
			Objective: "Second",
			Tools:     []string{"tool2"},
			DependsOn: []string{"phase1"},
			MinIterations: 1,
			MaxIterations: 3,
		})

	sorted, err := pipeline.TopologicalSort()
	assert.NoError(t, err)
	assert.Equal(t, 3, len(sorted))

	// phase1 should come before phase2 and phase3
	// phase2 should come before phase3
	phase1Idx := -1
	phase2Idx := -1
	phase3Idx := -1

	for i, phase := range sorted {
		switch phase.Name {
		case "phase1":
			phase1Idx = i
		case "phase2":
			phase2Idx = i
		case "phase3":
			phase3Idx = i
		}
	}

	assert.Less(t, phase1Idx, phase2Idx)
	assert.Less(t, phase1Idx, phase3Idx)
	assert.Less(t, phase2Idx, phase3Idx)
}

func TestPredefinedPipelines(t *testing.T) {
	// web_full pipeline
	webFull, err := GetPredefinedPipeline("web_full")
	assert.NoError(t, err)
	assert.Equal(t, "web_full", webFull.Name)
	assert.Greater(t, len(webFull.Phases), 0)

	err = webFull.Validate()
	assert.NoError(t, err)

	// web_quick pipeline
	webQuick, err := GetPredefinedPipeline("web_quick")
	assert.NoError(t, err)
	assert.Equal(t, "web_quick", webQuick.Name)

	err = webQuick.Validate()
	assert.NoError(t, err)

	// Non-existent pipeline
	_, err = GetPredefinedPipeline("nonexistent")
	assert.Error(t, err)
}

func TestPredefinedPipelineStructure(t *testing.T) {
	webFull, _ := GetPredefinedPipeline("web_full")

	// Check phases exist
	reconPhase, err := webFull.GetPhase("recon")
	assert.NoError(t, err)
	assert.Equal(t, "recon", reconPhase.Name)

	portScanPhase, err := webFull.GetPhase("port_scan")
	assert.NoError(t, err)
	assert.Contains(t, portScanPhase.DependsOn, "recon")

	servicePhase, err := webFull.GetPhase("service_discovery")
	assert.NoError(t, err)
	assert.Contains(t, servicePhase.DependsOn, "port_scan")

	vulnPhase, err := webFull.GetPhase("vulnerability_scan")
	assert.NoError(t, err)
	assert.Contains(t, vulnPhase.DependsOn, "service_discovery")
}
