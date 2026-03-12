package filters

import (
	"bufio"
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// GenericTextFilter provides a safe fallback for large outputs from tools
// that do not have a dedicated profile-specific filter.
type GenericTextFilter struct {
	*BaseFilter
}

func NewGenericTextFilter(outputDir string) *GenericTextFilter {
	bf := NewBaseFilter("generic_text", outputDir)
	bf.threshold = 4096
	bf.maxSummaryLen = 1800
	return &GenericTextFilter{BaseFilter: bf}
}

func (gf *GenericTextFilter) Filter(toolName string, output []byte) (string, string, error) {
	fullPath, err := gf.SaveFullOutput(toolName, output)
	if err != nil {
		return "", "", err
	}

	summary := gf.generateSummary(toolName, output, fullPath)
	return summary, fullPath, nil
}

func (gf *GenericTextFilter) generateSummary(toolName string, output []byte, fullPath string) string {
	lines := make([]string, 0, 12)
	keywordHits := make([]string, 0, 12)
	seen := make(map[string]bool)
	statusCounts := make(map[string]int)

	interesting := regexp.MustCompile(`(?i)(error|warning|fail|critical|high|medium|low|open|closed|timeout|refused|denied|found|discovered|vulnerab|expos|admin|login|token|secret|key|password|auth|http|api|graphql|swagger|debug|traceback|exception|cve-)`)
	statusCode := regexp.MustCompile(`\b([1-5][0-9]{2})\b`)

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if len(lines) < 5 {
			lines = append(lines, line)
		}

		if matches := statusCode.FindAllString(line, -1); len(matches) > 0 {
			for _, code := range matches {
				statusCounts[code]++
			}
		}

		if interesting.MatchString(line) && !seen[line] {
			seen[line] = true
			keywordHits = append(keywordHits, line)
			if len(keywordHits) >= 8 {
				break
			}
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Generic output summary for %s\n", toolName)
	fmt.Fprintf(&sb, "- Original size: %d bytes\n", len(output))

	if len(statusCounts) > 0 {
		codes := make([]string, 0, len(statusCounts))
		for code := range statusCounts {
			codes = append(codes, code)
		}
		sort.Strings(codes)
		pairs := make([]string, 0, len(codes))
		for _, code := range codes {
			pairs = append(pairs, fmt.Sprintf("%s=%d", code, statusCounts[code]))
		}
		fmt.Fprintf(&sb, "- Status-like codes: %s\n", strings.Join(pairs, ", "))
	}

	if len(keywordHits) > 0 {
		sb.WriteString("- Interesting lines:\n")
		for _, line := range keywordHits {
			fmt.Fprintf(&sb, "  %s\n", truncateLine(line, 180))
		}
	} else if len(lines) > 0 {
		sb.WriteString("- Leading lines:\n")
		for _, line := range lines {
			fmt.Fprintf(&sb, "  %s\n", truncateLine(line, 180))
		}
	}

	_ = fullPath
	return gf.TruncateSummary(sb.String())
}

func truncateLine(line string, max int) string {
	if len(line) <= max {
		return line
	}
	return line[:max] + "..."
}
