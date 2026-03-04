package integration

import (
	"context"
	"fmt"

	"github.com/ResistanceIsUseless/picoclaw/pkg/artifacts"
	"github.com/ResistanceIsUseless/picoclaw/pkg/blackboard"
	"github.com/ResistanceIsUseless/picoclaw/pkg/logger"
	"github.com/ResistanceIsUseless/picoclaw/pkg/orchestrator"
	"github.com/ResistanceIsUseless/picoclaw/pkg/providers"
	"github.com/ResistanceIsUseless/picoclaw/pkg/registry"
)

// CLAWAdapter bridges CLAW orchestrator with existing agent loop
// This allows gradual migration from legacy to CLAW architecture
type CLAWAdapter struct {
	orchestrator   *orchestrator.Orchestrator
	blackboard     *blackboard.Blackboard
	toolRegistry   *registry.ToolRegistry
	provider       providers.LLMProvider
	enabled        bool
}

// CLAWConfig configures CLAW adapter behavior
type CLAWConfig struct {
	Enabled       bool   // Enable CLAW mode
	Pipeline      string // Pipeline name (web_full, web_quick, or custom)
	PersistenceDir string // Directory for blackboard persistence
}

// NewCLAWAdapter creates a new CLAW adapter
func NewCLAWAdapter(cfg *CLAWConfig, provider providers.LLMProvider) (*CLAWAdapter, error) {
	if !cfg.Enabled {
		return &CLAWAdapter{enabled: false}, nil
	}

	// Initialize blackboard with persistence
	var persister blackboard.Persister
	if cfg.PersistenceDir != "" {
		var err error
		persister, err = blackboard.NewFilePersister(cfg.PersistenceDir)
		if err != nil {
			return nil, fmt.Errorf("failed to create blackboard persister: %w", err)
		}
	}
	bb := blackboard.New(persister)

	// Initialize tool registry
	toolRegistry := registry.NewToolRegistry()

	// Register security tools
	if err := registry.RegisterSecurityTools(toolRegistry); err != nil {
		return nil, fmt.Errorf("failed to register security tools: %w", err)
	}

	// Get pipeline
	var pipeline *orchestrator.Pipeline
	var err error
	if cfg.Pipeline != "" {
		pipeline, err = orchestrator.GetPredefinedPipeline(cfg.Pipeline)
		if err != nil {
			// Try to load custom pipeline
			return nil, fmt.Errorf("failed to load pipeline %q: %w", cfg.Pipeline, err)
		}
	} else {
		// Default to web_quick
		pipeline, _ = orchestrator.GetPredefinedPipeline("web_quick")
	}

	// Create orchestrator
	orch := orchestrator.NewOrchestrator(pipeline, bb, toolRegistry)

	// Set provider for model calls
	orch.SetProvider(provider)

	logger.InfoCF("claw", "CLAW adapter initialized",
		map[string]any{
			"pipeline":        pipeline.Name,
			"phases":          len(pipeline.Phases),
			"persistence_dir": cfg.PersistenceDir,
		})

	return &CLAWAdapter{
		orchestrator: orch,
		blackboard:   bb,
		toolRegistry: toolRegistry,
		provider:     provider,
		enabled:      true,
	}, nil
}

// IsEnabled returns true if CLAW mode is active
func (ca *CLAWAdapter) IsEnabled() bool {
	return ca.enabled
}

// ProcessMessage processes a message using CLAW orchestrator
// This replaces runAgentLoop when CLAW mode is enabled
func (ca *CLAWAdapter) ProcessMessage(ctx context.Context, userMessage string) (string, error) {
	if !ca.enabled {
		return "", fmt.Errorf("CLAW adapter is not enabled")
	}

	logger.InfoCF("claw", "Processing message in CLAW mode",
		map[string]any{
			"message_preview": truncateString(userMessage, 100),
		})

	// Parse operator target from message
	// For now, assume message is target specification
	// Example: "web:example.com" or "network:192.168.1.0/24"
	target, targetType := parseTargetFromMessage(userMessage)

	// Create OperatorTarget artifact
	operatorTarget := artifacts.NewOperatorTarget(target, targetType, "input")

	// Publish to blackboard to kick off pipeline
	if err := ca.blackboard.Publish(ctx, operatorTarget); err != nil {
		return "", fmt.Errorf("failed to publish operator target: %w", err)
	}

	// Execute orchestrator
	if err := ca.orchestrator.Execute(ctx); err != nil {
		logger.ErrorCF("claw", "Orchestrator execution failed",
			map[string]any{
				"error": err.Error(),
			})
		return "", fmt.Errorf("orchestrator execution failed: %w", err)
	}

	// Generate summary response
	summary := ca.orchestrator.Summary()

	logger.InfoCF("claw", "CLAW execution completed",
		map[string]any{
			"phases_completed": len(ca.orchestrator.GetCompletedPhases()),
		})

	return summary, nil
}

// GetOrchestrator returns the underlying orchestrator (for testing/inspection)
func (ca *CLAWAdapter) GetOrchestrator() *orchestrator.Orchestrator {
	return ca.orchestrator
}

// GetBlackboard returns the blackboard (for testing/inspection)
func (ca *CLAWAdapter) GetBlackboard() *blackboard.Blackboard {
	return ca.blackboard
}

// parseTargetFromMessage extracts target and type from user message
// TODO: Implement proper parsing logic
func parseTargetFromMessage(message string) (target string, targetType string) {
	// Simple parsing for now
	// Format: "web:example.com" or "network:192.168.1.0/24" or just "example.com"

	// Check for explicit type prefix
	if len(message) > 4 && message[3] == ':' {
		prefix := message[:3]
		switch prefix {
		case "web":
			return message[4:], "web"
		case "net":
			return message[4:], "network"
		}
	}

	if len(message) > 7 && message[6] == ':' {
		prefix := message[:6]
		if prefix == "source" {
			return message[7:], "source"
		}
	}

	if len(message) > 8 && message[7] == ':' {
		prefix := message[:7]
		switch prefix {
		case "network":
			return message[8:], "network"
		case "firmware":
			return message[9:], "firmware"
		}
	}

	if len(message) > 7 && message[6] == ':' {
		prefix := message[:6]
		if prefix == "binary" {
			return message[7:], "binary"
		}
	}

	// Default: assume web domain
	return message, "web"
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
