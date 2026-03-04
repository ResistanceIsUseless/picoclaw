package agent

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ResistanceIsUseless/picoclaw/pkg/logger"
)

// ToolStatus represents the execution state of a tool in the current phase
type ToolStatus string

const (
	StatusCompleted  ToolStatus = "COMPLETED"
	StatusReady      ToolStatus = "READY"
	StatusBlocked    ToolStatus = "BLOCKED"
	StatusNotStarted ToolStatus = "NOT_STARTED"
	StatusRunning    ToolStatus = "RUNNING"
	StatusFailed     ToolStatus = "FAILED"
)

// ToolCall represents a single tool invocation in the session
type ToolCall struct {
	ID           string
	ToolName     string
	StartTime    time.Time
	EndTime      time.Time
	Status       ToolStatus
	Dependencies []string // IDs of tool calls this depends on
	Result       string   // Summary of tool output
	Error        string   // Error message if failed
}

// DAGState represents the current state of tool execution in a phase
type DAGState struct {
	PhaseName   string
	ToolCalls   []*ToolCall
	AvailableTools []string // Tools that can be called in this phase
	Dependencies   map[string][]string // Tool name -> list of tool names it depends on
}

// NewDAGState creates a new DAG state for a phase
func NewDAGState(phaseName string, availableTools []string, dependencies map[string][]string) *DAGState {
	return &DAGState{
		PhaseName:      phaseName,
		ToolCalls:      make([]*ToolCall, 0),
		AvailableTools: availableTools,
		Dependencies:   dependencies,
	}
}

// AddToolCall records a new tool invocation
func (d *DAGState) AddToolCall(call *ToolCall) {
	d.ToolCalls = append(d.ToolCalls, call)
	logger.DebugCF("dag_state", "Added tool call",
		map[string]any{
			"tool":   call.ToolName,
			"status": call.Status,
		})
}

// UpdateToolCall updates the status of an existing tool call
func (d *DAGState) UpdateToolCall(callID string, status ToolStatus, result string, err error) error {
	for _, call := range d.ToolCalls {
		if call.ID == callID {
			call.Status = status
			call.Result = result
			if err != nil {
				call.Error = err.Error()
			}
			if status == StatusCompleted || status == StatusFailed {
				call.EndTime = time.Now()
			}
			return nil
		}
	}
	return fmt.Errorf("tool call %s not found", callID)
}

// GetToolStatus determines the status of a tool that hasn't been called yet
func (d *DAGState) GetToolStatus(toolName string) ToolStatus {
	// Check if tool has already been called
	for _, call := range d.ToolCalls {
		if call.ToolName == toolName {
			return call.Status
		}
	}

	// Check if dependencies are met
	deps, hasDeps := d.Dependencies[toolName]
	if !hasDeps || len(deps) == 0 {
		// No dependencies, ready to call
		return StatusReady
	}

	// Check if all dependencies are completed
	allDepsCompleted := true
	for _, depTool := range deps {
		depCompleted := false
		for _, call := range d.ToolCalls {
			if call.ToolName == depTool && call.Status == StatusCompleted {
				depCompleted = true
				break
			}
		}
		if !depCompleted {
			allDepsCompleted = false
			break
		}
	}

	if allDepsCompleted {
		return StatusReady
	}

	return StatusBlocked
}

// GetCompletedTools returns a list of successfully completed tool calls
func (d *DAGState) GetCompletedTools() []*ToolCall {
	completed := make([]*ToolCall, 0)
	for _, call := range d.ToolCalls {
		if call.Status == StatusCompleted {
			completed = append(completed, call)
		}
	}
	return completed
}

// GetReadyTools returns a list of tools that can be called now
func (d *DAGState) GetReadyTools() []string {
	ready := make([]string, 0)
	for _, toolName := range d.AvailableTools {
		if d.GetToolStatus(toolName) == StatusReady {
			ready = append(ready, toolName)
		}
	}
	return ready
}

// GetBlockedTools returns tools and their blocking dependencies
func (d *DAGState) GetBlockedTools() map[string][]string {
	blocked := make(map[string][]string)
	for _, toolName := range d.AvailableTools {
		if d.GetToolStatus(toolName) == StatusBlocked {
			// Find which dependencies are incomplete
			incompleteDeps := make([]string, 0)
			if deps, hasDeps := d.Dependencies[toolName]; hasDeps {
				for _, depTool := range deps {
					depCompleted := false
					for _, call := range d.ToolCalls {
						if call.ToolName == depTool && call.Status == StatusCompleted {
							depCompleted = true
							break
						}
					}
					if !depCompleted {
						incompleteDeps = append(incompleteDeps, depTool)
					}
				}
			}
			blocked[toolName] = incompleteDeps
		}
	}
	return blocked
}

// RenderState generates a human-readable state summary for the model prompt
func (d *DAGState) RenderState() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("## Current Phase State: %s\n\n", d.PhaseName))

	// Section 1: Completed tools
	completed := d.GetCompletedTools()
	if len(completed) > 0 {
		sb.WriteString("### COMPLETED ✓\n\n")
		// Sort by completion time (most recent first)
		sort.Slice(completed, func(i, j int) bool {
			return completed[i].EndTime.After(completed[j].EndTime)
		})
		for _, call := range completed {
			duration := call.EndTime.Sub(call.StartTime)
			sb.WriteString(fmt.Sprintf("  **%s** — returned %s (took %s)\n",
				call.ToolName,
				call.EndTime.Format("15:04:05"),
				duration.Round(time.Second)))
			if call.Result != "" {
				// Truncate result to 100 chars for brevity
				result := call.Result
				if len(result) > 100 {
					result = result[:97] + "..."
				}
				sb.WriteString(fmt.Sprintf("    → %s\n", result))
			}
			sb.WriteString("\n")
		}
	}

	// Section 2: Ready tools (can be called now)
	ready := d.GetReadyTools()
	if len(ready) > 0 {
		sb.WriteString("### READY (dependencies met — call these now)\n\n")
		for _, toolName := range ready {
			sb.WriteString(fmt.Sprintf("  **%s**", toolName))
			// Show what this tool depends on (now completed)
			if deps, hasDeps := d.Dependencies[toolName]; hasDeps && len(deps) > 0 {
				sb.WriteString(" [depends on: ")
				for i, dep := range deps {
					if i > 0 {
						sb.WriteString(", ")
					}
					sb.WriteString(dep + " ✓")
				}
				sb.WriteString("]")
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// Section 3: Blocked tools
	blocked := d.GetBlockedTools()
	if len(blocked) > 0 {
		sb.WriteString("### BLOCKED (waiting on dependencies)\n\n")
		for toolName, incompleteDeps := range blocked {
			sb.WriteString(fmt.Sprintf("  **%s** — waiting for: %s\n",
				toolName,
				strings.Join(incompleteDeps, ", ")))
		}
		sb.WriteString("\n")
	}

	// Section 4: Running tools
	running := make([]*ToolCall, 0)
	for _, call := range d.ToolCalls {
		if call.Status == StatusRunning {
			running = append(running, call)
		}
	}
	if len(running) > 0 {
		sb.WriteString("### RUNNING (in progress)\n\n")
		for _, call := range running {
			elapsed := time.Since(call.StartTime)
			sb.WriteString(fmt.Sprintf("  **%s** — started %s (running for %s)\n",
				call.ToolName,
				call.StartTime.Format("15:04:05"),
				elapsed.Round(time.Second)))
		}
		sb.WriteString("\n")
	}

	// Section 5: Failed tools
	failed := make([]*ToolCall, 0)
	for _, call := range d.ToolCalls {
		if call.Status == StatusFailed {
			failed = append(failed, call)
		}
	}
	if len(failed) > 0 {
		sb.WriteString("### FAILED ✗\n\n")
		for _, call := range failed {
			sb.WriteString(fmt.Sprintf("  **%s** — failed at %s\n",
				call.ToolName,
				call.EndTime.Format("15:04:05")))
			if call.Error != "" {
				sb.WriteString(fmt.Sprintf("    Error: %s\n", call.Error))
			}
		}
		sb.WriteString("\n")
	}

	// Section 6: Guidance
	if len(ready) > 0 {
		sb.WriteString("**Next Action**: Call one of the READY tools listed above.\n")
	} else if len(running) > 0 {
		sb.WriteString("**Next Action**: Wait for running tools to complete.\n")
	} else if len(blocked) > 0 {
		sb.WriteString("**Next Action**: Complete dependencies before calling blocked tools.\n")
	} else if len(completed) > 0 {
		sb.WriteString("**Next Action**: Phase may be complete. Review results and call `complete_phase` if objectives met.\n")
	}

	return sb.String()
}

// IsPhaseComplete checks if all required tools have been executed
func (d *DAGState) IsPhaseComplete(requiredTools []string) bool {
	completedTools := make(map[string]bool)
	for _, call := range d.ToolCalls {
		if call.Status == StatusCompleted {
			completedTools[call.ToolName] = true
		}
	}

	for _, required := range requiredTools {
		if !completedTools[required] {
			return false
		}
	}

	return true
}

// GetProgress returns completion percentage (0-100)
func (d *DAGState) GetProgress() float64 {
	if len(d.AvailableTools) == 0 {
		return 0.0
	}

	completed := 0
	for _, toolName := range d.AvailableTools {
		for _, call := range d.ToolCalls {
			if call.ToolName == toolName && call.Status == StatusCompleted {
				completed++
				break
			}
		}
	}

	return (float64(completed) / float64(len(d.AvailableTools))) * 100.0
}

// Clone creates a deep copy of the DAG state
func (d *DAGState) Clone() *DAGState {
	clone := &DAGState{
		PhaseName:      d.PhaseName,
		ToolCalls:      make([]*ToolCall, len(d.ToolCalls)),
		AvailableTools: make([]string, len(d.AvailableTools)),
		Dependencies:   make(map[string][]string),
	}

	copy(clone.AvailableTools, d.AvailableTools)

	for i, call := range d.ToolCalls {
		clone.ToolCalls[i] = &ToolCall{
			ID:           call.ID,
			ToolName:     call.ToolName,
			StartTime:    call.StartTime,
			EndTime:      call.EndTime,
			Status:       call.Status,
			Dependencies: append([]string{}, call.Dependencies...),
			Result:       call.Result,
			Error:        call.Error,
		}
	}

	for tool, deps := range d.Dependencies {
		clone.Dependencies[tool] = append([]string{}, deps...)
	}

	return clone
}

// Snapshot returns a serializable summary of the current state
func (d *DAGState) Snapshot() map[string]interface{} {
	return map[string]interface{}{
		"phase":          d.PhaseName,
		"total_calls":    len(d.ToolCalls),
		"completed":      len(d.GetCompletedTools()),
		"ready":          len(d.GetReadyTools()),
		"blocked":        len(d.GetBlockedTools()),
		"progress_pct":   d.GetProgress(),
		"last_updated":   time.Now(),
	}
}
