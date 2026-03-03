package filters

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// CrawlFilter filters web crawling output to extract interesting information
type CrawlFilter struct {
	*BaseFilter
	interestingPatterns []string
	maxURLsPerDomain    int
	sensitivePatterns   []string
}

// NewCrawlFilter creates a new web crawler output filter
func NewCrawlFilter(outputDir string) *CrawlFilter {
	return &CrawlFilter{
		BaseFilter:        NewBaseFilter("web_crawler", outputDir),
		maxURLsPerDomain:  100,
		interestingPatterns: []string{
			// Sensitive files
			`\.git/`,
			`\.env`,
			`\.config`,
			`backup\.(sql|zip|tar|gz)`,
			`dump\.(sql|db)`,
			`\.bak`,
			`\.old`,
			`\.swp`,
			`web\.config`,
			`\.htaccess`,
			// Admin/debug endpoints
			`/admin`,
			`/debug`,
			`/api`,
			`/graphql`,
			`/swagger`,
			`/metrics`,
			`/health`,
			// Interesting parameters
			`[?&](id|user|admin|token|key|secret|password|auth)=`,
		},
		sensitivePatterns: []string{
			`\.git/config`,
			`\.env`,
			`\.aws/`,
			`id_rsa`,
			`\.pem`,
			`\.key`,
			`backup\.sql`,
			`database\.sql`,
		},
	}
}

// FilteredCrawl represents the filtered crawl results
type FilteredCrawl struct {
	TotalURLs            int                 `json:"total_urls"`
	UniqueEndpoints      []string            `json:"unique_endpoints"`
	ParameterNames       []string            `json:"parameter_names"`
	SensitiveFiles       []string            `json:"sensitive_files"`
	UnusualStatusCodes   map[int][]string    `json:"unusual_status_codes"`
	InterestingHeaders   map[string][]string `json:"interesting_headers"`
	TechnologyFingerprints []string          `json:"technology_fingerprints"`
	Summary              string              `json:"summary"`
}

func (cf *CrawlFilter) Filter(toolName string, output []byte) (string, string, error) {
	// Save full output
	fullPath, err := cf.SaveFullOutput(toolName, output)
	if err != nil {
		return "", "", err
	}

	// Parse crawl output (assuming line-separated URLs or JSON)
	filtered := cf.analyzeCrawlOutput(output)

	// Generate summary
	summary := cf.generateSummary(filtered)

	return summary, fullPath, nil
}

func (cf *CrawlFilter) analyzeCrawlOutput(output []byte) *FilteredCrawl {
	result := &FilteredCrawl{
		UniqueEndpoints:    make([]string, 0),
		ParameterNames:     make([]string, 0),
		SensitiveFiles:     make([]string, 0),
		UnusualStatusCodes: make(map[int][]string),
		InterestingHeaders: make(map[string][]string),
		TechnologyFingerprints: make([]string, 0),
	}

	urls := make([]string, 0)
	paramSet := make(map[string]bool)
	endpointSet := make(map[string]bool)

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		// Try to parse as JSON first
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(line), &jsonData); err == nil {
			cf.parseJSONLine(jsonData, &urls, result)
			continue
		}

		// Otherwise treat as plain URL
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		urls = append(urls, line)
	}

	result.TotalURLs = len(urls)

	// Extract unique endpoints and parameters
	for _, urlStr := range urls {
		parsedURL, err := url.Parse(urlStr)
		if err != nil {
			continue
		}

		// Extract endpoint (path without query)
		endpoint := parsedURL.Path
		if endpoint != "" && !endpointSet[endpoint] {
			endpointSet[endpoint] = true
			result.UniqueEndpoints = append(result.UniqueEndpoints, endpoint)
		}

		// Extract parameter names
		for param := range parsedURL.Query() {
			if !paramSet[param] {
				paramSet[param] = true
				result.ParameterNames = append(result.ParameterNames, param)
			}
		}

		// Check for sensitive files
		if cf.isSensitiveURL(urlStr) {
			result.SensitiveFiles = append(result.SensitiveFiles, urlStr)
		}
	}

	return result
}

func (cf *CrawlFilter) parseJSONLine(data map[string]interface{}, urls *[]string, result *FilteredCrawl) {
	// Extract URL if present
	if urlVal, ok := data["url"].(string); ok {
		*urls = append(*urls, urlVal)
	}

	// Extract status code
	if status, ok := data["status"].(float64); ok {
		statusInt := int(status)
		if cf.isUnusualStatus(statusInt) {
			if urlVal, ok := data["url"].(string); ok {
				result.UnusualStatusCodes[statusInt] = append(
					result.UnusualStatusCodes[statusInt],
					urlVal,
				)
			}
		}
	}

	// Extract interesting headers
	if headers, ok := data["headers"].(map[string]interface{}); ok {
		for header, value := range headers {
			if cf.isInterestingHeader(header) {
				valStr := fmt.Sprintf("%v", value)
				result.InterestingHeaders[header] = append(
					result.InterestingHeaders[header],
					valStr,
				)
			}
		}
	}

	// Extract technology fingerprints
	if tech, ok := data["technology"].(string); ok {
		result.TechnologyFingerprints = append(result.TechnologyFingerprints, tech)
	}
}

func (cf *CrawlFilter) isSensitiveURL(urlStr string) bool {
	for _, pattern := range cf.sensitivePatterns {
		if matched, _ := regexp.MatchString(pattern, urlStr); matched {
			return true
		}
	}
	return false
}

func (cf *CrawlFilter) isUnusualStatus(status int) bool {
	// Focus on interesting status codes
	unusual := []int{401, 403, 500, 502, 503}
	for _, s := range unusual {
		if status == s {
			return true
		}
	}
	return false
}

func (cf *CrawlFilter) isInterestingHeader(header string) bool {
	interesting := []string{
		"Server",
		"X-Powered-By",
		"X-AspNet-Version",
		"X-Frame-Options",
		"Content-Security-Policy",
		"Access-Control-Allow-Origin",
		"Strict-Transport-Security",
	}

	headerLower := strings.ToLower(header)
	for _, h := range interesting {
		if strings.ToLower(h) == headerLower {
			return true
		}
	}
	return false
}

func (cf *CrawlFilter) generateSummary(filtered *FilteredCrawl) string {
	var summary strings.Builder

	summary.WriteString(fmt.Sprintf("Web Crawl Results Summary:\n"))
	summary.WriteString(fmt.Sprintf("- Total URLs crawled: %d\n", filtered.TotalURLs))
	summary.WriteString(fmt.Sprintf("- Unique endpoints: %d\n", len(filtered.UniqueEndpoints)))
	summary.WriteString(fmt.Sprintf("- Parameters found: %d\n", len(filtered.ParameterNames)))

	if len(filtered.SensitiveFiles) > 0 {
		summary.WriteString(fmt.Sprintf("\n⚠️  Sensitive Files Found (%d):\n", len(filtered.SensitiveFiles)))
		for i, file := range filtered.SensitiveFiles {
			if i >= 10 {
				summary.WriteString(fmt.Sprintf("  ... and %d more\n", len(filtered.SensitiveFiles)-10))
				break
			}
			summary.WriteString(fmt.Sprintf("  - %s\n", file))
		}
	}

	if len(filtered.ParameterNames) > 0 {
		summary.WriteString(fmt.Sprintf("\nInteresting Parameters (%d):\n", len(filtered.ParameterNames)))
		for i, param := range filtered.ParameterNames {
			if i >= 20 {
				summary.WriteString(fmt.Sprintf("  ... and %d more\n", len(filtered.ParameterNames)-20))
				break
			}
			summary.WriteString(fmt.Sprintf("  - %s\n", param))
		}
	}

	if len(filtered.UnusualStatusCodes) > 0 {
		summary.WriteString("\nUnusual Status Codes:\n")
		for status, urls := range filtered.UnusualStatusCodes {
			summary.WriteString(fmt.Sprintf("  - %d: %d URLs\n", status, len(urls)))
			if len(urls) <= 5 {
				for _, u := range urls {
					summary.WriteString(fmt.Sprintf("    - %s\n", u))
				}
			}
		}
	}

	if len(filtered.InterestingHeaders) > 0 {
		summary.WriteString("\nInteresting Headers:\n")
		for header, values := range filtered.InterestingHeaders {
			uniqueVals := cf.uniqueStrings(values)
			summary.WriteString(fmt.Sprintf("  - %s: %d unique values\n", header, len(uniqueVals)))
			if len(uniqueVals) <= 3 {
				for _, val := range uniqueVals {
					summary.WriteString(fmt.Sprintf("    - %s\n", val))
				}
			}
		}
	}

	if len(filtered.TechnologyFingerprints) > 0 {
		uniqueTech := cf.uniqueStrings(filtered.TechnologyFingerprints)
		summary.WriteString(fmt.Sprintf("\nTechnologies Detected (%d):\n", len(uniqueTech)))
		for i, tech := range uniqueTech {
			if i >= 10 {
				summary.WriteString(fmt.Sprintf("  ... and %d more\n", len(uniqueTech)-10))
				break
			}
			summary.WriteString(fmt.Sprintf("  - %s\n", tech))
		}
	}

	// Recommendations
	if len(filtered.SensitiveFiles) > 0 || len(filtered.UnusualStatusCodes) > 0 {
		summary.WriteString("\n🎯 Recommended Next Steps:\n")
		if len(filtered.SensitiveFiles) > 0 {
			summary.WriteString("  - Investigate sensitive file access\n")
		}
		if urls403, ok := filtered.UnusualStatusCodes[403]; ok && len(urls403) > 0 {
			summary.WriteString("  - Test authentication bypass on 403 endpoints\n")
		}
		if urls500, ok := filtered.UnusualStatusCodes[500]; ok && len(urls500) > 0 {
			summary.WriteString("  - Analyze 500 errors for information disclosure\n")
		}
		if len(filtered.ParameterNames) > 0 {
			summary.WriteString("  - Test parameters for injection vulnerabilities\n")
		}
	}

	return cf.TruncateSummary(summary.String())
}

func (cf *CrawlFilter) uniqueStrings(strs []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)
	for _, s := range strs {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
