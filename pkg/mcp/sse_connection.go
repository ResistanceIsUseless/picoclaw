package mcp

import (
	"context"
	"fmt"

	"github.com/ResistanceIsUseless/picoclaw/pkg/logger"
)

// SSEConnection implements MCP over Server-Sent Events
type SSEConnection struct {
	config *MCPServerConfig
	url    string
	apiKey string
}

// NewSSEConnection creates a new SSE-based MCP connection
func NewSSEConnection(config *MCPServerConfig) (*SSEConnection, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("URL required for SSE transport")
	}

	logger.InfoCF("mcp", "Created SSE connection",
		map[string]any{
			"server": config.Name,
			"url":    config.URL,
		})

	return &SSEConnection{
		config: config,
		url:    config.URL,
		apiKey: config.APIKey,
	}, nil
}

func (s *SSEConnection) ListTools(ctx context.Context) ([]MCPToolDefinition, error) {
	// TODO: Implement SSE-based tool listing
	return nil, fmt.Errorf("SSE transport not yet implemented")
}

func (s *SSEConnection) CallTool(ctx context.Context, name string, args map[string]any) ([]byte, error) {
	// TODO: Implement SSE-based tool calling
	return nil, fmt.Errorf("SSE transport not yet implemented")
}

func (s *SSEConnection) Close() error {
	logger.DebugCF("mcp", "Closed SSE connection",
		map[string]any{
			"server": s.config.Name,
		})
	return nil
}

func (s *SSEConnection) IsHealthy(ctx context.Context) bool {
	// TODO: Implement SSE health check
	return false
}
