package webui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/gorilla/websocket"

	"github.com/ResistanceIsUseless/picoclaw/pkg/blackboard"
	"github.com/ResistanceIsUseless/picoclaw/pkg/graph"
	"github.com/ResistanceIsUseless/picoclaw/pkg/logger"
	"github.com/ResistanceIsUseless/picoclaw/pkg/orchestrator"
	"github.com/ResistanceIsUseless/picoclaw/pkg/registry"
)

// Server manages the web UI HTTP server and WebSocket connections
type Server struct {
	orchestrator *orchestrator.Orchestrator
	blackboard   *blackboard.Blackboard
	graph        *graph.Graph
	registry     *registry.ToolRegistry
	hub          *Hub
	server       *http.Server
	startTime    time.Time
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

// NewServer creates a new web UI server
func NewServer(
	orch *orchestrator.Orchestrator,
	bb *blackboard.Blackboard,
	g *graph.Graph,
	toolRegistry *registry.ToolRegistry,
) *Server {
	hub := NewHub()

	return &Server{
		orchestrator: orch,
		blackboard:   bb,
		graph:        g,
		registry:     toolRegistry,
		hub:          hub,
		startTime:    time.Now(),
	}
}

// Start starts the web UI server
func (s *Server) Start(addr string) error {
	// Start WebSocket hub
	go s.hub.Run()

	// Create router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// API routes
	r.Route("/api", func(r chi.Router) {
		r.Get("/status", s.handleStatus)
		r.Get("/pipelines", s.handleListPipelines)
		r.Get("/pipeline/status", s.handlePipelineStatus)
		r.Get("/phase", s.handlePhaseDetail)
		r.Get("/artifacts", s.handleListArtifacts)
		r.Get("/artifacts/{id}", s.handleGetArtifact)
		r.Get("/graph/nodes", s.handleGraphNodes)
		r.Get("/graph/edges", s.handleGraphEdges)
		r.Get("/graph/frontier", s.handleGraphFrontier)
		r.Get("/tools", s.handleListTools)
	})

	// WebSocket routes
	r.Get("/ws/pipeline", s.handleWSPipeline)
	r.Get("/ws/logs", s.handleWSLogs)
	r.Get("/ws/graph", s.handleWSGraph)

	// Static files (will serve React frontend later)
	// r.Get("/*", s.handleStatic)

	// Create HTTP server
	s.server = &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.InfoCF("webui", "Starting web UI server",
		map[string]any{
			"addr": addr,
		})

	return s.server.ListenAndServe()
}

// Stop stops the web UI server
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

// GetEventEmitter returns an event emitter for this server
func (s *Server) GetEventEmitter() EventEmitter {
	return NewHubEmitter(s.hub)
}

// REST API Handlers

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := SystemStatus{
		Status:        "running",
		Version:       "0.1.0",
		Uptime:        time.Since(s.startTime).String(),
		ActiveClients: len(s.hub.clients),
		Timestamp:     time.Now(),
	}

	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleListPipelines(w http.ResponseWriter, r *http.Request) {
	// Return predefined pipelines
	pipelines := PipelineList{
		Pipelines: []PipelineInfo{
			{
				Name:        "web_quick",
				Description: "Quick web reconnaissance",
				Phases:      []string{"recon", "web_enum"},
			},
			{
				Name:        "web_full",
				Description: "Full web security assessment",
				Phases:      []string{"recon", "web_enum", "vuln_scan", "exploitation"},
			},
		},
	}

	writeJSON(w, http.StatusOK, pipelines)
}

func (s *Server) handlePipelineStatus(w http.ResponseWriter, r *http.Request) {
	status := SerializePipelineStatus(s.orchestrator, s.startTime)
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handlePhaseDetail(w http.ResponseWriter, r *http.Request) {
	current := s.orchestrator.GetCurrentPhase()
	if current == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "No phase currently executing",
		})
		return
	}

	detail := SerializePhaseDetail(current)
	writeJSON(w, http.StatusOK, detail)
}

func (s *Server) handleListArtifacts(w http.ResponseWriter, r *http.Request) {
	// Get query parameters
	phase := r.URL.Query().Get("phase")
	artifactType := r.URL.Query().Get("type")

	var artifacts []blackboard.ArtifactEnvelope
	var err error

	if phase != "" {
		artifacts, err = s.blackboard.GetByPhase(phase)
	} else if artifactType != "" {
		artifacts, err = s.blackboard.GetByType(artifactType)
	} else {
		artifacts, err = s.blackboard.GetAll()
	}

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	views := SerializeArtifacts(artifacts)
	writeJSON(w, http.StatusOK, views)
}

func (s *Server) handleGetArtifact(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	artifacts, err := s.blackboard.GetAll()
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "Artifact not found",
		})
		return
	}

	views := SerializeArtifacts(artifacts)
	for _, view := range views {
		if view.ID == id {
			writeJSON(w, http.StatusOK, view)
			return
		}
	}

	writeJSON(w, http.StatusNotFound, map[string]string{
		"error": "Artifact not found",
	})
}

func (s *Server) handleGraphNodes(w http.ResponseWriter, r *http.Request) {
	// Get frontier for highlighting
	entityRegistry := graph.NewEntityRegistry()
	frontier := s.graph.ComputeFrontier(entityRegistry)

	export := SerializeGraphExport(s.graph, frontier)
	writeJSON(w, http.StatusOK, export.Nodes)
}

func (s *Server) handleGraphEdges(w http.ResponseWriter, r *http.Request) {
	export := SerializeGraphExport(s.graph, nil)
	writeJSON(w, http.StatusOK, export.Edges)
}

func (s *Server) handleGraphFrontier(w http.ResponseWriter, r *http.Request) {
	entityRegistry := graph.NewEntityRegistry()
	frontier := s.graph.ComputeFrontier(entityRegistry)

	summary := frontier.Summary()
	recommendations := frontier.RecommendTools()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"summary":         summary,
		"recommendations": recommendations,
	})
}

func (s *Server) handleListTools(w http.ResponseWriter, r *http.Request) {
	tools := s.registry.ListAll()

	type ToolView struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Tier        string `json:"tier"`
	}

	views := make([]ToolView, 0, len(tools))
	for _, tool := range tools {
		views = append(views, ToolView{
			Name:        tool.Name,
			Description: tool.Description,
			Tier:        fmt.Sprintf("%d", tool.Tier),
		})
	}

	writeJSON(w, http.StatusOK, views)
}

// WebSocket Handlers

func (s *Server) handleWSPipeline(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.WarnCF("webui", "WebSocket upgrade failed",
			map[string]any{
				"error": err.Error(),
			})
		return
	}

	client := &Client{
		hub:  s.hub,
		conn: conn,
		send: make(chan []byte, 256),
	}

	s.hub.register <- client

	// Start pumps
	go client.writePump()
	go client.readPump()
}

func (s *Server) handleWSLogs(w http.ResponseWriter, r *http.Request) {
	// Same as pipeline for now - logs are broadcast through same hub
	s.handleWSPipeline(w, r)
}

func (s *Server) handleWSGraph(w http.ResponseWriter, r *http.Request) {
	// Same as pipeline for now - graph updates are broadcast through same hub
	s.handleWSPipeline(w, r)
}

// Helper functions

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
