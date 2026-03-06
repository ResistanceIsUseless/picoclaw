// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ResistanceIsUseless/picoclaw/pkg/blackboard"
	"github.com/ResistanceIsUseless/picoclaw/pkg/logger"
	"github.com/ResistanceIsUseless/picoclaw/pkg/providers"
	"github.com/ResistanceIsUseless/picoclaw/pkg/registry"
	"github.com/ResistanceIsUseless/picoclaw/pkg/tools"
)

// CommanderOrchestrator implements hierarchical multi-agent coordination
// It uses a Commander agent to dynamically route work to specialist agents
// based on blackboard state, replacing rigid pipeline execution.
type CommanderOrchestrator struct {
	provider         providers.LLMProvider
	blackboard       *blackboard.Blackboard
	toolRegistry     *tools.ToolRegistry // Execution registry with actual tools
	metadataRegistry *registry.ToolRegistry // Metadata registry for CLAW phase contracts (optional)
	promptsDir       string
	maxCycles        int // Maximum Commander → Specialist cycles to prevent infinite loops
}

// NewCommanderOrchestrator creates a new hierarchical orchestrator
func NewCommanderOrchestrator(
	provider providers.LLMProvider,
	bb *blackboard.Blackboard,
	toolRegistry *tools.ToolRegistry,
	maxCycles int,
) *CommanderOrchestrator {
	if maxCycles == 0 {
		maxCycles = 10 // Default to 10 cycles
	}

	return &CommanderOrchestrator{
		provider:         provider,
		blackboard:       bb,
		toolRegistry:     toolRegistry,
		metadataRegistry: nil, // Not used in Commander mode
		promptsDir:       "pkg/prompts",
		maxCycles:        maxCycles,
	}
}

// RoutingDecision represents the Commander's routing decision
type RoutingDecision struct {
	Action   string // "ROUTE" or "COMPLETE"
	Agent    string // Which specialist agent to route to
	Reason   string // Why this decision was made
	Focus    string // Specific task for the agent (if routing)
	Summary  string // Assessment summary (if complete)
}

// Execute runs the Commander-based orchestration loop
func (co *CommanderOrchestrator) Execute(ctx context.Context, userObjective string) (string, error) {
	logger.InfoCF("commander", "Starting hierarchical assessment", map[string]any{
		"objective":  userObjective,
		"max_cycles": co.maxCycles,
	})

	// Initialize with user objective
	co.blackboard.RecordUserObjective(userObjective)

	cycle := 0
	for cycle < co.maxCycles {
		cycle++

		logger.InfoCF("commander", "Commander cycle", map[string]any{
			"cycle": cycle,
		})

		// Get Commander's routing decision
		decision, err := co.getCommanderDecision(ctx, userObjective)
		if err != nil {
			return "", fmt.Errorf("commander decision failed: %w", err)
		}

		logger.InfoCF("commander", "Commander decision", map[string]any{
			"action": decision.Action,
			"agent":  decision.Agent,
			"reason": decision.Reason,
		})

		// Check if assessment is complete
		if decision.Action == "COMPLETE" {
			logger.InfoCF("commander", "Assessment complete", map[string]any{
				"cycles": cycle,
			})
			return co.formatFinalReport(decision), nil
		}

		// Route to specialist agent
		if decision.Action == "ROUTE" {
			err := co.executeSpecialist(ctx, decision)
			if err != nil {
				logger.ErrorCF("commander", "Specialist execution failed", map[string]any{
					"agent": decision.Agent,
					"error": err.Error(),
				})
				// Don't fail completely - let Commander decide next step
				co.blackboard.RecordError(decision.Agent, err.Error())
			}
		}
	}

	// Max cycles reached
	logger.WarnCF("commander", "Max cycles reached", map[string]any{
		"cycles": cycle,
	})

	return co.formatIncompleteReport(cycle), nil
}

// getCommanderDecision asks the Commander agent for routing decision
func (co *CommanderOrchestrator) getCommanderDecision(ctx context.Context, userObjective string) (*RoutingDecision, error) {
	// Load Commander prompt
	commanderPrompt, err := co.loadPrompt("commander_prompt.txt")
	if err != nil {
		return nil, fmt.Errorf("failed to load commander prompt: %w", err)
	}

	// Build blackboard summary
	summary := co.blackboard.GetSummary()

	// Construct Commander context
	messages := []providers.Message{
		{
			Role: "system",
			Content: commanderPrompt + "\n\n## Current Assessment State\n\n" +
				"**User Objective:** " + userObjective + "\n\n" +
				"**Blackboard Summary:**\n" + summary,
		},
		{
			Role:    "user",
			Content: "Based on the current state, what should happen next?",
		},
	}

	// Call LLM
	options := map[string]any{
		"max_tokens":  1024,
		"temperature": 0.7,
	}

	response, err := co.provider.Chat(ctx, messages, nil, co.provider.GetDefaultModel(), options)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Parse decision
	return co.parseCommanderResponse(response.Content)
}

// parseCommanderResponse parses the Commander's response into a RoutingDecision
func (co *CommanderOrchestrator) parseCommanderResponse(content string) (*RoutingDecision, error) {
	lines := strings.Split(content, "\n")
	decision := &RoutingDecision{}

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "**ROUTE:") {
			decision.Action = "ROUTE"
			// Extract agent name: "**ROUTE: Recon Agent**" -> "Recon Agent"
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				agent := strings.TrimSpace(parts[1])
				agent = strings.TrimSuffix(agent, "**")
				decision.Agent = agent
			}
		} else if strings.HasPrefix(line, "**COMPLETE**") {
			decision.Action = "COMPLETE"
		} else if strings.HasPrefix(line, "**REASON:") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				reason := strings.TrimSpace(strings.Join(parts[1:], ":"))
				reason = strings.TrimSuffix(reason, "**")
				decision.Reason = reason
			}
		} else if strings.HasPrefix(line, "**FOCUS:") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				focus := strings.TrimSpace(strings.Join(parts[1:], ":"))
				focus = strings.TrimSuffix(focus, "**")
				decision.Focus = focus
			}
		} else if strings.HasPrefix(line, "**SUMMARY:") {
			parts := strings.Split(line, ":")
			if len(parts) >= 2 {
				summary := strings.TrimSpace(strings.Join(parts[1:], ":"))
				summary = strings.TrimSuffix(summary, "**")
				decision.Summary = summary
			}
		}
	}

	// Validate decision
	if decision.Action == "" {
		return nil, fmt.Errorf("commander response missing action: %s", content)
	}

	if decision.Action == "ROUTE" && decision.Agent == "" {
		return nil, fmt.Errorf("commander ROUTE decision missing agent name")
	}

	return decision, nil
}

// executeSpecialist runs the specified specialist agent with tool execution loop
func (co *CommanderOrchestrator) executeSpecialist(ctx context.Context, decision *RoutingDecision) error {
	logger.InfoCF("commander", "Executing specialist", map[string]any{
		"agent": decision.Agent,
		"focus": decision.Focus,
	})

	// Map agent name to prompt file
	promptFile := co.getPromptFileForAgent(decision.Agent)
	if promptFile == "" {
		return fmt.Errorf("unknown agent: %s", decision.Agent)
	}

	// Load specialist prompt
	specialistPrompt, err := co.loadPrompt(promptFile)
	if err != nil {
		return fmt.Errorf("failed to load specialist prompt: %w", err)
	}

	// Build specialist context (just-in-time)
	blackboardSummary := co.blackboard.GetSummary()

	// Get available tools for this specialist
	toolDefs := co.getToolsForSpecialist()

	messages := []providers.Message{
		{
			Role: "system",
			Content: specialistPrompt + "\n\n## Current Context\n\n" +
				"**Your Task:** " + decision.Focus + "\n\n" +
				"**Blackboard Summary:**\n" + blackboardSummary,
		},
		{
			Role:    "user",
			Content: "Execute your specialist task: " + decision.Focus,
		},
	}

	// Tool execution loop (max 5 iterations)
	maxIterations := 5
	for iteration := 0; iteration < maxIterations; iteration++ {
		options := map[string]any{
			"max_tokens":  4096,
			"temperature": 0.7,
		}

		response, err := co.provider.Chat(ctx, messages, toolDefs, co.provider.GetDefaultModel(), options)
		if err != nil {
			return fmt.Errorf("specialist LLM call failed: %w", err)
		}

		// Check if specialist is done (no tool calls)
		if len(response.ToolCalls) == 0 {
			// Record final output to blackboard
			co.blackboard.RecordSpecialistOutput(decision.Agent, response.Content)

			logger.InfoCF("commander", "Specialist completed", map[string]any{
				"agent":       decision.Agent,
				"iterations":  iteration + 1,
				"output_size": len(response.Content),
			})
			return nil
		}

		// Execute tool calls
		toolResults := make([]string, 0, len(response.ToolCalls))
		for _, tc := range response.ToolCalls {
			// Normalize tool call
			normalizedTC := providers.NormalizeToolCall(tc)

			logger.InfoCF("commander", "Specialist executing tool", map[string]any{
				"agent":     decision.Agent,
				"tool":      normalizedTC.Name,
				"iteration": iteration + 1,
			})

			// Execute tool (simplified - no async, no user output)
			result := co.executeTool(ctx, normalizedTC.Name, normalizedTC.Arguments)
			toolResults = append(toolResults, result)
		}

		// Add assistant message with tool calls
		assistantMsg := providers.Message{
			Role:    "assistant",
			Content: response.Content,
		}
		messages = append(messages, assistantMsg)

		// Add tool results as user messages
		for i, tc := range response.ToolCalls {
			normalizedTC := providers.NormalizeToolCall(tc)
			toolMsg := providers.Message{
				Role: "user",
				Content: fmt.Sprintf("Tool %s returned:\n%s",
					normalizedTC.Name,
					toolResults[i]),
			}
			messages = append(messages, toolMsg)
		}

		logger.InfoCF("commander", "Specialist iteration complete", map[string]any{
			"agent":      decision.Agent,
			"iteration":  iteration + 1,
			"tool_calls": len(response.ToolCalls),
		})
	}

	// Max iterations reached
	logger.WarnCF("commander", "Specialist max iterations reached", map[string]any{
		"agent":          decision.Agent,
		"max_iterations": maxIterations,
	})

	return fmt.Errorf("specialist %s exceeded max iterations (%d)", decision.Agent, maxIterations)
}

// getPromptFileForAgent maps agent names to prompt files
func (co *CommanderOrchestrator) getPromptFileForAgent(agentName string) string {
	agentLower := strings.ToLower(agentName)

	if strings.Contains(agentLower, "recon") {
		return "recon_prompt.txt"
	}
	if strings.Contains(agentLower, "web") {
		return "web_analysis_prompt.txt"
	}
	if strings.Contains(agentLower, "api") {
		return "api_testing_prompt.txt"
	}
	if strings.Contains(agentLower, "vuln") || strings.Contains(agentLower, "validation") {
		return "vuln_validation_prompt.txt"
	}

	return ""
}

// loadPrompt loads a prompt file from the prompts directory
func (co *CommanderOrchestrator) loadPrompt(filename string) (string, error) {
	path := filepath.Join(co.promptsDir, filename)
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read prompt file %s: %w", path, err)
	}
	return string(content), nil
}

// formatFinalReport creates the final assessment report
func (co *CommanderOrchestrator) formatFinalReport(decision *RoutingDecision) string {
	report := "# Security Assessment Complete\n\n"
	report += "## Commander Decision\n\n"
	report += fmt.Sprintf("**Reason:** %s\n\n", decision.Reason)
	report += fmt.Sprintf("**Summary:** %s\n\n", decision.Summary)

	// Add blackboard summary
	report += "## Discoveries\n\n"
	report += co.blackboard.GetDetailedSummary()

	return report
}

// formatIncompleteReport creates a report when max cycles is reached
func (co *CommanderOrchestrator) formatIncompleteReport(cycles int) string {
	report := "# Security Assessment Incomplete\n\n"
	report += fmt.Sprintf("**Status:** Maximum cycles (%d) reached\n\n", cycles)
	report += "## Partial Results\n\n"
	report += co.blackboard.GetDetailedSummary()

	return report
}

// getToolsForSpecialist returns tool definitions available to specialists
// Unlike pipeline mode, specialists can access ALL registered tools dynamically
func (co *CommanderOrchestrator) getToolsForSpecialist() []providers.ToolDefinition {
	if co.toolRegistry == nil {
		return nil
	}

	// Get all tool definitions from the execution registry
	// ToProviderDefs() converts tools.Tool implementations to provider format
	return co.toolRegistry.ToProviderDefs()
}

// executeTool executes a tool and returns the result as a string
// Uses the tools.ToolRegistry.Execute() method for actual execution
func (co *CommanderOrchestrator) executeTool(ctx context.Context, toolName string, args map[string]any) string {
	if co.toolRegistry == nil {
		return "Error: Tool registry not available"
	}

	// Execute tool through registry
	// Note: We don't pass channel/chatID since Commander specialists don't have user context
	result := co.toolRegistry.Execute(ctx, toolName, args)

	if result.IsError {
		logger.ErrorCF("commander", "Tool execution failed", map[string]any{
			"tool":  toolName,
			"error": result.ForLLM,
		})
		return fmt.Sprintf("Error executing %s: %s", toolName, result.ForLLM)
	}

	logger.InfoCF("commander", "Tool executed successfully", map[string]any{
		"tool":        toolName,
		"output_size": len(result.ForLLM),
	})

	// Return the ForLLM content (this is what the specialist will see)
	return result.ForLLM
}
