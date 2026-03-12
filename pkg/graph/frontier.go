package graph

import (
	"fmt"
	"sort"

	"github.com/ResistanceIsUseless/picoclaw/pkg/logger"
)

// FrontierNode represents a node with unknown properties that should be explored
type FrontierNode struct {
	Node          *Node
	UnknownProps  []string
	InterestScore float64
	HighInterest  int // count of high-interest unknown properties
	Priority      int // calculated priority (higher = more important)
}

// Frontier represents the exploration frontier - nodes with unknown properties
type Frontier struct {
	nodes    []*FrontierNode
	registry *EntityRegistry
}

// NewFrontier creates a new exploration frontier
func NewFrontier(registry *EntityRegistry) *Frontier {
	return &Frontier{
		nodes:    make([]*FrontierNode, 0),
		registry: registry,
	}
}

// ComputeFrontier calculates the frontier from the current graph state
func (g *Graph) ComputeFrontier(registry *EntityRegistry) *Frontier {
	frontier := NewFrontier(registry)

	g.mu.RLock()
	defer g.mu.RUnlock()

	// Iterate over all nodes
	for nodeID, node := range g.nodes {
		unknownProps := g.GetUnknownProperties(nodeID)

		// Only include nodes with unknown properties
		if len(unknownProps) == 0 {
			continue
		}

		// Calculate interest score
		interestScore, highInterestCount := calculateInterest(node, unknownProps, registry)

		frontierNode := &FrontierNode{
			Node:          node,
			UnknownProps:  unknownProps,
			InterestScore: interestScore,
			HighInterest:  highInterestCount,
			Priority:      0, // will be calculated
		}

		frontier.nodes = append(frontier.nodes, frontierNode)
	}

	// Calculate priorities
	frontier.calculatePriorities()

	// Sort by priority (highest first)
	sort.Slice(frontier.nodes, func(i, j int) bool {
		return frontier.nodes[i].Priority > frontier.nodes[j].Priority
	})

	logger.InfoCF("graph", "Computed exploration frontier",
		map[string]any{
			"frontier_size": len(frontier.nodes),
		})

	return frontier
}

// calculateInterest computes the interest score for a node
func calculateInterest(node *Node, unknownProps []string, registry *EntityRegistry) (float64, int) {
	// Get base interest for entity type
	baseInterest, err := registry.GetDefaultInterest(node.EntityType)
	if err != nil {
		baseInterest = 0.5 // default if type not found
	}

	// Count high-interest unknown properties
	highInterestCount := 0
	for _, prop := range unknownProps {
		if registry.IsHighInterestProperty(node.EntityType, prop) {
			highInterestCount++
		}
	}

	// Calculate final interest score
	// Base + (high-interest properties * 0.1) + (total unknowns * 0.05)
	interestScore := baseInterest +
		float64(highInterestCount)*0.1 +
		float64(len(unknownProps))*0.05

	// Cap at 1.0
	if interestScore > 1.0 {
		interestScore = 1.0
	}

	return interestScore, highInterestCount
}

// calculatePriorities assigns priority ranks to frontier nodes
func (f *Frontier) calculatePriorities() {
	// Priority factors:
	// 1. High-interest property count (most important)
	// 2. Interest score
	// 3. Total unknown properties
	// 4. Entity type default interest

	maxHighInterest := 0
	for _, fn := range f.nodes {
		if fn.HighInterest > maxHighInterest {
			maxHighInterest = fn.HighInterest
		}
	}

	for _, fn := range f.nodes {
		// Normalize high-interest count to 0-100 scale
		highInterestScore := 0
		if maxHighInterest > 0 {
			highInterestScore = (fn.HighInterest * 100) / maxHighInterest
		}

		// Interest score already 0-1, scale to 0-50
		interestScorePart := int(fn.InterestScore * 50)

		// Unknown properties count (capped at 20)
		unknownCountPart := len(fn.UnknownProps)
		if unknownCountPart > 20 {
			unknownCountPart = 20
		}

		// Priority = high-interest (0-100) + interest score (0-50) + unknown count (0-20)
		fn.Priority = highInterestScore + interestScorePart + unknownCountPart
	}
}

// Top returns the top N frontier nodes by priority
func (f *Frontier) Top(n int) []*FrontierNode {
	if n > len(f.nodes) {
		n = len(f.nodes)
	}
	return f.nodes[:n]
}

// Size returns the total number of nodes in the frontier
func (f *Frontier) Size() int {
	return len(f.nodes)
}

// IsEmpty returns true if the frontier has no nodes
func (f *Frontier) IsEmpty() bool {
	return len(f.nodes) == 0
}

// Contains returns true if the frontier includes the given node ID.
func (f *Frontier) Contains(nodeID string) bool {
	for _, fn := range f.nodes {
		if fn.Node != nil && fn.Node.ID == nodeID {
			return true
		}
	}
	return false
}

// GetByEntityType returns frontier nodes of a specific entity type
func (f *Frontier) GetByEntityType(entityType EntityType) []*FrontierNode {
	result := make([]*FrontierNode, 0)
	for _, fn := range f.nodes {
		if fn.Node.EntityType == entityType {
			result = append(result, fn)
		}
	}
	return result
}

// GetByProperty returns frontier nodes that have a specific unknown property
func (f *Frontier) GetByProperty(propertyName string) []*FrontierNode {
	result := make([]*FrontierNode, 0)
	for _, fn := range f.nodes {
		for _, prop := range fn.UnknownProps {
			if prop == propertyName {
				result = append(result, fn)
				break
			}
		}
	}
	return result
}

// Summary returns a human-readable summary of the frontier
func (f *Frontier) Summary() string {
	if f.IsEmpty() {
		return "Frontier is empty - all discoverable properties resolved"
	}

	summary := fmt.Sprintf("Exploration Frontier: %d nodes\n", len(f.nodes))

	// Top 10 nodes by priority
	top := f.Top(10)
	summary += "\nTop Priority Nodes:\n"
	for i, fn := range top {
		summary += fmt.Sprintf("  %d. %s (%s) - Priority: %d, Interest: %.2f, Unknown: %d\n",
			i+1,
			fn.Node.Label,
			fn.Node.EntityType,
			fn.Priority,
			fn.InterestScore,
			len(fn.UnknownProps))

		// Show high-interest properties
		if fn.HighInterest > 0 {
			highInterestProps := make([]string, 0)
			for _, prop := range fn.UnknownProps {
				if f.registry.IsHighInterestProperty(fn.Node.EntityType, prop) {
					highInterestProps = append(highInterestProps, prop)
				}
			}
			summary += fmt.Sprintf("     High-interest props: %v\n", highInterestProps)
		}
	}

	// Entity type distribution
	typeDist := make(map[EntityType]int)
	for _, fn := range f.nodes {
		typeDist[fn.Node.EntityType]++
	}

	summary += "\nBy Entity Type:\n"
	for entityType, count := range typeDist {
		summary += fmt.Sprintf("  - %s: %d\n", entityType, count)
	}

	return summary
}

// RecommendTools suggests which tools should be run based on frontier state
func (f *Frontier) RecommendTools() []ToolRecommendation {
	recommendations := make([]ToolRecommendation, 0)

	// Analyze frontier to determine what tools are needed
	for _, fn := range f.nodes {
		for _, prop := range fn.UnknownProps {
			tools := getToolsForProperty(fn.Node.EntityType, prop)
			for _, tool := range tools {
				recommendations = append(recommendations, ToolRecommendation{
					Tool:         tool,
					Reason:       fmt.Sprintf("Resolve %s.%s for node %s", fn.Node.EntityType, prop, fn.Node.Label),
					TargetNodeID: fn.Node.ID,
					PropertyName: prop,
					Priority:     fn.Priority,
				})
			}
		}
	}

	// Sort by priority
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Priority > recommendations[j].Priority
	})

	// Deduplicate by tool name (keep highest priority)
	seen := make(map[string]bool)
	unique := make([]ToolRecommendation, 0)
	for _, rec := range recommendations {
		if !seen[rec.Tool] {
			seen[rec.Tool] = true
			unique = append(unique, rec)
		}
	}

	return unique
}

// ToolRecommendation suggests a tool to run based on frontier analysis
type ToolRecommendation struct {
	Tool         string
	Reason       string
	TargetNodeID string
	PropertyName string
	Priority     int
}

// getToolsForProperty maps entity properties to tools that can discover them
func getToolsForProperty(entityType EntityType, propertyName string) []string {
	// This mapping defines which tools can discover which properties
	// In a full implementation, this would be in a configuration file or registry

	toolMap := map[EntityType]map[string][]string{
		EntitySubdomain: {
			"ip_addresses": []string{"dig", "host"},
			"ports":        []string{"nmap"},
			"services":     []string{"nmap", "httpx"},
			"technologies": []string{"whatweb", "wappalyzer"},
			"endpoints":    []string{"katana", "gospider"},
		},
		EntityIP: {
			"open_ports": []string{"nmap", "masscan"},
			"os":         []string{"nmap"},
			"services":   []string{"nmap"},
		},
		EntityPort: {
			"service":         []string{"nmap"},
			"version":         []string{"nmap"},
			"vulnerabilities": []string{"nuclei", "nmap_vulners"},
		},
		EntityEndpoint: {
			"parameters":      []string{"katana", "paramspider"},
			"vulnerabilities": []string{"nuclei", "nikto"},
		},
		EntityParameter: {
			"injectable": []string{"sqlmap", "xsstrike"},
			"sink_type":  []string{"manual_analysis"},
		},
		EntityFunction: {
			"calls":                []string{"codeql"},
			"dangerous_sinks":      []string{"codeql", "semgrep"},
			"buffer_overflow_path": []string{"codeql"},
		},
		EntitySharedLibrary: {
			"cves":                []string{"cve_lookup"},
			"reachable_functions": []string{"ghidra", "rizin"},
		},
	}

	if propMap, exists := toolMap[entityType]; exists {
		if tools, exists := propMap[propertyName]; exists {
			return tools
		}
	}

	return []string{}
}
