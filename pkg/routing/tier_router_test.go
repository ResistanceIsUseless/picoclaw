package routing

import (
	"context"
	"fmt"
	"testing"

	"github.com/ResistanceIsUseless/picoclaw/pkg/config"
	"github.com/ResistanceIsUseless/picoclaw/pkg/providers"
)

// Mock provider for testing
type mockProvider struct {
	responses map[string]*providers.LLMResponse
	errors    map[string]error
	callCount map[string]int
}

func newMockProvider() *mockProvider {
	return &mockProvider{
		responses: make(map[string]*providers.LLMResponse),
		errors:    make(map[string]error),
		callCount: make(map[string]int),
	}
}

func (m *mockProvider) Chat(ctx context.Context, messages []providers.Message, tools []providers.ToolDefinition, model string, opts map[string]any) (*providers.LLMResponse, error) {
	key := model
	if m.errors[key] != nil {
		return nil, m.errors[key]
	}
	
	resp := m.responses[key]
	if resp == nil {
		// Default response
		resp = &providers.LLMResponse{
			Content: "Mock response",
			Usage: &providers.UsageInfo{
				PromptTokens:     10,
				CompletionTokens: 20,
				TotalTokens:     30,
			},
		}
	}
	
	m.callCount[key]++
	return resp, nil
}

func (m *mockProvider) setResponse(model string, resp *providers.LLMResponse) {
	m.responses[model] = resp
}

func (m *mockProvider) setError(model string, err error) {
	m.errors[model] = err
}

func (m *mockProvider) getCallCount(model string) int {
	return m.callCount[model]
}

func (m *mockProvider) GetDefaultModel() string {
	return "claude-3-haiku"
}

// Helper to create test routing config
func testRoutingConfig() *config.RoutingConfig {
	return &config.RoutingConfig{
		Enabled:    true,
		DefaultTier: "fast",
		Tiers: map[string]config.TierConfig{
			"fast": {
				ModelName: "claude-3-haiku",
				UseFor:    []string{"simple", "fast"},
				CostPerM: config.CostPerMInfo{
					Input:  0.25,
					Output: 1.25,
				},
			},
			"balanced": {
				ModelName: "claude-3-sonnet",
				UseFor:    []string{"analysis", "moderate"},
				CostPerM: config.CostPerMInfo{
					Input:  3.0,
					Output: 15.0,
				},
			},
			"powerful": {
				ModelName: "claude-3-opus",
				UseFor:    []string{"complex", "security"},
				CostPerM: config.CostPerMInfo{
					Input:  15.0,
					Output: 75.0,
				},
			},
		},
		EnableSupervision: true,
		SupervisorTier:    "powerful",
		ValidationConfidenceThreshold: 0.8,
		MinTaskComplexityForSupervision: 5,
	}
}

// Helper to create test model list
func testModelList() []config.ModelConfig {
	return []config.ModelConfig{
		{ModelName: "claude-3-haiku", Model: "claude-3-haiku"},
		{ModelName: "claude-3-sonnet", Model: "claude-3-sonnet"},
		{ModelName: "claude-3-opus", Model: "claude-3-opus"},
		{ModelName: "gpt-4", Model: "gpt-4"},
	}
}

func TestTierRouter_Init(t *testing.T) {
	cfg := testRoutingConfig()
	models := testModelList()
	provider := newMockProvider()
	
	router := NewTierRouter(cfg, models, map[string]providers.LLMProvider{"test": provider})
	
	if router == nil {
		t.Fatal("Expected router to be created")
	}
	
	if !router.IsEnabled() {
		t.Error("Expected router to be enabled")
	}
}

func TestTierRouter_ClassifyTask(t *testing.T) {
	cfg := testRoutingConfig()
	models := testModelList()
	provider := newMockProvider()
	router := NewTierRouter(cfg, models, map[string]providers.LLMProvider{"test": provider})
	
	tests := []struct {
		name     string
		ctx      AgentContext
		expected TaskType
	}{
		{
			name: "Simple task should use fast tier",
			ctx: AgentContext{
				TurnCount:      1,
				UserMessage:    "Hello, how are you?",
				ToolsAvailable: 0,
			},
			expected: TaskAnalysis, // Simple tasks typically classified as analysis
		},
		{
			name: "Security task should require supervision",
			ctx: AgentContext{
				TurnCount:      1,
				UserMessage:    "Find security vulnerabilities in this code",
				ToolsAvailable: 5,
				RequiresSupervision: true,
			},
			expected: TaskCodeReview, // Security tasks typically code review
		},
		{
			name: "Code execution task",
			ctx: AgentContext{
				TurnCount:      2,
				UserMessage:    "Run this Python script",
				LastToolOutput: "Script executed successfully",
				ToolsAvailable: 3,
			},
			expected: TaskAnalysis, // Code execution typically analysis
		},
		{
			name: "Complex multi-turn task",
			ctx: AgentContext{
				TurnCount:      5,
				UserMessage:    "Continue the analysis",
				LastToolOutput: "Found potential issues",
				ToolsAvailable: 8,
				RequiresSupervision: true,
			},
			expected: TaskAnalysis, // Complex tasks also analysis for now
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			taskType := router.ClassifyTask(tt.ctx)
			if taskType != tt.expected {
				t.Errorf("ClassifyTask() = %q, want %q", taskType, tt.expected)
			}
		})
	}
}

func TestTierRouter_RouteChat_NoSupervision(t *testing.T) {
	cfg := testRoutingConfig()
	models := testModelList()
	provider := newMockProvider()
	
	// Set up mock response
	provider.setResponse("claude-3-haiku", &providers.LLMResponse{
		Content: "Hello! How can I help you?",
		Usage: &providers.UsageInfo{
			PromptTokens:     10,
			CompletionTokens: 5,
			TotalTokens:     15,
		},
	})
	
	// Create providers map with model name as key
	providersMap := map[string]providers.LLMProvider{
		"claude-3-haiku": provider,
	}
	
	router := NewTierRouter(cfg, models, providersMap)
	
	messages := []providers.Message{
		{Role: "user", Content: "Hello"},
	}
	tools := []providers.ToolDefinition{}
	opts := map[string]any{}
	
	resp, err := router.RouteChat(context.Background(), "fast", messages, tools, opts, "test-session")
	if err != nil {
		t.Fatalf("RouteChat() failed: %v", err)
	}
	
	if resp.Content != "Hello! How can I help you?" {
		t.Errorf("Expected content to match mock response")
	}
	
	if provider.getCallCount("claude-3-haiku") != 1 {
		t.Errorf("Expected 1 call to claude-3-haiku, got %d", provider.getCallCount("claude-3-haiku"))
	}
}

func TestTierRouter_RouteWithSupervision_Success(t *testing.T) {
	cfg := testRoutingConfig()
	models := testModelList()
	provider := newMockProvider()
	costTracker := NewCostTracker()
	
	// Set up mock responses
	provider.setResponse("claude-3-haiku", &providers.LLMResponse{
		Content: "Here's the code analysis: no vulnerabilities found",
		Usage: &providers.UsageInfo{
			PromptTokens:     20,
			CompletionTokens: 30,
			TotalTokens:     50,
		},
	})
	
	provider.setResponse("claude-3-opus", &providers.LLMResponse{
		Content: `{"decision": "approve", "confidence": 0.95, "reasoning": "Analysis is accurate and complete"}`,
		Usage: &providers.UsageInfo{
			PromptTokens:     30,
			CompletionTokens: 20,
			TotalTokens:     50,
		},
	})
	
	// Create providers map with model names as keys
	providersMap := map[string]providers.LLMProvider{
		"claude-3-haiku": provider,
		"claude-3-opus":  provider,
	}
	
	router := NewTierRouter(cfg, models, providersMap)
	router.supervisor.costTracker = costTracker
	
	messages := []providers.Message{
		{Role: "user", Content: "Analyze this code for security vulnerabilities"},
	}
	tools := []providers.ToolDefinition{}
	opts := map[string]any{}
	
	ctx := AgentContext{
		TurnCount:      1,
		UserMessage:    "Analyze this code for security vulnerabilities",
		RequiresSupervision: true,
	}
	
	result, err := router.RouteWithSupervision(context.Background(), "balanced", messages, tools, opts, "test-session", ctx)
	if err != nil {
		t.Fatalf("RouteWithSupervision() failed: %v", err)
	}
	
	if !result.Validated {
		t.Error("Expected result to be validated")
	}
	
	if result.SupervisorModel != "claude-3-opus" {
		t.Errorf("Expected supervisor model claude-3-opus, got %q", result.SupervisorModel)
	}
	
	if result.WorkerModel != "claude-3-haiku" {
		t.Errorf("Expected worker model claude-3-haiku, got %q", result.WorkerModel)
	}
	
	if provider.getCallCount("claude-3-haiku") != 1 {
		t.Errorf("Expected 1 call to worker model, got %d", provider.getCallCount("claude-3-haiku"))
	}
	
	if provider.getCallCount("claude-3-opus") != 1 {
		t.Errorf("Expected 1 call to supervisor model, got %d", provider.getCallCount("claude-3-opus"))
	}
	
	// Check cost tracking
	sessionCost := costTracker.GetSessionCost("test-session")
	if sessionCost == nil {
		t.Fatal("Expected session cost to be tracked")
	}
	
	if sessionCost.Supervision.TotalSupervisions == 0 {
		t.Error("Expected supervision metrics to be tracked")
	}
}

func TestTierRouter_RouteWithSupervision_Correction(t *testing.T) {
	cfg := testRoutingConfig()
	models := testModelList()
	provider := newMockProvider()
	costTracker := NewCostTracker()
	
	// Set up mock responses - first attempt fails validation
	provider.setResponse("claude-3-haiku", &providers.LLMResponse{
		Content: "This code is perfectly safe, no issues at all",
		Usage: &providers.UsageInfo{
			PromptTokens:     20,
			CompletionTokens: 30,
			TotalTokens:     50,
		},
	})
	
	// Supervisor rejects first attempt
	provider.setResponse("claude-3-opus", &providers.LLMResponse{
		Content: `{"decision": "reject", "confidence": 0.9, "reasoning": "Analysis missed critical SQL injection vulnerability", "corrections": ["Add input validation", "Use parameterized queries"]}`,
		Usage: &providers.UsageInfo{
			PromptTokens:     30,
			CompletionTokens: 40,
			TotalTokens:     70,
		},
	})
	
	// Second attempt after correction
	provider.setResponse("claude-3-sonnet", &providers.LLMResponse{
		Content: "Found SQL injection vulnerability. Fixed with parameterized queries and input validation.",
		Usage: &providers.UsageInfo{
			PromptTokens:     25,
			CompletionTokens: 35,
			TotalTokens:     60,
		},
	})
	
	// Supervisor approves corrected version
	provider.responses["claude-3-opus-2"] = &providers.LLMResponse{
		Content: `{"decision": "approve", "confidence": 0.98, "reasoning": "Corrections properly address the security issues"}`,
		Usage: &providers.UsageInfo{
			PromptTokens:     35,
			CompletionTokens: 25,
			TotalTokens:     60,
		},
	}
	
	// Create providers map with model names as keys
	providersMap := map[string]providers.LLMProvider{
		"claude-3-haiku":  provider,
		"claude-3-sonnet": provider,
		"claude-3-opus":   provider,
	}
	
	router := NewTierRouter(cfg, models, providersMap)
	router.supervisor.costTracker = costTracker
	
	messages := []providers.Message{
		{Role: "user", Content: "Analyze this code for security vulnerabilities"},
	}
	tools := []providers.ToolDefinition{}
	opts := map[string]any{}
	
	ctx := AgentContext{
		TurnCount:      1,
		UserMessage:    "Analyze this code for security vulnerabilities",
		RequiresSupervision: true,
	}
	
	result, err := router.RouteWithSupervision(context.Background(), "balanced", messages, tools, opts, "test-session", ctx)
	if err != nil {
		t.Fatalf("RouteWithSupervision() failed: %v", err)
	}
	
	if !result.Validated {
		t.Error("Expected final result to be validated after correction")
	}
	
	if len(result.Corrections) == 0 {
		t.Error("Expected corrections to be recorded")
	}
	
	// Check that corrections were applied (len > 0 implies correction attempts)
	if len(result.Corrections) == 0 {
		t.Error("Expected correction attempts to be recorded via corrections")
	}
	
	// Check that both models were called
	if provider.getCallCount("claude-3-haiku") != 1 {
		t.Errorf("Expected 1 call to initial worker model, got %d", provider.getCallCount("claude-3-haiku"))
	}
	
	if provider.getCallCount("claude-3-sonnet") != 1 {
		t.Errorf("Expected 1 call to corrected worker model, got %d", provider.getCallCount("claude-3-sonnet"))
	}
	
	if provider.getCallCount("claude-3-opus") != 2 {
		t.Errorf("Expected 2 calls to supervisor model, got %d", provider.getCallCount("claude-3-opus"))
	}
}

func TestTierRouter_RouteWithSupervision_Fallback(t *testing.T) {
	cfg := testRoutingConfig()
	models := testModelList()
	provider := newMockProvider()
	costTracker := NewCostTracker()
	
	// Worker model succeeds
	provider.setResponse("claude-3-haiku", &providers.LLMResponse{
		Content: "Analysis complete",
		Usage: &providers.UsageInfo{
			PromptTokens:     20,
			CompletionTokens: 30,
			TotalTokens:     50,
		},
	})
	
	// Supervisor fails
	provider.setError("claude-3-opus", fmt.Errorf("supervisor unavailable"))
	
	// Create providers map with model names as keys
	providersMap := map[string]providers.LLMProvider{
		"claude-3-haiku": provider,
		"claude-3-opus":  provider,
	}
	
	router := NewTierRouter(cfg, models, providersMap)
	router.supervisor.costTracker = costTracker
	
	messages := []providers.Message{
		{Role: "user", Content: "Analyze this code"},
	}
	tools := []providers.ToolDefinition{}
	opts := map[string]any{}
	
	ctx := AgentContext{
		TurnCount:      1,
		UserMessage:    "Analyze this code",
		RequiresSupervision: true,
	}
	
	result, err := router.RouteWithSupervision(context.Background(), "balanced", messages, tools, opts, "test-session", ctx)
	if err != nil {
		t.Fatalf("RouteWithSupervision() failed: %v", err)
	}
	
	// Should fall back to original response
	if result.FinalOutput != "Analysis complete" {
		t.Errorf("Expected fallback to original response, got %q", result.FinalOutput)
	}
	
	if result.Validated {
		t.Error("Expected result not to be validated when supervisor fails")
	}
	
	// Check cost tracking records the failure
	sessionCost := costTracker.GetSessionCost("test-session")
	if sessionCost == nil {
		t.Fatal("Expected session cost to be tracked")
	}
	
	if sessionCost.Supervision.FailedValidations == 0 {
		t.Error("Expected supervision failure to be recorded")
	}
}

func TestTierRouter_CostTrackingIntegration(t *testing.T) {
	cfg := testRoutingConfig()
	models := testModelList()
	provider := newMockProvider()
	costTracker := NewCostTracker()
	
	// Set up responses with different costs
	provider.setResponse("claude-3-haiku", &providers.LLMResponse{
		Content: "Fast response",
		Usage: &providers.UsageInfo{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:     30,
		},
	})
	
	provider.setResponse("claude-3-opus", &providers.LLMResponse{
		Content: `{"decision": "approve", "confidence": 1.0}`,
		Usage: &providers.UsageInfo{
			PromptTokens:     50,
			CompletionTokens: 30,
			TotalTokens:     80,
		},
	})
	
	// Create providers map with model names as keys
	providersMap := map[string]providers.LLMProvider{
		"claude-3-haiku": provider,
		"claude-3-opus":  provider,
	}
	
	router := NewTierRouter(cfg, models, providersMap)
	router.supervisor.costTracker = costTracker
	
	messages := []providers.Message{
		{Role: "user", Content: "Test"},
	}
	tools := []providers.ToolDefinition{}
	opts := map[string]any{}
	
	ctx := AgentContext{
		TurnCount:      1,
		UserMessage:    "Test security analysis",
		RequiresSupervision: true,
	}
	
	// Execute supervised routing
	_, err := router.RouteWithSupervision(context.Background(), "balanced", messages, tools, opts, "test-session", ctx)
	if err != nil {
		t.Fatalf("RouteWithSupervision() failed: %v", err)
	}
	
	// Check cost tracking
	sessionCost := costTracker.GetSessionCost("test-session")
	if sessionCost == nil {
		t.Fatal("Expected session cost to be tracked")
	}
	
	// Should have both worker and supervisor costs
	if sessionCost.TotalCost <= 0 {
		t.Error("Expected total cost to be greater than 0")
	}
	
	if sessionCost.Supervision.TotalSupervisions != 1 {
		t.Errorf("Expected 1 supervised task, got %d", sessionCost.Supervision.TotalSupervisions)
	}
	
	if sessionCost.Supervision.TotalSupervisionCost <= 0 {
		t.Error("Expected supervision cost to be tracked")
	}
	
	// Check cost savings
	if sessionCost.Supervision.SupervisionSavings <= 0 {
		t.Error("Expected estimated savings to be calculated")
	}
}

func TestTierRouter_DisabledSupervision(t *testing.T) {
	cfg := testRoutingConfig()
	cfg.EnableSupervision = false
	models := testModelList()
	provider := newMockProvider()
	
	// Create providers map with model names as keys
	providersMap := map[string]providers.LLMProvider{
		"claude-3-haiku": provider,
	}
	
	router := NewTierRouter(cfg, models, providersMap)
	
	// Should route normally without supervision
	messages := []providers.Message{
		{Role: "user", Content: "Test"},
	}
	tools := []providers.ToolDefinition{}
	opts := map[string]any{}
	
	resp, err := router.RouteChat(context.Background(), "fast", messages, tools, opts, "test-session")
	if err != nil {
		t.Fatalf("RouteChat() failed with disabled supervision: %v", err)
	}
	
	if resp == nil {
		t.Error("Expected response from routing")
	}
}

func TestTierRouter_InvalidTier(t *testing.T) {
	cfg := testRoutingConfig()
	models := testModelList()
	provider := newMockProvider()
	
	// Create providers map
	providersMap := map[string]providers.LLMProvider{
		"claude-3-haiku": provider,
	}
	
	router := NewTierRouter(cfg, models, providersMap)
	
	messages := []providers.Message{
		{Role: "user", Content: "Test"},
	}
	tools := []providers.ToolDefinition{}
	opts := map[string]any{}
	
	_, err := router.RouteChat(context.Background(), "nonexistent-tier", messages, tools, opts, "test-session")
	if err == nil {
		t.Error("Expected error for invalid tier")
	}
}