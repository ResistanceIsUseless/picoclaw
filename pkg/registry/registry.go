package registry

import (
	"context"
	"fmt"
	"sync"

	"github.com/ResistanceIsUseless/picoclaw/pkg/logger"
)

// ToolDefinition defines a tool with its tier and metadata
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Tier        ToolTier               `json:"tier"`
	InputSchema map[string]interface{} `json:"input_schema"`
	OutputType  string                 `json:"output_type"` // artifact type it produces
	Parser      OutputParser           `json:"-"` // function to parse raw output
}

// OutputParser converts raw tool output into structured data
// Can produce either an artifact or a graph mutation
type OutputParser func(toolName string, output []byte) (interface{}, error)

// ToolRegistry manages tool definitions with tier-based access control
type ToolRegistry struct {
	mu    sync.RWMutex
	tools map[string]*ToolDefinition
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]*ToolDefinition),
	}
}

// Register adds a tool definition to the registry
func (r *ToolRegistry) Register(tool *ToolDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if tool.Name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	if _, exists := r.tools[tool.Name]; exists {
		logger.WarnCF("registry", "Tool already registered, overwriting",
			map[string]any{
				"tool": tool.Name,
			})
	}

	r.tools[tool.Name] = tool

	logger.DebugCF("registry", "Tool registered",
		map[string]any{
			"tool": tool.Name,
			"tier": tool.Tier.String(),
		})

	return nil
}

// Get retrieves a tool definition by name
func (r *ToolRegistry) Get(name string) (*ToolDefinition, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool %q not found in registry", name)
	}

	return tool, nil
}

// GetByTier returns all tools of a specific tier
func (r *ToolRegistry) GetByTier(tier ToolTier) []*ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*ToolDefinition
	for _, tool := range r.tools {
		if tool.Tier == tier {
			result = append(result, tool)
		}
	}

	return result
}

// GetVisibleTools returns tools that should be visible to the model
// This excludes TierHardwired (Tier 0) tools
func (r *ToolRegistry) GetVisibleTools() []*ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*ToolDefinition
	for _, tool := range r.tools {
		if tool.Tier.IsVisible() && tool.Tier != TierBanned {
			result = append(result, tool)
		}
	}

	return result
}

// GetRequestableTools returns tools that the model can request
// This includes TierAutoApprove (1), TierHuman (2), and TierOrchestrator (-1)
func (r *ToolRegistry) GetRequestableTools() []*ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*ToolDefinition
	for _, tool := range r.tools {
		if tool.Tier.CanModelRequest() {
			result = append(result, tool)
		}
	}

	return result
}

// ValidateExecution checks if a tool execution should be allowed
func (r *ToolRegistry) ValidateExecution(toolName string, args map[string]interface{}) (ValidationResult, error) {
	tool, err := r.Get(toolName)
	if err != nil {
		return ValidationResult{
			Allowed:      false,
			RejectReason: fmt.Sprintf("tool not found: %v", err),
		}, err
	}

	return ValidateToolExecution(tool.Tier, toolName, args), nil
}

// List returns all registered tool names
func (r *ToolRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}

	return names
}

// Count returns the total number of registered tools
func (r *ToolRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

// Summary returns a human-readable summary of the registry
func (r *ToolRegistry) Summary() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tierCounts := make(map[ToolTier]int)
	for _, tool := range r.tools {
		tierCounts[tool.Tier]++
	}

	summary := fmt.Sprintf("Tool Registry: %d tools\n", len(r.tools))
	for tier := TierOrchestrator; tier <= TierBanned; tier++ {
		if count, exists := tierCounts[tier]; exists {
			summary += fmt.Sprintf("  %s: %d tools\n", tier.String(), count)
		}
	}

	return summary
}

// RegisterBatch registers multiple tools at once
func (r *ToolRegistry) RegisterBatch(tools []*ToolDefinition) error {
	for _, tool := range tools {
		if err := r.Register(tool); err != nil {
			return fmt.Errorf("failed to register tool %q: %w", tool.Name, err)
		}
	}
	return nil
}

// ToolExecutor handles the actual execution of tools with tier enforcement
type ToolExecutor struct {
	registry         *ToolRegistry
	approvalHandler  ApprovalHandler
	executionHandler ExecutionHandler
}

// ApprovalHandler requests human approval for Tier 2 tools
type ApprovalHandler func(ctx context.Context, prompt string) (approved bool, err error)

// ExecutionHandler executes the actual tool (calls MCP, runs command, etc.)
type ExecutionHandler func(ctx context.Context, tool *ToolDefinition, args map[string]interface{}) ([]byte, error)

// NewToolExecutor creates a new tool executor
func NewToolExecutor(registry *ToolRegistry, approvalHandler ApprovalHandler, executionHandler ExecutionHandler) *ToolExecutor {
	return &ToolExecutor{
		registry:         registry,
		approvalHandler:  approvalHandler,
		executionHandler: executionHandler,
	}
}

// Execute runs a tool with full tier enforcement
func (te *ToolExecutor) Execute(ctx context.Context, toolName string, args map[string]interface{}) ([]byte, error) {
	// Get tool definition
	tool, err := te.registry.Get(toolName)
	if err != nil {
		return nil, err
	}

	// Validate execution is allowed
	validation := ValidateToolExecution(tool.Tier, toolName, args)

	if !validation.Allowed {
		logger.WarnCF("registry", "Tool execution rejected",
			map[string]any{
				"tool":   toolName,
				"tier":   tool.Tier.String(),
				"reason": validation.RejectReason,
			})
		return nil, fmt.Errorf("tool execution rejected: %s", validation.RejectReason)
	}

	// If human approval required, request it
	if validation.RequiresHuman {
		if te.approvalHandler == nil {
			return nil, fmt.Errorf("tool requires human approval but no approval handler configured")
		}

		approved, err := te.approvalHandler(ctx, validation.ApprovalPrompt)
		if err != nil {
			return nil, fmt.Errorf("approval request failed: %w", err)
		}

		if !approved {
			logger.InfoCF("registry", "Tool execution denied by operator",
				map[string]any{
					"tool": toolName,
					"tier": tool.Tier.String(),
				})
			return nil, fmt.Errorf("tool execution denied by operator")
		}

		logger.InfoCF("registry", "Tool execution approved by operator",
			map[string]any{
				"tool": toolName,
				"tier": tool.Tier.String(),
			})
	}

	// Execute tool
	logger.InfoCF("registry", "Executing tool",
		map[string]any{
			"tool": toolName,
			"tier": tool.Tier.String(),
		})

	output, err := te.executionHandler(ctx, tool, args)
	if err != nil {
		logger.ErrorCF("registry", "Tool execution failed",
			map[string]any{
				"tool":  toolName,
				"error": err.Error(),
			})
		return nil, fmt.Errorf("tool execution failed: %w", err)
	}

	logger.InfoCF("registry", "Tool execution completed",
		map[string]any{
			"tool":        toolName,
			"output_size": len(output),
		})

	return output, nil
}

// ParseOutput applies the tool's registered parser to raw output
func (te *ToolExecutor) ParseOutput(toolName string, output []byte) (interface{}, error) {
	tool, err := te.registry.Get(toolName)
	if err != nil {
		return nil, err
	}

	if tool.Parser == nil {
		// No parser registered, return raw output
		return output, nil
	}

	return tool.Parser(toolName, output)
}
