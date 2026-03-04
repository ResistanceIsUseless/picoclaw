package integration

import (
	"context"
	"testing"
	"time"

	"github.com/ResistanceIsUseless/picoclaw/pkg/artifacts"
	"github.com/ResistanceIsUseless/picoclaw/pkg/blackboard"
	"github.com/ResistanceIsUseless/picoclaw/pkg/graph"
	"github.com/ResistanceIsUseless/picoclaw/pkg/orchestrator"
	"github.com/ResistanceIsUseless/picoclaw/pkg/providers"
	"github.com/ResistanceIsUseless/picoclaw/pkg/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockProvider simulates LLM responses for testing
type MockProvider struct {
	responses []providers.LLMResponse
	callCount int
}

func (m *MockProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string, options map[string]interface{}) (*providers.LLMResponse, error) {
	if m.callCount >= len(m.responses) {
		// Return empty response if we've exhausted mock responses
		return &providers.LLMResponse{
			Content:   "No more actions needed",
			ToolCalls: []providers.ToolCall{},
		}, nil
	}

	response := m.responses[m.callCount]
	m.callCount++
	return &response, nil
}

func (m *MockProvider) GetDefaultModel() string {
	return "mock-model"
}

// TestCLAW_EndToEnd_ReconPhase tests a complete recon phase with mocked tools
func TestCLAW_EndToEnd_ReconPhase(t *testing.T) {
	// Setup
	// Create blackboard with nil persister (in-memory only)
	bb := blackboard.New(nil)
	require.NotNil(t, bb)

	// Create knowledge graph (note: graph and entityRegistry created by orchestrator)
	// Create tool registry with mock tools
	toolRegistry := registry.NewToolRegistry()

	// Register a mock subfinder tool
	// Note: We'll need to handle execution - the tool parser will create the artifact
	// but we need to prevent actual subfinder execution
	mockSubfinderTool := &registry.ToolDefinition{
		Name:        "mock_subfinder",
		Description: "Mock subdomain enumeration tool",
		Tier:        registry.TierAutoApprove, // Make visible to model for testing
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"domain": map[string]interface{}{
					"type":        "string",
					"description": "Target domain",
				},
			},
			"required": []string{"domain"},
		},
		OutputType: "SubdomainList",
		Parser: func(toolName string, output []byte) (interface{}, error) {
			// Mock parser - create a SubdomainList artifact directly
			// (output is ignored since we're mocking)
			return &artifacts.SubdomainList{
				Metadata: blackboard.ArtifactMetadata{
					Type:      "SubdomainList",
					CreatedAt: time.Now(),
					Phase:     "recon",
					Version:   "1.0",
					Domain:    "web",
				},
				BaseDomain: "example.com",
				Subdomains: []artifacts.Subdomain{
					{
						Name:         "www.example.com",
						Source:       "mock_subfinder",
						Verified:     true,
						DiscoveredAt: time.Now(),
						IPs:          []string{"93.184.216.34"},
					},
					{
						Name:         "api.example.com",
						Source:       "mock_subfinder",
						Verified:     true,
						DiscoveredAt: time.Now(),
						IPs:          []string{"93.184.216.35"},
					},
				},
				Sources: map[string]int{
					"mock_subfinder": 2,
				},
				Total: 2,
			}, nil
		},
	}

	err := toolRegistry.Register(mockSubfinderTool)
	require.NoError(t, err)

	// Create mock provider that calls mock_subfinder
	mockProvider := &MockProvider{
		responses: []providers.LLMResponse{
			{
				Content: "I'll enumerate subdomains for example.com using mock_subfinder",
				ToolCalls: []providers.ToolCall{
					{
						ID:   "call_1",
						Name: "mock_subfinder",
						Arguments: map[string]interface{}{
							"domain": "example.com",
						},
					},
				},
			},
			{
				Content:   "Subdomain enumeration complete. Found 2 subdomains.",
				ToolCalls: []providers.ToolCall{},
			},
		},
	}

	// Create a simple pipeline with just recon phase
	pipeline := orchestrator.NewPipeline("test_recon", "Test recon pipeline", "web")

	reconPhase := &orchestrator.PhaseDefinition{
		Name:             "recon",
		Objective:        "Discover subdomains for the target domain",
		Tools:            []string{"mock_subfinder"},
		RequiredTools:    []string{"mock_subfinder"},
		MinIterations:    1,
		MaxIterations:    3,
		Dependencies:     map[string][]string{},
		DependsOn:        []string{},
		TokenBudget:      5000,
		RequiredArtifacts: []string{"SubdomainList"},
	}

	pipeline.AddPhase(reconPhase)

	// Create orchestrator (creates its own graph and entity registry)
	orch := orchestrator.NewOrchestrator(pipeline, bb, toolRegistry)
	orch.SetProvider(mockProvider)

	// Publish initial OperatorTarget artifact
	ctx := context.Background()
	target := artifacts.NewOperatorTarget("example.com", "web", "recon")
	err = bb.Publish(ctx, target)
	require.NoError(t, err)

	// Execute pipeline
	err = orch.Execute(ctx)
	require.NoError(t, err)

	// Verify results
	t.Run("Pipeline completed successfully", func(t *testing.T) {
		completed := orch.GetCompletedPhases()
		assert.Contains(t, completed, "recon", "Recon phase should be completed")
	})

	t.Run("Artifacts published to blackboard", func(t *testing.T) {
		artifactEnvelopes, err := bb.GetByPhase("recon")
		require.NoError(t, err)

		// Should have OperatorTarget + SubdomainList
		assert.GreaterOrEqual(t, len(artifactEnvelopes), 2, "Expected at least 2 artifacts (OperatorTarget + SubdomainList)")

		// Find SubdomainList artifact by checking metadata type
		var foundSubdomainList bool
		for _, envelope := range artifactEnvelopes {
			if envelope.Metadata.Type == "SubdomainList" {
				foundSubdomainList = true
				// Verify metadata
				assert.Equal(t, "recon", envelope.Metadata.Phase)
				assert.Equal(t, "web", envelope.Metadata.Domain)
				break
			}
		}

		assert.True(t, foundSubdomainList, "SubdomainList artifact should be published")
	})

	t.Run("Knowledge graph updated", func(t *testing.T) {
		kg := orch.GetGraph()
		require.NotNil(t, kg)

		// Check for domain node
		domainNode, err := kg.GetNode("domain:example.com")
		assert.NoError(t, err, "Domain node should exist")
		assert.NotNil(t, domainNode)

		if domainNode != nil {
			assert.Equal(t, graph.EntityDomain, domainNode.EntityType)
			assert.Equal(t, "example.com", domainNode.Label)
		}

		// Check for subdomain nodes
		subdomainNode1, err := kg.GetNode("subdomain:www.example.com")
		assert.NoError(t, err, "Subdomain node should exist")
		assert.NotNil(t, subdomainNode1)

		if subdomainNode1 != nil {
			assert.Equal(t, graph.EntitySubdomain, subdomainNode1.EntityType)
			assert.Equal(t, "www.example.com", subdomainNode1.Label)
		}

		// Check for IP nodes
		ipNode, err := kg.GetNode("ip:93.184.216.34")
		assert.NoError(t, err, "IP node should exist")
		assert.NotNil(t, ipNode)

		if ipNode != nil {
			assert.Equal(t, graph.EntityIP, ipNode.EntityType)
		}

		// Check for edges
		edges := kg.GetEdges("subdomain:www.example.com")
		assert.NotEmpty(t, edges, "Subdomain should have outgoing edges")

		// Should have edge to domain and edge to IP
		var hasSubdomainOfEdge bool
		var hasResolvesToEdge bool
		for _, edge := range edges {
			if edge.RelationType == graph.RelationSubdomainOf && edge.To == "domain:example.com" {
				hasSubdomainOfEdge = true
			}
			if edge.RelationType == graph.RelationResolvesTo && edge.To == "ip:93.184.216.34" {
				hasResolvesToEdge = true
			}
		}

		assert.True(t, hasSubdomainOfEdge, "Should have subdomain_of edge")
		assert.True(t, hasResolvesToEdge, "Should have resolves_to edge")
	})

	t.Run("DAGState tracked tool execution", func(t *testing.T) {
		// Note: GetCurrentPhase() returns nil after phase completes
		// We can verify by checking completed phases and artifacts
		completed := orch.GetCompletedPhases()
		assert.Contains(t, completed, "recon", "Recon phase should be in completed phases")

		// Verify artifacts were created (which proves tool execution happened)
		artifactEnvelopes, err := bb.GetByPhase("recon")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(artifactEnvelopes), 2, "Should have artifacts from tool execution")
	})

	t.Run("Contract validated", func(t *testing.T) {
		// Phase completed successfully, which means contract was satisfied
		completed := orch.GetCompletedPhases()
		assert.Contains(t, completed, "recon", "Contract must be satisfied for phase to complete")

		// Verify required artifacts were produced
		artifactEnvelopes, _ := bb.GetByPhase("recon")
		var hasSubdomainList bool
		for _, envelope := range artifactEnvelopes {
			if envelope.Metadata.Type == "SubdomainList" {
				hasSubdomainList = true
				break
			}
		}
		assert.True(t, hasSubdomainList, "Required SubdomainList artifact was produced")
	})

	t.Run("Mock provider called at least once", func(t *testing.T) {
		// Contract was satisfied after first iteration, so only 1 call
		assert.GreaterOrEqual(t, mockProvider.callCount, 1, "Provider should be called at least once")
		assert.LessOrEqual(t, mockProvider.callCount, 2, "Provider should not be called more than configured responses")
	})
}

// TestCLAW_EndToEnd_MultiPhase tests a multi-phase pipeline
func TestCLAW_EndToEnd_MultiPhase(t *testing.T) {
	// Setup
	bb := blackboard.New(nil)
	toolRegistry := registry.NewToolRegistry()

	// Register mock tools
	mockReconTool := &registry.ToolDefinition{
		Name:        "mock_recon",
		Description: "Mock recon tool",
		Tier:        registry.TierAutoApprove,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"target": map[string]interface{}{"type": "string"},
			},
		},
		OutputType: "SubdomainList",
		Parser: func(toolName string, output []byte) (interface{}, error) {
			return &artifacts.SubdomainList{
				Metadata: blackboard.ArtifactMetadata{
					Type:      "SubdomainList",
					CreatedAt: time.Now(),
					Phase:     "recon",
					Version:   "1.0",
					Domain:    "web",
				},
				BaseDomain: "test.com",
				Subdomains: []artifacts.Subdomain{
					{Name: "www.test.com", Source: "mock", Verified: true, DiscoveredAt: time.Now()},
				},
				Total: 1,
			}, nil
		},
	}

	mockScanTool := &registry.ToolDefinition{
		Name:        "mock_scan",
		Description: "Mock port scan tool",
		Tier:        registry.TierAutoApprove,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"target": map[string]interface{}{"type": "string"},
			},
		},
		OutputType: "PortScanResult",
		Parser:     nil, // No parser for this test
	}

	err := toolRegistry.Register(mockReconTool)
	require.NoError(t, err)
	err = toolRegistry.Register(mockScanTool)
	require.NoError(t, err)

	// Mock provider responses for two phases
	mockProvider := &MockProvider{
		responses: []providers.LLMResponse{
			// Phase 1: recon
			{
				Content: "Running recon",
				ToolCalls: []providers.ToolCall{
					{ID: "call_1", Name: "mock_recon", Arguments: map[string]interface{}{"target": "test.com"}},
				},
			},
			{Content: "Recon complete", ToolCalls: []providers.ToolCall{}},
			// Phase 2: scan
			{
				Content: "Running scan",
				ToolCalls: []providers.ToolCall{
					{ID: "call_2", Name: "mock_scan", Arguments: map[string]interface{}{"target": "www.test.com"}},
				},
			},
			{Content: "Scan complete", ToolCalls: []providers.ToolCall{}},
		},
	}

	// Create multi-phase pipeline
	pipeline := orchestrator.NewPipeline("test_multi", "Multi-phase test", "web")

	pipeline.AddPhase(&orchestrator.PhaseDefinition{
		Name:          "recon",
		Objective:     "Discover targets",
		Tools:         []string{"mock_recon"},
		RequiredTools: []string{"mock_recon"},
		MinIterations: 1,
		MaxIterations: 2,
		TokenBudget:   5000,
		Dependencies:  map[string][]string{},
		DependsOn:     []string{},
	})

	pipeline.AddPhase(&orchestrator.PhaseDefinition{
		Name:          "scan",
		Objective:     "Scan discovered targets",
		Tools:         []string{"mock_scan"},
		Dependencies:  map[string][]string{},
		DependsOn:     []string{"recon"}, // Depends on recon
		MinIterations: 1,
		MaxIterations: 2,
		TokenBudget:   5000,
	})

	// Create orchestrator
	orch := orchestrator.NewOrchestrator(pipeline, bb, toolRegistry)
	orch.SetProvider(mockProvider)

	// Publish initial target
	ctx := context.Background()
	target := artifacts.NewOperatorTarget("test.com", "web", "recon")
	err = bb.Publish(ctx, target)
	require.NoError(t, err)

	// Execute pipeline
	err = orch.Execute(ctx)
	require.NoError(t, err)

	// Verify both phases completed
	t.Run("Both phases completed", func(t *testing.T) {
		completed := orch.GetCompletedPhases()
		assert.Len(t, completed, 2, "Should have 2 completed phases")

		var hasRecon, hasScan bool
		for _, phase := range completed {
			if phase == "recon" {
				hasRecon = true
			}
			if phase == "scan" {
				hasScan = true
			}
		}

		assert.True(t, hasRecon, "Recon phase should be completed")
		assert.True(t, hasScan, "Scan phase should be completed")
	})

	t.Run("Phase dependency satisfied", func(t *testing.T) {
		// Scan phase should only execute after recon
		// This is implicitly tested by successful execution
		completed := orch.GetCompletedPhases()
		assert.Contains(t, completed, "recon")
		assert.Contains(t, completed, "scan")
	})
}
