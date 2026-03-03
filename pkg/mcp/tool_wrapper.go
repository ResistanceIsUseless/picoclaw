package mcp

import (
	"context"

	"github.com/ResistanceIsUseless/picoclaw/pkg/tools"
)

// MCPToolWrapper wraps an MCP tool as a native Tool interface
type MCPToolWrapper struct {
	definition MCPToolDefinition
	manager    *MCPManager
}

// NewMCPToolWrapper creates a tool wrapper for an MCP tool
func NewMCPToolWrapper(definition MCPToolDefinition, manager *MCPManager) *MCPToolWrapper {
	return &MCPToolWrapper{
		definition: definition,
		manager:    manager,
	}
}

func (m *MCPToolWrapper) Name() string {
	// Prefix with server name to avoid conflicts
	return m.definition.Server + "_" + m.definition.Name
}

func (m *MCPToolWrapper) Description() string {
	return m.definition.Description
}

func (m *MCPToolWrapper) InputSchema() map[string]any {
	return m.definition.InputSchema
}

func (m *MCPToolWrapper) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	result, err := m.manager.CallTool(ctx, m.definition.Server, m.definition.Name, args)
	if err != nil {
		return tools.ErrorResult(err.Error())
	}
	return result
}

// RegisterMCPToolsInRegistry discovers and registers all MCP tools
func RegisterMCPToolsInRegistry(ctx context.Context, registry *tools.ToolRegistry, manager *MCPManager) error {
	allTools, err := manager.GetAllTools(ctx)
	if err != nil {
		return err
	}

	for _, toolDef := range allTools {
		wrapper := NewMCPToolWrapper(toolDef, manager)
		registry.Register(wrapper)
	}

	return nil
}
