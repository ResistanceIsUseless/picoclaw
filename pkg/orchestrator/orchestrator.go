package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ResistanceIsUseless/picoclaw/pkg/blackboard"
	"github.com/ResistanceIsUseless/picoclaw/pkg/graph"
	"github.com/ResistanceIsUseless/picoclaw/pkg/logger"
	"github.com/ResistanceIsUseless/picoclaw/pkg/parsers"
	"github.com/ResistanceIsUseless/picoclaw/pkg/phase"
	"github.com/ResistanceIsUseless/picoclaw/pkg/providers"
	"github.com/ResistanceIsUseless/picoclaw/pkg/registry"
)

// EventEmitter interface for sending events to web UI (defined in pkg/webui)
// Using interface to avoid circular dependency
type EventEmitter interface {
	EmitPhaseStart(phaseName string, objective string, iteration int)
	EmitPhaseComplete(phaseName string, status string, iteration int, duration string)
	EmitToolExecution(tool string, status string, summary string)
	EmitArtifact(artifactType string, phase string, count int)
	EmitGraphUpdate(mutation *graph.GraphMutation)
	EmitLog(level string, component string, message string, fields map[string]any)
}

// Orchestrator manages the lifecycle of phases in the CLAW pipeline
type Orchestrator struct {
	pipeline       *Pipeline
	blackboard     *blackboard.Blackboard
	graph          *graph.Graph
	registry       *registry.ToolRegistry
	entityRegistry *graph.EntityRegistry
	provider       providers.LLMProvider // Optional: for model calls during phase execution
	llmParser      *parsers.LLMParser    // Layer 2 compression: LLM-based parser for tools without structural parsers
	eventEmitter   EventEmitter          // Optional: for web UI event streaming

	mu              sync.RWMutex
	currentPhase    *PhaseExecution
	completedPhases []string
	phaseHistory    []*PhaseExecution
}

// PhaseExecution tracks the execution state of a single phase
type PhaseExecution struct {
	PhaseName      string
	StartTime      time.Time
	EndTime        time.Time
	Status         PhaseStatus
	State          *phase.DAGState
	Contract       *phase.PhaseContract
	ContextBuilder *phase.PhaseContextBuilder
	Iteration      int
	Error          error
	Artifacts      []blackboard.ArtifactEnvelope
}

// PhaseStatus represents the current status of a phase
type PhaseStatus string

const (
	PhaseNotStarted PhaseStatus = "NOT_STARTED"
	PhaseRunning    PhaseStatus = "RUNNING"
	PhaseCompleted  PhaseStatus = "COMPLETED"
	PhaseFailed     PhaseStatus = "FAILED"
	PhaseBlocked    PhaseStatus = "BLOCKED"
	PhaseEscalated  PhaseStatus = "ESCALATED"
)

// NewOrchestrator creates a new orchestrator
func NewOrchestrator(pipeline *Pipeline, bb *blackboard.Blackboard, toolRegistry *registry.ToolRegistry) *Orchestrator {
	return &Orchestrator{
		pipeline:        pipeline,
		blackboard:      bb,
		graph:           graph.NewGraph(),
		registry:        toolRegistry,
		entityRegistry:  graph.NewEntityRegistry(),
		provider:        nil, // Set via SetProvider if model calls needed
		completedPhases: make([]string, 0),
		phaseHistory:    make([]*PhaseExecution, 0),
	}
}

// SetProvider sets the LLM provider for model calls during phase execution
func (o *Orchestrator) SetProvider(provider providers.LLMProvider) {
	o.provider = provider
	// Initialize LLM parser with the provider
	if provider != nil {
		o.llmParser = parsers.NewLLMParser(provider, "")
	}
}

// SetEventEmitter sets the event emitter for web UI streaming
func (o *Orchestrator) SetEventEmitter(emitter EventEmitter) {
	o.eventEmitter = emitter
}

// Execute runs the entire pipeline from start to finish
func (o *Orchestrator) Execute(ctx context.Context) error {
	logger.InfoCF("orchestrator", "Starting pipeline execution",
		map[string]any{
			"pipeline": o.pipeline.Name,
			"phases":   len(o.pipeline.Phases),
		})

	// Validate pipeline before starting
	if err := o.pipeline.Validate(); err != nil {
		return fmt.Errorf("pipeline validation failed: %w", err)
	}

	// Execute phases in order
	for _, phaseDef := range o.pipeline.Phases {
		if err := o.executePhase(ctx, phaseDef); err != nil {
			logger.ErrorCF("orchestrator", "Phase execution failed",
				map[string]any{
					"phase": phaseDef.Name,
					"error": err.Error(),
				})

			return fmt.Errorf("phase %q failed: %w", phaseDef.Name, err)
		}

		// Check if context was cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	logger.InfoCF("orchestrator", "Pipeline execution completed",
		map[string]any{
			"pipeline":        o.pipeline.Name,
			"phases_executed": len(o.completedPhases),
		})

	return nil
}

// executePhase runs a single phase to completion
func (o *Orchestrator) executePhase(ctx context.Context, phaseDef *PhaseDefinition) error {
	logger.InfoCF("orchestrator", "Starting phase",
		map[string]any{
			"phase":     phaseDef.Name,
			"objective": phaseDef.Objective,
			"max_iters": phaseDef.MaxIterations,
		})

	// Check dependencies
	if err := o.checkDependencies(phaseDef); err != nil {
		return fmt.Errorf("dependency check failed: %w", err)
	}

	// Initialize phase execution
	phaseExec := &PhaseExecution{
		PhaseName: phaseDef.Name,
		StartTime: time.Now(),
		Status:    PhaseRunning,
		Iteration: 0,
	}

	// Create contract
	contract := o.createContract(phaseDef)
	phaseExec.Contract = contract

	// Create DAG state
	state := phase.NewDAGState(phaseDef.Name, phaseDef.ResolvedTools(), phaseDef.ResolvedDependencies())
	phaseExec.State = state

	// Create context builder
	contextBuilder := phase.NewPhaseContextBuilder(
		phaseDef.Name,
		phaseDef.Objective,
		phaseDef.TokenBudget,
	)
	phaseExec.ContextBuilder = contextBuilder

	// Store current phase
	o.mu.Lock()
	o.currentPhase = phaseExec
	o.mu.Unlock()

	// Emit phase start event
	if o.eventEmitter != nil {
		o.eventEmitter.EmitPhaseStart(phaseDef.Name, phaseDef.Objective, 0)
	}

	// Execute phase iterations
	for phaseExec.Iteration < phaseDef.MaxIterations {
		phaseExec.Iteration++

		logger.DebugCF("orchestrator", "Phase iteration",
			map[string]any{
				"phase":     phaseDef.Name,
				"iteration": phaseExec.Iteration,
			})

		// Emit iteration start
		if o.eventEmitter != nil {
			o.eventEmitter.EmitPhaseStart(phaseDef.Name, phaseDef.Objective, phaseExec.Iteration)
		}

		// Build context for this iteration
		frontier := o.graph.ComputeFrontier(o.entityRegistry)
		contextInput := &phase.PhaseContextInput{
			PhaseName:      phaseDef.Name,
			Objective:      phaseDef.Objective,
			Contract:       contract,
			State:          state,
			Blackboard:     o.blackboard,
			Graph:          o.graph,
			Frontier:       frontier,
			Registry:       o.registry,
			Iteration:      phaseExec.Iteration,
			PreviousPhases: o.completedPhases,
		}

		// Execute model call and tools (if provider configured)
		if o.provider != nil {
			if err := o.executeIteration(ctx, phaseDef, phaseExec, contextInput); err != nil {
				logger.ErrorCF("orchestrator", "Iteration execution failed",
					map[string]any{
						"phase":     phaseDef.Name,
						"iteration": phaseExec.Iteration,
						"error":     err.Error(),
					})
				// Continue to next iteration instead of failing entire phase
				// This allows recovery from transient errors
			}
		} else {
			// No provider configured - this is expected for tests
			logger.DebugCF("orchestrator", "No provider configured, skipping model call",
				map[string]any{
					"phase":     phaseDef.Name,
					"iteration": phaseExec.Iteration,
				})
		}

		// Check if phase contract is satisfied
		phaseCtx := &phase.PhaseContext{
			Phase:      phaseDef.Name,
			State:      state,
			Blackboard: o.blackboard,
			Artifacts:  o.getCurrentPhaseArtifacts(phaseDef.Name),
			Iteration:  phaseExec.Iteration,
		}

		if contract.CanComplete(phaseCtx) {
			logger.InfoCF("orchestrator", "Phase contract satisfied",
				map[string]any{
					"phase":     phaseDef.Name,
					"iteration": phaseExec.Iteration,
				})
			break
		}

		// Check for minimum iterations before allowing completion
		if phaseExec.Iteration >= contract.MinIterations {
			// Check if we've made progress
			progress := state.GetProgress()
			if progress >= 100.0 {
				logger.InfoCF("orchestrator", "Phase progress complete",
					map[string]any{
						"phase":    phaseDef.Name,
						"progress": progress,
					})
				break
			}
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			phaseExec.Status = PhaseFailed
			phaseExec.Error = ctx.Err()
			return ctx.Err()
		default:
		}
	}

	// Validate final contract
	phaseCtx := &phase.PhaseContext{
		Phase:      phaseDef.Name,
		State:      state,
		Blackboard: o.blackboard,
		Artifacts:  o.getCurrentPhaseArtifacts(phaseDef.Name),
		Iteration:  phaseExec.Iteration,
	}

	if err := contract.Validate(phaseCtx); err != nil {
		phaseExec.Status = PhaseFailed
		phaseExec.Error = err
		return fmt.Errorf("phase contract validation failed: %w", err)
	}

	// Mark phase as completed
	phaseExec.EndTime = time.Now()
	phaseExec.Status = PhaseCompleted
	phaseExec.Artifacts = o.getCurrentPhaseArtifacts(phaseDef.Name)

	o.mu.Lock()
	o.completedPhases = append(o.completedPhases, phaseDef.Name)
	o.phaseHistory = append(o.phaseHistory, phaseExec)
	o.currentPhase = nil
	o.mu.Unlock()

	duration := phaseExec.EndTime.Sub(phaseExec.StartTime).String()
	logger.InfoCF("orchestrator", "Phase completed",
		map[string]any{
			"phase":      phaseDef.Name,
			"iterations": phaseExec.Iteration,
			"duration":   duration,
			"artifacts":  len(phaseExec.Artifacts),
		})

	// Emit phase complete event
	if o.eventEmitter != nil {
		o.eventEmitter.EmitPhaseComplete(
			phaseDef.Name,
			string(phaseExec.Status),
			phaseExec.Iteration,
			duration,
		)
	}

	return nil
}

// checkDependencies verifies all phase dependencies are satisfied
func (o *Orchestrator) checkDependencies(phaseDef *PhaseDefinition) error {
	for _, dep := range phaseDef.DependsOn {
		found := false
		for _, completed := range o.completedPhases {
			if completed == dep {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("dependency %q not satisfied", dep)
		}
	}
	return nil
}

// createContract creates a phase contract from definition
func (o *Orchestrator) createContract(phaseDef *PhaseDefinition) *phase.PhaseContract {
	contract := phase.NewPhaseContract(phaseDef.Name).
		SetIterationLimits(phaseDef.MinIterations, phaseDef.MaxIterations)

	// Add required tools
	for _, tool := range phaseDef.RequiredTools {
		contract.AddRequiredTool(tool)
	}
	for _, profileName := range phaseDef.RequiredProfiles {
		contract.AddRequiredProfile(profileName)
	}
	for _, tool := range phaseDef.ResolvedOptionalTools() {
		contract.AddOptionalTool(tool)
	}

	// Add required artifacts
	for _, artifactType := range phaseDef.RequiredArtifacts {
		contract.AddRequiredArtifact(artifactType)
	}

	// Try to get predefined contract and merge with definition
	if predefined, err := phase.GetPredefinedContract(phaseDef.Name); err == nil {
		// Merge validation rules from predefined contract
		for _, rule := range predefined.SuccessCriteria {
			contract.AddValidationRule(rule)
		}
	}

	return contract
}

// executeIteration executes a single iteration of model call and tool execution
func (o *Orchestrator) executeIteration(ctx context.Context, phaseDef *PhaseDefinition, phaseExec *PhaseExecution, contextInput *phase.PhaseContextInput) error {
	// Build the context sections using PhaseContextBuilder
	sections, err := phaseExec.ContextBuilder.Build(contextInput)
	if err != nil {
		return fmt.Errorf("failed to build context: %w", err)
	}

	// Convert sections to a single prompt
	var promptParts []string
	for _, section := range sections {
		promptParts = append(promptParts, section.Content)
	}
	prompt := ""
	for _, part := range promptParts {
		prompt += part + "\n\n"
	}

	// Prepare messages for model
	messages := []providers.Message{
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// Get tool definitions from registry
	var toolDefs []providers.ToolDefinition
	resolvedTools := phaseDef.ResolvedTools()
	for _, toolName := range resolvedTools {
		toolDef, err := o.registry.Get(toolName)
		if err != nil {
			logger.WarnCF("orchestrator", "Tool not found in registry",
				map[string]any{
					"phase": phaseDef.Name,
					"tool":  toolName,
				})
			continue
		}

		// Skip Tier 0 (Hardwired) tools - they're invisible to the model
		if toolDef.Tier == registry.TierHardwired {
			continue
		}

		// Convert registry ToolDefinition to providers.ToolDefinition
		providerToolDef := providers.ToolDefinition{
			Type: "function",
			Function: providers.ToolFunctionDefinition{
				Name:        toolDef.Name,
				Description: toolDef.Description,
				Parameters:  toolDef.InputSchema,
			},
		}
		toolDefs = append(toolDefs, providerToolDef)
	}

	logger.DebugCF("orchestrator", "Tool definitions prepared",
		map[string]any{
			"phase":         phaseDef.Name,
			"total_tools":   len(resolvedTools),
			"visible_tools": len(toolDefs),
		})

	// Call the model
	model := o.provider.GetDefaultModel()
	options := map[string]any{
		"temperature": 0.7,
		"max_tokens":  4096,
	}

	response, err := o.provider.Chat(ctx, messages, toolDefs, model, options)
	if err != nil {
		return fmt.Errorf("model call failed: %w", err)
	}

	// Log model response
	logger.DebugCF("orchestrator", "Model response received",
		map[string]any{
			"phase":        phaseDef.Name,
			"iteration":    phaseExec.Iteration,
			"response_len": len(response.Content),
			"tool_calls":   len(response.ToolCalls),
		})

	// Execute requested tools
	for _, toolCall := range response.ToolCalls {
		if err := o.executeTool(ctx, phaseDef, phaseExec, toolCall); err != nil {
			logger.WarnCF("orchestrator", "Tool execution failed",
				map[string]any{
					"phase": phaseDef.Name,
					"tool":  toolCall.Name,
					"error": err.Error(),
				})
			// Continue with other tools instead of failing iteration
		}
	}

	return nil
}

// executeTool executes a single tool call and updates state
func (o *Orchestrator) executeTool(ctx context.Context, phaseDef *PhaseDefinition, phaseExec *PhaseExecution, toolCall providers.ToolCall) error {
	// Check if tool is available in phase
	toolAvailable := false
	for _, availableTool := range phaseDef.ResolvedTools() {
		if availableTool == toolCall.Name {
			toolAvailable = true
			break
		}
	}

	if !toolAvailable {
		return fmt.Errorf("tool %q not available in phase %q", toolCall.Name, phaseDef.Name)
	}

	// Create a tool call record in DAGState
	stateToolCall := &phase.ToolCall{
		ID:        fmt.Sprintf("%s-%d", toolCall.Name, phaseExec.Iteration),
		ToolName:  toolCall.Name,
		Status:    phase.StatusRunning,
		StartTime: time.Now(),
	}
	phaseExec.State.AddToolCall(stateToolCall)

	logger.InfoCF("orchestrator", "Executing tool",
		map[string]any{
			"phase":     phaseDef.Name,
			"tool":      toolCall.Name,
			"iteration": phaseExec.Iteration,
			"call_id":   stateToolCall.ID,
		})

	// Emit tool execution start event
	if o.eventEmitter != nil {
		o.eventEmitter.EmitToolExecution(toolCall.Name, "running", "")
	}

	// 1. Get tool definition from registry
	toolDef, err := o.registry.Get(toolCall.Name)
	if err != nil {
		phaseExec.State.UpdateToolCall(stateToolCall.ID, phase.StatusFailed, "", fmt.Errorf("tool not found: %w", err))
		return fmt.Errorf("failed to get tool %q: %w", toolCall.Name, err)
	}

	// 2. Get tool arguments (already parsed by provider)
	args := toolCall.Arguments
	if args == nil {
		args = make(map[string]interface{})
	}

	// 3. Execute tool
	rawOutput, err := registry.ExecuteTool(ctx, toolCall.Name, args)
	if err != nil {
		logger.ErrorCF("orchestrator", "Tool execution failed",
			map[string]any{
				"phase": phaseDef.Name,
				"tool":  toolCall.Name,
				"error": err.Error(),
			})
		phaseExec.State.UpdateToolCall(stateToolCall.ID, phase.StatusFailed, "", err)
		return fmt.Errorf("tool execution failed: %w", err)
	}

	logger.DebugCF("orchestrator", "Tool execution completed",
		map[string]any{
			"phase":       phaseDef.Name,
			"tool":        toolCall.Name,
			"output_size": len(rawOutput),
		})

	// 4. Parse output to artifacts
	var artifact interface{}
	var artifactSummary string
	var parseErr error

	if toolDef.Parser != nil {
		// Layer 1: Use structural parser if available
		artifact, parseErr = toolDef.Parser(toolCall.Name, rawOutput)
		if parseErr != nil {
			logger.WarnCF("orchestrator", "Structural parser failed",
				map[string]any{
					"phase": phaseDef.Name,
					"tool":  toolCall.Name,
					"error": parseErr.Error(),
				})
		}
	}

	// Layer 2: Fall back to LLM parser if no structural parser or if it failed
	if artifact == nil && o.llmParser != nil && len(rawOutput) > 0 {
		logger.InfoCF("orchestrator", "Using LLM parser (Layer 2 compression)",
			map[string]any{
				"phase": phaseDef.Name,
				"tool":  toolCall.Name,
			})

		// Determine expected artifact type (use ToolOutput as default)
		expectedType := toolDef.OutputType
		if expectedType == "" {
			expectedType = "ToolOutput"
		}

		artifact, parseErr = o.llmParser.ParseOutput(ctx, toolCall.Name, toolDef.Description, rawOutput, expectedType, phaseDef.Name)
		if parseErr != nil {
			logger.WarnCF("orchestrator", "LLM parser failed",
				map[string]any{
					"phase": phaseDef.Name,
					"tool":  toolCall.Name,
					"error": parseErr.Error(),
				})
			artifactSummary = fmt.Sprintf("Raw output: %d bytes (parsing failed)", len(rawOutput))
		}
	}

	// 5. Publish artifact to blackboard (if we got one)
	if artifact != nil {
		if artifactEnvelope, ok := artifact.(blackboard.Artifact); ok {
			if err := o.blackboard.Publish(ctx, artifactEnvelope); err != nil {
				logger.WarnCF("orchestrator", "Failed to publish artifact",
					map[string]any{
						"phase": phaseDef.Name,
						"tool":  toolCall.Name,
						"error": err.Error(),
					})
			} else {
				logger.InfoCF("orchestrator", "Artifact published",
					map[string]any{
						"phase": phaseDef.Name,
						"tool":  toolCall.Name,
						"type":  toolDef.OutputType,
					})

				// Emit artifact event
				if o.eventEmitter != nil {
					o.eventEmitter.EmitArtifact(toolDef.OutputType, phaseDef.Name, 1)
				}
			}

			artifactSummary = fmt.Sprintf("Artifact: %s", toolDef.OutputType)

			// 6. Update knowledge graph with tool results
			mutation, err := graph.ExtractMutation(artifactEnvelope)
			if err != nil {
				logger.WarnCF("orchestrator", "Failed to extract graph mutation",
					map[string]any{
						"phase": phaseDef.Name,
						"tool":  toolCall.Name,
						"error": err.Error(),
					})
			} else if mutation != nil && (len(mutation.Nodes) > 0 || len(mutation.Edges) > 0) {
				// Apply mutation to graph
				o.graph.ApplyMutation(mutation)

				logger.InfoCF("orchestrator", "Graph updated",
					map[string]any{
						"phase": phaseDef.Name,
						"tool":  toolCall.Name,
						"nodes": len(mutation.Nodes),
						"edges": len(mutation.Edges),
					})

				// Emit graph update event
				if o.eventEmitter != nil {
					o.eventEmitter.EmitGraphUpdate(mutation)
				}
			}
		}
	} else if artifactSummary == "" {
		// No artifact created and no summary set yet
		artifactSummary = fmt.Sprintf("Raw output: %d bytes (no parser available)", len(rawOutput))
	}

	// 7. Update DAGState status
	phaseExec.State.UpdateToolCall(stateToolCall.ID, phase.StatusCompleted, artifactSummary, nil)

	logger.InfoCF("orchestrator", "Tool execution complete",
		map[string]any{
			"phase":   phaseDef.Name,
			"tool":    toolCall.Name,
			"summary": artifactSummary,
		})

	// Emit tool execution complete event
	if o.eventEmitter != nil {
		o.eventEmitter.EmitToolExecution(toolCall.Name, "completed", artifactSummary)
	}

	return nil
}

// getCurrentPhaseArtifacts gets artifacts for current phase
func (o *Orchestrator) getCurrentPhaseArtifacts(phaseName string) []blackboard.ArtifactEnvelope {
	artifacts, err := o.blackboard.GetByPhase(phaseName)
	if err != nil {
		return []blackboard.ArtifactEnvelope{}
	}
	return artifacts
}

// GetCurrentPhase returns the currently executing phase
func (o *Orchestrator) GetCurrentPhase() *PhaseExecution {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.currentPhase
}

// GetCompletedPhases returns list of completed phase names
func (o *Orchestrator) GetCompletedPhases() []string {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return append([]string{}, o.completedPhases...)
}

// GetPhaseHistory returns execution history for all phases
func (o *Orchestrator) GetPhaseHistory() []*PhaseExecution {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return append([]*PhaseExecution{}, o.phaseHistory...)
}

// GetGraph returns the knowledge graph
func (o *Orchestrator) GetGraph() *graph.Graph {
	return o.graph
}

// GetBlackboard returns the blackboard
func (o *Orchestrator) GetBlackboard() *blackboard.Blackboard {
	return o.blackboard
}

// GetPipeline returns the active pipeline definition.
func (o *Orchestrator) GetPipeline() *Pipeline {
	return o.pipeline
}

// Escalate handles escalation requests from phases
func (o *Orchestrator) Escalate(phaseName string, reason string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.currentPhase != nil && o.currentPhase.PhaseName == phaseName {
		o.currentPhase.Status = PhaseEscalated
		o.currentPhase.Error = fmt.Errorf("escalated: %s", reason)

		logger.WarnCF("orchestrator", "Phase escalated",
			map[string]any{
				"phase":  phaseName,
				"reason": reason,
			})

		return nil
	}

	return fmt.Errorf("phase %q is not currently running", phaseName)
}

// Summary returns a summary of the orchestration state
func (o *Orchestrator) Summary() string {
	o.mu.RLock()
	defer o.mu.RUnlock()

	var result string
	result += fmt.Sprintf("Pipeline: %s\n", o.pipeline.Name)
	result += fmt.Sprintf("Completed Phases: %d/%d\n", len(o.completedPhases), len(o.pipeline.Phases))

	if o.currentPhase != nil {
		result += fmt.Sprintf("Current Phase: %s (iteration %d)\n",
			o.currentPhase.PhaseName,
			o.currentPhase.Iteration)
	}

	result += "\nPhase History:\n"
	for _, phaseExec := range o.phaseHistory {
		duration := phaseExec.EndTime.Sub(phaseExec.StartTime)
		result += fmt.Sprintf("  - %s: %s (%d iterations, %s)\n",
			phaseExec.PhaseName,
			phaseExec.Status,
			phaseExec.Iteration,
			duration.Round(time.Second))
	}

	return result
}
