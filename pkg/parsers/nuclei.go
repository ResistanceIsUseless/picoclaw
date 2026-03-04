package parsers

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"time"

	"github.com/ResistanceIsUseless/picoclaw/pkg/artifacts"
	"github.com/ResistanceIsUseless/picoclaw/pkg/blackboard"
)

// NucleiResult represents a single nuclei finding in JSON format
type NucleiResult struct {
	TemplateID   string `json:"template-id"`
	TemplatePath string `json:"template-path"`
	Info         struct {
		Name           string            `json:"name"`
		Author         []string          `json:"author"`
		Tags           []string          `json:"tags"`
		Description    string            `json:"description"`
		Reference      []string          `json:"reference"`
		Severity       string            `json:"severity"`
		Metadata       map[string]string `json:"metadata"`
		Classification struct {
			CVE  []string `json:"cve-id"`
			CWE  []string `json:"cwe-id"`
			CVSS struct {
				Score  float64 `json:"cvss-score"`
				Vector string  `json:"cvss-metrics"`
			} `json:"cvss-metrics"`
		} `json:"classification"`
		Remediation string `json:"remediation"`
	} `json:"info"`
	Type          string   `json:"type"`
	Host          string   `json:"host"`
	MatchedAt     string   `json:"matched-at"`
	MatchedLine   string   `json:"matched-line"`
	ExtractedResults []string `json:"extracted-results"`
	IP            string   `json:"ip"`
	Timestamp     string   `json:"timestamp"`
	CurlCommand   string   `json:"curl-command"`
	MatcherStatus bool     `json:"matcher-status"`
	MatcherName   string   `json:"matcher-name"`
}

// ParseNucleiOutput parses nuclei JSON output into VulnerabilityList artifact
// Nuclei outputs one JSON object per line when using -json flag
func ParseNucleiOutput(toolName string, output []byte, phase string) (*artifacts.VulnerabilityList, error) {
	vulnerabilities := make([]artifacts.Vulnerability, 0)
	bySeverity := make(map[string]int)
	byDomain := make(map[string]int)
	confirmed := 0

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var result NucleiResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			// Skip malformed lines
			continue
		}

		// Map nuclei severity to standard severity levels
		severity := normalizeSeverity(result.Info.Severity)

		// Build evidence
		evidence := make([]string, 0)
		if result.MatchedAt != "" {
			evidence = append(evidence, "Matched at: "+result.MatchedAt)
		}
		if result.MatchedLine != "" {
			evidence = append(evidence, "Matched line: "+result.MatchedLine)
		}
		if len(result.ExtractedResults) > 0 {
			evidence = append(evidence, "Extracted: "+strings.Join(result.ExtractedResults, ", "))
		}
		if result.CurlCommand != "" {
			evidence = append(evidence, "Curl: "+result.CurlCommand)
		}

		// Build affected targets
		affected := []string{result.Host}
		if result.IP != "" && result.IP != result.Host {
			affected = append(affected, result.IP)
		}

		// Determine confidence based on matcher status
		confidence := "MEDIUM"
		if result.MatcherStatus {
			confidence = "HIGH"
			if severity == "CRITICAL" || severity == "HIGH" {
				confirmed++
			}
		}

		// Parse timestamp
		timestamp := time.Now()
		if result.Timestamp != "" {
			if t, err := time.Parse(time.RFC3339, result.Timestamp); err == nil {
				timestamp = t
			}
		}

		vuln := artifacts.Vulnerability{
			ID:          result.TemplateID,
			Title:       result.Info.Name,
			Severity:    severity,
			CVE:         result.Info.Classification.CVE,
			CWE:         result.Info.Classification.CWE,
			OWASP:       extractOWASP(result.Info.Tags),
			Description: result.Info.Description,
			Impact:      buildImpact(result.Info.Severity, result.Info.Description),
			Affected:    affected,
			Evidence:    evidence,
			Remediation: result.Info.Remediation,
			References:  result.Info.Reference,
			Confidence:  confidence,
			Domain:      "web",
			DiscoveredAt: timestamp,
			DiscoveredBy: toolName,
		}

		vulnerabilities = append(vulnerabilities, vuln)
		bySeverity[severity]++
		byDomain["web"]++
	}

	return &artifacts.VulnerabilityList{
		Metadata: blackboard.ArtifactMetadata{
			Type:      "VulnerabilityList",
			CreatedAt: time.Now(),
			Phase:     phase,
			Version:   "1.0",
			Domain:    "web",
		},
		Vulnerabilities: vulnerabilities,
		Summary: artifacts.VulnSummary{
			Total:      len(vulnerabilities),
			BySeverity: bySeverity,
			ByDomain:   byDomain,
			Confirmed:  confirmed,
		},
	}, nil
}

// ParseNucleiToWebFindings converts nuclei output to WebFindings format
// This is an alternative that combines findings with endpoint data
func ParseNucleiToWebFindings(toolName string, output []byte, phase string) (*artifacts.WebFindings, error) {
	findings := make([]artifacts.WebFinding, 0)
	endpoints := make([]artifacts.Endpoint, 0)
	endpointMap := make(map[string]bool) // dedupe endpoints

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var result NucleiResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			continue
		}

		// Create endpoint if not seen before
		if !endpointMap[result.Host] {
			endpointMap[result.Host] = true
			endpoints = append(endpoints, artifacts.Endpoint{
				URL:          result.Host,
				Method:       "GET", // nuclei default
				StatusCode:   200,   // assumption
				DiscoveredAt: time.Now(),
				Source:       toolName,
			})
		}

		// Map nuclei template to finding type
		findingType := mapTemplateToType(result.TemplateID, result.Info.Tags)
		severity := normalizeSeverity(result.Info.Severity)

		// Parse timestamp
		timestamp := time.Now()
		if result.Timestamp != "" {
			if t, err := time.Parse(time.RFC3339, result.Timestamp); err == nil {
				timestamp = t
			}
		}

		// Determine confidence
		confidence := "MEDIUM"
		if result.MatcherStatus {
			confidence = "HIGH"
		}

		finding := artifacts.WebFinding{
			ID:          result.TemplateID,
			Type:        findingType,
			Severity:    severity,
			Title:       result.Info.Name,
			Description: result.Info.Description,
			URL:         result.Host,
			Evidence:    result.MatchedAt,
			Impact:      buildImpact(result.Info.Severity, result.Info.Description),
			Remediation: result.Info.Remediation,
			References:  result.Info.Reference,
			Confidence:  confidence,
			CWE:         result.Info.Classification.CWE,
			OWASP:       extractOWASP(result.Info.Tags),
			Tool:        toolName,
			Timestamp:   timestamp,
		}

		findings = append(findings, finding)
	}

	return &artifacts.WebFindings{
		Metadata: blackboard.ArtifactMetadata{
			Type:      "WebFindings",
			CreatedAt: time.Now(),
			Phase:     phase,
			Version:   "1.0",
			Domain:    "web",
		},
		Endpoints:    endpoints,
		Parameters:   make([]artifacts.Parameter, 0),
		Technologies: make([]artifacts.Technology, 0),
		Findings:     findings,
		Crawled: artifacts.CrawlStats{
			TotalURLs:   len(endpoints),
			UniqueHosts: len(endpointMap),
		},
	}, nil
}

// normalizeSeverity maps nuclei severity levels to standard levels
func normalizeSeverity(severity string) string {
	severity = strings.ToUpper(severity)
	switch severity {
	case "CRITICAL":
		return "CRITICAL"
	case "HIGH":
		return "HIGH"
	case "MEDIUM":
		return "MEDIUM"
	case "LOW":
		return "LOW"
	case "INFO", "INFORMATIONAL":
		return "INFO"
	default:
		return "MEDIUM"
	}
}

// extractOWASP extracts OWASP categories from nuclei tags
func extractOWASP(tags []string) []string {
	owasp := make([]string, 0)
	for _, tag := range tags {
		if strings.HasPrefix(strings.ToLower(tag), "owasp") {
			owasp = append(owasp, tag)
		}
	}
	return owasp
}

// buildImpact constructs an impact statement from severity and description
func buildImpact(severity, description string) string {
	severity = strings.ToLower(severity)
	switch severity {
	case "critical":
		return "Critical security issue that requires immediate attention. " + description
	case "high":
		return "High severity security issue that should be addressed promptly. " + description
	case "medium":
		return "Medium severity security issue that should be remediated. " + description
	case "low":
		return "Low severity security issue. Consider remediation as part of routine maintenance. " + description
	default:
		return description
	}
}

// mapTemplateToType maps nuclei template IDs and tags to finding types
func mapTemplateToType(templateID string, tags []string) string {
	templateID = strings.ToLower(templateID)

	// Check tags first
	for _, tag := range tags {
		tag = strings.ToLower(tag)
		if strings.Contains(tag, "xss") {
			return "XSS"
		}
		if strings.Contains(tag, "sqli") || strings.Contains(tag, "sql-injection") {
			return "SQLi"
		}
		if strings.Contains(tag, "ssrf") {
			return "SSRF"
		}
		if strings.Contains(tag, "rce") {
			return "RCE"
		}
		if strings.Contains(tag, "lfi") {
			return "LFI"
		}
		if strings.Contains(tag, "xxe") {
			return "XXE"
		}
		if strings.Contains(tag, "csrf") {
			return "CSRF"
		}
	}

	// Check template ID
	if strings.Contains(templateID, "xss") {
		return "XSS"
	}
	if strings.Contains(templateID, "sqli") || strings.Contains(templateID, "sql") {
		return "SQLi"
	}
	if strings.Contains(templateID, "ssrf") {
		return "SSRF"
	}
	if strings.Contains(templateID, "rce") {
		return "RCE"
	}
	if strings.Contains(templateID, "lfi") {
		return "LFI"
	}
	if strings.Contains(templateID, "cve-") {
		return "CVE"
	}
	if strings.Contains(templateID, "exposure") {
		return "Information Disclosure"
	}
	if strings.Contains(templateID, "misconfig") {
		return "Misconfiguration"
	}
	if strings.Contains(templateID, "takeover") {
		return "Subdomain Takeover"
	}

	return "Security Finding"
}
