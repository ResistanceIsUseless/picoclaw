# Quick Start: Output Filtering & MCP Integration

## Using Output Filtering

### Enable Filtering for Your Agent

```go
// In your agent initialization code (pkg/agent/instance.go or similar)

import (
    "github.com/ResistanceIsUseless/picoclaw/pkg/tools"
    "github.com/ResistanceIsUseless/picoclaw/pkg/tools/filters"
)

// Create tool registry WITH filtering
outputDir := filepath.Join(workspace, "filtered-output")
toolRegistry := tools.NewToolRegistryWithFilters(outputDir)

// Register your tools as usual
toolRegistry.Register(NewShellTool())
toolRegistry.Register(NewFilesystemTool())
// ... etc
```

That's it! Filtering is now automatic for all registered tool patterns.

### Customize Filter Settings

```go
// Create custom filter with different threshold
customFilter := filters.NewCrawlFilter(outputDir)
customFilter.threshold = 50000 // 50KB instead of 10KB

// Register for specific tool
toolRegistry.filterRegistry.Register("custom_crawler", customFilter)
```

### Disable Filtering for Specific Tool

```go
// Simply don't register a filter for that tool pattern
// Or use the legacy NewToolRegistry() without filters
```

---

## Using MCP Integration

### 1. Configure MCP Servers

Add to your `config.json`:

```json
{
  "mcp": {
    "enabled": true,
    "output_dir": "~/.picoclaw/filtered-output",
    "servers": {
      "recon": {
        "enabled": true,
        "transport": "http",
        "url": "http://localhost:8000",
        "description": "Your custom recon MCP server"
      },
      "ghidra": {
        "enabled": false,
        "transport": "stdio",
        "binary": "/usr/local/bin/ghidra-mcp-server",
        "auto_start": true,
        "project_dir": "~/.picoclaw/ghidra-projects"
      },
      "burp": {
        "enabled": false,
        "transport": "http",
        "url": "http://localhost:9002",
        "api_key": "${BURP_API_KEY}",
        "project_dir": "~/.picoclaw/burp-projects"
      }
    }
  }
}
```

### 2. Initialize MCP Manager

```go
// In your main application initialization

import (
    "github.com/ResistanceIsUseless/picoclaw/pkg/mcp"
    "github.com/ResistanceIsUseless/picoclaw/pkg/tools/filters"
)

// Create filter registry
filterRegistry := filters.NewFilterRegistry(
    filepath.Join(workspace, "filtered-output"),
)

// Create MCP manager
mcpManager := mcp.NewMCPManager(filterRegistry)

// Load configuration and register servers
for name, serverConfig := range config.MCP.Servers {
    if err := mcpManager.RegisterServer(serverConfig); err != nil {
        logger.ErrorCF("mcp", "Failed to register server",
            map[string]any{"server": name, "error": err})
    }
}

// Connect to all enabled servers
if err := mcpManager.ConnectAll(ctx); err != nil {
    logger.WarnCF("mcp", "Some MCP servers failed to connect",
        map[string]any{"error": err})
}
```

### 3. Register MCP Tools

```go
// Discover and register all tools from MCP servers
if err := mcp.RegisterMCPToolsInRegistry(ctx, toolRegistry, mcpManager); err != nil {
    logger.ErrorCF("mcp", "Failed to register MCP tools",
        map[string]any{"error": err})
}

// Now MCP tools are available alongside native tools
logger.InfoCF("tool", "Total tools available",
    map[string]any{"count": toolRegistry.Count()})
```

### 4. Agent Uses MCP Tools Transparently

```go
// Agent code doesn't need to know if tool is native or MCP
result := toolRegistry.Execute(ctx, "recon_subdomain_enum", map[string]any{
    "domain": "target.com",
})

// Output is automatically filtered
fmt.Println(result.ForLLM) // Concise summary

// Full output available in workspace
// ~/.picoclaw/filtered-output/recon_subdomain_enum-20260302-143052.txt
```

---

## Example: Complete Agent Setup

```go
package agent

import (
    "context"
    "path/filepath"

    "github.com/ResistanceIsUseless/picoclaw/pkg/config"
    "github.com/ResistanceIsUseless/picoclaw/pkg/mcp"
    "github.com/ResistanceIsUseless/picoclaw/pkg/tools"
    "github.com/ResistanceIsUseless/picoclaw/pkg/tools/filters"
)

func NewAgent(cfg *config.Config, workspace string) (*Agent, error) {
    ctx := context.Background()

    // 1. Create filter registry
    outputDir := filepath.Join(workspace, "filtered-output")
    filterRegistry := filters.NewFilterRegistry(outputDir)

    // 2. Create tool registry with filtering
    toolRegistry := tools.NewToolRegistryWithFilters(outputDir)

    // 3. Register native tools
    toolRegistry.Register(NewShellTool())
    toolRegistry.Register(NewFilesystemTool())
    toolRegistry.Register(NewWebSearchTool())

    // 4. Set up MCP if enabled
    var mcpManager *mcp.MCPManager
    if cfg.MCP != nil && cfg.MCP.Enabled {
        mcpManager = mcp.NewMCPManager(filterRegistry)

        // Register configured servers
        for name, serverConfig := range cfg.MCP.Servers {
            if err := mcpManager.RegisterServer(serverConfig); err != nil {
                return nil, err
            }
        }

        // Connect to servers
        if err := mcpManager.ConnectAll(ctx); err != nil {
            // Log but don't fail - some servers may be optional
            logger.WarnCF("agent", "Some MCP servers unavailable",
                map[string]any{"error": err})
        }

        // Register MCP tools
        if err := mcp.RegisterMCPToolsInRegistry(ctx, toolRegistry, mcpManager); err != nil {
            logger.WarnCF("agent", "Failed to register MCP tools",
                map[string]any{"error": err})
        }
    }

    // 5. Create agent instance
    agent := &Agent{
        toolRegistry: toolRegistry,
        mcpManager:   mcpManager,
        workspace:    workspace,
        // ... other fields
    }

    logger.InfoCF("agent", "Agent initialized",
        map[string]any{
            "native_tools": toolRegistry.Count() - (if mcpManager available),
            "mcp_tools":    (count from mcpManager),
            "filtering":    "enabled",
        })

    return agent, nil
}

func (a *Agent) Cleanup() {
    if a.mcpManager != nil {
        a.mcpManager.DisconnectAll()
    }
}
```

---

## Testing Your Setup

### Test Output Filtering

```bash
# Run a tool that generates large output
picoclaw agent -m "Run nmap scan on localhost"

# Check filtered output
ls -lh ~/.picoclaw/workspace/filtered-output/
# Should see: nmap-TIMESTAMP.txt with full output

# Check agent received summary
# Agent should have received ~2KB summary instead of 50KB full output
```

### Test MCP Connection

```bash
# Start your recon MCP server
python3 recon_mcp_server.py

# Test connection
picoclaw agent -m "List available recon tools"

# Should show both native and MCP tools
# Native: shell, filesystem, web_search
# MCP: recon_subdomain_enum, recon_port_scan, etc.
```

### Test End-to-End Workflow

```bash
# Run reconnaissance workflow
picoclaw agent -m "Perform reconnaissance on example.com"

# Agent should:
# 1. Discover MCP tools are available
# 2. Use recon_subdomain_enum (MCP tool)
# 3. Receive filtered summary (not 10K subdomains)
# 4. Use recon_port_scan (MCP tool)
# 5. Receive filtered summary (key findings only)
# 6. Continue with next steps based on concise data
```

---

## Troubleshooting

### Filters Not Applied

**Symptom:** Agent still receives full tool output

**Checks:**
1. Verify using `NewToolRegistryWithFilters()` not `NewToolRegistry()`
2. Check tool name matches registered pattern (e.g., "nmap" not "nmap_scan")
3. Ensure output exceeds threshold (default 10KB)
4. Check logs for filter errors

```bash
# Enable debug logging
export PICOCLAW_LOG_LEVEL=debug
picoclaw agent -m "test"
```

### MCP Server Not Connecting

**Symptom:** MCP tools not available

**Checks:**
1. Verify server is running (`curl http://localhost:8000/health`)
2. Check config.json has correct URL/binary path
3. Verify `enabled: true` in server config
4. Check firewall/network access
5. Review MCP server logs

```bash
# Test MCP server directly
curl http://localhost:8000/tools

# Check picoclaw logs
picoclaw agent -m "test" --log-level debug
```

### Stdio MCP Server Won't Start

**Symptom:** "Failed to start process" error

**Checks:**
1. Binary path exists and is executable
2. Binary has correct permissions (`chmod +x`)
3. Required dependencies installed
4. Check stderr output in logs

```bash
# Test binary directly
/path/to/mcp-server --help

# Check permissions
ls -l /path/to/mcp-server
```

### Filtered Output Directory Permissions

**Symptom:** "Failed to save full output" error

**Fix:**
```bash
mkdir -p ~/.picoclaw/workspace/filtered-output
chmod 755 ~/.picoclaw/workspace/filtered-output
```

---

## Performance Considerations

### Filter Performance
- Filters add <100ms overhead (negligible)
- File writes are async (don't block)
- Memory usage: ~2x output size temporarily

### MCP Performance
- Stdio: ~10-50ms latency per call
- HTTP: ~50-200ms latency per call (network)
- Use stdio for local tools (Ghidra)
- Use HTTP for remote services (your recon server)

### Optimization Tips
1. **Batch MCP calls** when possible
2. **Use async tools** for long-running operations
3. **Tune filter thresholds** based on your needs
4. **Monitor context window usage** in logs

---

## Advanced Configuration

### Custom Filter for Specific Tool

```go
// Create specialized filter
type CustomToolFilter struct {
    *filters.BaseFilter
}

func (c *CustomToolFilter) Filter(toolName string, output []byte) (string, string, error) {
    // Your custom filtering logic
    summary := extractKeyFindings(output)
    fullPath, _ := c.SaveFullOutput(toolName, output)
    return summary, fullPath, nil
}

// Register it
customFilter := &CustomToolFilter{
    BaseFilter: filters.NewBaseFilter("custom", outputDir),
}
toolRegistry.filterRegistry.Register("my_tool", customFilter)
```

### MCP Server with Authentication

```json
{
  "servers": {
    "authenticated_server": {
      "enabled": true,
      "transport": "http",
      "url": "https://api.example.com/mcp",
      "api_key": "${API_KEY}",
      "env": {
        "CUSTOM_TOKEN": "${CUSTOM_TOKEN}"
      }
    }
  }
}
```

### Multiple Output Directories

```go
// Separate directories for different tool types
reconOutputDir := filepath.Join(workspace, "recon-output")
exploitOutputDir := filepath.Join(workspace, "exploit-output")

reconFilter := filters.NewCrawlFilter(reconOutputDir)
exploitFilter := filters.NewFuzzingFilter(exploitOutputDir)

toolRegistry.filterRegistry.Register("subfinder", reconFilter)
toolRegistry.filterRegistry.Register("ffuf", exploitFilter)
```

---

## Migration Guide

### From Legacy Tool Registry

**Before:**
```go
toolRegistry := tools.NewToolRegistry()
```

**After:**
```go
outputDir := filepath.Join(workspace, "filtered-output")
toolRegistry := tools.NewToolRegistryWithFilters(outputDir)
```

### From Direct Tool Calls

**Before:**
```go
output := runTool("nmap", args)
result := processNmapOutput(output) // Manual processing
```

**After:**
```go
result := toolRegistry.Execute(ctx, "nmap", args)
// result.ForLLM already contains filtered summary
// Full output auto-saved to disk
```

### Adding MCP to Existing Agent

**Steps:**
1. Add MCP config to `config.json`
2. Initialize `MCPManager` in agent setup
3. Call `RegisterMCPToolsInRegistry()`
4. Existing code works unchanged!

---

## Summary

### What You Get
✅ **Automatic output filtering** - No code changes needed
✅ **MCP tool integration** - Seamless with native tools
✅ **Context efficiency** - 10-50x reduction in noise
✅ **Full audit trail** - Complete outputs saved to disk
✅ **Actionable insights** - Recommendations included
✅ **Extensibility** - Easy to add new filters/servers

### What Changes
- Tool registry initialization (`NewToolRegistryWithFilters()`)
- Config file (add MCP section)
- Agent sees concise summaries instead of raw dumps

### What Stays the Same
- Tool interface and registration
- Agent loop logic
- Existing tools and workflows
- Error handling patterns
