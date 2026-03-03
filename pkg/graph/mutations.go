package graph

import (
	"fmt"
	"time"

	"github.com/ResistanceIsUseless/picoclaw/pkg/logger"
)

// GraphMutation represents a change to the knowledge graph
// Tool output parsers produce mutations instead of directly modifying the graph
type GraphMutation struct {
	Type         MutationType           `json:"type"`
	Nodes        []*NodeMutation        `json:"nodes,omitempty"`
	Edges        []*EdgeMutation        `json:"edges,omitempty"`
	Properties   []*PropertyMutation    `json:"properties,omitempty"`
	DiscoveredBy string                 `json:"discovered_by"` // tool name
	Timestamp    time.Time              `json:"timestamp"`
}

// MutationType defines the type of graph mutation
type MutationType string

const (
	MutationAddNodes      MutationType = "add_nodes"
	MutationAddEdges      MutationType = "add_edges"
	MutationSetProperties MutationType = "set_properties"
	MutationMarkUnknown   MutationType = "mark_unknown"
	MutationBatch         MutationType = "batch" // multiple mutation types in one
)

// NodeMutation represents adding a node to the graph
type NodeMutation struct {
	ID           string                 `json:"id"`
	EntityType   EntityType             `json:"entity_type"`
	Label        string                 `json:"label"`
	Properties   map[string]interface{} `json:"properties"`
	Confirmed    bool                   `json:"confirmed"`
}

// EdgeMutation represents adding an edge to the graph
type EdgeMutation struct {
	ID           string                 `json:"id"`
	From         string                 `json:"from"`
	To           string                 `json:"to"`
	RelationType RelationType           `json:"relation_type"`
	Properties   map[string]interface{} `json:"properties"`
}

// PropertyMutation represents setting a property on a node
type PropertyMutation struct {
	NodeID       string      `json:"node_id"`
	PropertyName string      `json:"property_name"`
	Value        interface{} `json:"value"`
	Confidence   float64     `json:"confidence"`
	NeedsConfirm bool        `json:"needs_confirm"`
}

// ApplyMutation applies a mutation to the graph
func (g *Graph) ApplyMutation(mutation *GraphMutation) error {
	switch mutation.Type {
	case MutationAddNodes:
		return g.applyNodeMutations(mutation.Nodes, mutation.DiscoveredBy, mutation.Timestamp)

	case MutationAddEdges:
		return g.applyEdgeMutations(mutation.Edges, mutation.DiscoveredBy, mutation.Timestamp)

	case MutationSetProperties:
		return g.applyPropertyMutations(mutation.Properties, mutation.DiscoveredBy, mutation.Timestamp)

	case MutationBatch:
		// Apply all mutation types in sequence
		if err := g.applyNodeMutations(mutation.Nodes, mutation.DiscoveredBy, mutation.Timestamp); err != nil {
			return fmt.Errorf("failed to apply node mutations: %w", err)
		}
		if err := g.applyEdgeMutations(mutation.Edges, mutation.DiscoveredBy, mutation.Timestamp); err != nil {
			return fmt.Errorf("failed to apply edge mutations: %w", err)
		}
		if err := g.applyPropertyMutations(mutation.Properties, mutation.DiscoveredBy, mutation.Timestamp); err != nil {
			return fmt.Errorf("failed to apply property mutations: %w", err)
		}
		return nil

	default:
		return fmt.Errorf("unknown mutation type: %s", mutation.Type)
	}
}

func (g *Graph) applyNodeMutations(nodes []*NodeMutation, discoveredBy string, timestamp time.Time) error {
	for _, nodeMut := range nodes {
		node := &Node{
			ID:           nodeMut.ID,
			EntityType:   nodeMut.EntityType,
			Label:        nodeMut.Label,
			Properties:   nodeMut.Properties,
			DiscoveredAt: timestamp,
			DiscoveredBy: discoveredBy,
			Confirmed:    nodeMut.Confirmed,
		}

		if err := g.AddNode(node); err != nil {
			logger.WarnCF("graph", "Failed to add node from mutation",
				map[string]any{
					"node_id": nodeMut.ID,
					"error":   err.Error(),
				})
			continue
		}

		logger.DebugCF("graph", "Added node from mutation",
			map[string]any{
				"node_id":       nodeMut.ID,
				"entity_type":   nodeMut.EntityType,
				"discovered_by": discoveredBy,
			})
	}

	return nil
}

func (g *Graph) applyEdgeMutations(edges []*EdgeMutation, discoveredBy string, timestamp time.Time) error {
	for _, edgeMut := range edges {
		edge := &Edge{
			ID:           edgeMut.ID,
			From:         edgeMut.From,
			To:           edgeMut.To,
			RelationType: edgeMut.RelationType,
			Properties:   edgeMut.Properties,
			DiscoveredAt: timestamp,
			DiscoveredBy: discoveredBy,
		}

		if err := g.AddEdge(edge); err != nil {
			logger.WarnCF("graph", "Failed to add edge from mutation",
				map[string]any{
					"edge_id": edgeMut.ID,
					"from":    edgeMut.From,
					"to":      edgeMut.To,
					"error":   err.Error(),
				})
			continue
		}

		logger.DebugCF("graph", "Added edge from mutation",
			map[string]any{
				"edge_id":       edgeMut.ID,
				"relation":      edgeMut.RelationType,
				"discovered_by": discoveredBy,
			})
	}

	return nil
}

func (g *Graph) applyPropertyMutations(properties []*PropertyMutation, discoveredBy string, timestamp time.Time) error {
	for _, propMut := range properties {
		value := PropertyValue{
			Value:        propMut.Value,
			ResolvedAt:   timestamp,
			ResolvedBy:   discoveredBy,
			Confidence:   propMut.Confidence,
			NeedsConfirm: propMut.NeedsConfirm,
		}

		if err := g.SetProperty(propMut.NodeID, propMut.PropertyName, value); err != nil {
			logger.WarnCF("graph", "Failed to set property from mutation",
				map[string]any{
					"node_id":  propMut.NodeID,
					"property": propMut.PropertyName,
					"error":    err.Error(),
				})
			continue
		}

		logger.DebugCF("graph", "Set property from mutation",
			map[string]any{
				"node_id":       propMut.NodeID,
				"property":      propMut.PropertyName,
				"discovered_by": discoveredBy,
			})
	}

	return nil
}

// NewNodeMutation creates a new node mutation
func NewNodeMutation(id string, entityType EntityType, label string, properties map[string]interface{}) *NodeMutation {
	return &NodeMutation{
		ID:         id,
		EntityType: entityType,
		Label:      label,
		Properties: properties,
		Confirmed:  false,
	}
}

// NewEdgeMutation creates a new edge mutation
func NewEdgeMutation(id, from, to string, relationType RelationType) *EdgeMutation {
	return &EdgeMutation{
		ID:           id,
		From:         from,
		To:           to,
		RelationType: relationType,
		Properties:   make(map[string]interface{}),
	}
}

// NewPropertyMutation creates a new property mutation
func NewPropertyMutation(nodeID, propertyName string, value interface{}, confidence float64) *PropertyMutation {
	return &PropertyMutation{
		NodeID:       nodeID,
		PropertyName: propertyName,
		Value:        value,
		Confidence:   confidence,
		NeedsConfirm: confidence < 1.0,
	}
}

// BatchMutation creates a batch mutation from multiple mutation types
func BatchMutation(discoveredBy string, nodes []*NodeMutation, edges []*EdgeMutation, properties []*PropertyMutation) *GraphMutation {
	return &GraphMutation{
		Type:         MutationBatch,
		Nodes:        nodes,
		Edges:        edges,
		Properties:   properties,
		DiscoveredBy: discoveredBy,
		Timestamp:    time.Now(),
	}
}

// MarkUnknownProperties marks properties as unknown based on entity definition
func (g *Graph) MarkUnknownProperties(nodeID string, registry *EntityRegistry) error {
	node, err := g.GetNode(nodeID)
	if err != nil {
		return err
	}

	discoverableProps, err := registry.GetDiscoverableProperties(node.EntityType)
	if err != nil {
		return err
	}

	// Mark all discoverable properties that aren't already known as unknown
	for _, prop := range discoverableProps {
		if _, known := g.GetProperty(nodeID, prop); !known {
			if err := g.MarkPropertyUnknown(nodeID, prop); err != nil {
				logger.WarnCF("graph", "Failed to mark property unknown",
					map[string]any{
						"node_id":  nodeID,
						"property": prop,
						"error":    err.Error(),
					})
			}
		}
	}

	return nil
}

// ApplyMutationBatch applies multiple mutations atomically
func (g *Graph) ApplyMutationBatch(mutations []*GraphMutation) error {
	for _, mutation := range mutations {
		if err := g.ApplyMutation(mutation); err != nil {
			return fmt.Errorf("failed to apply mutation: %w", err)
		}
	}

	logger.InfoCF("graph", "Applied mutation batch",
		map[string]any{
			"mutation_count": len(mutations),
			"nodes":          g.NodeCount(),
			"edges":          g.EdgeCount(),
		})

	return nil
}
