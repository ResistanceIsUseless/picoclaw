package webui

import (
	"github.com/ResistanceIsUseless/picoclaw/pkg/graph"
)

// EventEmitter interface for sending events to the web UI
type EventEmitter interface {
	EmitPhaseStart(phaseName string, objective string, iteration int)
	EmitPhaseComplete(phaseName string, status string, iteration int, duration string)
	EmitToolExecution(tool string, status string, summary string)
	EmitArtifact(artifactType string, phase string, count int)
	EmitGraphUpdate(mutation *graph.GraphMutation)
	EmitLog(level string, component string, message string, fields map[string]any)
}

// HubEmitter sends events to WebSocket hub
type HubEmitter struct {
	hub *Hub
}

// NewHubEmitter creates a new hub emitter
func NewHubEmitter(hub *Hub) *HubEmitter {
	return &HubEmitter{hub: hub}
}

// EmitPhaseStart emits phase start event
func (e *HubEmitter) EmitPhaseStart(phaseName string, objective string, iteration int) {
	e.hub.Broadcast("phase_start", map[string]interface{}{
		"phase":     phaseName,
		"objective": objective,
		"iteration": iteration,
	})
}

// EmitPhaseComplete emits phase completion event
func (e *HubEmitter) EmitPhaseComplete(phaseName string, status string, iteration int, duration string) {
	e.hub.Broadcast("phase_complete", map[string]interface{}{
		"phase":     phaseName,
		"status":    status,
		"iteration": iteration,
		"duration":  duration,
	})
}

// EmitToolExecution emits tool execution event
func (e *HubEmitter) EmitToolExecution(tool string, status string, summary string) {
	e.hub.Broadcast("tool_execution", map[string]interface{}{
		"tool":    tool,
		"status":  status,
		"summary": summary,
	})
}

// EmitArtifact emits artifact creation event
func (e *HubEmitter) EmitArtifact(artifactType string, phase string, count int) {
	e.hub.Broadcast("artifact", map[string]interface{}{
		"type":  artifactType,
		"phase": phase,
		"count": count,
	})
}

// EmitGraphUpdate emits graph mutation event
func (e *HubEmitter) EmitGraphUpdate(mutation *graph.GraphMutation) {
	e.hub.Broadcast("graph_update", map[string]interface{}{
		"nodes": len(mutation.Nodes),
		"edges": len(mutation.Edges),
	})
}

// EmitLog emits log event
func (e *HubEmitter) EmitLog(level string, component string, message string, fields map[string]any) {
	e.hub.Broadcast("log", map[string]interface{}{
		"level":     level,
		"component": component,
		"message":   message,
		"fields":    fields,
	})
}

// NullEmitter is a no-op emitter for CLI mode
type NullEmitter struct{}

// NewNullEmitter creates a null emitter
func NewNullEmitter() *NullEmitter {
	return &NullEmitter{}
}

func (e *NullEmitter) EmitPhaseStart(phaseName string, objective string, iteration int)       {}
func (e *NullEmitter) EmitPhaseComplete(phaseName string, status string, iteration int, duration string) {}
func (e *NullEmitter) EmitToolExecution(tool string, status string, summary string)          {}
func (e *NullEmitter) EmitArtifact(artifactType string, phase string, count int)             {}
func (e *NullEmitter) EmitGraphUpdate(mutation *graph.GraphMutation)                         {}
func (e *NullEmitter) EmitLog(level string, component string, message string, fields map[string]any) {}
