# AGENTS.md - StrikeClaw/PicoClaw Development Guide

This document provides essential information for AI agents working effectively in the StrikeClaw/PicoClaw codebase.

## Project Overview

StrikeClaw is a methodology-driven AI agent framework forked from PicoClaw. It's built in Go and provides:

- **Workflow Engine**: Multi-phase methodology execution with state persistence
- **Tier-Based Model Routing**: Route tasks to different models by complexity (heavy/medium/light)
- **Local Model Support**: LM Studio, Ollama, and OpenAI-compatible endpoint support
- **Terminal UI**: Charm/Bubble Tea-based TUI with chat, mission progress, and cost tracking
- **19 Built-in Tools**: File operations, shell execution, web search, hardware access, etc.
- **Multi-Channel Support**: Telegram, Discord, Slack, WeChat, WhatsApp, and more

## Essential Commands

### Build & Development
```bash
# Build for current platform
make build

# Build for all platforms  
make build-all

# Install to ~/.local/bin
make install

# Run tests
make test

# Run linting
make lint

# Fix linting issues
make fix

# Format code
make fmt

# Run static analysis
make vet

# Complete check (deps + fmt + vet + test)
make check

# Clean build artifacts
make clean

# Run the application
make run ARGS="agent --help"
```

### Development Workflow
```bash
# Generate embedded files (required before build)
make generate

# Update dependencies
make update-deps

# Download and verify dependencies
make deps
```

## Code Organization

### Core Structure
```
pkg/
├── agent/           # Agent loop and registry
├── config/          # Configuration management
├── providers/       # LLM provider implementations
├── tools/           # Built-in tool implementations
├── session/         # Session management and persistence
├── routing/         # Tier-based model routing
├── workflow/       # Workflow engine
├── channels/        # Communication channel handlers
├── skills/          # Skill discovery and installation
├── tui/             # Terminal UI components
├── bus/             # Message bus for inter-component communication
├── state/           # State management
├── logger/          # Structured logging
├── utils/           # Utility functions
└── constants/       # Application constants
```

### CLI Structure
```
cmd/picoclaw/
├── main.go                    # Main entry point
└── internal/
    ├── agent/                 # Agent command logic
    ├── auth/                  # Authentication commands
    ├── config/                # Configuration commands
    ├── skills/                # Skill management
    ├── gateway/               # Gateway server
    ├── cron/                  # Scheduled tasks
    ├── status/                # Status commands
    ├── migrate/               # Migration utilities
    ├── onboard/               # Onboarding wizard
    ├── version/               # Version info
    └── helpers.go            # Shared utilities
```

## Configuration Patterns

### Configuration Loading
Configuration is loaded via `internal.LoadConfig()` which:
1. Reads from `~/.picoclaw/config.json`
2. Applies environment variables with `PICOCLAW_` prefix
3. Supports model-centric configuration via `model_list`

### Key Configuration Structures
```go
type Config struct {
    Agents    AgentsConfig    `json:"agents"`
    ModelList []ModelConfig   `json:"model_list"`
    Routing   RoutingConfig   `json:"routing,omitempty"`
    Channels  ChannelsConfig  `json:"channels"`
    Tools     ToolsConfig     `json:"tools"`
    // ... other fields
}
```

### Model Configuration
- **Primary Model**: `cfg.Agents.Defaults.ModelName`
- **Model List**: Array of `ModelConfig` with provider details
- **Tier Routing**: `cfg.Routing.Enabled` determines multi-model mode
- **Fallback Support**: Models can have fallbacks defined

### Environment Variables
Use `PICOCLAW_` prefix with nested structure separated by underscores:
```bash
PICOCLAW_AGENTS_DEFAULTS_MODEL_NAME=claude-sonnet
PICOCLAW_ROUTING_ENABLED=true
```

## Naming Conventions

### Go Code
- **Packages**: `lowercase`, descriptive (`agentloop`, `sessionmanager`)
- **Functions**: `CamelCase`, exported start with capital (`ProcessDirect`, `GetStartupInfo`)
- **Variables**: `camelCase` for private, `CamelCase` for exported
- **Interfaces**: Simple `-er` suffix (`LLMProvider`, `ToolRegistry`)
- **Constants**: `SCREAMING_SNAKE_CASE` (`DEFAULT_MODEL`, `MAX_TOKENS`)

### Configuration
- **JSON Keys**: `snake_case`
- **Environment Variables**: `SCREAMING_SNAKE_CASE` with `PICOCLAW_` prefix
- **Model Names**: `kebab-case` (`claude-sonnet`, `gpt-4`)

### File Organization
- **Command Files**: `command.go` + `helpers.go`
- **Test Files**: `*_test.go` alongside implementation
- **Example Configs**: `config.example.json`, `config.tier-routing.example.json`

## Key Patterns

### Agent Loop Pattern
The main agent execution follows this pattern:
1. Load configuration via `internal.LoadConfig()`
2. Create provider via `providers.CreateProvider(cfg)`
3. Create agent loop via `agent.NewAgentLoop(cfg, msgBus, provider)`
4. Process messages via `agentLoop.ProcessDirect(ctx, message, sessionKey)`

### Tool Registration Pattern
Tools are registered in the agent registry:
```go
func registerSharedTools(cfg *config.Config, msgBus *bus.MessageBus, 
    registry *AgentRegistry, provider providers.LLMProvider) {
    // Tools are registered with the registry for all agents
}
```

### Provider Pattern
LLM providers implement the `LLMProvider` interface:
```go
type LLMProvider interface {
    Chat(ctx context.Context, messages []Message, tools []ToolDefinition, 
        model string, options map[string]any) (*LLMResponse, error)
    GetDefaultModel() string
}
```

### Session Management Pattern
Sessions are managed by `SessionManager`:
```go
type SessionManager struct {
    sessions map[string]*Session
    mu       sync.RWMutex
    storage  string // Persistent storage path
}
```

## Testing Approach

### Test Structure
- **Unit Tests**: Located in `*_test.go` files alongside implementation
- **Integration Tests**: Separate files with `_integration_test.go` suffix
- **Mock Providers**: Use `mock_provider_test.go` for testing without real LLM calls

### Test Running
```bash
# Run all tests
make test

# Run specific package tests
go test ./pkg/agent

# Run with verbose output
go test -v ./pkg/agent

# Run integration tests
go test -tags=integration ./pkg/providers
```

### Test Patterns
- **Table-Driven Tests**: Common pattern for testing multiple scenarios
- **Mock Interfaces**: Use interface mocks to isolate dependencies
- **Test Helpers**: Shared utilities in `test_helpers.go` files
- **Integration Tests**: Use build tags to separate from unit tests

## Important Gotchas

### Current Known Issues
1. **Missing Import**: `pkg/session/manager.go` is missing `"fmt"` import (line 301)
   - **Fix**: Add `"fmt"` to import block
   - **Impact**: Compilation fails until fixed

2. **Unused Functions**: Several parsing functions in `pkg/tools/i2c.go` and `pkg/tools/spi.go` are unused
   - **Files**: `parseI2CAddress`, `parseI2CBus`, `parseSPIArgs`
   - **Impact**: Linter warnings, but no functional impact

3. **Unused Parameters**: Some cross-platform stub functions have unused parameters
   - **Files**: `pkg/tools/i2c_other.go`, `pkg/tools/spi_other.go`
   - **Impact**: Linter warnings, but intentional for cross-platform compatibility

### Configuration Gotchas
1. **Model Field Name**: Use `ModelName` not `Model` in agent defaults
   - **Location**: `cfg.Agents.Defaults.ModelName`
   - **Reason**: `Model` field is deprecated but kept for backward compatibility

2. **Provider Migration**: Old `providers` section is deprecated
   - **New Way**: Use `model_list` array
   - **Migration**: Run built-in migration tools

3. **Environment Variables**: Must use `PICOCLAW_` prefix
   - **Example**: `PICOCLAW_AGENTS_DEFAULTS_MODEL_NAME`
   - **Gotcha**: Nested structure uses underscores, not dots

### Build Gotchas
1. **Required Generate Step**: Must run `make generate` before build
   - **Reason**: Embedded workspace files need generation
   - **Impact**: Build will fail without generated files

2. **CGO Disabled**: Build uses `CGO_ENABLED=0`
   - **Reason**: Create minimal, portable binaries
   - **Impact**: Cannot use CGO-dependent packages

3. **Go Version**: Requires Go 1.25.7+
   - **Check**: `go version` in Makefile
   - **Impact**: Will fail to build with older Go versions

### Runtime Gotchas
1. **Session Auto-Fresh**: Workflows create automatic session keys
   - **Pattern**: `cli:workflow_{name}_{timestamp}`
   - **Reason**: Avoid history pollution between workflow runs

2. **Tool Execution**: Tools run directly without permission prompts
   - **Reason**: Autonomous execution by design
   - **Risk**: Ensure tool deny patterns are properly configured

3. **Tier Routing**: Only enabled when `cfg.Routing.Enabled == true`
   - **Check**: Multiple model mode is determined by routing enabled flag
   - **Models**: Must have models defined in both `model_list` and routing tiers

## Development Workflow

### Making Changes
1. **Understand the Area**: Check related files in the same directory
2. **Follow Patterns**: Use existing code patterns for consistency
3. **Add Tests**: Create or update tests for new functionality
4. **Check Imports**: Ensure all necessary imports are included
5. **Run Linting**: Use `make lint` to catch style issues
6. **Test Build**: Run `make build` to verify compilation
7. **Run Tests**: Execute `make test` to ensure functionality

### Adding New Tools
1. **Implement Tool Interface**: Create in `pkg/tools/`
2. **Register Tool**: Add to `registerSharedTools()` in agent loop
3. **Add Configuration**: If needed, add to `cfg.Tools`
4. **Write Tests**: Create comprehensive test coverage
5. **Update Documentation**: Add to README or docs if user-facing

### Adding New Providers
1. **Implement LLMProvider**: Create in `pkg/providers/`
2. **Add to Factory**: Update `provider_factory.go`
3. **Add Config**: Support in configuration structure
4. **Test Integration**: Use integration tests with real API
5. **Document**: Add setup instructions to relevant docs

### Configuration Changes
1. **Backward Compatibility**: Keep old fields for migration
2. **Environment Support**: Add `env:` tags to struct fields
3. **Validation**: Add validation in config loading
4. **Migration**: Provide migration tools for breaking changes
5. **Examples**: Update example configurations

## Project-Specific Context

### Architecture Decisions
- **Agent-Centric**: Everything revolves around agent execution
- **Tool-Based**: Capabilities exposed through tools, not direct functions
- **Configuration-Driven**: Behavior controlled by config, not code
- **Multi-Model**: Designed to work with multiple LLM providers
- **Channel-Agnostic**: Core logic separate from communication channels

### Design Principles
- **Lightweight**: Minimal dependencies and overhead
- **Autonomous**: Tools execute without confirmation
- **Methodology-Driven**: Workflows guide agent behavior
- **Cost-Optimized**: Tier routing reduces expensive model usage
- **Extensible**: Easy to add new tools and providers

### Key Dependencies
- **Cobra**: CLI framework
- **Charm (Bubble Tea/Lip Gloss)**: Terminal UI
- **Anthropic SDK**: Claude API integration
- **OpenAI SDK**: OpenAI-compatible API support
- **Various Channel SDKs**: Telegram, Discord, Slack, etc.

This guide covers the essential information needed to work effectively in the StrikeClaw/PicoClaw codebase. Always refer to existing code patterns and tests when implementing new functionality.