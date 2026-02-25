package routing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
)

// TaskType represents the classification of an LLM task for tier routing
type TaskType string

const (
	TaskPlanning      TaskType = "planning"       // Strategic decisions, initial planning
	TaskAnalysis      TaskType = "analysis"       // Deep analysis, reasoning about findings
	TaskExploitation  TaskType = "exploitation"   // Testing vulnerabilities, exploit development
	TaskReportWriting TaskType = "report_writing" // Final reporting, documentation

	TaskToolSelection TaskType = "tool_selection" // Choosing which tool to use
	TaskCodeReview    TaskType = "code_review"    // Analyzing JavaScript, code, configs
	TaskJSAnalysis    TaskType = "js_analysis"    // Specific JavaScript analysis

	TaskParsing       TaskType = "parsing"        // Parsing tool output
	TaskSummary       TaskType = "summary"        // Summarizing large output
	TaskFormatting    TaskType = "formatting"     // Formatting responses
	TaskTriage        TaskType = "triage"         // Quick triage decisions
)

// AgentContext provides information about the current agent state for task classification
type AgentContext struct {
	TurnCount        int    // Number of turns in current session
	LastToolOutput   string // Output from last tool execution
	PhaseChanged     bool   // Whether workflow phase just changed
	UserMessage      string // Current user message
	ToolsAvailable   int    // Number of tools in current context
	ReportRequested  bool   // Whether user requested a report
	SessionStarted   bool   // Whether this is the start of a session
}

// TierRouter handles task classification and routing to appropriate model tiers
type TierRouter struct {
	config    *config.RoutingConfig
	modelList []config.ModelConfig
	providers map[string]providers.LLMProvider
	costs     *CostTracker
	component string // Component name for logging
}

// NewTierRouter creates a new tier router
func NewTierRouter(
	routingCfg *config.RoutingConfig,
	modelList []config.ModelConfig,
	providerMap map[string]providers.LLMProvider,
) *TierRouter {
	return &TierRouter{
		config:    routingCfg,
		modelList: modelList,
		providers: providerMap,
		costs:     NewCostTracker(),
		component: "tier-router",
	}
}

// ClassifyTask determines the task type from the current agent context
// Uses rule-based classification (fast, deterministic, zero-cost)
func (tr *TierRouter) ClassifyTask(ctx AgentContext) TaskType {
	// Explicit report request
	if ctx.ReportRequested {
		return TaskReportWriting
	}

	// Start of session or phase change = planning
	if ctx.TurnCount == 0 || ctx.SessionStarted || ctx.PhaseChanged {
		return TaskPlanning
	}

	// Large tool output = parsing/summarizing
	if len(ctx.LastToolOutput) > 2000 {
		if len(ctx.LastToolOutput) > 10000 {
			return TaskSummary // Very large, needs summarization
		}
		return TaskParsing // Medium size, needs parsing
	}

	// Keywords in user message
	userLower := strings.ToLower(ctx.UserMessage)
	if strings.Contains(userLower, "analyze") || strings.Contains(userLower, "examine") {
		return TaskAnalysis
	}
	if strings.Contains(userLower, "test") || strings.Contains(userLower, "exploit") || strings.Contains(userLower, "vulnerability") {
		return TaskExploitation
	}
	if strings.Contains(userLower, "javascript") || strings.Contains(userLower, "js file") {
		return TaskJSAnalysis
	}
	if strings.Contains(userLower, "code") || strings.Contains(userLower, "review") {
		return TaskCodeReview
	}
	if strings.Contains(userLower, "which tool") || strings.Contains(userLower, "what command") {
		return TaskToolSelection
	}

	// Default: analysis for reasoning tasks
	return TaskAnalysis
}

// SelectTier returns the tier configuration for a given task type
func (tr *TierRouter) SelectTier(taskType TaskType) (string, *config.TierConfig, error) {
	if !tr.config.Enabled {
		// Routing disabled, use default tier
		if tr.config.DefaultTier != "" {
			if tier, ok := tr.config.Tiers[tr.config.DefaultTier]; ok {
				return tr.config.DefaultTier, &tier, nil
			}
		}
		return "", nil, fmt.Errorf("routing disabled and no valid default tier")
	}

	// Find tier that handles this task type
	for tierName, tierCfg := range tr.config.Tiers {
		for _, taskName := range tierCfg.UseFor {
			if strings.EqualFold(taskName, string(taskType)) {
				return tierName, &tierCfg, nil
			}
		}
	}

	// Fallback to default tier
	if tr.config.DefaultTier != "" {
		if tier, ok := tr.config.Tiers[tr.config.DefaultTier]; ok {
			logger.DebugCF(tr.component, "No tier found for task type, using default", map[string]any{
				"task": taskType,
				"tier": tr.config.DefaultTier,
			})
			return tr.config.DefaultTier, &tier, nil
		}
	}

	return "", nil, fmt.Errorf("no tier found for task type %s and no valid default tier", taskType)
}

// RouteChat executes an LLM chat request with tier-based routing
func (tr *TierRouter) RouteChat(
	ctx context.Context,
	taskType TaskType,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	options map[string]any,
	sessionKey string,
) (*providers.LLMResponse, error) {
	tierName, tierCfg, err := tr.SelectTier(taskType)
	if err != nil {
		return nil, fmt.Errorf("tier selection failed: %w", err)
	}

	provider, ok := tr.providers[tierCfg.ModelName]
	if !ok {
		return nil, fmt.Errorf("provider not found for model %s", tierCfg.ModelName)
	}

	logger.InfoCF(tr.component, "Routing to tier", map[string]any{
		"task":  taskType,
		"tier":  tierName,
		"model": tierCfg.ModelName,
	})

	start := time.Now()
	resp, err := provider.Chat(ctx, messages, tools, tierCfg.ModelName, options)
	elapsed := time.Since(start)

	if err != nil {
		logger.ErrorCF(tr.component, "Tier routing chat failed", map[string]any{
			"task":  taskType,
			"tier":  tierName,
			"model": tierCfg.ModelName,
			"error": err.Error(),
		})
		return nil, err
	}

	// Track cost
	tr.costs.Record(sessionKey, tierCfg.ModelName, tierName, *tierCfg, *resp.Usage, elapsed)

	logger.DebugCF(tr.component, "Tier routing chat complete", map[string]any{
		"task":          taskType,
		"tier":          tierName,
		"model":         tierCfg.ModelName,
		"input_tokens":  resp.Usage.PromptTokens,
		"output_tokens": resp.Usage.CompletionTokens,
		"latency":       elapsed.String(),
	})

	return resp, nil
}

// GetCostTracker returns the cost tracker for session-level cost reporting
func (tr *TierRouter) GetCostTracker() *CostTracker {
	return tr.costs
}

// IsEnabled returns whether tier routing is enabled
func (tr *TierRouter) IsEnabled() bool {
	return tr.config != nil && tr.config.Enabled
}
