package graph

import (
	"fmt"
	"time"

	"github.com/ResistanceIsUseless/picoclaw/pkg/artifacts"
	"github.com/ResistanceIsUseless/picoclaw/pkg/blackboard"
)

// ExtractMutation extracts graph mutations from artifacts
// This converts structured artifact data into graph nodes, edges, and properties
func ExtractMutation(artifact blackboard.Artifact) (*GraphMutation, error) {
	mutation := &GraphMutation{
		Type:      MutationBatch,
		Nodes:     make([]*NodeMutation, 0),
		Edges:     make([]*EdgeMutation, 0),
		Properties: make([]*PropertyMutation, 0),
		Timestamp: time.Now(),
	}

	// Type switch on artifact to extract relevant entities
	switch a := artifact.(type) {
	case *artifacts.SubdomainList:
		return extractFromSubdomainList(a, mutation)

	case *artifacts.OperatorTarget:
		return extractFromOperatorTarget(a, mutation)

	// Add more artifact types as needed
	default:
		// Unknown artifact type - return empty mutation
		return mutation, nil
	}
}

// extractFromSubdomainList extracts subdomain entities and relationships
func extractFromSubdomainList(list *artifacts.SubdomainList, mutation *GraphMutation) (*GraphMutation, error) {
	// Create/update base domain node
	baseDomainID := fmt.Sprintf("domain:%s", list.BaseDomain)
	mutation.Nodes = append(mutation.Nodes, &NodeMutation{
		ID:         baseDomainID,
		EntityType: EntityDomain,
		Label:      list.BaseDomain,
		Properties: map[string]interface{}{
			"name": list.BaseDomain,
		},
		Confirmed: true,
	})

	// Create subdomain nodes and link to base domain
	for _, subdomain := range list.Subdomains {
		subdomainID := fmt.Sprintf("subdomain:%s", subdomain.Name)

		// Create subdomain node
		subdomainProps := map[string]interface{}{
			"name":          subdomain.Name,
			"source":        subdomain.Source,
			"verified":      subdomain.Verified,
			"discovered_at": subdomain.DiscoveredAt,
		}

		// Add IPs if available
		if len(subdomain.IPs) > 0 {
			subdomainProps["ips"] = subdomain.IPs
		}

		mutation.Nodes = append(mutation.Nodes, &NodeMutation{
			ID:         subdomainID,
			EntityType: EntitySubdomain,
			Label:      subdomain.Name,
			Properties: subdomainProps,
			Confirmed:  subdomain.Verified,
		})

		// Create edge: subdomain -> base domain
		mutation.Edges = append(mutation.Edges, &EdgeMutation{
			ID:           fmt.Sprintf("edge:%s->%s", subdomainID, baseDomainID),
			From:         subdomainID,
			To:           baseDomainID,
			RelationType: RelationSubdomainOf,
			Properties: map[string]interface{}{
				"source": subdomain.Source,
			},
		})

		// Mark properties as discovered
		mutation.Properties = append(mutation.Properties, &PropertyMutation{
			NodeID:       subdomainID,
			PropertyName: "discoverable_properties",
			Value: map[string]interface{}{
				"known":   []string{"name", "source", "verified", "discovered_at"},
				"unknown": []string{"ip_addresses", "ports", "services", "tech_stack"},
			},
			Confidence:   1.0,
			NeedsConfirm: false,
		})

		// If IPs are present, create IP nodes
		if len(subdomain.IPs) > 0 {
			for _, ip := range subdomain.IPs {
				ipID := fmt.Sprintf("ip:%s", ip)
				mutation.Nodes = append(mutation.Nodes, &NodeMutation{
					ID:         ipID,
					EntityType: EntityIP,
					Label:      ip,
					Properties: map[string]interface{}{
						"address": ip,
					},
					Confirmed: true,
				})

				// Edge: subdomain -> IP
				mutation.Edges = append(mutation.Edges, &EdgeMutation{
					ID:           fmt.Sprintf("edge:%s->%s", subdomainID, ipID),
					From:         subdomainID,
					To:           ipID,
					RelationType: RelationResolvesTo,
					Properties:   map[string]interface{}{},
				})
			}
		}
	}

	return mutation, nil
}

// extractFromOperatorTarget extracts the initial target entity
func extractFromOperatorTarget(target *artifacts.OperatorTarget, mutation *GraphMutation) (*GraphMutation, error) {
	targetID := fmt.Sprintf("%s:%s", target.TargetType, target.Target)

	// Determine entity type from target type
	var entityType EntityType
	switch target.TargetType {
	case "web":
		entityType = EntityDomain
	case "network":
		entityType = EntityIP
	default:
		entityType = EntityDomain // Default fallback
	}

	mutation.Nodes = append(mutation.Nodes, &NodeMutation{
		ID:         targetID,
		EntityType: entityType,
		Label:      target.Target,
		Properties: map[string]interface{}{
			"target": target.Target,
		},
		Confirmed: true,
	})

	// Mark initial properties as unknown (to be discovered)
	var unknownProps []string
	switch target.TargetType {
	case "web":
		unknownProps = []string{"subdomains", "infrastructure", "technologies"}
	case "network":
		unknownProps = []string{"hosts", "open_ports", "services"}
	}

	if len(unknownProps) > 0 {
		mutation.Properties = append(mutation.Properties, &PropertyMutation{
			NodeID:       targetID,
			PropertyName: "discoverable_properties",
			Value: map[string]interface{}{
				"known":   []string{"target"},
				"unknown": unknownProps,
			},
			Confidence:   1.0,
			NeedsConfirm: false,
		})
	}

	return mutation, nil
}

// Additional extractors can be added for other artifact types:
// - extractFromPortScanResult() - creates port and service nodes
// - extractFromServiceFingerprint() - adds technology stack info
// - extractFromVulnerabilityList() - creates vulnerability nodes
// - extractFromExploitResult() - marks vulnerabilities as exploitable
