package filters

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ResistanceIsUseless/picoclaw/pkg/logger"
)

// OutputFilter defines the interface for filtering tool output
type OutputFilter interface {
	// Filter processes tool output and returns a summary for the LLM
	// Returns: summary for LLM context, path to full output, error
	Filter(toolName string, output []byte) (summary string, fullPath string, err error)

	// ShouldFilter determines if filtering should be applied based on output size
	ShouldFilter(outputSize int) bool

	// Name returns the filter name for logging
	Name() string
}

// FilteredResult represents the result of applying a filter
type FilteredResult struct {
	Summary      string    `json:"summary"`
	FullPath     string    `json:"full_path"`
	OriginalSize int       `json:"original_size"`
	FilteredSize int       `json:"filtered_size"`
	Timestamp    time.Time `json:"timestamp"`
	ToolName     string    `json:"tool_name"`
	FilterName   string    `json:"filter_name"`
}

// BaseFilter provides common functionality for all filters
type BaseFilter struct {
	name          string
	threshold     int    // Size threshold in bytes
	outputDir     string // Directory to store full outputs
	maxSummaryLen int    // Maximum summary length
}

// NewBaseFilter creates a new base filter with default settings
func NewBaseFilter(name string, outputDir string) *BaseFilter {
	return &BaseFilter{
		name:          name,
		threshold:     10240, // 10KB default
		outputDir:     outputDir,
		maxSummaryLen: 2000, // 2KB summary max
	}
}

func (bf *BaseFilter) Name() string {
	return bf.name
}

func (bf *BaseFilter) ShouldFilter(outputSize int) bool {
	return outputSize > bf.threshold
}

// SaveFullOutput saves the complete output to a file and returns the path
func (bf *BaseFilter) SaveFullOutput(toolName string, output []byte) (string, error) {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(bf.outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s.txt", toolName, timestamp)
	fullPath := filepath.Join(bf.outputDir, filename)

	// Write output to file
	if err := os.WriteFile(fullPath, output, 0644); err != nil {
		return "", fmt.Errorf("failed to save full output: %w", err)
	}

	logger.DebugCF("filter", "Saved full output",
		map[string]any{
			"tool":   toolName,
			"path":   fullPath,
			"size":   len(output),
			"filter": bf.name,
		})

	return fullPath, nil
}

// TruncateSummary ensures summary doesn't exceed max length
func (bf *BaseFilter) TruncateSummary(summary string) string {
	if len(summary) <= bf.maxSummaryLen {
		return summary
	}
	return summary[:bf.maxSummaryLen] + "\n... (truncated)"
}

// FilterRegistry manages output filters for different tool types
type FilterRegistry struct {
	filters       map[string]OutputFilter
	defaultFilter OutputFilter
	outputDir     string
}

// NewFilterRegistry creates a new filter registry
func NewFilterRegistry(outputDir string) *FilterRegistry {
	return &FilterRegistry{
		filters:   make(map[string]OutputFilter),
		outputDir: outputDir,
	}
}

// GetOutputDir returns the output directory for filtered results
func (fr *FilterRegistry) GetOutputDir() string {
	return fr.outputDir
}

// Register adds a filter to the registry
func (fr *FilterRegistry) Register(toolPattern string, filter OutputFilter) {
	fr.filters[toolPattern] = filter
	logger.DebugCF("filter", "Registered filter",
		map[string]any{
			"pattern": toolPattern,
			"filter":  filter.Name(),
		})
}

// RegisterDefault sets the fallback filter for tools without a specific filter.
func (fr *FilterRegistry) RegisterDefault(filter OutputFilter) {
	fr.defaultFilter = filter
	logger.DebugCF("filter", "Registered default filter",
		map[string]any{
			"filter": filter.Name(),
		})
}

// Get returns a filter for the given tool name
func (fr *FilterRegistry) Get(toolName string) (OutputFilter, bool) {
	filter, exists := fr.filters[toolName]
	if exists {
		return filter, true
	}
	if fr.defaultFilter != nil {
		return fr.defaultFilter, true
	}
	return filter, exists
}

// ApplyFilter applies filtering if a filter exists and threshold is met
func (fr *FilterRegistry) ApplyFilter(toolName string, output []byte) (string, error) {
	filter, exists := fr.Get(toolName)
	if !exists {
		// No filter registered, return original output
		return string(output), nil
	}

	if !filter.ShouldFilter(len(output)) {
		// Below threshold, return original
		logger.DebugCF("filter", "Output below threshold, not filtering",
			map[string]any{
				"tool": toolName,
				"size": len(output),
			})
		return string(output), nil
	}

	// Apply filter
	summary, fullPath, err := filter.Filter(toolName, output)
	if err != nil {
		logger.ErrorCF("filter", "Failed to apply filter",
			map[string]any{
				"tool":  toolName,
				"error": err.Error(),
			})
		return string(output), err
	}

	logger.InfoCF("filter", "Applied output filter",
		map[string]any{
			"tool":          toolName,
			"filter":        filter.Name(),
			"original_size": len(output),
			"summary_size":  len(summary),
			"full_path":     fullPath,
		})

	// Return summary with reference to full output
	result := fmt.Sprintf("%s\n\n[Full output saved to: %s]", summary, fullPath)
	return result, nil
}

// SaveFilterMetadata saves metadata about the filtering operation
func (fr *FilterRegistry) SaveFilterMetadata(result *FilteredResult) error {
	metadataPath := filepath.Join(fr.outputDir, "filter_metadata.jsonl")

	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	f, err := os.OpenFile(metadataPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open metadata file: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}
