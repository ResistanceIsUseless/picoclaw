package mcp

import (
	"context"
	"fmt"
	"sync"

	"github.com/ResistanceIsUseless/picoclaw/pkg/logger"
	"github.com/ResistanceIsUseless/picoclaw/pkg/tools"
	"github.com/ResistanceIsUseless/picoclaw/pkg/tools/filters"
)

// TransportType defines the MCP connection transport
type TransportType string

const (
	TransportStdio TransportType = "stdio"
	TransportHTTP  TransportType = "http"
	TransportSSE   TransportType = "sse"
)

// MCPServerConfig defines configuration for an MCP server
type MCPServerConfig struct {
	Name       string            `json:"name"`
	Enabled    bool              `json:"enabled"`
	Transport  TransportType     `json:"transport"`
	URL        string            `json:"url,omitempty"`        // HTTP/SSE endpoint
	Binary     string            `json:"binary,omitempty"`     // Path to stdio binary
	Args       []string          `json:"args,omitempty"`       // Binary arguments
	Env        map[string]string `json:"env,omitempty"`        // Environment variables
	APIKey     string            `json:"api_key,omitempty"`    // Authentication
	AutoStart  bool              `json:"auto_start,omitempty"` // Launch if not running
	ProjectDir string            `json:"project_dir,omitempty"` // Working directory
}

// MCPConnection represents an active connection to an MCP server
type MCPConnection interface {
	// ListTools returns available tools from this MCP server
	ListTools(ctx context.Context) ([]MCPToolDefinition, error)

	// CallTool executes a tool on the MCP server
	CallTool(ctx context.Context, name string, args map[string]any) ([]byte, error)

	// Close terminates the connection
	Close() error

	// IsHealthy checks if the connection is alive
	IsHealthy(ctx context.Context) bool
}

// MCPToolDefinition represents a tool exposed by an MCP server
type MCPToolDefinition struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	InputSchema map[string]any    `json:"inputSchema"`
	Server      string            `json:"server"` // Which server provides this tool
}

// MCPManager manages connections to multiple MCP servers
type MCPManager struct {
	configs        map[string]*MCPServerConfig
	connections    map[string]MCPConnection
	filterRegistry *filters.FilterRegistry
	mu             sync.RWMutex
}

// NewMCPManager creates a new MCP manager
func NewMCPManager(filterRegistry *filters.FilterRegistry) *MCPManager {
	return &MCPManager{
		configs:        make(map[string]*MCPServerConfig),
		connections:    make(map[string]MCPConnection),
		filterRegistry: filterRegistry,
	}
}

// RegisterServer adds an MCP server configuration
func (m *MCPManager) RegisterServer(config *MCPServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !config.Enabled {
		logger.DebugCF("mcp", "Server disabled, skipping registration",
			map[string]any{
				"server": config.Name,
			})
		return nil
	}

	m.configs[config.Name] = config

	logger.InfoCF("mcp", "Registered MCP server",
		map[string]any{
			"server":    config.Name,
			"transport": config.Transport,
		})

	return nil
}

// ConnectServer establishes a connection to an MCP server
func (m *MCPManager) ConnectServer(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	config, exists := m.configs[name]
	if !exists {
		return fmt.Errorf("server %q not registered", name)
	}

	// Check if already connected
	if conn, exists := m.connections[name]; exists && conn.IsHealthy(ctx) {
		logger.DebugCF("mcp", "Server already connected",
			map[string]any{
				"server": name,
			})
		return nil
	}

	// Auto-start if configured
	if config.AutoStart {
		if err := m.autoStartServer(ctx, config); err != nil {
			logger.WarnCF("mcp", "Failed to auto-start server",
				map[string]any{
					"server": name,
					"error":  err.Error(),
				})
		}
	}

	// Create connection based on transport type
	var conn MCPConnection
	var err error

	switch config.Transport {
	case TransportStdio:
		conn, err = NewStdioConnection(config)
	case TransportHTTP:
		conn, err = NewHTTPConnection(config)
	case TransportSSE:
		conn, err = NewSSEConnection(config)
	default:
		return fmt.Errorf("unsupported transport: %s", config.Transport)
	}

	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", name, err)
	}

	m.connections[name] = conn

	logger.InfoCF("mcp", "Connected to MCP server",
		map[string]any{
			"server":    name,
			"transport": config.Transport,
		})

	return nil
}

// DisconnectServer closes the connection to an MCP server
func (m *MCPManager) DisconnectServer(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	conn, exists := m.connections[name]
	if !exists {
		return fmt.Errorf("server %q not connected", name)
	}

	if err := conn.Close(); err != nil {
		return fmt.Errorf("failed to disconnect from %s: %w", name, err)
	}

	delete(m.connections, name)

	logger.InfoCF("mcp", "Disconnected from MCP server",
		map[string]any{
			"server": name,
		})

	return nil
}

// ConnectAll connects to all registered and enabled servers
func (m *MCPManager) ConnectAll(ctx context.Context) error {
	m.mu.RLock()
	serverNames := make([]string, 0, len(m.configs))
	for name := range m.configs {
		serverNames = append(serverNames, name)
	}
	m.mu.RUnlock()

	var firstErr error
	for _, name := range serverNames {
		if err := m.ConnectServer(ctx, name); err != nil {
			logger.ErrorCF("mcp", "Failed to connect to server",
				map[string]any{
					"server": name,
					"error":  err.Error(),
				})
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	return firstErr
}

// DisconnectAll closes all connections
func (m *MCPManager) DisconnectAll() error {
	m.mu.RLock()
	serverNames := make([]string, 0, len(m.connections))
	for name := range m.connections {
		serverNames = append(serverNames, name)
	}
	m.mu.RUnlock()

	var firstErr error
	for _, name := range serverNames {
		if err := m.DisconnectServer(name); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	return firstErr
}

// CallTool executes a tool on a specific MCP server
func (m *MCPManager) CallTool(ctx context.Context, server, tool string, args map[string]any) (*tools.ToolResult, error) {
	m.mu.RLock()
	conn, exists := m.connections[server]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("not connected to server %q", server)
	}

	logger.InfoCF("mcp", "Calling MCP tool",
		map[string]any{
			"server": server,
			"tool":   tool,
		})

	// Call the tool
	output, err := conn.CallTool(ctx, tool, args)
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("MCP tool call failed: %v", err)), nil
	}

	// Apply filtering if available
	result := string(output)
	if m.filterRegistry != nil {
		filtered, err := m.filterRegistry.ApplyFilter(tool, output)
		if err != nil {
			logger.WarnCF("mcp", "Failed to apply filter to MCP tool output",
				map[string]any{
					"server": server,
					"tool":   tool,
					"error":  err.Error(),
				})
		} else {
			result = filtered
		}
	}

	return &tools.ToolResult{
		ForLLM:  result,
		IsError: false,
	}, nil
}

// GetAllTools returns all available tools from all connected servers
func (m *MCPManager) GetAllTools(ctx context.Context) ([]MCPToolDefinition, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	allTools := make([]MCPToolDefinition, 0)

	for serverName, conn := range m.connections {
		tools, err := conn.ListTools(ctx)
		if err != nil {
			logger.ErrorCF("mcp", "Failed to list tools from server",
				map[string]any{
					"server": serverName,
					"error":  err.Error(),
				})
			continue
		}

		// Tag tools with their server
		for i := range tools {
			tools[i].Server = serverName
		}

		allTools = append(allTools, tools...)
	}

	logger.InfoCF("mcp", "Retrieved all MCP tools",
		map[string]any{
			"total_tools": len(allTools),
			"servers":     len(m.connections),
		})

	return allTools, nil
}

// GetConnection returns a connection to a specific server
func (m *MCPManager) GetConnection(name string) (MCPConnection, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conn, exists := m.connections[name]
	return conn, exists
}

// IsConnected checks if a server is connected
func (m *MCPManager) IsConnected(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.connections[name]
	return exists
}

// GetConnectedServers returns names of all connected servers
func (m *MCPManager) GetConnectedServers() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	servers := make([]string, 0, len(m.connections))
	for name := range m.connections {
		servers = append(servers, name)
	}
	return servers
}

// autoStartServer attempts to launch an MCP server
func (m *MCPManager) autoStartServer(ctx context.Context, config *MCPServerConfig) error {
	if config.Transport != TransportStdio {
		return fmt.Errorf("auto-start only supported for stdio transport")
	}

	if config.Binary == "" {
		return fmt.Errorf("binary path not specified for auto-start")
	}

	logger.InfoCF("mcp", "Auto-starting MCP server",
		map[string]any{
			"server": config.Name,
			"binary": config.Binary,
		})

	// Server will be started by StdioConnection
	return nil
}
