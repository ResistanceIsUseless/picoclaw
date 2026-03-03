package graph

import (
	"fmt"
	"sync"
	"time"
)

// Graph represents the knowledge graph for exploration-based reconnaissance
// It stores entities (domains, IPs, ports, functions, etc.) and their relationships
type Graph struct {
	mu         sync.RWMutex
	nodes      map[string]*Node           // nodeID -> Node
	edges      map[string]map[string]*Edge // fromID -> toID -> Edge
	entities   map[EntityType][]*Node     // entityType -> nodes of that type
	properties map[string]PropertyStore   // nodeID -> properties
}

// Node represents an entity in the knowledge graph
type Node struct {
	ID           string                 `json:"id"`
	EntityType   EntityType             `json:"entity_type"`
	Label        string                 `json:"label"` // human-readable name
	Properties   map[string]interface{} `json:"properties"`
	DiscoveredAt time.Time              `json:"discovered_at"`
	DiscoveredBy string                 `json:"discovered_by"` // which tool/phase
	Confirmed    bool                   `json:"confirmed"`     // is this entity verified?
}

// Edge represents a relationship between two nodes
type Edge struct {
	ID           string                 `json:"id"`
	From         string                 `json:"from"` // source node ID
	To           string                 `json:"to"`   // target node ID
	RelationType RelationType           `json:"relation_type"`
	Properties   map[string]interface{} `json:"properties"`
	DiscoveredAt time.Time              `json:"discovered_at"`
	DiscoveredBy string                 `json:"discovered_by"`
}

// PropertyStore holds properties for a node with metadata
type PropertyStore struct {
	Known   map[string]PropertyValue `json:"known"`   // property -> value
	Unknown []string                 `json:"unknown"` // list of unknown property names
}

// PropertyValue represents a property with its metadata
type PropertyValue struct {
	Value        interface{} `json:"value"`
	ResolvedAt   time.Time   `json:"resolved_at"`
	ResolvedBy   string      `json:"resolved_by"` // which tool resolved it
	Confidence   float64     `json:"confidence"`  // 0.0 - 1.0
	NeedsConfirm bool        `json:"needs_confirm"` // should this be verified?
}

// NewGraph creates a new knowledge graph
func NewGraph() *Graph {
	return &Graph{
		nodes:      make(map[string]*Node),
		edges:      make(map[string]map[string]*Edge),
		entities:   make(map[EntityType][]*Node),
		properties: make(map[string]PropertyStore),
	}
}

// AddNode adds a node to the graph
func (g *Graph) AddNode(node *Node) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if node.ID == "" {
		return fmt.Errorf("node ID cannot be empty")
	}

	// Check if node already exists
	if existing, exists := g.nodes[node.ID]; exists {
		// Update existing node properties
		for k, v := range node.Properties {
			existing.Properties[k] = v
		}
		return nil
	}

	g.nodes[node.ID] = node
	g.entities[node.EntityType] = append(g.entities[node.EntityType], node)

	// Initialize property store if not exists
	if _, exists := g.properties[node.ID]; !exists {
		g.properties[node.ID] = PropertyStore{
			Known:   make(map[string]PropertyValue),
			Unknown: make([]string, 0),
		}
	}

	return nil
}

// AddEdge adds an edge between two nodes
func (g *Graph) AddEdge(edge *Edge) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if edge.From == "" || edge.To == "" {
		return fmt.Errorf("edge from/to cannot be empty")
	}

	// Verify nodes exist
	if _, exists := g.nodes[edge.From]; !exists {
		return fmt.Errorf("source node %q not found", edge.From)
	}
	if _, exists := g.nodes[edge.To]; !exists {
		return fmt.Errorf("target node %q not found", edge.To)
	}

	// Create edge map for source if not exists
	if g.edges[edge.From] == nil {
		g.edges[edge.From] = make(map[string]*Edge)
	}

	g.edges[edge.From][edge.To] = edge

	return nil
}

// GetNode retrieves a node by ID
func (g *Graph) GetNode(nodeID string) (*Node, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	node, exists := g.nodes[nodeID]
	if !exists {
		return nil, fmt.Errorf("node %q not found", nodeID)
	}

	return node, nil
}

// GetNodesByType returns all nodes of a specific entity type
func (g *Graph) GetNodesByType(entityType EntityType) []*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()

	nodes := g.entities[entityType]
	result := make([]*Node, len(nodes))
	copy(result, nodes)

	return result
}

// GetEdges returns all edges from a node
func (g *Graph) GetEdges(fromNodeID string) []*Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()

	edgeMap, exists := g.edges[fromNodeID]
	if !exists {
		return nil
	}

	edges := make([]*Edge, 0, len(edgeMap))
	for _, edge := range edgeMap {
		edges = append(edges, edge)
	}

	return edges
}

// GetEdgesByType returns all edges of a specific relation type from a node
func (g *Graph) GetEdgesByType(fromNodeID string, relationType RelationType) []*Edge {
	g.mu.RLock()
	defer g.mu.RUnlock()

	edgeMap, exists := g.edges[fromNodeID]
	if !exists {
		return nil
	}

	edges := make([]*Edge, 0)
	for _, edge := range edgeMap {
		if edge.RelationType == relationType {
			edges = append(edges, edge)
		}
	}

	return edges
}

// SetProperty sets a property value for a node
func (g *Graph) SetProperty(nodeID string, propertyName string, value PropertyValue) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.nodes[nodeID]; !exists {
		return fmt.Errorf("node %q not found", nodeID)
	}

	store, exists := g.properties[nodeID]
	if !exists {
		store = PropertyStore{
			Known:   make(map[string]PropertyValue),
			Unknown: make([]string, 0),
		}
	}

	store.Known[propertyName] = value

	// Remove from unknown list if it was there
	unknownFiltered := make([]string, 0)
	for _, prop := range store.Unknown {
		if prop != propertyName {
			unknownFiltered = append(unknownFiltered, prop)
		}
	}
	store.Unknown = unknownFiltered

	g.properties[nodeID] = store

	return nil
}

// GetProperty retrieves a property value for a node
func (g *Graph) GetProperty(nodeID string, propertyName string) (PropertyValue, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	store, exists := g.properties[nodeID]
	if !exists {
		return PropertyValue{}, false
	}

	value, exists := store.Known[propertyName]
	return value, exists
}

// MarkPropertyUnknown marks a property as unknown for a node
func (g *Graph) MarkPropertyUnknown(nodeID string, propertyName string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.nodes[nodeID]; !exists {
		return fmt.Errorf("node %q not found", nodeID)
	}

	store, exists := g.properties[nodeID]
	if !exists {
		store = PropertyStore{
			Known:   make(map[string]PropertyValue),
			Unknown: make([]string, 0),
		}
	}

	// Check if already marked as unknown
	for _, prop := range store.Unknown {
		if prop == propertyName {
			return nil // already marked
		}
	}

	store.Unknown = append(store.Unknown, propertyName)
	g.properties[nodeID] = store

	return nil
}

// GetUnknownProperties returns all unknown properties for a node
func (g *Graph) GetUnknownProperties(nodeID string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	store, exists := g.properties[nodeID]
	if !exists {
		return nil
	}

	result := make([]string, len(store.Unknown))
	copy(result, store.Unknown)

	return result
}

// NodeCount returns the total number of nodes
func (g *Graph) NodeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.nodes)
}

// EdgeCount returns the total number of edges
func (g *Graph) EdgeCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	count := 0
	for _, edgeMap := range g.edges {
		count += len(edgeMap)
	}
	return count
}

// GetNeighbors returns all nodes connected to the given node
func (g *Graph) GetNeighbors(nodeID string) []*Node {
	g.mu.RLock()
	defer g.mu.RUnlock()

	edgeMap, exists := g.edges[nodeID]
	if !exists {
		return nil
	}

	neighbors := make([]*Node, 0, len(edgeMap))
	for toNodeID := range edgeMap {
		if node, exists := g.nodes[toNodeID]; exists {
			neighbors = append(neighbors, node)
		}
	}

	return neighbors
}

// Summary returns a human-readable summary of the graph
func (g *Graph) Summary() string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	summary := fmt.Sprintf("Knowledge Graph: %d nodes, %d edges\n", len(g.nodes), g.EdgeCount())

	entityCounts := make(map[EntityType]int)
	for entityType, nodes := range g.entities {
		entityCounts[entityType] = len(nodes)
	}

	summary += "\nEntities by type:\n"
	for entityType, count := range entityCounts {
		summary += fmt.Sprintf("  - %s: %d\n", entityType, count)
	}

	return summary
}

// Clear removes all nodes and edges (used for testing)
func (g *Graph) Clear() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.nodes = make(map[string]*Node)
	g.edges = make(map[string]map[string]*Edge)
	g.entities = make(map[EntityType][]*Node)
	g.properties = make(map[string]PropertyStore)
}
