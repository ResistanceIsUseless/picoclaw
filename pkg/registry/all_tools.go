package registry

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RegisterAllTools discovers and registers all available security tools
// This includes: security tools in ~/go/bin, shell commands, and hardcoded tools
func RegisterAllTools(registry *ToolRegistry) error {
	// 1. Register hardcoded security tools with parsers (Tier 0 and Tier 1)
	if err := RegisterSecurityTools(registry); err != nil {
		return fmt.Errorf("failed to register security tools: %w", err)
	}

	// 2. Register shell tool for generic command execution
	if err := RegisterShellTool(registry); err != nil {
		return fmt.Errorf("failed to register shell tool: %w", err)
	}

	// 3. Auto-discover and register all tools from ~/go/bin
	if err := RegisterDiscoveredTools(registry); err != nil {
		// Don't fail if discovery fails - just log warning
		fmt.Printf("Warning: failed to auto-discover tools: %v\n", err)
	}

	return nil
}

// RegisterShellTool registers the shell command execution tool
func RegisterShellTool(registry *ToolRegistry) error {
	shellTool := &ToolDefinition{
		Name:        "shell",
		Description: "Execute shell commands (bash/zsh). Supports pipes, grep, awk, sed, curl, jq, etc. Use for ad-hoc analysis and tool chaining.",
		Tier:        TierAutoApprove, // Tier 1: visible and auto-approved
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "Shell command to execute (e.g., 'curl -s https://example.com | jq .data')",
				},
				"working_dir": map[string]interface{}{
					"type":        "string",
					"description": "Working directory for command execution (optional)",
				},
			},
			"required": []string{"command"},
		},
		OutputType: "raw", // Shell output is unstructured - returns raw text
		Parser:     nil,   // No parsing - raw output
	}

	return registry.Register(shellTool)
}

// RegisterDiscoveredTools auto-discovers and registers tools from common locations
func RegisterDiscoveredTools(registry *ToolRegistry) error {
	// Tool search paths
	searchPaths := []string{
		filepath.Join(os.Getenv("HOME"), "go", "bin"),
		"/opt/homebrew/bin",
		"/usr/local/bin",
	}

	// Known security tool names and their descriptions
	knownTools := map[string]string{
		// Recon tools
		"gau":          "Get All URLs - fetch known URLs from AlienVault, Wayback, Common Crawl",
		"waybackurls":  "Fetch all URLs from Wayback Machine for a domain",
		"assetfinder":  "Find domains and subdomains related to a target",
		"chaos":        "ProjectDiscovery Chaos DNS dataset client",
		"shuffledns":   "Fast subdomain bruteforcer with wildcard detection",
		"puredns":      "Fast domain resolver with wildcard filtering",
		"dnsx":         "Fast DNS toolkit for running various DNS queries",
		"hakrevdns":    "Reverse DNS lookups on CIDR ranges",
		"asnmap":       "Map IP addresses to ASN information",
		"certgraph":    "Map certificate relationships for domain discovery",
		"cdncheck":     "Check if IP/domain is behind a CDN",

		// Web crawling
		"katana":       "Next-generation web crawler with JS rendering",
		"gospider":     "Fast web spider written in Go",
		"hakrawler":    "Fast web crawler for gathering URLs",

		// JavaScript analysis
		"jsluice":      "Extract URLs, paths, secrets from JavaScript",
		"subjs":        "Fetch JavaScript files from URLs/subdomains",

		// Fuzzing and parameter discovery
		"ffuf":         "Fast web fuzzer written in Go",
		"gobuster":     "Directory/file & DNS bruteforcing tool",
		"fuzzparam":    "Parameter fuzzing tool",
		"kxss":         "XSS reflection parameter discovery",
		"qsreplace":    "Replace query string values in URLs",

		// Utilities
		"unfurl":       "Pull out bits of URLs (domain, path, query, etc.)",
		"anew":         "Add new lines to files, skipping duplicates",
		"gf":           "Grep through output using patterns",
		"notify":       "Send notifications to various platforms",
		"cidr2ip":      "Convert CIDR notation to IP addresses",
		"proxycheck":   "Check if hosts are proxying requests",
		"webanalyze":   "Technology detection on websites",
		"webscope":     "Scope analyzer for web recon",
		"subscope":     "Check if subdomains are in scope",

		// Analysis tools
		"pdtm":         "ProjectDiscovery Template Manager",
	}

	discoveredCount := 0
	for _, searchPath := range searchPaths {
		entries, err := os.ReadDir(searchPath)
		if err != nil {
			continue // Path doesn't exist or not accessible
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			toolName := entry.Name()

			// Skip if already registered (hardcoded tools take precedence)
			if _, err := registry.Get(toolName); err == nil {
				continue
			}

			// Only register known security tools
			description, isKnown := knownTools[toolName]
			if !isKnown {
				continue
			}

			// Register as Tier 1 (auto-approve) generic tool
			tool := &ToolDefinition{
				Name:        toolName,
				Description: description,
				Tier:        TierAutoApprove,
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"args": map[string]interface{}{
							"type":        "array",
							"description": fmt.Sprintf("Command-line arguments for %s", toolName),
							"items": map[string]interface{}{
								"type": "string",
							},
						},
					},
				},
				OutputType: "raw",
				Parser:     nil, // Generic tools return raw output
			}

			if err := registry.Register(tool); err != nil {
				fmt.Printf("Warning: failed to register %s: %v\n", toolName, err)
				continue
			}

			discoveredCount++
		}
	}

	fmt.Printf("Auto-discovered %d security tools\n", discoveredCount)
	return nil
}

// GetToolPath finds the full path to a tool executable
func GetToolPath(toolName string) (string, error) {
	// Check common locations
	searchPaths := []string{
		filepath.Join(os.Getenv("HOME"), "go", "bin", toolName),
		filepath.Join("/opt/homebrew/bin", toolName),
		filepath.Join("/usr/local/bin", toolName),
		filepath.Join("/usr/bin", toolName),
	}

	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// Not found in common locations - check PATH
	path, err := exec.LookPath(toolName)
	if err != nil {
		return "", fmt.Errorf("tool %s not found in PATH or common locations", toolName)
	}

	return path, nil
}

// IsShellCommand checks if a tool should be executed via shell
func IsShellCommand(command string) bool {
	// Commands with pipes, redirects, or shell operators need shell execution
	shellIndicators := []string{"|", ">", "<", "&&", "||", ";", "$", "`"}
	for _, indicator := range shellIndicators {
		if strings.Contains(command, indicator) {
			return true
		}
	}
	return false
}
