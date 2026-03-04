package registry

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/ResistanceIsUseless/picoclaw/pkg/parsers"
)

// RegisterSecurityTools registers common security assessment tools
func RegisterSecurityTools(registry *ToolRegistry) error {
	tools := []*ToolDefinition{
		{
			Name:        "subfinder",
			Description: "Fast subdomain enumeration tool using passive sources",
			Tier:        TierHardwired, // Tier 0: Invisible to model, output as ground truth
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"domain": map[string]interface{}{
						"type":        "string",
						"description": "Target domain to enumerate subdomains for",
					},
				},
				"required": []string{"domain"},
			},
			OutputType: "SubdomainList",
			Parser: func(toolName string, output []byte) (interface{}, error) {
				// Extract domain from execution context - for now use placeholder
				return parsers.ParseSubfinderOutput(toolName, output, "", "recon")
			},
		},
		{
			Name:        "amass",
			Description: "In-depth subdomain enumeration with DNS resolution",
			Tier:        TierHardwired, // Tier 0
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"domain": map[string]interface{}{
						"type":        "string",
						"description": "Target domain to enumerate",
					},
				},
				"required": []string{"domain"},
			},
			OutputType: "SubdomainList",
			Parser: func(toolName string, output []byte) (interface{}, error) {
				return parsers.ParseAmassOutput(toolName, output, "", "recon")
			},
		},
		{
			Name:        "nmap",
			Description: "Network port scanner",
			Tier:        TierAutoApprove, // Tier 1: Visible and auto-approved
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"target": map[string]interface{}{
						"type":        "string",
						"description": "IP address or hostname to scan",
					},
					"ports": map[string]interface{}{
						"type":        "string",
						"description": "Port range (e.g., '1-1000', 'top-100', or specific ports)",
						"default":     "top-1000",
					},
				},
				"required": []string{"target"},
			},
			OutputType: "PortScanResult",
			Parser:     nil, // TODO: Implement nmap parser
		},
		{
			Name:        "httpx",
			Description: "Fast HTTP probe and fingerprinting tool",
			Tier:        TierAutoApprove, // Tier 1
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"targets": map[string]interface{}{
						"type":        "array",
						"description": "List of hosts to probe",
						"items": map[string]interface{}{
							"type": "string",
						},
					},
				},
				"required": []string{"targets"},
			},
			OutputType: "ServiceFingerprint",
			Parser:     nil, // TODO: Implement httpx parser
		},
		{
			Name:        "nuclei",
			Description: "Vulnerability scanner based on templates",
			Tier:        TierAutoApprove, // Tier 1
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"target": map[string]interface{}{
						"type":        "string",
						"description": "Target URL or host to scan",
					},
					"severity": map[string]interface{}{
						"type":        "string",
						"description": "Severity filter (critical, high, medium, low, info)",
						"default":     "critical,high",
					},
				},
				"required": []string{"target"},
			},
			OutputType: "VulnerabilityList",
			Parser:     nil, // TODO: Implement nuclei parser
		},
	}

	for _, tool := range tools {
		if err := registry.Register(tool); err != nil {
			return fmt.Errorf("failed to register %s: %w", tool.Name, err)
		}
	}

	return nil
}

// ExecuteTool executes a security tool and returns raw output
// This is a simple implementation that runs tools via exec
func ExecuteTool(ctx context.Context, toolName string, args map[string]interface{}) ([]byte, error) {
	switch toolName {
	case "subfinder":
		domain, ok := args["domain"].(string)
		if !ok {
			return nil, fmt.Errorf("subfinder requires 'domain' parameter")
		}
		cmd := exec.CommandContext(ctx, "subfinder", "-d", domain, "-silent")
		return cmd.Output()

	case "amass":
		domain, ok := args["domain"].(string)
		if !ok {
			return nil, fmt.Errorf("amass requires 'domain' parameter")
		}
		cmd := exec.CommandContext(ctx, "amass", "enum", "-passive", "-d", domain)
		return cmd.Output()

	case "nmap":
		target, ok := args["target"].(string)
		if !ok {
			return nil, fmt.Errorf("nmap requires 'target' parameter")
		}
		ports := "top-1000"
		if p, ok := args["ports"].(string); ok {
			ports = p
		}
		// Simple nmap scan
		cmd := exec.CommandContext(ctx, "nmap", "-p", ports, target)
		return cmd.Output()

	case "httpx":
		// httpx typically reads from stdin or file
		// For now, return placeholder
		return nil, fmt.Errorf("httpx execution not yet implemented")

	case "nuclei":
		target, ok := args["target"].(string)
		if !ok {
			return nil, fmt.Errorf("nuclei requires 'target' parameter")
		}
		severity := "critical,high"
		if s, ok := args["severity"].(string); ok {
			severity = s
		}
		cmd := exec.CommandContext(ctx, "nuclei", "-u", target, "-severity", severity, "-silent")
		return cmd.Output()

	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}
