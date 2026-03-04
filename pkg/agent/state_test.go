package agent

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewDAGState(t *testing.T) {
	tools := []string{"subfinder", "nmap", "httpx"}
	deps := map[string][]string{
		"nmap":  {"subfinder"},
		"httpx": {"nmap"},
	}

	state := NewDAGState("recon", tools, deps)

	assert.Equal(t, "recon", state.PhaseName)
	assert.Equal(t, 3, len(state.AvailableTools))
	assert.Equal(t, 0, len(state.ToolCalls))
	assert.Equal(t, 2, len(state.Dependencies))
}

func TestGetToolStatus(t *testing.T) {
	tools := []string{"subfinder", "nmap", "httpx"}
	deps := map[string][]string{
		"nmap":  {"subfinder"},
		"httpx": {"nmap"},
	}

	state := NewDAGState("recon", tools, deps)

	// Initially, only subfinder (no deps) should be ready
	assert.Equal(t, StatusReady, state.GetToolStatus("subfinder"))
	assert.Equal(t, StatusBlocked, state.GetToolStatus("nmap"))
	assert.Equal(t, StatusBlocked, state.GetToolStatus("httpx"))

	// Complete subfinder
	call1 := &ToolCall{
		ID:        "call-1",
		ToolName:  "subfinder",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(2 * time.Second),
		Status:    StatusCompleted,
		Result:    "Found 15 subdomains",
	}
	state.AddToolCall(call1)

	// Now nmap should be ready
	assert.Equal(t, StatusCompleted, state.GetToolStatus("subfinder"))
	assert.Equal(t, StatusReady, state.GetToolStatus("nmap"))
	assert.Equal(t, StatusBlocked, state.GetToolStatus("httpx"))

	// Complete nmap
	call2 := &ToolCall{
		ID:        "call-2",
		ToolName:  "nmap",
		StartTime: time.Now(),
		EndTime:   time.Now().Add(5 * time.Second),
		Status:    StatusCompleted,
		Result:    "Scanned 15 hosts, 47 open ports",
	}
	state.AddToolCall(call2)

	// Now httpx should be ready
	assert.Equal(t, StatusReady, state.GetToolStatus("httpx"))
}

func TestGetReadyTools(t *testing.T) {
	tools := []string{"subfinder", "amass", "nmap"}
	deps := map[string][]string{
		"nmap": {"subfinder", "amass"},
	}

	state := NewDAGState("recon", tools, deps)

	// Both subfinder and amass should be ready initially
	ready := state.GetReadyTools()
	assert.Equal(t, 2, len(ready))
	assert.Contains(t, ready, "subfinder")
	assert.Contains(t, ready, "amass")

	// Complete subfinder
	call1 := &ToolCall{
		ID:       "call-1",
		ToolName: "subfinder",
		Status:   StatusCompleted,
	}
	state.AddToolCall(call1)

	// Still only amass is ready (nmap needs both deps)
	ready = state.GetReadyTools()
	assert.Equal(t, 1, len(ready))
	assert.Contains(t, ready, "amass")

	// Complete amass
	call2 := &ToolCall{
		ID:       "call-2",
		ToolName: "amass",
		Status:   StatusCompleted,
	}
	state.AddToolCall(call2)

	// Now nmap is ready
	ready = state.GetReadyTools()
	assert.Equal(t, 1, len(ready))
	assert.Contains(t, ready, "nmap")
}

func TestGetBlockedTools(t *testing.T) {
	tools := []string{"subfinder", "nmap", "httpx", "nuclei"}
	deps := map[string][]string{
		"nmap":   {"subfinder"},
		"httpx":  {"nmap"},
		"nuclei": {"httpx"},
	}

	state := NewDAGState("recon", tools, deps)

	blocked := state.GetBlockedTools()
	assert.Equal(t, 3, len(blocked))
	assert.Contains(t, blocked["nmap"], "subfinder")
	assert.Contains(t, blocked["httpx"], "nmap")
	assert.Contains(t, blocked["nuclei"], "httpx")
}

func TestRenderState(t *testing.T) {
	tools := []string{"subfinder", "nmap", "httpx"}
	deps := map[string][]string{
		"nmap":  {"subfinder"},
		"httpx": {"nmap"},
	}

	state := NewDAGState("recon", tools, deps)

	// Add completed tool
	call1 := &ToolCall{
		ID:        "call-1",
		ToolName:  "subfinder",
		StartTime: time.Now().Add(-10 * time.Second),
		EndTime:   time.Now().Add(-8 * time.Second),
		Status:    StatusCompleted,
		Result:    "Found 15 subdomains",
	}
	state.AddToolCall(call1)

	// Add running tool
	call2 := &ToolCall{
		ID:        "call-2",
		ToolName:  "nmap",
		StartTime: time.Now().Add(-5 * time.Second),
		Status:    StatusRunning,
	}
	state.AddToolCall(call2)

	rendered := state.RenderState()

	// Check that all sections are present
	assert.Contains(t, rendered, "Current Phase State: recon")
	assert.Contains(t, rendered, "COMPLETED ✓")
	assert.Contains(t, rendered, "subfinder")
	assert.Contains(t, rendered, "Found 15 subdomains")
	assert.Contains(t, rendered, "RUNNING")
	assert.Contains(t, rendered, "nmap")
	assert.Contains(t, rendered, "BLOCKED")
	assert.Contains(t, rendered, "httpx")
}

func TestRenderStateWithFailure(t *testing.T) {
	tools := []string{"subfinder", "nmap"}
	deps := map[string][]string{}

	state := NewDAGState("recon", tools, deps)

	// Add failed tool
	call1 := &ToolCall{
		ID:        "call-1",
		ToolName:  "subfinder",
		StartTime: time.Now().Add(-5 * time.Second),
		EndTime:   time.Now().Add(-3 * time.Second),
		Status:    StatusFailed,
		Error:     "DNS resolution timeout",
	}
	state.AddToolCall(call1)

	rendered := state.RenderState()

	assert.Contains(t, rendered, "FAILED ✗")
	assert.Contains(t, rendered, "subfinder")
	assert.Contains(t, rendered, "DNS resolution timeout")
}

func TestIsPhaseComplete(t *testing.T) {
	tools := []string{"subfinder", "nmap", "httpx"}
	state := NewDAGState("recon", tools, nil)

	// Not complete initially
	assert.False(t, state.IsPhaseComplete(tools))

	// Add completed tools
	state.AddToolCall(&ToolCall{
		ID:       "call-1",
		ToolName: "subfinder",
		Status:   StatusCompleted,
	})
	state.AddToolCall(&ToolCall{
		ID:       "call-2",
		ToolName: "nmap",
		Status:   StatusCompleted,
	})

	// Still not complete (httpx missing)
	assert.False(t, state.IsPhaseComplete(tools))

	state.AddToolCall(&ToolCall{
		ID:       "call-3",
		ToolName: "httpx",
		Status:   StatusCompleted,
	})

	// Now complete
	assert.True(t, state.IsPhaseComplete(tools))
}

func TestGetProgress(t *testing.T) {
	tools := []string{"subfinder", "nmap", "httpx", "nuclei"}
	state := NewDAGState("recon", tools, nil)

	// 0% initially
	assert.Equal(t, 0.0, state.GetProgress())

	// 25% after one tool
	state.AddToolCall(&ToolCall{
		ID:       "call-1",
		ToolName: "subfinder",
		Status:   StatusCompleted,
	})
	assert.Equal(t, 25.0, state.GetProgress())

	// 50% after two tools
	state.AddToolCall(&ToolCall{
		ID:       "call-2",
		ToolName: "nmap",
		Status:   StatusCompleted,
	})
	assert.Equal(t, 50.0, state.GetProgress())

	// 75% after three tools
	state.AddToolCall(&ToolCall{
		ID:       "call-3",
		ToolName: "httpx",
		Status:   StatusCompleted,
	})
	assert.Equal(t, 75.0, state.GetProgress())

	// 100% after all tools
	state.AddToolCall(&ToolCall{
		ID:       "call-4",
		ToolName: "nuclei",
		Status:   StatusCompleted,
	})
	assert.Equal(t, 100.0, state.GetProgress())
}

func TestUpdateToolCall(t *testing.T) {
	state := NewDAGState("recon", []string{"subfinder"}, nil)

	call := &ToolCall{
		ID:        "call-1",
		ToolName:  "subfinder",
		StartTime: time.Now(),
		Status:    StatusRunning,
	}
	state.AddToolCall(call)

	// Update to completed
	err := state.UpdateToolCall("call-1", StatusCompleted, "Found 15 subdomains", nil)
	assert.NoError(t, err)
	assert.Equal(t, StatusCompleted, call.Status)
	assert.Equal(t, "Found 15 subdomains", call.Result)
	assert.False(t, call.EndTime.IsZero())

	// Try to update non-existent call
	err = state.UpdateToolCall("call-999", StatusCompleted, "", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestClone(t *testing.T) {
	tools := []string{"subfinder", "nmap"}
	deps := map[string][]string{
		"nmap": {"subfinder"},
	}

	original := NewDAGState("recon", tools, deps)
	original.AddToolCall(&ToolCall{
		ID:       "call-1",
		ToolName: "subfinder",
		Status:   StatusCompleted,
	})

	clone := original.Clone()

	// Verify clone is independent
	assert.Equal(t, original.PhaseName, clone.PhaseName)
	assert.Equal(t, len(original.ToolCalls), len(clone.ToolCalls))
	assert.Equal(t, len(original.AvailableTools), len(clone.AvailableTools))

	// Modify clone shouldn't affect original
	clone.AddToolCall(&ToolCall{
		ID:       "call-2",
		ToolName: "nmap",
		Status:   StatusCompleted,
	})

	assert.Equal(t, 1, len(original.ToolCalls))
	assert.Equal(t, 2, len(clone.ToolCalls))
}

func TestSnapshot(t *testing.T) {
	tools := []string{"subfinder", "nmap", "httpx"}
	state := NewDAGState("recon", tools, nil)

	state.AddToolCall(&ToolCall{
		ID:       "call-1",
		ToolName: "subfinder",
		Status:   StatusCompleted,
	})

	snapshot := state.Snapshot()

	assert.Equal(t, "recon", snapshot["phase"])
	assert.Equal(t, 1, snapshot["total_calls"])
	assert.Equal(t, 1, snapshot["completed"])
	assert.Equal(t, 2, snapshot["ready"])
	assert.NotNil(t, snapshot["last_updated"])
}

func TestRenderStateGuidance(t *testing.T) {
	tools := []string{"subfinder", "nmap"}
	deps := map[string][]string{
		"nmap": {"subfinder"},
	}

	state := NewDAGState("recon", tools, deps)

	// Initially, should suggest calling ready tools
	rendered := state.RenderState()
	assert.Contains(t, rendered, "Call one of the READY tools")

	// With running tool, should suggest waiting
	state.AddToolCall(&ToolCall{
		ID:        "call-1",
		ToolName:  "subfinder",
		StartTime: time.Now(),
		Status:    StatusRunning,
	})
	rendered = state.RenderState()
	assert.Contains(t, rendered, "Wait for running tools")

	// With all completed, should suggest phase completion
	state.UpdateToolCall("call-1", StatusCompleted, "Done", nil)
	state.AddToolCall(&ToolCall{
		ID:       "call-2",
		ToolName: "nmap",
		Status:   StatusCompleted,
	})
	rendered = state.RenderState()
	assert.Contains(t, rendered, "Phase may be complete")
	assert.Contains(t, rendered, "complete_phase")
}

func TestMultipleDependencies(t *testing.T) {
	tools := []string{"subfinder", "amass", "crtsh", "nmap"}
	deps := map[string][]string{
		"nmap": {"subfinder", "amass", "crtsh"}, // nmap needs all three
	}

	state := NewDAGState("recon", tools, deps)

	// nmap should be blocked initially
	assert.Equal(t, StatusBlocked, state.GetToolStatus("nmap"))

	// Complete subfinder and amass
	state.AddToolCall(&ToolCall{ID: "1", ToolName: "subfinder", Status: StatusCompleted})
	state.AddToolCall(&ToolCall{ID: "2", ToolName: "amass", Status: StatusCompleted})

	// nmap still blocked (needs crtsh)
	assert.Equal(t, StatusBlocked, state.GetToolStatus("nmap"))

	// Complete crtsh
	state.AddToolCall(&ToolCall{ID: "3", ToolName: "crtsh", Status: StatusCompleted})

	// Now nmap is ready
	assert.Equal(t, StatusReady, state.GetToolStatus("nmap"))
}

func TestRenderStateTruncation(t *testing.T) {
	tools := []string{"subfinder"}
	state := NewDAGState("recon", tools, nil)

	// Add tool with very long result
	longResult := strings.Repeat("subdomain.example.com\n", 100)
	state.AddToolCall(&ToolCall{
		ID:        "call-1",
		ToolName:  "subfinder",
		StartTime: time.Now().Add(-5 * time.Second),
		EndTime:   time.Now(),
		Status:    StatusCompleted,
		Result:    longResult,
	})

	rendered := state.RenderState()

	// Result should be truncated
	assert.Contains(t, rendered, "...")
	// But not contain the full result
	assert.Less(t, len(rendered), len(longResult))
}
