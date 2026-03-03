package filters

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// CodeAnalysisFilter filters static code analysis results
type CodeAnalysisFilter struct {
	*BaseFilter
	severityThreshold string
	maxFindingsPerType int
	deduplicateBy     string
}

// NewCodeAnalysisFilter creates a new code analysis output filter
func NewCodeAnalysisFilter(outputDir string) *CodeAnalysisFilter {
	return &CodeAnalysisFilter{
		BaseFilter:         NewBaseFilter("code_analysis", outputDir),
		severityThreshold:  "MEDIUM",
		maxFindingsPerType: 10,
		deduplicateBy:      "pattern",
	}
}

// CodeFinding represents a single code analysis finding
type CodeFinding struct {
	Tool        string   `json:"tool"`
	File        string   `json:"file"`
	Line        int      `json:"line"`
	Column      int      `json:"column"`
	RuleID      string   `json:"rule_id"`
	Severity    string   `json:"severity"`
	Category    string   `json:"category"`
	Message     string   `json:"message"`
	CodeSnippet string   `json:"code_snippet"`
	CWE         []string `json:"cwe"`
	OWASP       []string `json:"owasp"`
}

// FilteredCodeAnalysis represents the filtered code analysis results
type FilteredCodeAnalysis struct {
	TotalFindings      int                      `json:"total_findings"`
	CriticalVulns      []CodeFinding            `json:"critical_vulns"`
	HighVulns          []CodeFinding            `json:"high_vulns"`
	UniquePatterns     map[string][]CodeFinding `json:"unique_patterns"`
	ExploitablePaths   []ExploitChain           `json:"exploitable_paths"`
	QuickWins          []CodeFinding            `json:"quick_wins"`
	SeverityDist       map[string]int           `json:"severity_distribution"`
	CategoryDist       map[string]int           `json:"category_distribution"`
	Summary            string                   `json:"summary"`
}

// ExploitChain represents a source-to-sink vulnerability path
type ExploitChain struct {
	Type        string        `json:"type"`
	Source      CodeFinding   `json:"source"`
	Sinks       []CodeFinding `json:"sinks"`
	Severity    string        `json:"severity"`
	Description string        `json:"description"`
}

func (caf *CodeAnalysisFilter) Filter(toolName string, output []byte) (string, string, error) {
	// Save full output
	fullPath, err := caf.SaveFullOutput(toolName, output)
	if err != nil {
		return "", "", err
	}

	// Parse code analysis output
	filtered := caf.analyzeCodeOutput(output)

	// Generate summary
	summary := caf.generateSummary(filtered)

	return summary, fullPath, nil
}

func (caf *CodeAnalysisFilter) analyzeCodeOutput(output []byte) *FilteredCodeAnalysis {
	result := &FilteredCodeAnalysis{
		CriticalVulns:  make([]CodeFinding, 0),
		HighVulns:      make([]CodeFinding, 0),
		UniquePatterns: make(map[string][]CodeFinding),
		ExploitablePaths: make([]ExploitChain, 0),
		QuickWins:      make([]CodeFinding, 0),
		SeverityDist:   make(map[string]int),
		CategoryDist:   make(map[string]int),
	}

	findings := make([]CodeFinding, 0)

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		// Try JSON format (semgrep, bandit, gosec JSON output)
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(line), &jsonData); err == nil {
			if finding := caf.parseJSONFinding(jsonData); finding != nil {
				findings = append(findings, *finding)
			}
			continue
		}

		// Try SARIF format
		if strings.Contains(line, "\"$schema\"") && strings.Contains(line, "sarif") {
			sarifFindings := caf.parseSARIF([]byte(line))
			findings = append(findings, sarifFindings...)
			continue
		}

		// Parse text format (grep-like output)
		if finding := caf.parseTextFinding(line); finding != nil {
			findings = append(findings, *finding)
		}
	}

	result.TotalFindings = len(findings)

	// Categorize findings
	for _, finding := range findings {
		// Count by severity
		result.SeverityDist[finding.Severity]++

		// Count by category
		result.CategoryDist[finding.Category]++

		// Separate by severity
		switch finding.Severity {
		case "CRITICAL":
			result.CriticalVulns = append(result.CriticalVulns, finding)
		case "HIGH":
			result.HighVulns = append(result.HighVulns, finding)
		}

		// Group by pattern
		pattern := finding.RuleID
		if pattern == "" {
			pattern = finding.Category
		}
		result.UniquePatterns[pattern] = append(result.UniquePatterns[pattern], finding)

		// Identify quick wins (easy to exploit)
		if caf.isQuickWin(finding) {
			result.QuickWins = append(result.QuickWins, finding)
		}
	}

	// Find exploit chains (source to sink)
	result.ExploitablePaths = caf.findExploitChains(findings)

	return result
}

func (caf *CodeAnalysisFilter) parseJSONFinding(data map[string]interface{}) *CodeFinding {
	finding := &CodeFinding{}

	// Semgrep format
	if check, ok := data["check_id"].(string); ok {
		finding.RuleID = check
	} else if ruleID, ok := data["rule_id"].(string); ok {
		finding.RuleID = ruleID
	}

	if path, ok := data["path"].(string); ok {
		finding.File = path
	}

	if start, ok := data["start"].(map[string]interface{}); ok {
		if line, ok := start["line"].(float64); ok {
			finding.Line = int(line)
		}
		if col, ok := start["col"].(float64); ok {
			finding.Column = int(col)
		}
	}

	if extra, ok := data["extra"].(map[string]interface{}); ok {
		if severity, ok := extra["severity"].(string); ok {
			finding.Severity = strings.ToUpper(severity)
		}
		if message, ok := extra["message"].(string); ok {
			finding.Message = message
		}
		if metadata, ok := extra["metadata"].(map[string]interface{}); ok {
			if cwe, ok := metadata["cwe"].([]interface{}); ok {
				finding.CWE = make([]string, len(cwe))
				for i, c := range cwe {
					finding.CWE[i] = fmt.Sprintf("%v", c)
				}
			}
		}
	}

	// Detect category from rule ID or message
	finding.Category = caf.detectCategory(finding)

	return finding
}

func (caf *CodeAnalysisFilter) parseSARIF(data []byte) []CodeFinding {
	// SARIF is complex - simplified parsing
	findings := make([]CodeFinding, 0)

	var sarif map[string]interface{}
	if err := json.Unmarshal(data, &sarif); err != nil {
		return findings
	}

	runs, ok := sarif["runs"].([]interface{})
	if !ok {
		return findings
	}

	for _, run := range runs {
		runMap, ok := run.(map[string]interface{})
		if !ok {
			continue
		}

		results, ok := runMap["results"].([]interface{})
		if !ok {
			continue
		}

		for _, result := range results {
			resultMap, ok := result.(map[string]interface{})
			if !ok {
				continue
			}

			finding := &CodeFinding{}

			if ruleID, ok := resultMap["ruleId"].(string); ok {
				finding.RuleID = ruleID
			}

			if message, ok := resultMap["message"].(map[string]interface{}); ok {
				if text, ok := message["text"].(string); ok {
					finding.Message = text
				}
			}

			findings = append(findings, *finding)
		}
	}

	return findings
}

func (caf *CodeAnalysisFilter) parseTextFinding(line string) *CodeFinding {
	// Format: file:line:column: message
	// Or: file:line: [SEVERITY] message

	parts := strings.SplitN(line, ":", 4)
	if len(parts) < 3 {
		return nil
	}

	finding := &CodeFinding{
		File: parts[0],
	}

	var lineNum int
	fmt.Sscanf(parts[1], "%d", &lineNum)
	finding.Line = lineNum

	if len(parts) >= 4 {
		var col int
		fmt.Sscanf(parts[2], "%d", &col)
		finding.Column = col
		finding.Message = strings.TrimSpace(parts[3])
	} else {
		finding.Message = strings.TrimSpace(parts[2])
	}

	// Extract severity from message
	finding.Severity = caf.extractSeverity(finding.Message)
	finding.Category = caf.detectCategory(finding)

	return finding
}

func (caf *CodeAnalysisFilter) extractSeverity(message string) string {
	msgLower := strings.ToLower(message)
	if strings.Contains(msgLower, "critical") {
		return "CRITICAL"
	}
	if strings.Contains(msgLower, "high") {
		return "HIGH"
	}
	if strings.Contains(msgLower, "medium") {
		return "MEDIUM"
	}
	if strings.Contains(msgLower, "low") {
		return "LOW"
	}
	return "INFO"
}

func (caf *CodeAnalysisFilter) detectCategory(finding *CodeFinding) string {
	text := strings.ToLower(finding.RuleID + " " + finding.Message)

	categories := map[string][]string{
		"SQL_INJECTION": {"sql", "injection", "sqli"},
		"XSS": {"xss", "cross-site scripting", "html injection"},
		"COMMAND_INJECTION": {"command injection", "shell injection", "exec"},
		"PATH_TRAVERSAL": {"path traversal", "directory traversal", "../"},
		"AUTHENTICATION": {"auth", "authentication", "credential", "password"},
		"AUTHORIZATION": {"authz", "authorization", "access control", "privilege"},
		"CRYPTO": {"crypto", "encryption", "hash", "random"},
		"DESERIALIZATION": {"deserialize", "unserialize", "pickle"},
		"XXE": {"xxe", "xml external entity"},
		"SSRF": {"ssrf", "server-side request forgery"},
		"HARDCODED_SECRET": {"hardcoded", "secret", "credential", "api key"},
		"INSECURE_RANDOM": {"insecure random", "weak random", "predictable"},
	}

	for category, keywords := range categories {
		for _, keyword := range keywords {
			if strings.Contains(text, keyword) {
				return category
			}
		}
	}

	return "MISCELLANEOUS"
}

func (caf *CodeAnalysisFilter) isQuickWin(finding CodeFinding) bool {
	quickWinCategories := []string{
		"HARDCODED_SECRET",
		"SQL_INJECTION",
		"COMMAND_INJECTION",
		"PATH_TRAVERSAL",
	}

	for _, cat := range quickWinCategories {
		if finding.Category == cat && (finding.Severity == "CRITICAL" || finding.Severity == "HIGH") {
			return true
		}
	}

	return false
}

func (caf *CodeAnalysisFilter) findExploitChains(findings []CodeFinding) []ExploitChain {
	chains := make([]ExploitChain, 0)

	// Group by file to find potential chains
	byFile := make(map[string][]CodeFinding)
	for _, finding := range findings {
		byFile[finding.File] = append(byFile[finding.File], finding)
	}

	// Look for source-to-sink patterns
	for file, fileFindings := range byFile {
		sources := make([]CodeFinding, 0)
		sinks := make([]CodeFinding, 0)

		for _, finding := range fileFindings {
			if caf.isSource(finding) {
				sources = append(sources, finding)
			}
			if caf.isSink(finding) {
				sinks = append(sinks, finding)
			}
		}

		// If we have both sources and sinks in the same file, it's a potential chain
		if len(sources) > 0 && len(sinks) > 0 {
			chain := ExploitChain{
				Type:        "USER_INPUT_TO_DANGEROUS_SINK",
				Source:      sources[0],
				Sinks:       sinks,
				Severity:    "HIGH",
				Description: fmt.Sprintf("User input in %s flows to dangerous operations", file),
			}
			chains = append(chains, chain)
		}
	}

	return chains
}

func (caf *CodeAnalysisFilter) isSource(finding CodeFinding) bool {
	sources := []string{"user input", "request", "param", "query", "form", "cookie", "header"}
	text := strings.ToLower(finding.Message + " " + finding.RuleID)
	for _, source := range sources {
		if strings.Contains(text, source) {
			return true
		}
	}
	return false
}

func (caf *CodeAnalysisFilter) isSink(finding CodeFinding) bool {
	return finding.Category == "SQL_INJECTION" ||
		finding.Category == "COMMAND_INJECTION" ||
		finding.Category == "XSS" ||
		finding.Category == "PATH_TRAVERSAL"
}

func (caf *CodeAnalysisFilter) generateSummary(filtered *FilteredCodeAnalysis) string {
	var summary strings.Builder

	summary.WriteString(fmt.Sprintf("Code Analysis Results Summary:\n"))
	summary.WriteString(fmt.Sprintf("- Total findings: %d\n", filtered.TotalFindings))

	// Severity distribution
	if len(filtered.SeverityDist) > 0 {
		summary.WriteString("\nSeverity Distribution:\n")
		severities := []string{"CRITICAL", "HIGH", "MEDIUM", "LOW", "INFO"}
		for _, sev := range severities {
			if count, exists := filtered.SeverityDist[sev]; exists {
				summary.WriteString(fmt.Sprintf("  - %s: %d\n", sev, count))
			}
		}
	}

	// Critical vulnerabilities
	if len(filtered.CriticalVulns) > 0 {
		summary.WriteString(fmt.Sprintf("\n🚨 Critical Vulnerabilities (%d):\n", len(filtered.CriticalVulns)))
		for i, vuln := range filtered.CriticalVulns {
			if i >= 10 {
				summary.WriteString(fmt.Sprintf("  ... and %d more\n", len(filtered.CriticalVulns)-10))
				break
			}
			summary.WriteString(fmt.Sprintf("  - [%s] %s:%d - %s\n", vuln.Category, vuln.File, vuln.Line, vuln.Message))
		}
	}

	// High vulnerabilities
	if len(filtered.HighVulns) > 0 {
		summary.WriteString(fmt.Sprintf("\n⚠️  High Severity Vulnerabilities (%d):\n", len(filtered.HighVulns)))
		for i, vuln := range filtered.HighVulns {
			if i >= 10 {
				summary.WriteString(fmt.Sprintf("  ... and %d more\n", len(filtered.HighVulns)-10))
				break
			}
			summary.WriteString(fmt.Sprintf("  - [%s] %s:%d - %s\n", vuln.Category, vuln.File, vuln.Line, vuln.Message))
		}
	}

	// Unique patterns (deduplicated)
	if len(filtered.UniquePatterns) > 0 {
		summary.WriteString(fmt.Sprintf("\nUnique Vulnerability Patterns (%d):\n", len(filtered.UniquePatterns)))
		// Sort patterns by count
		type patternCount struct {
			pattern string
			count   int
		}
		patterns := make([]patternCount, 0)
		for pattern, findings := range filtered.UniquePatterns {
			patterns = append(patterns, patternCount{pattern, len(findings)})
		}
		sort.Slice(patterns, func(i, j int) bool {
			return patterns[i].count > patterns[j].count
		})

		for i, pc := range patterns {
			if i >= 15 {
				summary.WriteString(fmt.Sprintf("  ... and %d more patterns\n", len(patterns)-15))
				break
			}
			summary.WriteString(fmt.Sprintf("  - %s: %d occurrences\n", pc.pattern, pc.count))
		}
	}

	// Exploit chains
	if len(filtered.ExploitablePaths) > 0 {
		summary.WriteString(fmt.Sprintf("\n🎯 Exploitable Paths (%d):\n", len(filtered.ExploitablePaths)))
		for i, chain := range filtered.ExploitablePaths {
			if i >= 5 {
				summary.WriteString(fmt.Sprintf("  ... and %d more chains\n", len(filtered.ExploitablePaths)-5))
				break
			}
			summary.WriteString(fmt.Sprintf("  - [%s] %s\n", chain.Severity, chain.Description))
			summary.WriteString(fmt.Sprintf("    Source: %s:%d\n", chain.Source.File, chain.Source.Line))
			summary.WriteString(fmt.Sprintf("    Sinks: %d dangerous operations\n", len(chain.Sinks)))
		}
	}

	// Quick wins
	if len(filtered.QuickWins) > 0 {
		summary.WriteString(fmt.Sprintf("\n💰 Quick Wins - Easy to Exploit (%d):\n", len(filtered.QuickWins)))
		for i, win := range filtered.QuickWins {
			if i >= 10 {
				summary.WriteString(fmt.Sprintf("  ... and %d more\n", len(filtered.QuickWins)-10))
				break
			}
			summary.WriteString(fmt.Sprintf("  - [%s] %s:%d\n", win.Category, win.File, win.Line))
		}
	}

	// Recommendations
	summary.WriteString("\n🎯 Recommended Next Steps:\n")
	if len(filtered.CriticalVulns) > 0 {
		summary.WriteString("  - Immediately address critical vulnerabilities\n")
	}
	if len(filtered.ExploitablePaths) > 0 {
		summary.WriteString("  - Prioritize fixing exploitable source-to-sink paths\n")
	}
	if len(filtered.QuickWins) > 0 {
		summary.WriteString("  - Focus on quick wins for immediate security improvements\n")
	}
	summary.WriteString("  - Review deduplicated patterns to fix entire vulnerability classes\n")

	return caf.TruncateSummary(summary.String())
}
