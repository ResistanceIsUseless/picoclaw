package filters

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// FuzzingFilter filters fuzzing output to extract anomalies and vulnerabilities
type FuzzingFilter struct {
	*BaseFilter
	minSeverity      string
	baselineSize     int
	baselineTiming   int
	maxExamplesPerType int
}

// NewFuzzingFilter creates a new fuzzing output filter
func NewFuzzingFilter(outputDir string) *FuzzingFilter {
	return &FuzzingFilter{
		BaseFilter:         NewBaseFilter("fuzzing", outputDir),
		minSeverity:        "MEDIUM",
		maxExamplesPerType: 5,
	}
}

// FuzzResult represents a single fuzzing result
type FuzzResult struct {
	URL          string `json:"url"`
	Method       string `json:"method"`
	Payload      string `json:"payload"`
	StatusCode   int    `json:"status_code"`
	ResponseSize int    `json:"response_size"`
	ResponseTime int    `json:"response_time_ms"`
	ErrorType    string `json:"error_type"`
	Snippet      string `json:"snippet"`
}

// FilteredFuzz represents the filtered fuzzing results
type FilteredFuzz struct {
	TotalRequests      int                  `json:"total_requests"`
	StatusCodeDist     map[int]int          `json:"status_code_distribution"`
	UniqueErrors       []ErrorPattern       `json:"unique_errors"`
	AnomalousResponses []FuzzResult         `json:"anomalous_responses"`
	PotentialVulns     []VulnFinding        `json:"potential_vulns"`
	RecommendedFocus   []string             `json:"recommended_focus"`
	Summary            string               `json:"summary"`
}

// ErrorPattern represents a unique error pattern
type ErrorPattern struct {
	Type        string   `json:"type"`
	Count       int      `json:"count"`
	Examples    []string `json:"examples"`
	Severity    string   `json:"severity"`
	Description string   `json:"description"`
}

// VulnFinding represents a potential vulnerability
type VulnFinding struct {
	Type        string       `json:"type"`
	Severity    string       `json:"severity"`
	Affected    []FuzzResult `json:"affected"`
	Description string       `json:"description"`
	Exploitation string      `json:"exploitation"`
}

func (ff *FuzzingFilter) Filter(toolName string, output []byte) (string, string, error) {
	// Save full output
	fullPath, err := ff.SaveFullOutput(toolName, output)
	if err != nil {
		return "", "", err
	}

	// Parse fuzzing output
	filtered := ff.analyzeFuzzingOutput(output)

	// Generate summary
	summary := ff.generateSummary(filtered)

	return summary, fullPath, nil
}

func (ff *FuzzingFilter) analyzeFuzzingOutput(output []byte) *FilteredFuzz {
	result := &FilteredFuzz{
		StatusCodeDist:     make(map[int]int),
		UniqueErrors:       make([]ErrorPattern, 0),
		AnomalousResponses: make([]FuzzResult, 0),
		PotentialVulns:     make([]VulnFinding, 0),
		RecommendedFocus:   make([]string, 0),
	}

	results := make([]FuzzResult, 0)
	errorPatterns := make(map[string]*ErrorPattern)

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		// Try JSON format first (ffuf, wfuzz with JSON output)
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(line), &jsonData); err == nil {
			if fuzzResult := ff.parseJSONResult(jsonData); fuzzResult != nil {
				results = append(results, *fuzzResult)
			}
			continue
		}

		// Parse text format (common fuzzing tool output)
		if fuzzResult := ff.parseTextResult(line); fuzzResult != nil {
			results = append(results, *fuzzResult)
		}
	}

	result.TotalRequests = len(results)

	// Calculate status code distribution
	for _, r := range results {
		result.StatusCodeDist[r.StatusCode]++
	}

	// Calculate baseline stats for anomaly detection
	if len(results) > 0 {
		ff.calculateBaseline(results)
	}

	// Analyze results
	for _, r := range results {
		// Check for error patterns
		if errorType := ff.detectErrorType(r); errorType != "" {
			if _, exists := errorPatterns[errorType]; !exists {
				errorPatterns[errorType] = &ErrorPattern{
					Type:     errorType,
					Examples: make([]string, 0),
				}
			}
			errorPatterns[errorType].Count++
			if len(errorPatterns[errorType].Examples) < ff.maxExamplesPerType {
				errorPatterns[errorType].Examples = append(errorPatterns[errorType].Examples, r.URL)
			}
		}

		// Check for anomalies
		if ff.isAnomalous(r) {
			result.AnomalousResponses = append(result.AnomalousResponses, r)
		}
	}

	// Convert error patterns to slice
	for _, pattern := range errorPatterns {
		pattern.Severity = ff.assessErrorSeverity(pattern.Type)
		pattern.Description = ff.describeError(pattern.Type)
		result.UniqueErrors = append(result.UniqueErrors, *pattern)
	}

	// Detect potential vulnerabilities
	result.PotentialVulns = ff.detectVulnerabilities(results, errorPatterns)

	// Generate recommendations
	result.RecommendedFocus = ff.generateRecommendations(result)

	return result
}

func (ff *FuzzingFilter) parseJSONResult(data map[string]interface{}) *FuzzResult {
	result := &FuzzResult{}

	if url, ok := data["url"].(string); ok {
		result.URL = url
	} else if input, ok := data["input"].(string); ok {
		result.URL = input
	}

	if method, ok := data["method"].(string); ok {
		result.Method = method
	}

	if payload, ok := data["payload"].(string); ok {
		result.Payload = payload
	}

	if status, ok := data["status"].(float64); ok {
		result.StatusCode = int(status)
	} else if status, ok := data["status_code"].(float64); ok {
		result.StatusCode = int(status)
	}

	if size, ok := data["length"].(float64); ok {
		result.ResponseSize = int(size)
	} else if size, ok := data["size"].(float64); ok {
		result.ResponseSize = int(size)
	}

	if time, ok := data["time"].(float64); ok {
		result.ResponseTime = int(time)
	}

	return result
}

func (ff *FuzzingFilter) parseTextResult(line string) *FuzzResult {
	// Common fuzzing tool output: [STATUS] URL [SIZE] [TIME]
	// Example: "[200] http://target.com/test [1234] [150ms]"

	urlRegex := regexp.MustCompile(`https?://[^\s]+`)
	statusRegex := regexp.MustCompile(`\[(\d{3})\]`)
	sizeRegex := regexp.MustCompile(`\[(\d+)\s*[Bb]\]`)

	urlMatch := urlRegex.FindString(line)
	if urlMatch == "" {
		return nil
	}

	result := &FuzzResult{
		URL: urlMatch,
	}

	if statusMatch := statusRegex.FindStringSubmatch(line); len(statusMatch) > 1 {
		var status int
		fmt.Sscanf(statusMatch[1], "%d", &status)
		result.StatusCode = status
	}

	if sizeMatch := sizeRegex.FindStringSubmatch(line); len(sizeMatch) > 1 {
		var size int
		fmt.Sscanf(sizeMatch[1], "%d", &size)
		result.ResponseSize = size
	}

	return result
}

func (ff *FuzzingFilter) calculateBaseline(results []FuzzResult) {
	sizes := make([]int, 0)
	timings := make([]int, 0)

	for _, r := range results {
		if r.StatusCode == 200 || r.StatusCode == 404 {
			sizes = append(sizes, r.ResponseSize)
			if r.ResponseTime > 0 {
				timings = append(timings, r.ResponseTime)
			}
		}
	}

	if len(sizes) > 0 {
		sort.Ints(sizes)
		ff.baselineSize = sizes[len(sizes)/2] // Median
	}

	if len(timings) > 0 {
		sort.Ints(timings)
		ff.baselineTiming = timings[len(timings)/2] // Median
	}
}

func (ff *FuzzingFilter) detectErrorType(r FuzzResult) string {
	snippet := strings.ToLower(r.Snippet)

	errorSignatures := map[string][]string{
		"SQL_INJECTION": {
			"sql syntax",
			"mysql",
			"postgresql",
			"ora-",
			"sqlite",
			"syntax error",
			"quoted string not properly terminated",
			"unclosed quotation mark",
		},
		"XSS": {
			"<script",
			"javascript:",
			"onerror=",
			"onclick=",
		},
		"PATH_TRAVERSAL": {
			"directory traversal",
			"path traversal",
			"../",
			"..\\",
		},
		"COMMAND_INJECTION": {
			"sh:",
			"bash:",
			"command not found",
			"/bin/",
		},
		"XXE": {
			"xml",
			"entity",
			"doctype",
		},
		"SSRF": {
			"connection refused",
			"connection timeout",
			"dns",
		},
		"INFO_DISCLOSURE": {
			"stack trace",
			"exception",
			"error in",
			"warning:",
			"debug",
		},
	}

	for errorType, signatures := range errorSignatures {
		for _, sig := range signatures {
			if strings.Contains(snippet, sig) {
				return errorType
			}
		}
	}

	return ""
}

func (ff *FuzzingFilter) isAnomalous(r FuzzResult) bool {
	// Unusual status codes
	if r.StatusCode == 500 || r.StatusCode == 503 {
		return true
	}

	// Size anomaly (significantly different from baseline)
	if ff.baselineSize > 0 {
		sizeDiff := abs(r.ResponseSize - ff.baselineSize)
		if float64(sizeDiff)/float64(ff.baselineSize) > 0.5 { // 50% difference
			return true
		}
	}

	// Timing anomaly
	if ff.baselineTiming > 0 && r.ResponseTime > 0 {
		if r.ResponseTime > ff.baselineTiming*3 { // 3x slower
			return true
		}
	}

	return false
}

func (ff *FuzzingFilter) assessErrorSeverity(errorType string) string {
	severityMap := map[string]string{
		"SQL_INJECTION":     "CRITICAL",
		"COMMAND_INJECTION": "CRITICAL",
		"XXE":               "HIGH",
		"XSS":               "HIGH",
		"SSRF":              "HIGH",
		"PATH_TRAVERSAL":    "HIGH",
		"INFO_DISCLOSURE":   "MEDIUM",
	}

	if severity, exists := severityMap[errorType]; exists {
		return severity
	}
	return "LOW"
}

func (ff *FuzzingFilter) describeError(errorType string) string {
	descriptions := map[string]string{
		"SQL_INJECTION":     "SQL error messages indicate potential injection vulnerability",
		"COMMAND_INJECTION": "Command execution indicators suggest OS command injection",
		"XXE":               "XML parsing errors may indicate XXE vulnerability",
		"XSS":               "Script tags reflected in response indicate XSS vulnerability",
		"SSRF":              "Internal connection attempts suggest SSRF vulnerability",
		"PATH_TRAVERSAL":    "File system access patterns indicate path traversal",
		"INFO_DISCLOSURE":   "Stack traces and debug info leak sensitive information",
	}

	if desc, exists := descriptions[errorType]; exists {
		return desc
	}
	return "Unclassified error pattern"
}

func (ff *FuzzingFilter) detectVulnerabilities(results []FuzzResult, patterns map[string]*ErrorPattern) []VulnFinding {
	vulns := make([]VulnFinding, 0)

	for _, pattern := range patterns {
		if pattern.Count == 0 {
			continue
		}

		// Only report significant findings
		if pattern.Severity == "CRITICAL" || pattern.Severity == "HIGH" || pattern.Count > 5 {
			vuln := VulnFinding{
				Type:        pattern.Type,
				Severity:    pattern.Severity,
				Description: pattern.Description,
				Affected:    make([]FuzzResult, 0),
			}

			// Find affected requests
			for _, r := range results {
				if ff.detectErrorType(r) == pattern.Type {
					vuln.Affected = append(vuln.Affected, r)
					if len(vuln.Affected) >= ff.maxExamplesPerType {
						break
					}
				}
			}

			vulns = append(vulns, vuln)
		}
	}

	return vulns
}

func (ff *FuzzingFilter) generateRecommendations(filtered *FilteredFuzz) []string {
	recommendations := make([]string, 0)

	// Based on vulnerabilities
	for _, vuln := range filtered.PotentialVulns {
		switch vuln.Type {
		case "SQL_INJECTION":
			recommendations = append(recommendations, "Perform manual SQL injection testing with sqlmap")
		case "XSS":
			recommendations = append(recommendations, "Test XSS payloads with browser automation")
		case "COMMAND_INJECTION":
			recommendations = append(recommendations, "Attempt command injection with various payloads")
		case "PATH_TRAVERSAL":
			recommendations = append(recommendations, "Enumerate file system access with path traversal")
		}
	}

	// Based on status codes
	if count403, exists := filtered.StatusCodeDist[403]; exists && count403 > 10 {
		recommendations = append(recommendations, fmt.Sprintf("Focus on %d 403 Forbidden responses for potential auth bypass", count403))
	}

	if count500, exists := filtered.StatusCodeDist[500]; exists && count500 > 5 {
		recommendations = append(recommendations, fmt.Sprintf("Investigate %d 500 errors for information disclosure", count500))
	}

	// Based on anomalies
	if len(filtered.AnomalousResponses) > 0 {
		recommendations = append(recommendations, "Manually review anomalous responses for unique vulnerabilities")
	}

	return recommendations
}

func (ff *FuzzingFilter) generateSummary(filtered *FilteredFuzz) string {
	var summary strings.Builder

	summary.WriteString(fmt.Sprintf("Fuzzing Results Summary:\n"))
	summary.WriteString(fmt.Sprintf("- Total requests: %d\n", filtered.TotalRequests))

	// Status code distribution
	if len(filtered.StatusCodeDist) > 0 {
		summary.WriteString("\nStatus Code Distribution:\n")
		// Sort status codes for consistent output
		codes := make([]int, 0, len(filtered.StatusCodeDist))
		for code := range filtered.StatusCodeDist {
			codes = append(codes, code)
		}
		sort.Ints(codes)
		for _, code := range codes {
			count := filtered.StatusCodeDist[code]
			summary.WriteString(fmt.Sprintf("  - %d: %d requests\n", code, count))
		}
	}

	// Unique errors
	if len(filtered.UniqueErrors) > 0 {
		summary.WriteString(fmt.Sprintf("\n⚠️  Unique Error Patterns (%d):\n", len(filtered.UniqueErrors)))
		for i, pattern := range filtered.UniqueErrors {
			if i >= 10 {
				summary.WriteString(fmt.Sprintf("  ... and %d more\n", len(filtered.UniqueErrors)-10))
				break
			}
			summary.WriteString(fmt.Sprintf("  - [%s] %s: %d occurrences\n", pattern.Severity, pattern.Type, pattern.Count))
			if len(pattern.Examples) > 0 {
				summary.WriteString(fmt.Sprintf("    Examples: %s\n", strings.Join(pattern.Examples[:min(3, len(pattern.Examples))], ", ")))
			}
		}
	}

	// Potential vulnerabilities
	if len(filtered.PotentialVulns) > 0 {
		summary.WriteString(fmt.Sprintf("\n🎯 Potential Vulnerabilities (%d):\n", len(filtered.PotentialVulns)))
		for i, vuln := range filtered.PotentialVulns {
			if i >= 10 {
				summary.WriteString(fmt.Sprintf("  ... and %d more\n", len(filtered.PotentialVulns)-10))
				break
			}
			summary.WriteString(fmt.Sprintf("  - [%s] %s: %d affected endpoints\n", vuln.Severity, vuln.Type, len(vuln.Affected)))
			summary.WriteString(fmt.Sprintf("    %s\n", vuln.Description))
		}
	}

	// Anomalous responses
	if len(filtered.AnomalousResponses) > 0 {
		summary.WriteString(fmt.Sprintf("\n🔍 Anomalous Responses (%d):\n", len(filtered.AnomalousResponses)))
		for i, resp := range filtered.AnomalousResponses {
			if i >= 5 {
				summary.WriteString(fmt.Sprintf("  ... and %d more\n", len(filtered.AnomalousResponses)-5))
				break
			}
			summary.WriteString(fmt.Sprintf("  - %s [%d] (size: %d)\n", resp.URL, resp.StatusCode, resp.ResponseSize))
		}
	}

	// Recommendations
	if len(filtered.RecommendedFocus) > 0 {
		summary.WriteString("\n🎯 Recommended Next Steps:\n")
		for _, rec := range filtered.RecommendedFocus {
			summary.WriteString(fmt.Sprintf("  - %s\n", rec))
		}
	}

	return ff.TruncateSummary(summary.String())
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
