# Agent Instructions for PicoClaw/StrikeClaw Development

This document provides essential guidance for AI agents working in the PicoClaw/StrikeClaw codebase. PicoClaw is an ultra-lightweight personal AI assistant written in Go, designed to run on resource-constrained hardware, currently being extended into StrikeClaw - a methodology-driven agent framework.

## Project Overview

**Core Architecture**: Multi-model AI agent framework with extensible tool system, designed for resource-constrained environments (10MB RAM, $10 devices).

**Key Components**:
- **Agent Loop**: Think/Act/Observe pattern for iterative AI processing
- **Multi-Provider LLM Support**: Abstraction layer for OpenAI, Anthropic, DeepSeek, Gemini, local models
- **Model Routing**: Task-based model selection with tier-based cost optimization
- **Tool System**: Extensible tool registry with security sandboxing
- **Channel Integration**: Discord, Telegram, Slack, WeChat, QQ, DingTalk, Line, Feishu
- **Workspace Management**: Persistent context with AGENTS.md, SOUL.md, IDENTITY.md files

## Essential Commands

### Build & Development
```bash
# Primary build commands
make build                    # Build for current platform
make build-all                # Cross-compile for all platforms
make install                  # Install to ~/.local/bin

# Development workflow
make deps                     # Download and verify dependencies
make check                    # Full pre-commit check (deps + fmt + vet + test)
make fmt                      # Format with golangci-lint
make vet                      # Static analysis
make lint                     # Full linter (uses .golangci.yaml)
make test                     # Run all tests

# Testing
go test -run TestName -v ./pkg/package/    # Specific test
go test -cover ./...                        # With coverage
go test -bench=. -benchmem ./...           # Benchmarks

# Running
make run ARGS="agent -m 'hello'"           # Build and run
./build/picoclaw agent -m "What is 2+2?"   # Direct execution
```

### CLI Usage
```bash
picoclaw agent -m "message"                # Send message to agent
picoclaw auth login                        # OAuth login for providers
picoclaw config discover                   # Discover configuration
picoclaw skills install <skill>            # Install new skill
picoclaw cron add "*/30 * * * *" "task"    # Add scheduled task
picoclaw status                             # System status
```

## Project Structure

### Core Packages (`pkg/`)

```
pkg/
├── agent/          # Agent core - loop, instance, registry, context
├── providers/      # LLM providers - Anthropic, OpenAI, fallback, routing
├── tools/          # Tool system - registry, filesystem, shell, web, edit
├── channels/       # Messaging integrations - Discord, Telegram, Slack, etc.
├── session/        # Conversation persistence and management
├── skills/         # Skill discovery, installation, execution
├── routing/        # Model routing and tier-based selection
├── config/         # Configuration management and defaults
├── utils/          # Utilities - message, string, media, download
├── state/          # Persistent state management
├── workflow/       # Workflow engine and parsing
├── auth/           # OAuth token management
├── bus/            # Message bus for inter-component communication
├── cron/           # Scheduled task execution
├── heartbeat/      # Periodic task service
├── health/         # Health check server
├── logger/         # Structured logging
├── migrate/        # Configuration migration
├── devices/        # Hardware device integration (USB, I2C, SPI)
└── voice/          # Voice transcription
```

### CLI Structure (`cmd/picoclaw/`)
```
cmd/picoclaw/
├── main.go              # Root command with Cobra
├── internal/
│   ├── agent/           # Agent CLI commands
│   ├── auth/            # Authentication commands (login, logout, status)
│   ├── config/          # Configuration commands
│   ├── gateway/         # Gateway management
│   ├── skills/          # Skill management (install, list, search, remove)
│   ├── cron/            # Scheduled task management
│   ├── migrate/         # Migration utilities
│   ├── status/          # System status
│   └── version/         # Version information
└── onboard/             # First-time setup and workspace creation
```

### Workspace Structure
Default: `~/.picoclaw/workspace/`
```
workspace/
├── sessions/          # Conversation history and persistence
├── memory/            # Long-term memory (MEMORY.md)
├── state/             # Persistent state files
├── cron/              # Scheduled jobs database
├── skills/            # User-installed skills
├── AGENTS.md          # This file - agent behavior guide
├── HEARTBEAT.md       # Periodic task prompts (checked every 30min)
├── IDENTITY.md        # Agent identity and persona
├── SOUL.md            # Agent personality and values
├── TOOLS.md           # Available tool descriptions
└── USER.md            # User preferences and context
```

## Code Patterns & Conventions

### Go Development Standards
- **Go Version**: Requires Go 1.25.7+
- **CGO**: Disabled (`CGO_ENABLED=0`) for minimal binary size
- **Build Tags**: Use `-tags stdjson` for builds
- **Package Organization**: Follow existing `pkg/` structure
- **Minimal Dependencies**: Keep binary size small - avoid heavy dependencies

### Error Handling
```go
// Use structured logging with context
logger.ErrorCF("component", "Error message", 
    map[string]any{"key": "value"})

// Return errors with context
return fmt.Errorf("operation failed: %w", err)
```

### Configuration
- **Primary Config**: `~/.picoclaw/config.json`
- **Environment Variables**: `PICOCLAW_*` prefixes override config
- **API Keys**: Use `_env` suffix in config to reference environment variables
- **Model Definition**: Define models in `model_list` array with vendor/model format

### Tool Development
```go
// Implement Tool interface
type MyTool struct {
    // Tool state
}

func (t *MyTool) GetDefinition() *ToolDefinition {
    return &ToolDefinition{
        Name:        "my_tool",
        Description: "Tool description",
        Parameters: map[string]ToolParameter{
            "param": {
                Type:        "string",
                Required:    true,
                Description: "Parameter description",
            },
        },
    }
}

func (t *MyTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
    // Tool implementation
    // Always check workspace boundaries for security
}
```

### Provider Development
```go
// Implement LLMProvider interface
type MyProvider struct {
    // Provider state
}

func (p *MyProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
    // LLM completion implementation
}

func (p *MyProvider) Models() []ModelRef {
    // Return supported models
}
```

### Security Patterns
**Critical Security Requirements**:
- **File Operations**: MUST check workspace boundaries (see `tools/filesystem.go`)
- **Shell Execution**: MUST block dangerous commands (see `tools/shell.go`)
- **Channel Access**: MUST validate allow_from lists
- **OAuth Tokens**: Store encrypted in `pkg/auth/store.go`
- **Path Traversal**: Never allow `../` in file paths

### Testing Patterns
- **Test Location**: Alongside implementation (`*_test.go`)
- **Test Framework**: Use `testify` for assertions
- **Mock Objects**: Create mock implementations for providers/channels
- **Edge Cases**: Test empty inputs, error conditions, concurrent access
- **API Tests**: Some tests require environment variables for API keys

## Important Gotchas

### Build & Runtime
1. **Cross-compilation**: Use `make build-all` for multiple platforms
2. **Workspace Permissions**: `~/.picoclaw/workspace/` must be writable
3. **Config Migration**: Run `picoclaw migrate config` when upgrading
4. **Debug Logging**: Enable via config or `PICOCLAW_LOG_LEVEL=debug`

### Development Pitfalls
1. **Agent Loop Changes**: The agent loop affects all interactions - test carefully
2. **Tool Registration**: Register tools in agent instance or shared registration
3. **Provider Fallback**: The fallback chain handles provider failures automatically
4. **Session Management**: Context window management happens automatically
5. **Concurrent Access**: Use appropriate locking in shared components

### Configuration Issues
1. **Model List vs Providers**: Prefer `model_list` over legacy `providers` section
2. **API Key Format**: Use `_env` suffix for environment variable references
3. **Channel Enablement**: Set `enabled: true` in channel configuration
4. **Workspace Restriction**: `restrict_to_workspace: true` for security

### Common Development Tasks

### Adding a New Tool
1. Create tool struct in `pkg/tools/`
2. Implement `Tool` interface (GetDefinition, Execute)
3. Register in agent instance or shared tool registration
4. Add security checks if file/shell operations
5. Write comprehensive tests with edge cases

### Adding a New LLM Provider
1. Implement `LLMProvider` interface in `pkg/providers/`
2. Add vendor prefix to model routing in `pkg/routing/`
3. Update config schema in `pkg/config/config.go`
4. Add example to `config/config.example.json`
5. Test with: `picoclaw provider test`

### Adding a New Channel
1. Create channel implementation in `pkg/channels/`
2. Implement channel interface (Start, Stop, message handling)
3. Add config struct in `pkg/config/config.go`
4. Register in channel manager
5. Add setup documentation

### Modifying Agent Loop
1. **Entry Point**: `AgentLoop.ProcessMessage()` in `pkg/agent/loop.go`
2. **Tool Execution**: `executeTool()` handles tool calls and results
3. **Context Assembly**: `ContextBuilder` in `pkg/agent/context.go`
4. **Session Persistence**: After each turn via `session.SessionManager`

## Configuration Examples

### Basic Model Configuration
```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": true,
      "model_name": "claude-sonnet-4.6",
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 20
    }
  },
  "model_list": [
    {
      "model_name": "claude-sonnet-4.6",
      "model": "anthropic/claude-sonnet-4.6",
      "api_key": "sk-ant-env:ANTHROPIC_API_KEY"
    }
  ]
}
```

### Multi-Model Routing
```json
{
  "routing": {
    "enabled": true,
    "default_tier": "balanced",
    "tiers": {
      "fast": {
        "models": ["deepseek/deepseek-chat"],
        "max_cost": 0.001
      },
      "balanced": {
        "models": ["anthropic/claude-sonnet-4.6"],
        "max_cost": 0.01
      },
      "quality": {
        "models": ["openai/gpt-5.2"],
        "max_cost": 0.05
      }
    }
  }
}
```

## Troubleshooting

### Build Issues
- **Dependency Problems**: Run `make deps` to ensure dependencies are current
- **Go Version**: Requires Go 1.25.7+ - check with `go version`
- **Generate Step**: Always run `go generate ./...` before building

### Test Failures
- **Missing API Keys**: Some tests require environment variables
- **Channel Tests**: May need mock servers for testing
- **Session Tests**: Write to temporary directories

### Runtime Issues
- **Workspace Permissions**: Ensure `~/.picoclaw/workspace/` is writable
- **Config Missing**: Create `~/.picoclaw/config.json` from `config/config.example.json`
- **Provider Errors**: Check API keys and network connectivity

## Architecture Notes

### StrikeClaw Extension
PicoClaw is being extended into StrikeClaw with these additions:
- **Multi-Model Routing**: Intelligent model selection based on task type
- **MCP Integration**: Extensible tooling via Model Context Protocol
- **Workflow Engine**: Methodology-driven task execution with state tracking
- **Enhanced TUI**: Polished terminal interface using Charm libraries

### Performance Considerations
- **Binary Size**: Keep minimal - use `CGO_ENABLED=0`
- **Memory Usage**: Designed for 10MB RAM environments
- **Concurrent Operations**: Use goroutines but manage carefully
- **Caching**: Implement caching for expensive operations

### Security Principles
- **Workspace Isolation**: All operations restricted to workspace by default
- **Command Filtering**: Dangerous shell commands blocked automatically
- **Token Security**: OAuth tokens encrypted at rest
- **Input Validation**: All external inputs validated before processing

## Resources

- **Documentation**: `docs/` directory with architecture and usage guides
- **Examples**: `examples/workflows/` for common workflow patterns
- **Configuration**: `config/config.example.json` for complete reference
- **Issues**: GitHub issues and discussions
- **Community**: Discord and WeChat links in README

Remember: This is a production system used by real people. Test thoroughly, follow security best practices, and maintain backward compatibility when making changes.