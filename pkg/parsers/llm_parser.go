package parsers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ResistanceIsUseless/picoclaw/pkg/artifacts"
	"github.com/ResistanceIsUseless/picoclaw/pkg/blackboard"
	"github.com/ResistanceIsUseless/picoclaw/pkg/logger"
	"github.com/ResistanceIsUseless/picoclaw/pkg/providers"
)

// LLMParser uses an LLM to parse raw tool output into structured artifacts
// This is Layer 2 compression - falls back when structural parsers aren't available
type LLMParser struct {
	provider providers.LLMProvider
	model    string
}

// NewLLMParser creates a new LLM-based parser
func NewLLMParser(provider providers.LLMProvider, model string) *LLMParser {
	if model == "" {
		model = provider.GetDefaultModel()
	}
	return &LLMParser{
		provider: provider,
		model:    model,
	}
}

// ParseOutput uses LLM to extract structured data from raw tool output
func (p *LLMParser) ParseOutput(ctx context.Context, toolName, toolDescription string, rawOutput []byte, expectedArtifactType string, phase string) (blackboard.Artifact, error) {
	// Truncate output if too large (keep first 50KB)
	const maxOutputSize = 50000
	outputStr := string(rawOutput)
	if len(outputStr) > maxOutputSize {
		logger.WarnCF("llm_parser", "Truncating large output",
			map[string]any{
				"tool":          toolName,
				"original_size": len(outputStr),
				"truncated_to":  maxOutputSize,
			})
		outputStr = outputStr[:maxOutputSize] + "\n\n[... output truncated ...]"
	}

	// Build extraction prompt
	prompt := p.buildExtractionPrompt(toolName, toolDescription, outputStr, expectedArtifactType)

	messages := []providers.Message{
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// Call LLM for parsing (no tools needed)
	options := map[string]any{
		"temperature": 0.1, // Low temperature for consistent parsing
		"max_tokens":  4096,
	}

	logger.DebugCF("llm_parser", "Calling LLM to parse tool output",
		map[string]any{
			"tool":          toolName,
			"output_size":   len(rawOutput),
			"artifact_type": expectedArtifactType,
		})

	response, err := p.provider.Chat(ctx, messages, nil, p.model, options)
	if err != nil {
		return nil, fmt.Errorf("LLM parsing failed: %w", err)
	}

	// Extract JSON from response
	artifact, err := p.extractArtifact(response.Content, expectedArtifactType, toolName, phase)
	if err != nil {
		logger.WarnCF("llm_parser", "Failed to extract artifact from LLM response",
			map[string]any{
				"tool":  toolName,
				"error": err.Error(),
			})
		// Fall back to raw output artifact
		return p.createRawOutputArtifact(toolName, rawOutput, phase), nil
	}

	logger.InfoCF("llm_parser", "Successfully parsed tool output with LLM",
		map[string]any{
			"tool":          toolName,
			"artifact_type": expectedArtifactType,
		})

	return artifact, nil
}

// buildExtractionPrompt creates a prompt for LLM to extract structured data
func (p *LLMParser) buildExtractionPrompt(toolName, toolDescription, output, artifactType string) string {
	var sb strings.Builder

	sb.WriteString("# Tool Output Parsing Task\n\n")
	sb.WriteString("You are a security assessment agent parsing tool output into structured artifacts.\n\n")

	sb.WriteString(fmt.Sprintf("**Tool**: %s\n", toolName))
	sb.WriteString(fmt.Sprintf("**Description**: %s\n\n", toolDescription))

	sb.WriteString("## Raw Tool Output\n\n")
	sb.WriteString("```\n")
	sb.WriteString(output)
	sb.WriteString("\n```\n\n")

	sb.WriteString(fmt.Sprintf("## Your Task: Extract %s\n\n", artifactType))

	// Provide artifact-specific instructions
	switch artifactType {
	case "SubdomainList":
		sb.WriteString("Extract all discovered subdomains from the output.\n\n")
		sb.WriteString("Return a JSON object with this structure:\n")
		sb.WriteString("```json\n")
		sb.WriteString("{\n")
		sb.WriteString("  \"subdomains\": [\"sub1.example.com\", \"sub2.example.com\"],\n")
		sb.WriteString("  \"count\": 2\n")
		sb.WriteString("}\n")
		sb.WriteString("```\n")

	case "PortScanResult":
		sb.WriteString("Extract all open ports, services, and OS information from the output.\n\n")
		sb.WriteString("Return a JSON object with this structure:\n")
		sb.WriteString("```json\n")
		sb.WriteString("{\n")
		sb.WriteString("  \"host\": \"192.168.1.1\",\n")
		sb.WriteString("  \"open_ports\": [{\"port\": 80, \"protocol\": \"tcp\", \"service\": \"http\", \"version\": \"nginx 1.18\"}],\n")
		sb.WriteString("  \"os\": \"Linux 5.4\"\n")
		sb.WriteString("}\n")
		sb.WriteString("```\n")

	case "WebFindings":
		sb.WriteString("Extract all web endpoints, technologies, and interesting findings from the output.\n\n")
		sb.WriteString("Return a JSON object with this structure:\n")
		sb.WriteString("```json\n")
		sb.WriteString("{\n")
		sb.WriteString("  \"endpoints\": [{\"url\": \"https://example.com\", \"status_code\": 200, \"title\": \"Example\"}],\n")
		sb.WriteString("  \"technologies\": [\"nginx\", \"php\"],\n")
		sb.WriteString("  \"interesting_headers\": [\"X-Powered-By: PHP/7.4\"]\n")
		sb.WriteString("}\n")
		sb.WriteString("```\n")

	case "VulnerabilityList":
		sb.WriteString("Extract all vulnerabilities found from the output.\n\n")
		sb.WriteString("Return a JSON object with this structure:\n")
		sb.WriteString("```json\n")
		sb.WriteString("{\n")
		sb.WriteString("  \"vulnerabilities\": [\n")
		sb.WriteString("    {\n")
		sb.WriteString("      \"title\": \"SQL Injection\",\n")
		sb.WriteString("      \"severity\": \"high\",\n")
		sb.WriteString("      \"url\": \"https://example.com/login\",\n")
		sb.WriteString("      \"description\": \"SQL injection in login form\"\n")
		sb.WriteString("    }\n")
		sb.WriteString("  ]\n")
		sb.WriteString("}\n")
		sb.WriteString("```\n")

	case "ToolOutput":
		sb.WriteString("Extract key findings from the output and summarize them.\n\n")
		sb.WriteString("Return a JSON object with this structure:\n")
		sb.WriteString("```json\n")
		sb.WriteString("{\n")
		sb.WriteString("  \"summary\": \"Brief summary of findings\",\n")
		sb.WriteString("  \"key_findings\": [\"finding 1\", \"finding 2\"],\n")
		sb.WriteString("  \"interesting_data\": {\"any\": \"relevant data\"}\n")
		sb.WriteString("}\n")
		sb.WriteString("```\n")

	default:
		// Generic extraction for unknown artifact types
		sb.WriteString("Extract the most important information from the output.\n\n")
		sb.WriteString("Return a JSON object with key findings.\n")
	}

	sb.WriteString("\n## Important Rules\n\n")
	sb.WriteString("1. **Only extract data that actually appears in the output** - never invent or guess\n")
	sb.WriteString("2. **Return ONLY valid JSON** - no markdown, no explanations, just the JSON object\n")
	sb.WriteString("3. **If output is empty or contains no useful data**, return an empty structure\n")
	sb.WriteString("4. **Preserve exact values** from output (URLs, IPs, version numbers, etc.)\n")
	sb.WriteString("5. **Focus on security-relevant information** - skip noise and irrelevant data\n\n")

	return sb.String()
}

// extractArtifact parses LLM response and creates typed artifact
func (p *LLMParser) extractArtifact(llmResponse, artifactType, toolName, phase string) (blackboard.Artifact, error) {
	// Extract JSON from response (might be wrapped in markdown)
	jsonStr := extractJSON(llmResponse)
	if jsonStr == "" {
		return nil, fmt.Errorf("no valid JSON found in LLM response")
	}

	// Parse based on expected artifact type
	switch artifactType {
	case "SubdomainList":
		return p.parseSubdomainList(jsonStr, toolName, phase)
	case "PortScanResult":
		return p.parsePortScanResult(jsonStr, toolName, phase)
	case "WebFindings":
		return p.parseWebFindings(jsonStr, toolName, phase)
	case "VulnerabilityList":
		return p.parseVulnerabilityList(jsonStr, toolName, phase)
	case "ToolOutput":
		return p.parseToolOutput(jsonStr, toolName, phase)
	default:
		return p.parseToolOutput(jsonStr, toolName, phase)
	}
}

// parseSubdomainList parses LLM-extracted subdomain data
func (p *LLMParser) parseSubdomainList(jsonStr, toolName, phase string) (blackboard.Artifact, error) {
	var data struct {
		Subdomains []string `json:"subdomains"`
		Count      int      `json:"count"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("failed to parse SubdomainList JSON: %w", err)
	}

	// Extract domain from first subdomain
	baseDomain := ""
	if len(data.Subdomains) > 0 {
		parts := strings.Split(data.Subdomains[0], ".")
		if len(parts) >= 2 {
			baseDomain = strings.Join(parts[len(parts)-2:], ".")
		}
	}

	// Convert string list to Subdomain objects
	subdomains := make([]artifacts.Subdomain, 0, len(data.Subdomains))
	for _, name := range data.Subdomains {
		subdomains = append(subdomains, artifacts.Subdomain{
			Name:         name,
			IPs:          []string{},
			Source:       toolName,
			Verified:     false,
			DiscoveredAt: time.Now(),
		})
	}

	sources := map[string]int{toolName: len(data.Subdomains)}

	return &artifacts.SubdomainList{
		Metadata: blackboard.ArtifactMetadata{
			Type:      "SubdomainList",
			Phase:     phase,
			Domain:    "web",
			CreatedAt: time.Now(),
		},
		BaseDomain: baseDomain,
		Subdomains: subdomains,
		Sources:    sources,
		Total:      len(subdomains),
	}, nil
}

// parsePortScanResult parses LLM-extracted port scan data
func (p *LLMParser) parsePortScanResult(jsonStr, toolName, phase string) (blackboard.Artifact, error) {
	var data map[string]interface{}

	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("failed to parse PortScanResult JSON: %w", err)
	}

	// For complex structures like PortScanResult, use ToolOutput
	// The structural parser (nmap_parser) handles the full complexity
	return artifacts.NewToolOutput(toolName, data, phase), nil
}

// parseWebFindings parses LLM-extracted web findings
func (p *LLMParser) parseWebFindings(jsonStr, toolName, phase string) (blackboard.Artifact, error) {
	var data map[string]interface{}

	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("failed to parse WebFindings JSON: %w", err)
	}

	// For complex structures like WebFindings, use ToolOutput
	// The structural parser (httpx_parser) handles the full complexity
	return artifacts.NewToolOutput(toolName, data, phase), nil
}

// parseVulnerabilityList parses LLM-extracted vulnerability data
func (p *LLMParser) parseVulnerabilityList(jsonStr, toolName, phase string) (blackboard.Artifact, error) {
	var data map[string]interface{}

	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("failed to parse VulnerabilityList JSON: %w", err)
	}

	// For complex structures like VulnerabilityList, use ToolOutput
	// The structural parser (nuclei_parser) handles the full complexity
	return artifacts.NewToolOutput(toolName, data, phase), nil
}

// parseToolOutput parses generic tool output
func (p *LLMParser) parseToolOutput(jsonStr, toolName, phase string) (blackboard.Artifact, error) {
	var data map[string]interface{}

	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, fmt.Errorf("failed to parse ToolOutput JSON: %w", err)
	}

	return artifacts.NewToolOutput(toolName, data, phase), nil
}

// createRawOutputArtifact creates a fallback artifact with raw output
func (p *LLMParser) createRawOutputArtifact(toolName string, rawOutput []byte, phase string) blackboard.Artifact {
	// Truncate if too large
	const maxRawSize = 10000
	outputStr := string(rawOutput)
	if len(outputStr) > maxRawSize {
		outputStr = outputStr[:maxRawSize] + "\n[... truncated ...]"
	}

	data := map[string]interface{}{
		"raw_output": outputStr,
		"note":       "Parser failed - storing raw output",
	}

	return artifacts.NewToolOutput(toolName, data, phase)
}

// extractJSON extracts JSON from markdown-wrapped text or plain response
func extractJSON(text string) string {
	// Try to find JSON in markdown code blocks
	if strings.Contains(text, "```json") {
		start := strings.Index(text, "```json")
		if start != -1 {
			start += 7 // Skip past ```json
			end := strings.Index(text[start:], "```")
			if end != -1 {
				return strings.TrimSpace(text[start : start+end])
			}
		}
	}

	// Try to find JSON in plain code blocks
	if strings.Contains(text, "```") {
		start := strings.Index(text, "```")
		if start != -1 {
			start += 3 // Skip past ```
			// Skip optional language tag
			if newline := strings.Index(text[start:], "\n"); newline != -1 {
				start += newline + 1
			}
			end := strings.Index(text[start:], "```")
			if end != -1 {
				return strings.TrimSpace(text[start : start+end])
			}
		}
	}

	// Try to find JSON object/array in plain text
	if idx := strings.Index(text, "{"); idx != -1 {
		// Find matching closing brace
		braceCount := 0
		for i := idx; i < len(text); i++ {
			if text[i] == '{' {
				braceCount++
			} else if text[i] == '}' {
				braceCount--
				if braceCount == 0 {
					return strings.TrimSpace(text[idx : i+1])
				}
			}
		}
	}

	if idx := strings.Index(text, "["); idx != -1 {
		// Find matching closing bracket
		bracketCount := 0
		for i := idx; i < len(text); i++ {
			if text[i] == '[' {
				bracketCount++
			} else if text[i] == ']' {
				bracketCount--
				if bracketCount == 0 {
					return strings.TrimSpace(text[idx : i+1])
				}
			}
		}
	}

	return ""
}
