package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ResistanceIsUseless/picoclaw/pkg/logger"
)

// HTTPConnection implements MCP over HTTP/REST
type HTTPConnection struct {
	config *MCPServerConfig
	client *http.Client
	url    string
	apiKey string
}

// NewHTTPConnection creates a new HTTP-based MCP connection
func NewHTTPConnection(config *MCPServerConfig) (*HTTPConnection, error) {
	if config.URL == "" {
		return nil, fmt.Errorf("URL required for HTTP transport")
	}

	return &HTTPConnection{
		config: config,
		client: &http.Client{
			Timeout: 5 * time.Minute,
		},
		url:    config.URL,
		apiKey: config.APIKey,
	}, nil
}

func (h *HTTPConnection) ListTools(ctx context.Context) ([]MCPToolDefinition, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", h.url+"/tools", nil)
	if err != nil {
		return nil, err
	}

	if h.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+h.apiKey)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Tools []MCPToolDefinition `json:"tools"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	logger.DebugCF("mcp", "Listed tools from HTTP server",
		map[string]any{
			"server":     h.config.Name,
			"tool_count": len(result.Tools),
		})

	return result.Tools, nil
}

func (h *HTTPConnection) CallTool(ctx context.Context, name string, args map[string]any) ([]byte, error) {
	payload := map[string]any{
		"tool": name,
		"args": args,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", h.url+"/call", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if h.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+h.apiKey)
	}

	logger.DebugCF("mcp", "Calling HTTP MCP tool",
		map[string]any{
			"server": h.config.Name,
			"tool":   name,
		})

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool: %w", err)
	}
	defer resp.Body.Close()

	output, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(output))
	}

	return output, nil
}

func (h *HTTPConnection) Close() error {
	// HTTP connections are stateless, nothing to close
	logger.DebugCF("mcp", "Closed HTTP connection",
		map[string]any{
			"server": h.config.Name,
		})
	return nil
}

func (h *HTTPConnection) IsHealthy(ctx context.Context) bool {
	req, err := http.NewRequestWithContext(ctx, "GET", h.url+"/health", nil)
	if err != nil {
		return false
	}

	if h.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+h.apiKey)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}
