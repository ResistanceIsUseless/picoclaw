package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ResistanceIsUseless/picoclaw/pkg/config"
	"github.com/ResistanceIsUseless/picoclaw/pkg/logger"
	"github.com/ResistanceIsUseless/picoclaw/pkg/providers"
)

// SupervisionRouter handles hierarchical oversight where powerful models supervise lighter models
type SupervisionRouter struct {
	tierRouter *TierRouter
	validator  *TaskValidator
	costTracker *CostTracker
	component  string
}

// TaskValidator validates and corrects outputs from lighter models
type TaskValidator struct {
	rules        []ValidationRule
	confidence   map[TaskType]float64
	component    string
}

// ValidationRule defines validation criteria for task outputs
type ValidationRule struct {
	TaskType     TaskType
	MinConfidence float64
	RequiresValidation bool
	ValidationTasks []TaskType // Tasks that can validate this output
}

// SupervisionResult represents the result of a supervised execution
type SupervisionResult struct {
	OriginalTask   TaskType
	SupervisorTask TaskType
	Validated      bool
	Corrections    []string
	FinalOutput    string
	SupervisorModel string
	WorkerModel    string
	ValidationScore float64
	SupervisorConfidence float64
}

// ValidationDecision represents the parsed validation decision from a supervisor
// ValidationDecision represents the parsed validation decision from a supervisor
type ValidationDecision struct {
	Approved     bool     `json:"approved"`
	Confidence   float64  `json:"confidence"`
	Corrections  []string `json:"corrections"`
	FinalOutput  string   `json:"final_output"`
}

// TaskType represents the classification of an LLM task for tier routing
type TaskType string

const (
	// Strategic tasks (require powerful models)
	TaskPlanning      TaskType = "planning"       // Strategic decisions, initial planning
	TaskAnalysis      TaskType = "analysis"       // Deep analysis, reasoning about findings
	TaskExploitation  TaskType = "exploitation"   // Testing vulnerabilities, exploit development
	TaskReportWriting TaskType = "report_writing" // Final reporting, documentation
	TaskSupervision   TaskType = "supervision"    // Oversight of lighter model execution
	
	// Intermediate tasks (moderate model power)
	TaskToolSelection TaskType = "tool_selection" // Choosing which tool to use
	TaskCodeReview    TaskType = "code_review"    // Analyzing JavaScript, code, configs
	TaskJSAnalysis    TaskType = "js_analysis"    // Specific JavaScript analysis
	TaskValidation    TaskType = "validation"     // Validating lighter model outputs
	
	// Lightweight tasks (can use local/lighter models)
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
	RequiresSupervision bool // Whether task needs oversight validation
	ConfidenceScore  float64 // Confidence level of current task classification
	TaskComplexity   int    // Estimated complexity (1-10)
	DependentTasks   []TaskType // Tasks that depend on this one
}

// TierRouter handles task classification and routing to appropriate model tiers
type TierRouter struct {
	config    *config.RoutingConfig
	modelList []config.ModelConfig
	providers map[string]providers.LLMProvider
	costs     *CostTracker
	component string // Component name for logging
	supervisor *SupervisionRouter // Hierarchical oversight routing
}

// NewTaskValidator creates a new task validator with default rules
func NewTaskValidator() *TaskValidator {
	validator := &TaskValidator{
		rules: []ValidationRule{
			{
				TaskType:            TaskAnalysis,
				MinConfidence:      0.8,
				RequiresValidation: true,
				ValidationTasks:    []TaskType{TaskSupervision},
			},
			{
				TaskType:            TaskExploitation,
				MinConfidence:      0.9,
				RequiresValidation: true,
				ValidationTasks:    []TaskType{TaskSupervision},
			},
			{
				TaskType:            TaskPlanning,
				MinConfidence:      0.7,
				RequiresValidation: false,
				ValidationTasks:    []TaskType{TaskValidation},
			},
			{
				TaskType:            TaskCodeReview,
				MinConfidence:      0.75,
				RequiresValidation: true,
				ValidationTasks:    []TaskType{TaskValidation},
			},
			{
				TaskType:            TaskToolSelection,
				MinConfidence:      0.6,
				RequiresValidation: false,
				ValidationTasks:    []TaskType{TaskValidation},
			},
		},
		confidence: map[TaskType]float64{
			TaskPlanning:      0.9,
			TaskAnalysis:      0.7,
			TaskExploitation:  0.6,
			TaskReportWriting: 0.8,
			TaskSupervision:  0.95,
			TaskValidation:   0.85,
			TaskToolSelection: 0.75,
			TaskCodeReview:    0.7,
			TaskJSAnalysis:    0.75,
			TaskParsing:       0.9,
			TaskSummary:       0.8,
			TaskFormatting:    0.95,
			TaskTriage:        0.85,
		},
		component: "task-validator",
	}
	return validator
}

// NewTierRouter creates a new tier router
func NewTierRouter(
	routingCfg *config.RoutingConfig,
	modelList []config.ModelConfig,
	providerMap map[string]providers.LLMProvider,
) *TierRouter {
	router := &TierRouter{
		config:    routingCfg,
		modelList: modelList,
		providers: providerMap,
		costs:     NewCostTracker(),
		component: "tier-router",
	}
	
	// Initialize supervision router if hierarchical routing is enabled
	if routingCfg != nil && routingCfg.Enabled && routingCfg.EnableSupervision {
		router.supervisor = &SupervisionRouter{
			tierRouter: router,
			validator:  NewTaskValidator(),
			costTracker: router.costs,
			component:  "supervision-router",
		}
		// Set validation confidence threshold if specified
		if routingCfg.ValidationConfidenceThreshold > 0 {
			for i := range router.supervisor.validator.rules {
				router.supervisor.validator.rules[i].MinConfidence = routingCfg.ValidationConfidenceThreshold
			}
		}
	}
	
	return router
}

// ClassifyTask determines the task type from the current agent context
// Uses rule-based classification (fast, deterministic, zero-cost)
func (tr *TierRouter) ClassifyTask(ctx AgentContext) TaskType {
	// Initialize default values
	if ctx.ConfidenceScore == 0 {
		ctx.ConfidenceScore = 0.5
	}
	if ctx.TaskComplexity == 0 {
		ctx.TaskComplexity = 5 // Medium complexity by default
	}
	
	// Explicit report request
	if ctx.ReportRequested {
		return TaskReportWriting
	}

	// Start of session or phase change = planning
	if ctx.TurnCount == 0 || ctx.SessionStarted || ctx.PhaseChanged {
		ctx.TaskComplexity = 8 // High complexity for planning
		return TaskPlanning
	}

	// Large tool output = parsing/summarizing
	if len(ctx.LastToolOutput) > 2000 {
		if len(ctx.LastToolOutput) > 10000 {
			ctx.TaskComplexity = 7 // High complexity for large summaries
			return TaskSummary
		}
		ctx.TaskComplexity = 4 // Medium complexity for parsing
		return TaskParsing
	}

	// Keywords in user message - enhanced with complexity scoring
	userLower := strings.ToLower(ctx.UserMessage)
	complexityModifiers := map[string]int{
		"deep":      2, "thorough": 2, "comprehensive": 3,
		"quick":    -1, "simple": -1, "basic": -2,
		"exploit":   3, "vulnerability": 3, "security": 2,
		"analyze":   1, "review": 1, "test": 1,
	}
	
	// Calculate complexity from keywords
	for keyword, modifier := range complexityModifiers {
		if strings.Contains(userLower, keyword) {
			ctx.TaskComplexity += modifier
			// Clamp complexity between 1-10
			if ctx.TaskComplexity < 1 {
				ctx.TaskComplexity = 1
			} else if ctx.TaskComplexity > 10 {
				ctx.TaskComplexity = 10
			}
		}
	}
	
	// Determine if supervision is needed
	ctx.RequiresSupervision = tr.requiresSupervision(ctx)

	if strings.Contains(userLower, "analyze") || strings.Contains(userLower, "examine") {
		ctx.ConfidenceScore = 0.7
		return TaskAnalysis
	}
	if strings.Contains(userLower, "test") || strings.Contains(userLower, "exploit") || strings.Contains(userLower, "vulnerability") {
		ctx.ConfidenceScore = 0.6
		ctx.RequiresSupervision = true
		return TaskExploitation
	}
	if strings.Contains(userLower, "javascript") || strings.Contains(userLower, "js file") {
		ctx.ConfidenceScore = 0.75
		return TaskJSAnalysis
	}
	if strings.Contains(userLower, "code") || strings.Contains(userLower, "review") {
		ctx.ConfidenceScore = 0.7
		return TaskCodeReview
	}
	if strings.Contains(userLower, "which tool") || strings.Contains(userLower, "what command") {
		ctx.ConfidenceScore = 0.8
		return TaskToolSelection
	}

	// Default: analysis for reasoning tasks
	ctx.ConfidenceScore = 0.6
	return TaskAnalysis
}

// requiresSupervision determines if a task needs supervision based on context
func (tr *TierRouter) requiresSupervision(ctx AgentContext) bool {
	// Check if supervision is enabled in config
	if tr.config == nil || !tr.config.EnableSupervision {
		return false
	}
	
	// Use configured minimum complexity if available
	minComplexity := 7 // Default
	if tr.config.MinTaskComplexityForSupervision > 0 {
		minComplexity = tr.config.MinTaskComplexityForSupervision
	}
	
	// High complexity tasks always need supervision
	if ctx.TaskComplexity >= minComplexity {
		return true
	}
	
	// Low confidence tasks need supervision
	if ctx.ConfidenceScore < 0.6 {
		return true
	}
	
	// Critical tasks that could have security implications
	userLower := strings.ToLower(ctx.UserMessage)
	criticalKeywords := []string{"exploit", "vulnerability", "attack", "hack", "breach"}
	for _, keyword := range criticalKeywords {
		if strings.Contains(userLower, keyword) {
			return true
		}
	}
	
	// Multi-turn tasks in critical phases
	if ctx.TurnCount > 5 && ctx.TaskComplexity > 6 {
		return true
	}
	
	return false
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

// RouteWithSupervision executes a task with hierarchical oversight
// Powerful models supervise and validate outputs from lighter models
func (tr *TierRouter) RouteWithSupervision(
	ctx context.Context,
	taskType TaskType,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	options map[string]any,
	sessionKey string,
	agentCtx AgentContext,
) (*SupervisionResult, error) {
	if tr.supervisor == nil {
		// Fallback to regular routing if supervision is disabled
		resp, err := tr.RouteChat(ctx, taskType, messages, tools, options, sessionKey)
		if err != nil {
			return nil, err
		}
		return &SupervisionResult{
			OriginalTask:   taskType,
			SupervisorTask: taskType,
			Validated:      true,
			FinalOutput:    resp.Content,
			SupervisorModel: "direct",
			WorkerModel:    "direct",
		}, nil
	}
	
	return tr.supervisor.ExecuteWithSupervision(ctx, taskType, messages, tools, options, sessionKey, agentCtx)
}

// ExecuteWithSupervision routes a task with hierarchical oversight
func (sr *SupervisionRouter) ExecuteWithSupervision(
	ctx context.Context,
	taskType TaskType,
	messages []providers.Message,
	tools []providers.ToolDefinition,
	options map[string]any,
	sessionKey string,
	agentCtx AgentContext,
) (*SupervisionResult, error) {
	
	// Check if this task requires supervision
	validationRule := sr.validator.getValidationRule(taskType)
	if validationRule == nil || !validationRule.RequiresValidation {
		// Execute directly without supervision
		resp, err := sr.tierRouter.RouteChat(ctx, taskType, messages, tools, options, sessionKey)
		if err != nil {
			return nil, err
		}
		return &SupervisionResult{
			OriginalTask:   taskType,
			SupervisorTask: taskType,
			Validated:      true,
			FinalOutput:    resp.Content,
			SupervisorModel: "none",
			WorkerModel:    sr.getModelForTask(taskType),
		}, nil
	}
	
	// First, execute with lighter model
	resp, err := sr.tierRouter.RouteChat(ctx, taskType, messages, tools, options, sessionKey)
	if err != nil {
		return nil, err
	}
	
	// Now validate with supervisor model
	supervisionResult, err := sr.validateOutput(ctx, taskType, resp, messages, tools, options, sessionKey)
	if err != nil {
		return nil, fmt.Errorf("supervision validation failed: %w", err)
	}
	
	return supervisionResult, nil
}

// validateOutput validates a lighter model's output using a powerful supervisor model
func (sr *SupervisionRouter) validateOutput(
	ctx context.Context,
	originalTask TaskType,
	workerResp *providers.LLMResponse,
	originalMessages []providers.Message,
	tools []providers.ToolDefinition,
	options map[string]any,
	sessionKey string,
) (*SupervisionResult, error) {
	
	// Create validation prompt
	validationPrompt := sr.createValidationPrompt(originalTask, workerResp.Content)
	
	// Add validation message to conversation
	validationMessages := append(originalMessages, providers.Message{
		Role:    "user",
		Content: validationPrompt,
	})
	
	// Try to validate with supervisor model, with retries
	var supervisorResp *providers.LLMResponse
	var err error
	
	maxRetries := 2
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Route to supervisor model
		supervisorResp, err = sr.tierRouter.RouteChat(ctx, TaskSupervision, validationMessages, tools, options, sessionKey)
		if err == nil {
			break // Success, exit retry loop
		}
		
		logger.WarnCF(sr.component, "Supervisor validation attempt failed", map[string]any{
			"attempt": attempt,
			"max_retries": maxRetries,
			"error": err.Error(),
			"task": originalTask,
		})
		
		if attempt == maxRetries {
			// All retries failed, use fallback strategy
			logger.ErrorCF(sr.component, "All supervisor validation attempts failed, using fallback", map[string]any{
				"task": originalTask,
				"final_error": err.Error(),
			})
			return sr.createFallbackResult(originalTask, workerResp, "supervisor_unavailable")
		}
		
		// Wait before retry (if this were async, we'd add a delay here)
		// For now, just continue immediately
	}
	
	// Parse supervisor's decision
	validationDecision, err := sr.parseValidationDecision(supervisorResp.Content)
	if err != nil {
		logger.WarnCF(sr.component, "Failed to parse validation decision, using fallback", map[string]any{
			"error": err.Error(),
			"task": originalTask,
		})
		return sr.createFallbackResult(originalTask, workerResp, "parse_error")
	}
	
	// Check if validation passed
	if validationDecision.Approved && validationDecision.Confidence >= 0.7 {
		// Validation successful
		return &SupervisionResult{
			OriginalTask:        originalTask,
			SupervisorTask:      TaskSupervision,
			Validated:           true,
			Corrections:         validationDecision.Corrections,
			FinalOutput:         validationDecision.FinalOutput,
			SupervisorModel:     sr.tierRouter.selectSupervisorModel(),
			WorkerModel:        sr.tierRouter.selectWorkerModel(originalTask),
			ValidationScore:     validationDecision.Confidence,
			SupervisorConfidence: validationDecision.Confidence,
		}, nil
	} else {
		// Validation failed or low confidence
		logger.WarnCF(sr.component, "Supervisor rejected output or low confidence", map[string]any{
			"approved": validationDecision.Approved,
			"confidence": validationDecision.Confidence,
			"task": originalTask,
		})
		
		// For high-stakes tasks, we might want to escalate rather than fallback
		if sr.isHighStakesTask(originalTask) {
			return nil, fmt.Errorf("high-stakes task %s failed validation with confidence %.2f", originalTask, validationDecision.Confidence)
		}
		
		// For other tasks, use the supervisor's corrected output if available
		if validationDecision.FinalOutput != "" && validationDecision.FinalOutput != workerResp.Content {
			logger.InfoCF(sr.component, "Using supervisor-corrected output", map[string]any{
				"task": originalTask,
				"has_corrections": len(validationDecision.Corrections) > 0,
			})
			return &SupervisionResult{
				OriginalTask:        originalTask,
				SupervisorTask:      TaskSupervision,
				Validated:           false, // Not fully validated, but corrected
				Corrections:         validationDecision.Corrections,
				FinalOutput:         validationDecision.FinalOutput,
				SupervisorModel:     sr.tierRouter.selectSupervisorModel(),
				WorkerModel:        sr.tierRouter.selectWorkerModel(originalTask),
				ValidationScore:     validationDecision.Confidence,
				SupervisorConfidence: validationDecision.Confidence,
			}, nil
		} else {
			// No corrected output available, use fallback
			return sr.createFallbackResult(originalTask, workerResp, "validation_rejected")
		}
	}
}

// createValidationPrompt creates a prompt for the supervisor to validate worker output
func (sr *SupervisionRouter) createValidationPrompt(taskType TaskType, workerOutput string) string {
	return fmt.Sprintf(`Please validate the following %s task output:

WORKER OUTPUT:
%s

Validation Requirements:
1. Check for accuracy, correctness, and completeness
2. Identify any potential issues, errors, or security concerns
3. If issues found, provide specific corrections
4. Approve if output is correct, or provide improved version

Respond in JSON format:
{
  "approved": true/false,
  "confidence": 0.0-1.0,
  "corrections": ["specific correction 1", "specific correction 2"],
  "final_output": "approved or corrected output"
}`, taskType, workerOutput)
}

// parseValidationDecision parses the supervisor's validation decision
func (sr *SupervisionRouter) parseValidationDecision(supervisorContent string) (*ValidationDecision, error) {
	// Try to parse JSON response from supervisor
	var decision ValidationDecision
	
	// First, try to extract JSON from the response
	jsonStart := strings.Index(supervisorContent, "{")
	jsonEnd := strings.LastIndex(supervisorContent, "}")
	
	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		// No valid JSON found, use fallback approval
		logger.WarnCF(sr.component, "No valid JSON found in supervisor response, using fallback", nil)
		return &ValidationDecision{
			Approved:    true,
			Confidence:  0.7, // Lower confidence for fallback
			Corrections: []string{},
			FinalOutput: supervisorContent,
		}, nil
	}
	
	jsonStr := supervisorContent[jsonStart : jsonEnd+1]
	err := json.Unmarshal([]byte(jsonStr), &decision)
	if err != nil {
		logger.WarnCF(sr.component, "Failed to parse supervisor JSON response, using fallback", map[string]any{
			"error": err.Error(),
			"json_preview": jsonStr[:min(200, len(jsonStr))],
		})
		// Use fallback approval
		return &ValidationDecision{
			Approved:    true,
			Confidence:  0.6, // Even lower confidence for parse failure
			Corrections: []string{"Failed to parse validation response"},
			FinalOutput: supervisorContent,
		}, nil
	}
	
	// Validate the parsed decision
	if decision.Confidence < 0 || decision.Confidence > 1 {
		decision.Confidence = 0.8 // Default confidence if out of range
	}
	if decision.FinalOutput == "" {
		decision.FinalOutput = supervisorContent
	}
	
	return &decision, nil
}

// min helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Helper methods for model selection
func (sr *SupervisionRouter) getModelForTask(taskType TaskType) string {
	return sr.tierRouter.selectWorkerModel(taskType)
}

func (tr *TierRouter) selectSupervisorModel() string {
	// Return most powerful model (typically GPT-4 or Claude 3 Opus)
	for _, model := range tr.modelList {
		if strings.Contains(strings.ToLower(model.ModelName), "gpt-4") || 
		   strings.Contains(strings.ToLower(model.ModelName), "claude-3-opus") ||
		   strings.Contains(strings.ToLower(model.ModelName), "claude-3.5-sonnet") {
			return model.ModelName
		}
	}
	return tr.modelList[0].ModelName // Fallback to first model
}

func (tr *TierRouter) selectWorkerModel(taskType TaskType) string {
	// Select appropriate model based on task type
	// For lighter tasks, prefer local or faster models
	lightTasks := map[TaskType]bool{
		TaskParsing:     true,
		TaskSummary:     true,
		TaskFormatting:  true,
		TaskTriage:      true,
	}
	
	if lightTasks[taskType] {
		// Prefer lighter models for these tasks
		for _, model := range tr.modelList {
			if strings.Contains(strings.ToLower(model.ModelName), "haiku") ||
			   strings.Contains(strings.ToLower(model.ModelName), "3.5-turbo") ||
			   strings.Contains(strings.ToLower(model.ModelName), "local") {
				return model.ModelName
			}
		}
	}
	
	// Default to first available model
	return tr.modelList[0].ModelName
}

// Validation helper methods
func (tv *TaskValidator) getValidationRule(taskType TaskType) *ValidationRule {
	for _, rule := range tv.rules {
		if rule.TaskType == taskType {
			return &rule
		}
	}
	return nil
}

// createFallbackResult creates a fallback supervision result when validation fails
func (sr *SupervisionRouter) createFallbackResult(originalTask TaskType, workerResp *providers.LLMResponse, reason string) (*SupervisionResult, error) {
	return &SupervisionResult{
		OriginalTask:        originalTask,
		SupervisorTask:      TaskSupervision,
		Validated:           false,
		FinalOutput:         workerResp.Content,
		SupervisorModel:     "fallback",
		WorkerModel:        sr.getModelForTask(originalTask),
		ValidationScore:     0.5,
		SupervisorConfidence: 0.5,
	}, nil
}

// isHighStakesTask determines if a task is high-stakes and should fail rather than fallback
func (sr *SupervisionRouter) isHighStakesTask(taskType TaskType) bool {
	// High-stakes tasks are those where errors could cause security issues or data loss
	highStakesTasks := map[TaskType]bool{
		TaskExploitation: true,
		TaskAnalysis:     true,
		TaskPlanning:     true,
	}
	
	return highStakesTasks[taskType]
}

// recordSupervisionMetrics records supervision metrics in the cost tracker
func (sr *SupervisionRouter) recordSupervisionMetrics(
	sessionKey string,
	validationSuccess bool,
	validationFailed bool,
	fallbackUsed bool,
	correctionsCount int,
	supervisionCost float64,
	confidenceScore float64,
	costSavings float64,
) {
	if sr.costTracker != nil {
		sr.costTracker.RecordSupervision(
			sessionKey,
			validationSuccess,
			validationFailed,
			fallbackUsed,
			correctionsCount,
			supervisionCost,
			confidenceScore,
			costSavings,
		)
	}
}
