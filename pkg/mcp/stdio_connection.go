package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"github.com/ResistanceIsUseless/picoclaw/pkg/logger"
)

// StdioConnection implements MCP over stdio (subprocess communication)
type StdioConnection struct {
	config  *MCPServerConfig
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	stderr  io.ReadCloser
	scanner *bufio.Scanner
	mu      sync.Mutex
}

// NewStdioConnection creates a new stdio-based MCP connection
func NewStdioConnection(config *MCPServerConfig) (*StdioConnection, error) {
	if config.Binary == "" {
		return nil, fmt.Errorf("binary path required for stdio transport")
	}

	conn := &StdioConnection{
		config: config,
	}

	if err := conn.start(); err != nil {
		return nil, err
	}

	return conn, nil
}

func (s *StdioConnection) start() error {
	s.cmd = exec.Command(s.config.Binary, s.config.Args...)

	// Set environment variables
	if len(s.config.Env) > 0 {
		env := make([]string, 0, len(s.config.Env))
		for k, v := range s.config.Env {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		s.cmd.Env = append(s.cmd.Env, env...)
	}

	// Set working directory if specified
	if s.config.ProjectDir != "" {
		s.cmd.Dir = s.config.ProjectDir
	}

	var err error

	// Connect to stdin
	s.stdin, err = s.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdin: %w", err)
	}

	// Connect to stdout
	s.stdout, err = s.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout: %w", err)
	}

	// Connect to stderr for logging
	s.stderr, err = s.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr: %w", err)
	}

	// Start the process
	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	s.scanner = bufio.NewScanner(s.stdout)

	// Log stderr in background
	go s.logStderr()

	logger.InfoCF("mcp", "Started stdio MCP server",
		map[string]any{
			"server": s.config.Name,
			"binary": s.config.Binary,
			"pid":    s.cmd.Process.Pid,
		})

	return nil
}

func (s *StdioConnection) logStderr() {
	scanner := bufio.NewScanner(s.stderr)
	for scanner.Scan() {
		logger.DebugCF("mcp", "MCP server stderr",
			map[string]any{
				"server":  s.config.Name,
				"message": scanner.Text(),
			})
	}
}

func (s *StdioConnection) sendRequest(req map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// MCP protocol uses JSON-RPC over stdio
	data = append(data, '\n')

	if _, err := s.stdin.Write(data); err != nil {
		return fmt.Errorf("failed to write request: %w", err)
	}

	return nil
}

func (s *StdioConnection) readResponse() (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.scanner.Scan() {
		if err := s.scanner.Err(); err != nil {
			return nil, fmt.Errorf("failed to read response: %w", err)
		}
		return nil, fmt.Errorf("EOF reading response")
	}

	var response map[string]any
	if err := json.Unmarshal(s.scanner.Bytes(), &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return response, nil
}

func (s *StdioConnection) ListTools(ctx context.Context) ([]MCPToolDefinition, error) {
	req := map[string]any{
		"method": "tools/list",
		"id":     1,
	}

	if err := s.sendRequest(req); err != nil {
		return nil, err
	}

	response, err := s.readResponse()
	if err != nil {
		return nil, err
	}

	// Extract tools from response
	result, ok := response["result"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	toolsData, ok := result["tools"].([]any)
	if !ok {
		return nil, fmt.Errorf("tools not found in response")
	}

	tools := make([]MCPToolDefinition, 0, len(toolsData))
	for _, t := range toolsData {
		toolMap, ok := t.(map[string]any)
		if !ok {
			continue
		}

		tool := MCPToolDefinition{
			Name:        getString(toolMap, "name"),
			Description: getString(toolMap, "description"),
		}

		if schema, ok := toolMap["inputSchema"].(map[string]any); ok {
			tool.InputSchema = schema
		}

		tools = append(tools, tool)
	}

	logger.DebugCF("mcp", "Listed tools from stdio server",
		map[string]any{
			"server":     s.config.Name,
			"tool_count": len(tools),
		})

	return tools, nil
}

func (s *StdioConnection) CallTool(ctx context.Context, name string, args map[string]any) ([]byte, error) {
	req := map[string]any{
		"method": "tools/call",
		"id":     2,
		"params": map[string]any{
			"name":      name,
			"arguments": args,
		},
	}

	logger.DebugCF("mcp", "Calling stdio MCP tool",
		map[string]any{
			"server": s.config.Name,
			"tool":   name,
		})

	if err := s.sendRequest(req); err != nil {
		return nil, err
	}

	response, err := s.readResponse()
	if err != nil {
		return nil, err
	}

	// Check for error
	if errData, ok := response["error"]; ok {
		return nil, fmt.Errorf("MCP error: %v", errData)
	}

	// Extract result
	result, ok := response["result"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	// Get content
	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		return nil, fmt.Errorf("no content in response")
	}

	// Extract text from first content item
	firstContent, ok := content[0].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid content format")
	}

	text, ok := firstContent["text"].(string)
	if !ok {
		return nil, fmt.Errorf("no text in content")
	}

	return []byte(text), nil
}

func (s *StdioConnection) Close() error {
	if s.stdin != nil {
		s.stdin.Close()
	}

	if s.cmd != nil && s.cmd.Process != nil {
		if err := s.cmd.Process.Kill(); err != nil {
			logger.WarnCF("mcp", "Failed to kill MCP server process",
				map[string]any{
					"server": s.config.Name,
					"error":  err.Error(),
				})
		}
	}

	logger.InfoCF("mcp", "Closed stdio connection",
		map[string]any{
			"server": s.config.Name,
		})

	return nil
}

func (s *StdioConnection) IsHealthy(ctx context.Context) bool {
	if s.cmd == nil || s.cmd.Process == nil {
		return false
	}

	// Check if process is still running
	// This is a simple check - could be enhanced
	return s.cmd.ProcessState == nil || !s.cmd.ProcessState.Exited()
}

func getString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
