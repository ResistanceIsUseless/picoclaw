package phase

import (
	"strings"
	"testing"
	"time"

	"github.com/ResistanceIsUseless/picoclaw/pkg/artifacts"
	"github.com/ResistanceIsUseless/picoclaw/pkg/blackboard"
	"github.com/ResistanceIsUseless/picoclaw/pkg/graph"
	"github.com/stretchr/testify/assert"
)

func TestNewPhaseContextBuilder(t *testing.T) {
	builder := NewPhaseContextBuilder("recon", "Discover subdomains", 10000)

	assert.Equal(t, "recon", builder.PhaseName)
	assert.Equal(t, "Discover subdomains", builder.PhaseObjective)
	assert.Equal(t, 10000, builder.TokenBudget)
	assert.True(t, builder.KVCacheEnabled)
}

func TestBuild(t *testing.T) {
	builder := NewPhaseContextBuilder("recon", "Discover subdomains", 10000)

	state := NewDAGState("recon", []string{"subfinder", "amass"}, nil)
	contract := NewPhaseContract("recon").
		AddRequiredTool("subfinder").
		AddRequiredArtifact(artifacts.ArtifactSubdomainList)

	bb := blackboard.New(nil)
	g := graph.NewGraph()

	input := &PhaseContextInput{
		PhaseName:      "recon",
		Objective:      "Discover subdomains",
		Contract:       contract,
		State:          state,
		Blackboard:     bb,
		Graph:          g,
		Frontier:       nil,
		Registry:       nil,
		Iteration:      1,
		PreviousPhases: []string{},
	}

	sections, err := builder.Build(input)

	assert.NoError(t, err)
	assert.Greater(t, len(sections), 0)

	// Check that key sections are present
	sectionNames := make(map[string]bool)
	for _, section := range sections {
		sectionNames[section.Name] = true
	}

	assert.True(t, sectionNames["system_prompt"])
	assert.True(t, sectionNames["phase_context"])
	assert.True(t, sectionNames["dag_state"])
	assert.True(t, sectionNames["contract_status"])
}

func TestBuildSystemPrompt(t *testing.T) {
	builder := NewPhaseContextBuilder("recon", "Discover subdomains", 10000)

	state := NewDAGState("recon", []string{"subfinder"}, nil)
	contract := NewPhaseContract("recon").AddRequiredTool("subfinder")

	input := &PhaseContextInput{
		PhaseName: "recon",
		Objective: "Discover subdomains",
		Contract:  contract,
		State:     state,
		Iteration: 1,
	}

	prompt := builder.buildSystemPrompt(input)

	assert.Contains(t, prompt, "Security Assessment Agent")
	assert.Contains(t, prompt, "Core Principles")
	assert.Contains(t, prompt, "Tool calls are the only way to progress")
	assert.Contains(t, prompt, "Never hallucinate data")
	assert.Contains(t, prompt, "Orchestrator Tools")
	assert.Contains(t, prompt, "complete_phase")
}

func TestBuildPhaseContext(t *testing.T) {
	builder := NewPhaseContextBuilder("recon", "Discover subdomains", 10000)

	contract := NewPhaseContract("recon").
		AddRequiredTool("subfinder").
		AddRequiredArtifact(artifacts.ArtifactSubdomainList).
		SetIterationLimits(1, 5)

	input := &PhaseContextInput{
		PhaseName: "recon",
		Objective: "Discover subdomains",
		Contract:  contract,
		Iteration: 2,
	}

	phaseCtx := builder.buildPhaseContext(input)

	assert.Contains(t, phaseCtx, "Current Phase: recon")
	assert.Contains(t, phaseCtx, "Discover subdomains")
	assert.Contains(t, phaseCtx, "**Iteration**: 2")
	assert.Contains(t, phaseCtx, "Required Tool Executions")
	assert.Contains(t, phaseCtx, "subfinder")
	assert.Contains(t, phaseCtx, "Required Artifact Production")
	assert.Contains(t, phaseCtx, artifacts.ArtifactSubdomainList)
	assert.Contains(t, phaseCtx, "Minimum: 1 iterations")
	assert.Contains(t, phaseCtx, "Maximum: 5 iterations")
}

func TestBuildInputArtifacts(t *testing.T) {
	builder := NewPhaseContextBuilder("port_scan", "Scan for open ports", 10000)

	// Create persister so we can track phases
	tempDir := t.TempDir()
	persister, err := blackboard.NewFilePersister(tempDir)
	assert.NoError(t, err)
	bb := blackboard.New(persister)

	// Add artifact from previous phase
	sublist := &artifacts.SubdomainList{
		Metadata: blackboard.ArtifactMetadata{
			Type:      "SubdomainList",
			CreatedAt: time.Now(),
			Phase:     "recon",
			Domain:    "web",
		},
		BaseDomain: "example.com",
		Subdomains: []artifacts.Subdomain{}, // Initialize empty slice to pass validation
		Total:      5,
	}
	err = bb.Publish(nil, sublist)
	assert.NoError(t, err)

	input := &PhaseContextInput{
		PhaseName:      "port_scan",
		Blackboard:     bb,
		PreviousPhases: []string{"recon"},
	}

	artifactsCtx := builder.buildInputArtifacts(input)

	assert.Contains(t, artifactsCtx, "Input Artifacts")
	assert.Contains(t, artifactsCtx, "From recon phase")
	assert.Contains(t, artifactsCtx, "SubdomainList")
}

func TestBuildGraphState(t *testing.T) {
	builder := NewPhaseContextBuilder("recon", "Discover subdomains", 10000)

	g := graph.NewGraph()

	// Add some nodes
	g.AddNode(&graph.Node{
		ID:         "subdomain-1",
		EntityType: graph.EntitySubdomain,
		Label:      "api.example.com",
	})
	g.AddNode(&graph.Node{
		ID:         "subdomain-2",
		EntityType: graph.EntitySubdomain,
		Label:      "www.example.com",
	})

	input := &PhaseContextInput{
		Graph: g,
	}

	graphCtx := builder.buildGraphState(input)

	assert.Contains(t, graphCtx, "Knowledge Graph State")
	assert.Contains(t, graphCtx, "Nodes: 2")
	assert.Contains(t, graphCtx, "Edges: 0")
}

func TestBuildFrontierState(t *testing.T) {
	builder := NewPhaseContextBuilder("recon", "Discover subdomains", 10000)

	g := graph.NewGraph()
	registry := graph.NewEntityRegistry()

	// Add node with unknown properties
	g.AddNode(&graph.Node{
		ID:         "subdomain-1",
		EntityType: graph.EntitySubdomain,
		Label:      "api.example.com",
	})
	g.MarkPropertyUnknown("subdomain-1", "ip_addresses")
	g.MarkPropertyUnknown("subdomain-1", "ports")

	frontier := g.ComputeFrontier(registry)

	input := &PhaseContextInput{
		Frontier: frontier,
	}

	frontierCtx := builder.buildFrontierState(input)

	assert.Contains(t, frontierCtx, "Exploration Frontier")
	assert.Contains(t, frontierCtx, "Recommended Tools")
}

func TestCalculateTotalTokens(t *testing.T) {
	builder := NewPhaseContextBuilder("recon", "Discover subdomains", 10000)

	sections := []ContextSection{
		{Name: "section1", TokenCost: 100},
		{Name: "section2", TokenCost: 200},
		{Name: "section3", TokenCost: 300},
	}

	total := builder.calculateTotalTokens(sections)

	assert.Equal(t, 600, total)
}

func TestEstimateTokens(t *testing.T) {
	// Rough estimate: 4 chars ≈ 1 token
	text := strings.Repeat("a", 400)
	tokens := estimateTokens(text)

	assert.Equal(t, 100, tokens)
}

func TestRenderSections(t *testing.T) {
	sections := []ContextSection{
		{
			Name:     "section1",
			Priority: 100,
			Content:  "Section 1 content",
		},
		{
			Name:     "section2",
			Priority: 50,
			Content:  "Section 2 content",
		},
	}

	rendered := RenderSections(sections)

	assert.Contains(t, rendered, "Section 1 content")
	assert.Contains(t, rendered, "Section 2 content")
	assert.Contains(t, rendered, "---")
}

func TestPhaseContextCache(t *testing.T) {
	cache := NewPhaseContextCache()

	sections := []ContextSection{
		{
			Name:      "static_section",
			Content:   "Static content",
			Cacheable: true,
		},
		{
			Name:      "dynamic_section",
			Content:   "Dynamic content",
			Cacheable: false,
		},
	}

	// First call - should mark as cacheable
	marked := cache.MarkCacheable(sections)
	assert.True(t, marked[0].Cacheable)
	assert.False(t, marked[1].Cacheable)

	stats := cache.GetStats()
	assert.Equal(t, 0, stats["hits"])
	assert.Equal(t, 1, stats["misses"])

	// Second call with same content - should be cache hit
	marked = cache.MarkCacheable(sections)
	assert.True(t, marked[0].Cacheable)

	stats = cache.GetStats()
	assert.Equal(t, 1, stats["hits"])
	assert.Equal(t, 1, stats["misses"])

	// Third call with changed content - should be cache miss
	sections[0].Content = "Changed content"
	marked = cache.MarkCacheable(sections)
	assert.False(t, marked[0].Cacheable)

	stats = cache.GetStats()
	assert.Equal(t, 1, stats["hits"])
	assert.Equal(t, 2, stats["misses"])
}

func TestSectionPriority(t *testing.T) {
	builder := NewPhaseContextBuilder("recon", "Discover subdomains", 10000)

	state := NewDAGState("recon", []string{"subfinder"}, nil)
	contract := NewPhaseContract("recon").AddRequiredTool("subfinder")
	bb := blackboard.New(nil)
	g := graph.NewGraph()

	input := &PhaseContextInput{
		PhaseName:      "recon",
		Objective:      "Discover subdomains",
		Contract:       contract,
		State:          state,
		Blackboard:     bb,
		Graph:          g,
		Iteration:      1,
		PreviousPhases: []string{},
	}

	sections, err := builder.Build(input)
	assert.NoError(t, err)

	// Verify system prompt has highest priority
	var systemPromptPriority int
	for _, section := range sections {
		if section.Name == "system_prompt" {
			systemPromptPriority = section.Priority
			break
		}
	}

	// All other sections should have lower priority
	for _, section := range sections {
		if section.Name != "system_prompt" {
			assert.Less(t, section.Priority, systemPromptPriority)
		}
	}
}

func TestCacheableFlags(t *testing.T) {
	builder := NewPhaseContextBuilder("recon", "Discover subdomains", 10000)

	state := NewDAGState("recon", []string{"subfinder"}, nil)
	contract := NewPhaseContract("recon").AddRequiredTool("subfinder")
	bb := blackboard.New(nil)
	g := graph.NewGraph()

	input := &PhaseContextInput{
		PhaseName:  "recon",
		Objective:  "Discover subdomains",
		Contract:   contract,
		State:      state,
		Blackboard: bb,
		Graph:      g,
		Iteration:  1,
	}

	sections, err := builder.Build(input)
	assert.NoError(t, err)

	// Verify cacheable flags
	for _, section := range sections {
		switch section.Name {
		case "system_prompt", "phase_context":
			assert.True(t, section.Cacheable, "Section %s should be cacheable", section.Name)
		case "dag_state", "contract_status":
			assert.False(t, section.Cacheable, "Section %s should not be cacheable", section.Name)
		}
	}
}

func TestTokenBudgetWarning(t *testing.T) {
	// Create builder with very small token budget
	builder := NewPhaseContextBuilder("recon", "Discover subdomains", 10)

	state := NewDAGState("recon", []string{"subfinder"}, nil)
	contract := NewPhaseContract("recon").AddRequiredTool("subfinder")
	bb := blackboard.New(nil)
	g := graph.NewGraph()

	input := &PhaseContextInput{
		PhaseName:  "recon",
		Objective:  "Discover subdomains",
		Contract:   contract,
		State:      state,
		Blackboard: bb,
		Graph:      g,
		Iteration:  1,
	}

	sections, err := builder.Build(input)
	assert.NoError(t, err)

	// Should still return sections even if over budget
	assert.Greater(t, len(sections), 0)

	// Total tokens should exceed budget
	total := builder.calculateTotalTokens(sections)
	assert.Greater(t, total, builder.TokenBudget)
}
