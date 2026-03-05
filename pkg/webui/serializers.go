package webui

import (
	"time"

	"github.com/ResistanceIsUseless/picoclaw/pkg/blackboard"
	"github.com/ResistanceIsUseless/picoclaw/pkg/graph"
	"github.com/ResistanceIsUseless/picoclaw/pkg/orchestrator"
	"github.com/ResistanceIsUseless/picoclaw/pkg/phase"
)

// PipelineStatus represents the current state of a pipeline execution
type PipelineStatus struct {
	Name            string    `json:"name"`
	Status          string    `json:"status"`
	CurrentPhase    string    `json:"current_phase"`
	CompletedPhases []string  `json:"completed_phases"`
	Progress        float64   `json:"progress"`
	StartTime       time.Time `json:"start_time"`
	ArtifactCount   int       `json:"artifact_count"`
	GraphNodes      int       `json:"graph_nodes"`
}

// PhaseDetail represents detailed phase execution state
type PhaseDetail struct {
	Name          string            `json:"name"`
	Status        string            `json:"status"`
	Iteration     int               `json:"iteration"`
	MaxIterations int               `json:"max_iterations"`
	DAGState      *DAGStateView     `json:"dag_state"`
	Contract      *ContractView     `json:"contract"`
	Tools         []ToolExecution   `json:"tools"`
}

// DAGStateView represents the DAG state for API
type DAGStateView struct {
	PhaseName string         `json:"phase_name"`
	Tools     []ToolCallView `json:"tools"`
	Progress  float64        `json:"progress"`
}

// ToolCallView represents a tool call for API
type ToolCallView struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Status  string    `json:"status"`
	Summary string    `json:"summary,omitempty"`
	Started time.Time `json:"started"`
	Ended   time.Time `json:"ended,omitempty"`
}

// ContractView represents contract status for API
type ContractView struct {
	Satisfied         bool     `json:"satisfied"`
	RequiredTools     []string `json:"required_tools"`
	RequiredArtifacts []string `json:"required_artifacts"`
	Progress          float64  `json:"progress"`
	MinIterations     int      `json:"min_iterations"`
	MaxIterations     int      `json:"max_iterations"`
}

// ToolExecution represents a completed tool execution
type ToolExecution struct {
	Tool       string    `json:"tool"`
	Status     string    `json:"status"`
	Summary    string    `json:"summary"`
	OutputSize int       `json:"output_size"`
	Timestamp  time.Time `json:"timestamp"`
}

// GraphExport represents graph data for visualization
type GraphExport struct {
	Nodes []NodeView `json:"nodes"`
	Edges []EdgeView `json:"edges"`
}

// NodeView represents a graph node for API
type NodeView struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Label      string                 `json:"label"`
	Properties map[string]interface{} `json:"properties"`
	IsFrontier bool                   `json:"is_frontier"`
}

// EdgeView represents a graph edge for API
type EdgeView struct {
	ID         string                 `json:"id"`
	Source     string                 `json:"source"`
	Target     string                 `json:"target"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
}

// ArtifactView represents an artifact for API
type ArtifactView struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Phase     string                 `json:"phase"`
	Domain    string                 `json:"domain"`
	CreatedAt time.Time              `json:"created_at"`
	Data      map[string]interface{} `json:"data"`
}

// PipelineList represents available pipelines
type PipelineList struct {
	Pipelines []PipelineInfo `json:"pipelines"`
}

// PipelineInfo represents pipeline metadata
type PipelineInfo struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Phases      []string `json:"phases"`
}

// SystemStatus represents system health
type SystemStatus struct {
	Status        string    `json:"status"`
	Version       string    `json:"version"`
	Uptime        string    `json:"uptime"`
	ActiveClients int       `json:"active_clients"`
	Timestamp     time.Time `json:"timestamp"`
}

// SerializePipelineStatus converts orchestrator state to API response
func SerializePipelineStatus(orch *orchestrator.Orchestrator, startTime time.Time) *PipelineStatus {
	current := orch.GetCurrentPhase()
	completed := orch.GetCompletedPhases()
	bb := orch.GetBlackboard()
	g := orch.GetGraph()

	status := &PipelineStatus{
		Name:            "current_pipeline",
		Status:          "idle",
		CompletedPhases: completed,
		StartTime:       startTime,
		GraphNodes:      g.NodeCount(),
	}

	// Count artifacts
	artifacts, _ := bb.GetAll()
	status.ArtifactCount = len(artifacts)

	if current != nil {
		status.Status = string(current.Status)
		status.CurrentPhase = current.PhaseName
		// Calculate progress based on completed phases and current iteration
		// This is simplified - could be more sophisticated
		totalPhases := len(completed) + 1 // completed + current
		status.Progress = float64(len(completed)) / float64(totalPhases)
		if current.Iteration > 0 {
			phaseProgress := float64(current.Iteration) / float64(current.Contract.MaxIterations)
			status.Progress += phaseProgress / float64(totalPhases)
		}
	} else {
		status.Progress = 1.0
		status.Status = "completed"
	}

	return status
}

// SerializePhaseDetail converts phase execution to API response
func SerializePhaseDetail(phaseExec *orchestrator.PhaseExecution) *PhaseDetail {
	detail := &PhaseDetail{
		Name:          phaseExec.PhaseName,
		Status:        string(phaseExec.Status),
		Iteration:     phaseExec.Iteration,
		MaxIterations: phaseExec.Contract.MaxIterations,
	}

	// Serialize DAG state
	if phaseExec.State != nil {
		detail.DAGState = &DAGStateView{
			PhaseName: phaseExec.PhaseName,
			Tools:     make([]ToolCallView, 0),
			Progress:  phaseExec.State.GetProgress(),
		}

		// Add tool calls
		for _, toolCall := range phaseExec.State.GetToolCalls() {
			detail.DAGState.Tools = append(detail.DAGState.Tools, ToolCallView{
				ID:      toolCall.ID,
				Name:    toolCall.ToolName,
				Status:  string(toolCall.Status),
				Summary: toolCall.OutputSummary,
				Started: toolCall.StartTime,
				Ended:   toolCall.EndTime,
			})
		}
	}

	// Serialize contract
	if phaseExec.Contract != nil {
		phaseCtx := &phase.PhaseContext{
			Phase:      phaseExec.PhaseName,
			State:      phaseExec.State,
			Artifacts:  phaseExec.Artifacts,
			Iteration:  phaseExec.Iteration,
		}

		detail.Contract = &ContractView{
			Satisfied:         phaseExec.Contract.CanComplete(phaseCtx),
			RequiredTools:     phaseExec.Contract.RequiredTools,
			RequiredArtifacts: phaseExec.Contract.RequiredArtifacts,
			MinIterations:     phaseExec.Contract.MinIterations,
			MaxIterations:     phaseExec.Contract.MaxIterations,
		}

		// Calculate contract progress
		satisfied := 0
		total := len(phaseExec.Contract.RequiredTools) + len(phaseExec.Contract.RequiredArtifacts)
		if total > 0 {
			// This is simplified - could check actual completion
			if phaseExec.State != nil {
				satisfied = len(phaseExec.State.GetCompletedTools())
			}
			detail.Contract.Progress = float64(satisfied) / float64(total)
		}
	}

	return detail
}

// SerializeGraphExport converts graph to API response
func SerializeGraphExport(g *graph.Graph, frontier *graph.Frontier) *GraphExport {
	export := &GraphExport{
		Nodes: make([]NodeView, 0),
		Edges: make([]EdgeView, 0),
	}

	// Get all nodes
	for _, node := range g.GetAllNodes() {
		isFrontier := false
		if frontier != nil {
			isFrontier = frontier.Contains(node.ID)
		}

		export.Nodes = append(export.Nodes, NodeView{
			ID:         node.ID,
			Type:       node.Type,
			Label:      node.Label,
			Properties: node.Properties,
			IsFrontier: isFrontier,
		})
	}

	// Get all edges
	for _, edge := range g.GetAllEdges() {
		export.Edges = append(export.Edges, EdgeView{
			ID:         edge.ID,
			Source:     edge.Source,
			Target:     edge.Target,
			Type:       edge.Type,
			Properties: edge.Properties,
		})
	}

	return export
}

// SerializeArtifacts converts artifacts to API response
func SerializeArtifacts(artifacts []blackboard.ArtifactEnvelope) []ArtifactView {
	views := make([]ArtifactView, 0, len(artifacts))

	for _, artifact := range artifacts {
		view := ArtifactView{
			ID:        artifact.Metadata.ID,
			Type:      artifact.Metadata.Type,
			Phase:     artifact.Metadata.Phase,
			Domain:    artifact.Metadata.Domain,
			CreatedAt: artifact.Metadata.CreatedAt,
			Data:      make(map[string]interface{}),
		}

		// Extract key data fields based on type
		// This is simplified - could unmarshal specific artifact types
		view.Data["raw"] = string(artifact.Data)

		views = append(views, view)
	}

	return views
}
