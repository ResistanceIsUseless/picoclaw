package phase

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ResistanceIsUseless/picoclaw/pkg/blackboard"
	"github.com/ResistanceIsUseless/picoclaw/pkg/graph"
	"github.com/ResistanceIsUseless/picoclaw/pkg/logger"
	"github.com/ResistanceIsUseless/picoclaw/pkg/registry"
)

// PhaseContextBuilder assembles phase-scoped context for model prompts
// Replaces flat BuildMessages with structured, cacheable context for CLAW phases
type PhaseContextBuilder struct {
	PhaseName      string
	PhaseObjective string
	TokenBudget    int
	KVCacheEnabled bool
}

// PhaseContextInput contains all inputs for building phase context
type PhaseContextInput struct {
	PhaseName      string
	Objective      string
	Contract       *PhaseContract
	State          *DAGState
	Blackboard     *blackboard.Blackboard
	Graph          *graph.Graph
	Frontier       *graph.Frontier
	Registry       *registry.ToolRegistry
	Iteration      int
	PreviousPhases []string // phases that have already completed
}

// ContextSection represents a section of the context prompt
type ContextSection struct {
	Name      string
	Priority  int // higher priority sections come first
	Content   string
	Cacheable bool // can this section be KV cached?
	TokenCost int  // approximate token count
}

// NewPhaseContextBuilder creates a new phase-scoped context builder
func NewPhaseContextBuilder(phaseName string, objective string, tokenBudget int) *PhaseContextBuilder {
	return &PhaseContextBuilder{
		PhaseName:      phaseName,
		PhaseObjective: objective,
		TokenBudget:    tokenBudget,
		KVCacheEnabled: true,
	}
}

// Build assembles the complete context for a phase iteration
func (b *PhaseContextBuilder) Build(input *PhaseContextInput) ([]ContextSection, error) {
	sections := make([]ContextSection, 0)

	// Section 1: System prompt (highest priority, cacheable)
	systemPrompt := b.buildSystemPrompt(input)
	sections = append(sections, ContextSection{
		Name:      "system_prompt",
		Priority:  1000,
		Content:   systemPrompt,
		Cacheable: true,
		TokenCost: estimateTokens(systemPrompt),
	})

	// Section 2: Phase objective and contract (cacheable)
	phaseContext := b.buildPhaseContext(input)
	sections = append(sections, ContextSection{
		Name:      "phase_context",
		Priority:  900,
		Content:   phaseContext,
		Cacheable: true,
		TokenCost: estimateTokens(phaseContext),
	})

	// Section 3: Input artifacts from previous phases (cacheable if unchanged)
	if len(input.PreviousPhases) > 0 {
		inputArtifacts := b.buildInputArtifacts(input)
		sections = append(sections, ContextSection{
			Name:      "input_artifacts",
			Priority:  800,
			Content:   inputArtifacts,
			Cacheable: true,
			TokenCost: estimateTokens(inputArtifacts),
		})
	}

	// Section 4: Knowledge graph state (cacheable if graph unchanged)
	if input.Graph != nil && input.Graph.NodeCount() > 0 {
		graphState := b.buildGraphState(input)
		sections = append(sections, ContextSection{
			Name:      "graph_state",
			Priority:  700,
			Content:   graphState,
			Cacheable: true,
			TokenCost: estimateTokens(graphState),
		})
	}

	// Section 5: Frontier and tool recommendations (semi-cacheable)
	if input.Frontier != nil && !input.Frontier.IsEmpty() {
		frontierState := b.buildFrontierState(input)
		sections = append(sections, ContextSection{
			Name:      "frontier_state",
			Priority:  600,
			Content:   frontierState,
			Cacheable: false, // changes frequently
			TokenCost: estimateTokens(frontierState),
		})
	}

	// Section 6: DAG state (NOT cacheable - changes every iteration)
	dagState := input.State.RenderState()
	sections = append(sections, ContextSection{
		Name:      "dag_state",
		Priority:  500,
		Content:   dagState,
		Cacheable: false,
		TokenCost: estimateTokens(dagState),
	})

	// Section 7: Contract status (NOT cacheable - changes with progress)
	contractStatus := input.Contract.GetCompletionStatus(&PhaseContext{
		Phase:      input.PhaseName,
		State:      input.State,
		Blackboard: input.Blackboard,
		Artifacts:  b.getCurrentPhaseArtifacts(input),
		Iteration:  input.Iteration,
	})
	sections = append(sections, ContextSection{
		Name:      "contract_status",
		Priority:  400,
		Content:   contractStatus,
		Cacheable: false,
		TokenCost: estimateTokens(contractStatus),
	})

	// Apply token budget if needed
	sections = b.applyTokenBudget(sections)

	logger.InfoCF("phase_context", "Built phase context",
		map[string]any{
			"phase":         input.PhaseName,
			"iteration":     input.Iteration,
			"section_count": len(sections),
			"total_tokens":  b.calculateTotalTokens(sections),
		})

	return sections, nil
}

// buildSystemPrompt creates the system-level instructions
func (b *PhaseContextBuilder) buildSystemPrompt(input *PhaseContextInput) string {
	var sb strings.Builder

	sb.WriteString("# Security Assessment Agent\n\n")
	sb.WriteString("You are an autonomous security assessment agent executing a structured pipeline.\n\n")

	sb.WriteString("## Core Principles\n\n")
	sb.WriteString("1. **Tool calls are the only way to progress** - You MUST call tools to discover information\n")
	sb.WriteString("2. **Never hallucinate data** - Only use information from tool outputs\n")
	sb.WriteString("3. **Follow the DAG state** - Only call tools marked as READY\n")
	sb.WriteString("4. **Respect the contract** - Phase is complete when contract requirements are met\n")
	sb.WriteString("5. **Use the graph** - Consult frontier for exploration priorities\n\n")

	sb.WriteString("## Available Tools\n\n")
	if input.Registry != nil {
		// List available tools for this phase
		tools := input.Contract.RequiredTools
		tools = append(tools, input.Contract.OptionalTools...)
		for _, profileName := range input.Contract.RequiredProfiles {
			sb.WriteString(fmt.Sprintf("- **%s profile**: at least one matching tool must be executed\n", profileName))
		}
		for _, toolName := range tools {
			// Would fetch tool definition from registry
			sb.WriteString(fmt.Sprintf("- **%s**: [Tool description would go here]\n", toolName))
		}
	}
	sb.WriteString("\n")

	sb.WriteString("## Orchestrator Tools (Tier -1)\n\n")
	sb.WriteString("These tools control phase flow:\n")
	sb.WriteString("- **complete_phase**: Mark phase as complete (only when contract satisfied)\n")
	sb.WriteString("- **escalate**: Request human assistance for blockers\n")
	sb.WriteString("- **validate_artifact**: Check artifact quality before proceeding\n\n")

	return sb.String()
}

// buildPhaseContext creates phase-specific context
func (b *PhaseContextBuilder) buildPhaseContext(input *PhaseContextInput) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Current Phase: %s\n\n", input.PhaseName))
	sb.WriteString(fmt.Sprintf("**Objective**: %s\n\n", input.Objective))
	sb.WriteString(fmt.Sprintf("**Iteration**: %d\n\n", input.Iteration))

	sb.WriteString("## Phase Requirements\n\n")
	sb.WriteString("To complete this phase, you must:\n\n")

	if len(input.Contract.RequiredTools) > 0 {
		sb.WriteString("### Required Tool Executions\n")
		for _, tool := range input.Contract.RequiredTools {
			sb.WriteString(fmt.Sprintf("- %s\n", tool))
		}
		sb.WriteString("\n")
	}

	if len(input.Contract.RequiredProfiles) > 0 {
		sb.WriteString("### Required Tool Profiles\n")
		for _, profileName := range input.Contract.RequiredProfiles {
			sb.WriteString(fmt.Sprintf("- %s (at least one matching tool)\n", profileName))
		}
		sb.WriteString("\n")
	}

	if len(input.Contract.RequiredArtifacts) > 0 {
		sb.WriteString("### Required Artifact Production\n")
		for _, artifactType := range input.Contract.RequiredArtifacts {
			sb.WriteString(fmt.Sprintf("- %s\n", artifactType))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("### Iteration Limits\n"))
	sb.WriteString(fmt.Sprintf("- Minimum: %d iterations\n", input.Contract.MinIterations))
	sb.WriteString(fmt.Sprintf("- Maximum: %d iterations\n", input.Contract.MaxIterations))
	sb.WriteString("\n")

	return sb.String()
}

// buildInputArtifacts includes artifacts from previous phases
func (b *PhaseContextBuilder) buildInputArtifacts(input *PhaseContextInput) string {
	var sb strings.Builder

	sb.WriteString("## Input Artifacts\n\n")
	sb.WriteString("These artifacts were produced by previous phases:\n\n")

	// Get artifacts from completed phases
	for _, phaseName := range input.PreviousPhases {
		artifacts, err := input.Blackboard.GetByPhase(phaseName)
		if err == nil && len(artifacts) > 0 {
			sb.WriteString(fmt.Sprintf("### From %s phase:\n", phaseName))
			for _, envelope := range artifacts {
				sb.WriteString(fmt.Sprintf("- **%s** (created: %s)\n",
					envelope.Metadata.Type,
					envelope.Metadata.CreatedAt.Format(time.RFC3339)))

				// Include artifact summary
				summary := b.summarizeArtifact(&envelope)
				if summary != "" {
					sb.WriteString(fmt.Sprintf("  %s\n", summary))
				}
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// buildGraphState renders current knowledge graph
func (b *PhaseContextBuilder) buildGraphState(input *PhaseContextInput) string {
	var sb strings.Builder

	sb.WriteString("## Knowledge Graph State\n\n")
	sb.WriteString(fmt.Sprintf("Nodes: %d | Edges: %d\n\n",
		input.Graph.NodeCount(),
		input.Graph.EdgeCount()))

	// Show summary of entities by type
	sb.WriteString("### Discovered Entities\n\n")

	// Would iterate nodes and group by type
	// For now, placeholder
	sb.WriteString("(Graph summary showing entity counts by type)\n\n")

	return sb.String()
}

// buildFrontierState shows exploration priorities
func (b *PhaseContextBuilder) buildFrontierState(input *PhaseContextInput) string {
	var sb strings.Builder

	sb.WriteString("## Exploration Frontier\n\n")
	sb.WriteString(input.Frontier.Summary())
	sb.WriteString("\n")

	// Tool recommendations
	recommendations := input.Frontier.RecommendTools()
	if len(recommendations) > 0 {
		sb.WriteString("### Recommended Tools (based on frontier)\n\n")
		for i, rec := range recommendations {
			if i >= 5 {
				break // Top 5 only
			}
			sb.WriteString(fmt.Sprintf("- **%s**: %s (priority: %d)\n",
				rec.Tool, rec.Reason, rec.Priority))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// getCurrentPhaseArtifacts gets artifacts produced in current phase
func (b *PhaseContextBuilder) getCurrentPhaseArtifacts(input *PhaseContextInput) []blackboard.ArtifactEnvelope {
	artifacts, err := input.Blackboard.GetByPhase(input.PhaseName)
	if err != nil {
		return []blackboard.ArtifactEnvelope{}
	}
	return artifacts
}

// summarizeArtifact creates a brief summary of an artifact
func (b *PhaseContextBuilder) summarizeArtifact(envelope *blackboard.ArtifactEnvelope) string {
	// Would unmarshal and summarize based on type
	// For now, just show metadata
	return fmt.Sprintf("Domain: %s, Phase: %s",
		envelope.Metadata.Domain,
		envelope.Metadata.Phase)
}

// applyTokenBudget trims sections if over budget
func (b *PhaseContextBuilder) applyTokenBudget(sections []ContextSection) []ContextSection {
	if b.TokenBudget <= 0 {
		return sortSectionsByPriority(sections)
	}

	sections = sortSectionsByPriority(sections)
	totalTokens := b.calculateTotalTokens(sections)
	if totalTokens <= b.TokenBudget {
		return sections // Under budget
	}

	core := make([]ContextSection, 0)
	optional := make([]ContextSection, 0)
	for _, section := range sections {
		if section.Priority >= 800 {
			core = append(core, section)
		} else {
			optional = append(optional, section)
		}
	}

	trimmed := append([]ContextSection{}, core...)
	currentTokens := b.calculateTotalTokens(trimmed)
	for _, section := range optional {
		if currentTokens+section.TokenCost <= b.TokenBudget {
			trimmed = append(trimmed, section)
			currentTokens += section.TokenCost
		}
	}

	logger.WarnCF("phase_context", "Token budget exceeded",
		map[string]any{
			"budget":  b.TokenBudget,
			"actual":  totalTokens,
			"trimmed": currentTokens,
			"dropped": len(sections) - len(trimmed),
		})

	return sortSectionsByPriority(trimmed)
}

// calculateTotalTokens sums token costs across sections
func (b *PhaseContextBuilder) calculateTotalTokens(sections []ContextSection) int {
	total := 0
	for _, section := range sections {
		total += section.TokenCost
	}
	return total
}

// RenderSections converts sections to final prompt string
func RenderSections(sections []ContextSection) string {
	var sb strings.Builder

	for _, section := range sortSectionsByPriority(sections) {
		sb.WriteString(section.Content)
		sb.WriteString("\n---\n\n")
	}

	return sb.String()
}

func sortSectionsByPriority(sections []ContextSection) []ContextSection {
	sorted := append([]ContextSection(nil), sections...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Priority > sorted[j].Priority
	})
	return sorted
}

// estimateTokens rough token estimation (4 chars ≈ 1 token)
func estimateTokens(text string) int {
	return len(text) / 4
}

// PhaseContextCache manages KV cache for context sections
type PhaseContextCache struct {
	cachedSections map[string]string // section name -> content hash
	hitCount       int
	missCount      int
}

// NewPhaseContextCache creates a new phase context cache
func NewPhaseContextCache() *PhaseContextCache {
	return &PhaseContextCache{
		cachedSections: make(map[string]string),
	}
}

// MarkCacheable marks sections that can use KV cache
func (c *PhaseContextCache) MarkCacheable(sections []ContextSection) []ContextSection {
	marked := make([]ContextSection, len(sections))
	copy(marked, sections)

	for i, section := range marked {
		if section.Cacheable {
			// Check if content changed
			hash := hashContent(section.Content)
			if prevHash, exists := c.cachedSections[section.Name]; exists {
				if prevHash == hash {
					// Cache hit - can reuse
					marked[i].Cacheable = true
					c.hitCount++
				} else {
					// Cache miss - content changed
					marked[i].Cacheable = false
					c.cachedSections[section.Name] = hash
					c.missCount++
				}
			} else {
				// First time seeing this section
				c.cachedSections[section.Name] = hash
				marked[i].Cacheable = true
				c.missCount++
			}
		}
	}

	return marked
}

// GetStats returns cache statistics
func (c *PhaseContextCache) GetStats() map[string]int {
	return map[string]int{
		"hits":   c.hitCount,
		"misses": c.missCount,
	}
}

// hashContent creates a simple hash of content for cache comparison
func hashContent(content string) string {
	// Simple hash - in production would use crypto hash
	return fmt.Sprintf("%d", len(content))
}
