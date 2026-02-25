# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

PicoClaw is an ultra-lightweight personal AI assistant written in Go, designed to run on resource-constrained hardware (10MB RAM, $10 devices). This is a fork of picoclaw being extended into **StrikeClaw** - a methodology-driven agent framework with multi-model routing, MCP tool support, and a polished terminal UI.

**Key Architecture:**
- **Agent Loop**: Think/Act/Observe pattern in `pkg/agent/loop.go`
- **Multi-provider LLM**: Abstraction layer in `pkg/providers/` supporting OpenAI, Anthropic, Zhipu, local models, etc.
- **Model Routing**: Task-based model selection in `pkg/routing/`
- **Tool System**: Extensible tool registry in `pkg/tools/` with built-in tools (file ops, shell, web search)
- **Messaging Channels**: Discord, Telegram, Slack, QQ, DingTalk, etc. in `pkg/channels/`
- **Session Management**: Conversation persistence in `pkg/session/`
- **Workspace System**: Agent workspace with AGENTS.md, SOUL.md, IDENTITY.md context files

## Build and Development Commands

```bash
# Build for current platform
make build                    # Outputs to build/picoclaw-{platform}-{arch}

# Build for all platforms
make build-all                # Cross-compile for linux/darwin/windows

# Development workflow
make deps                     # Download dependencies
make check                    # Run deps + fmt + vet + test (full pre-commit check)
make fmt                      # Format code with golangci-lint
make vet                      # Static analysis
make lint                     # Full linter (uses .golangci.yaml config)
make test                     # Run all tests

# Install locally
make install                  # Install to ~/.local/bin
make uninstall                # Remove binary
make uninstall-all            # Remove binary + workspace (~/.picoclaw)

# Run without installing
make run ARGS="agent -m 'hello'"
./build/picoclaw agent -m "What is 2+2?"
```

## Running Tests

```bash
# All tests
make test

# Specific test
go test -run TestSessionPersistence -v ./pkg/session/

# Specific package
go test -v ./pkg/agent/

# With coverage
go test -cover ./...

# Benchmarks
go test -bench=. -benchmem -run='^$' ./...
```

## Project Structure & Architecture

### Core Components

**Agent System** (`pkg/agent/`):
- `loop.go`: Main agent loop - processes messages, executes tools, manages iterations
- `instance.go`: Agent instance with workspace, sessions, context, tools
- `registry.go`: Multi-agent management - default agent + named agents
- `context.go`: Builds system prompts from workspace files (AGENTS.md, SOUL.md, etc.)

**LLM Providers** (`pkg/providers/`):
- Provider abstraction supporting multiple LLM backends
- Native implementations for Anthropic, OpenAI, Zhipu, DeepSeek, Gemini, Groq, etc.
- OpenAI-compatible wrapper for LM Studio, NVIDIA NIM, OpenRouter, Azure, Bedrock
- Fallback chain with cooldown tracking
- Load balancing across multiple API endpoints

**Routing System** (`pkg/routing/`):
- Multi-model routing based on agent configuration
- Task-based model selection (planning vs parsing vs analysis)
- Cost tracking per model tier

**Tool System** (`pkg/tools/`):
- `registry.go`: Tool discovery and invocation
- Built-in tools: `filesystem.go`, `shell.go`, `web.go`, `message.go`, `edit.go`
- `subagent.go`: Spawn subagents for async/parallel work
- `cron.go`: Scheduled task execution
- Security sandbox: workspace restriction, dangerous command blocking

**Session Management** (`pkg/session/`):
- Conversation history persistence
- Context window management
- Session resumption

**Channels** (`pkg/channels/`):
- Messaging platform integrations (Discord, Telegram, Slack, etc.)
- Webhook handlers and bot implementations
- Allow-list based access control

**Skills System** (`pkg/skills/`):
- Skill discovery, installation, execution
- Built-in and user-installed skills
- GitHub Copilot SDK integration

### Key Files

- `cmd/picoclaw/main.go`: CLI entrypoint with cobra commands
- `config/config.example.json`: Complete configuration reference
- `STRIKECLAW_ARCHITECTURE.md`: Detailed architecture plan for StrikeClaw fork
- `.golangci.yaml`: Linter configuration

### Workspace Structure

Default workspace: `~/.picoclaw/workspace/`

```
workspace/
├── sessions/          # Conversation history
├── memory/           # Long-term memory (MEMORY.md)
├── state/            # Persistent state
├── cron/             # Scheduled jobs database
├── skills/           # Custom skills
├── AGENTS.md         # Agent behavior guide
├── HEARTBEAT.md      # Periodic task prompts (checked every 30min)
├── IDENTITY.md       # Agent identity
├── SOUL.md           # Agent soul/personality
├── TOOLS.md          # Tool descriptions
└── USER.md           # User preferences
```

## Development Guidelines

### Code Organization

- **Package structure**: Follow existing `pkg/` organization
- **Minimal dependencies**: Keep binary size small - avoid heavy dependencies
- **Provider abstraction**: New LLM providers implement `LLMProvider` interface
- **Tool registration**: Register tools in agent instance or loop initialization
- **Security-first**: All file/exec tools check workspace boundaries

### Security Considerations

**Critical security patterns:**
- File path operations MUST check workspace boundaries (see `tools/filesystem.go`)
- Shell execution MUST block dangerous commands (see `tools/shell.go`)
- Channel handlers MUST validate allow_from lists
- OAuth tokens stored in `pkg/auth/store.go` with proper encryption
- Never commit credentials - use `~/.picoclaw/config.json` or environment variables

**Recent security fixes:**
- Commit `244eb0b`: Fixed path traversal in file operations
- Commit `740cdca`: Removed redundant tool definitions from system prompt

### Testing Expectations

- Tests are located alongside implementation files (`*_test.go`)
- Use `testify` for assertions (`github.com/stretchr/testify/assert`)
- Mock interfaces for provider/channel testing
- Focus on edge cases: empty inputs, error conditions, concurrent access

### AI-Generated Code

This project embraces AI-assisted development (see CONTRIBUTING.md). When working with AI-generated code:
- **Read and understand** every line before committing
- **Test in real environment** - don't rely on AI's correctness claims
- **Security review** - AI often generates insecure patterns (path traversal, injection)
- **Disclose AI involvement** in PR descriptions

## Configuration

**Primary config file:** `~/.picoclaw/config.json`

**Key configuration sections:**
- `agents.defaults`: Default agent settings (workspace, model, temperature, iterations)
- `model_list`: Model definitions with vendor/model format (e.g., `anthropic/claude-sonnet-4.6`)
- `providers`: Legacy provider configuration (deprecated, use `model_list`)
- `channels`: Messaging platform credentials and settings
- `tools.web`: Web search API keys (Brave, Tavily, DuckDuckGo)
- `heartbeat`: Periodic task configuration

**Environment variables:**
- `PICOCLAW_*`: Config overrides (e.g., `PICOCLAW_AGENTS_DEFAULTS_WORKSPACE`)
- API keys can use `_env` suffix in config to reference env vars

## Common Tasks

### Adding a New LLM Provider

1. Implement `LLMProvider` interface in `pkg/providers/`
2. Add vendor prefix to model routing in `pkg/routing/`
3. Update config schema in `pkg/config/config.go`
4. Add example to `config/config.example.json`
5. Test with: `picoclaw provider test`

### Adding a New Tool

1. Create tool struct in `pkg/tools/`
2. Implement `Tool` interface (GetDefinition, Execute)
3. Register in agent instance or shared tool registration
4. Add to tool registry in `registerSharedTools()` if global

### Adding a New Channel

1. Create channel implementation in `pkg/channels/`
2. Implement channel interface (Start, Stop, message handling)
3. Add config struct in `pkg/config/config.go`
4. Register in channel manager
5. Add setup docs in README.md

### Modifying the Agent Loop

**Be careful**: The agent loop is the core execution path. Changes here affect all interactions.

- Entry point: `AgentLoop.ProcessMessage()` in `pkg/agent/loop.go`
- Tool execution: `executeTool()` handles tool calls and results
- Context assembly: `ContextBuilder` in `pkg/agent/context.go`
- Session persistence: After each turn via `session.SessionManager`

## StrikeClaw Fork Development

**Active development area**: Extending picoclaw → StrikeClaw per `STRIKECLAW_ARCHITECTURE.md`

**Key additions planned:**
- Multi-model routing with task classification (`internal/llm/router.go`)
- MCP client for extensible tooling (`internal/mcp/`)
- Workflow engine with methodology state tracking (`internal/workflow/`)
- Charm-based TUI (Bubble Tea, Lip Gloss, Glamour) (`internal/tui/`)

**When working on StrikeClaw features:**
- Keep picoclaw compatibility - changes should be additive
- New packages go in `internal/` not `pkg/`
- Configuration backward compatibility required
- Refer to architecture doc before implementing

## Troubleshooting

**Build issues:**
- Run `make deps` to ensure dependencies are up to date
- Check Go version: requires Go 1.25.7+
- CGO is disabled: `CGO_ENABLED=0`

**Test failures:**
- Some tests require API keys in environment
- Channel tests may need mock servers
- Session tests write to temp directories

**Runtime issues:**
- Check workspace permissions: `~/.picoclaw/workspace/` must be writable
- Verify config file exists: `~/.picoclaw/config.json`
- Enable debug logging: Set log level in config or via environment

## Git Workflow

- **Main branch**: `main` - active development
- **Release branches**: `release/x.y` - stable releases
- **Merge strategy**: Squash merge for most PRs
- **Protected branches**: Direct pushes not permitted
- **PR requirements**: CI passing, maintainer approval, template complete

## Resources

- Original inspiration: [nanobot](https://github.com/HKUDS/nanobot)
- Official website: [picoclaw.io](https://picoclaw.io)
- Community: Discord, WeChat (links in README.md)
- Issues and discussions: GitHub
