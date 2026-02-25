# StrikeClaw — Architecture & Development Plan

> Picoclaw fork with multi-model routing, MCP tool support, methodology-driven workflows, and a Charm-powered TUI.

---

## 1. Project Summary

**StrikeClaw** is a general-purpose autonomous agent framework forked from [picoclaw](https://github.com/sipeed/picoclaw). It inherits picoclaw's lightweight Go architecture, messaging integrations (Discord, Telegram, Slack, etc.), session persistence, and workspace system — then adds the capabilities no existing framework provides in combination:

| Capability | Source |
|-----------|--------|
| Agent loop, tools, messaging, sessions, workspace | Picoclaw (inherited) |
| Multi-provider LLM support with task-based routing | New (inspired by PentestGPT CCR + your harness) |
| MCP client for extensible tooling | New (inspired by PentestAgent pattern) |
| Methodology/workflow engine with state tracking | New |
| Polished terminal UI | New (Charm libraries: Bubble Tea, Lip Gloss, Glamour) |

**Not a framework.** A single binary that runs autonomous, multi-step workflows against any task domain — security, CI/CD, data analysis, devops — routing each subtask to the most cost-effective model.

---

## 2. Core Objectives (Scope Guard)

Everything in this plan serves exactly these objectives. If a feature doesn't advance one of them, it's out of scope.

| # | Objective | Acceptance Criteria |
|---|-----------|-------------------|
| O1 | **Multi-model routing** | Agent automatically selects model tier per task type. Config-driven. Supports: LM Studio, OpenRouter, NVIDIA NIM, Anthropic, OpenAI, Azure, Bedrock. |
| O2 | **Workflow enforcement** | Agent follows user-defined multi-phase methodologies with branching. State tracked in files. Phases have completion criteria. |
| O3 | **Autonomous tool chaining** | 50+ iteration loops. No approval gates. Deep chaining based on model analysis of results. |
| O4 | **MCP tool extensibility** | Load tools from MCP servers (stdio, http, SSE). Bridge into agent tool registry alongside built-in tools. |
| O5 | **Polished TUI** | Charm-based terminal UI showing: active model/tier, current workflow phase, tool execution stream, mission state. Professional enough for company demos. |
| O6 | **Messaging control** | Inherited from picoclaw. Discord/Telegram for remote monitoring, findings alerts, and basic commands. |

**Explicitly out of scope for v1:** Multi-agent orchestration, web UI, fine-tuning, training, browser automation.

---

## 3. Architecture

### 3.1 High-Level

```
┌─────────────────────────────────────────────────────────────────┐
│                        StrikeClaw Binary                         │
│                                                                  │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌────────────────┐  │
│  │   TUI    │  │ Gateway  │  │  Agent   │  │    CLI/Batch   │  │
│  │(Charm BT)│  │(Discord/ │  │   Loop   │  │  (one-shot or  │  │
│  │          │  │ Telegram)│  │          │  │   piped input) │  │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └──────┬─────────┘  │
│       └──────────────┴──────┬─────┘               │            │
│                             │                      │            │
│                    ┌────────▼────────┐              │            │
│                    │  Agent Core     │◄─────────────┘            │
│                    │                 │                           │
│                    │  - Think/Act/   │                           │
│                    │    Observe loop │                           │
│                    │  - Session mgmt │                           │
│                    │  - Tool dispatch│                           │
│                    └───┬────┬────┬───┘                           │
│                        │    │    │                               │
│           ┌────────────┘    │    └────────────┐                  │
│           ▼                 ▼                  ▼                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐       │
│  │ LLM Router   │  │ Tool Registry│  │ Workflow Engine   │       │
│  │              │  │              │  │                   │       │
│  │ Classify task│  │ Built-in:    │  │ Parse methodology │       │
│  │ Select tier  │  │  - exec      │  │ Track phases      │       │
│  │ Call provider│  │  - file ops  │  │ Manage branches   │       │
│  │              │  │  - web       │  │ Enforce completion│       │
│  │ Providers:   │  │ MCP bridge:  │  │                   │       │
│  │  - Anthropic │  │  - stdio     │  │ State persisted   │       │
│  │  - OpenAI    │  │  - http      │  │ in mission files  │       │
│  │  - LM Studio │  │  - sse       │  │                   │       │
│  │  - NIM       │  │              │  │                   │       │
│  │  - OpenRouter│  │              │  │                   │       │
│  │  - Azure     │  │              │  │                   │       │
│  │  - Bedrock   │  │              │  │                   │       │
│  └──────────────┘  └──────────────┘  └──────────────────┘       │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 Directory Structure

```
strikeclaw/
├── cmd/strikeclaw/
│   └── main.go                    # Entrypoint, CLI commands
│
├── internal/
│   ├── agent/
│   │   ├── loop.go                # MODIFIED: core think/act/observe loop
│   │   ├── context.go             # MODIFIED: context assembly from session + workflow state
│   │   └── session.go             # INHERITED: session persistence
│   │
│   ├── llm/
│   │   ├── router.go              # NEW: task classifier → model tier → provider dispatch
│   │   ├── provider.go            # NEW: provider interface (replaces picoclaw's single-provider)
│   │   ├── anthropic.go           # NEW: native Anthropic Messages API client
│   │   ├── openai_compat.go       # MODIFIED: OpenAI-compatible (LM Studio, NIM, OpenRouter, Azure, Bedrock)
│   │   └── config.go              # NEW: routing config schema
│   │
│   ├── mcp/
│   │   ├── client.go              # NEW: MCP client (stdio, http, SSE transports)
│   │   ├── registry.go            # NEW: discover + register MCP tools into agent tool set
│   │   └── config.go              # NEW: mcp_servers.json loader
│   │
│   ├── workflow/
│   │   ├── engine.go              # NEW: methodology state machine
│   │   ├── parser.go              # NEW: parse workflow .md files into phase/branch structures
│   │   └── state.go               # NEW: MISSION.md + state.json read/write
│   │
│   ├── tools/
│   │   ├── exec.go                # INHERITED: shell execution
│   │   ├── file.go                # INHERITED: file read/write/edit
│   │   ├── web.go                 # INHERITED: web search
│   │   ├── message.go             # INHERITED: messaging
│   │   └── mcp_bridge.go          # NEW: adapts MCP tool results to agent tool interface
│   │
│   ├── tui/
│   │   ├── app.go                 # NEW: Bubble Tea main model
│   │   ├── views/
│   │   │   ├── chat.go            # NEW: conversation/output stream view
│   │   │   ├── status.go          # NEW: status bar (model, phase, cost, elapsed)
│   │   │   ├── mission.go         # NEW: mission/workflow state panel
│   │   │   └── tools.go           # NEW: tool execution log view
│   │   ├── styles.go              # NEW: Lip Gloss theme definitions
│   │   └── keys.go                # NEW: keybinding definitions
│   │
│   └── channels/
│       ├── discord.go             # INHERITED
│       ├── telegram.go            # INHERITED
│       └── ...                    # Other channels inherited from picoclaw
│
├── workspace/                     # Default workspace template
│   ├── AGENTS.md
│   ├── SOUL.md
│   ├── TOOLS.md
│   ├── IDENTITY.md
│   ├── USER.md
│   ├── HEARTBEAT.md
│   ├── workflows/                 # NEW: workflow definition templates
│   │   └── example.md
│   └── missions/                  # NEW: active mission state (auto-created)
│
├── config.example.json            # Full config reference
├── go.mod
├── go.sum
├── Makefile
├── Dockerfile
└── README.md
```

### 3.3 What Changes vs Picoclaw

| Area | Change Type | Details |
|------|-------------|---------|
| `internal/llm/` | **Rewrite** | Picoclaw has a single LLM client. Replace with provider abstraction + router. |
| `internal/mcp/` | **New package** | MCP client doesn't exist in picoclaw. Build from scratch. |
| `internal/workflow/` | **New package** | Workflow engine is entirely new. |
| `internal/tui/` | **New package** | Picoclaw's TUI is minimal. Replace with Charm-based UI. |
| `internal/agent/loop.go` | **Modify** | Insert routing + workflow hooks into the existing agent loop. |
| `internal/agent/context.go` | **Modify** | Add workflow state and mission context to prompt assembly. |
| `internal/tools/` | **Extend** | Add `mcp_bridge.go`. Existing tools untouched. |
| `internal/channels/` | **Inherit** | No changes. Discord/Telegram/Slack work as-is. |
| `cmd/strikeclaw/main.go` | **Modify** | Add new CLI subcommands: `mission`, `workflow`, `tui`. |
| `workspace/` | **Extend** | Add `workflows/` and `missions/` directories to template. |

---

## 4. Component Designs

### 4.1 LLM Router

The router classifies each agent turn by task type, then dispatches to the configured model tier.

**Config:**

```json
{
  "routing": {
    "tiers": {
      "heavy": {
        "provider": "anthropic",
        "model": "claude-sonnet-4-5-20250929",
        "use_for": ["planning", "analysis", "exploitation", "report_writing"]
      },
      "medium": {
        "provider": "lm_studio",
        "model": "qwen3-32b",
        "api_base": "http://localhost:1234/v1",
        "use_for": ["tool_selection", "code_review", "js_analysis"]
      },
      "light": {
        "provider": "nvidia_nim",
        "model": "meta/llama-3.1-8b-instruct",
        "api_base": "https://integrate.api.nvidia.com/v1",
        "api_key_env": "NIM_API_KEY",
        "use_for": ["output_parsing", "summarization", "formatting"]
      }
    },
    "default_tier": "heavy",
    "fallback_provider": {
      "provider": "openrouter",
      "model": "anthropic/claude-sonnet-4-5",
      "api_key_env": "OPENROUTER_API_KEY"
    }
  }
}
```

**Classification logic** (rule-based, no LLM call needed):

```go
// internal/llm/router.go

// ClassifyTask determines the task type from the current agent context.
// Rule-based — deterministic and zero-cost.
func (r *Router) ClassifyTask(ctx *AgentContext) TaskType {
    // Just received a tool result with large output?
    if ctx.LastToolOutput != "" && len(ctx.LastToolOutput) > 2000 {
        return TaskOutputParsing
    }
    // Starting a new phase or beginning of session?
    if ctx.WorkflowPhaseChanged || ctx.TurnCount == 0 {
        return TaskPlanning
    }
    // Model explicitly requested to write a report?
    if ctx.UserRequestedReport {
        return TaskReportWriting
    }
    // Default: let the heavy model handle reasoning
    return TaskAnalysis
}
```

**Provider interface:**

```go
// internal/llm/provider.go

// Provider is the interface all LLM backends implement.
type Provider interface {
    // Chat sends a completion request and returns the response.
    Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
    // ChatWithTools sends a request with tool schemas for function calling.
    ChatWithTools(ctx context.Context, req *ChatRequest, tools []ToolSchema) (*ChatResponse, error)
    // Name returns the provider identifier for logging.
    Name() string
}

// Implementations:
// - AnthropicProvider: native Messages API (system param, content blocks, tool_use)
// - OpenAICompatProvider: covers OpenRouter, LM Studio, NIM, Azure, Bedrock
//   Configured via api_base + api_key. Provider-specific quirks handled by flags:
//   - nim_thinking: adds chat_template_kwargs for NIM reasoning models
//   - azure_api_version: adds api-version query param
//   - bedrock_signing: AWS SigV4 request signing
```

### 4.2 MCP Client

Implements the client side of the Model Context Protocol to discover and invoke tools from MCP servers.

**Config:**

```json
{
  "mcp": {
    "servers": {
      "nmap": {
        "transport": "stdio",
        "command": "npx",
        "args": ["-y", "gc-nmap-mcp"],
        "env": {"NMAP_PATH": "/usr/bin/nmap"}
      },
      "github": {
        "transport": "http",
        "url": "https://mcp.github.com/sse"
      },
      "custom_tools": {
        "transport": "stdio",
        "command": "./my-mcp-server",
        "args": ["--workspace", "~/.strikeclaw/workspace"]
      }
    }
  }
}
```

**Architecture:**

```go
// internal/mcp/client.go

// Client manages connections to MCP servers and exposes their tools.
type Client struct {
    servers map[string]*ServerConnection
}

// DiscoverTools connects to all configured servers, runs tools/list,
// and returns tool schemas compatible with the agent's tool registry.
func (c *Client) DiscoverTools() ([]tools.Schema, error) { ... }

// InvokeTool calls a specific tool on the appropriate MCP server.
// Handles JSON-RPC framing, timeout, and error mapping.
func (c *Client) InvokeTool(serverName, toolName string, args map[string]any) (*tools.Result, error) { ... }

// internal/tools/mcp_bridge.go

// Bridge adapts MCP tools into the agent's native tool interface.
// Each MCP tool becomes a regular tool the agent can call via function calling.
func RegisterMCPTools(registry *tools.Registry, mcpClient *mcp.Client) error {
    schemas, err := mcpClient.DiscoverTools()
    // For each MCP tool, register a wrapper that:
    // 1. Receives args from the LLM's function call
    // 2. Invokes the MCP server
    // 3. Returns the result in the agent's expected format
    // 4. Logs the call for session persistence
    ...
}
```

### 4.3 Workflow Engine

Parses user-defined methodology documents and tracks execution state.

**Workflow definition** (`workflows/security_pentest.md`):

```markdown
---
name: web-pentest
description: Web application penetration testing methodology
phases: [recon, web_analysis, api_testing, vuln_scanning, validation]
---

## Phase: recon
### Steps
- subdomain_enumeration: Run subfinder/amass against target domain
- dns_resolution: Resolve all discovered subdomains to IPs
- port_scanning: nmap service detection on all live hosts
- http_probing: httpx with tech detection on all web ports

### Completion Criteria
All discovered subdomains port-scanned. All web services HTTP-probed.

### Branches On Discovery
- web_service_found → web_analysis
- graphql_endpoint → api_testing
- outdated_software → vuln_scanning (targeted)
- credentials_found → validate immediately

## Phase: web_analysis
...
```

**State tracking** (`missions/{id}/state.json`):

```json
{
  "mission_id": "target-2026-02-25",
  "workflow": "web-pentest",
  "current_phase": "web_analysis",
  "phases": {
    "recon": {
      "status": "complete",
      "steps": {
        "subdomain_enumeration": {"status": "done", "summary": "Found 12 subdomains"},
        "dns_resolution": {"status": "done", "summary": "8 live hosts"},
        "port_scanning": {"status": "done", "summary": "See discoveries.md"},
        "http_probing": {"status": "done", "summary": "5 web services identified"}
      }
    },
    "web_analysis": {
      "status": "in_progress",
      "steps": {
        "crawling": {"status": "in_progress", "summary": "Crawling api.target.com"}
      }
    }
  },
  "branches": [
    {
      "id": "BRANCH-001",
      "trigger": "GraphQL at api.target.com:443/graphql",
      "target_phase": "api_testing",
      "status": "pending",
      "priority": "high"
    }
  ],
  "findings_count": {"critical": 0, "high": 1, "medium": 2, "low": 0, "info": 3},
  "cost_usd": 0.47,
  "elapsed_minutes": 34
}
```

**Engine interface:**

```go
// internal/workflow/engine.go

// Engine manages workflow state and injects phase awareness into agent context.
type Engine struct {
    definition *WorkflowDef   // Parsed from .md
    state      *MissionState  // Loaded from state.json
}

// InjectContext returns the workflow-relevant context string to prepend
// to the agent's system prompt. Includes: current phase, pending steps,
// open branches, completion criteria, and discovered knowledge summary.
func (e *Engine) InjectContext() string { ... }

// Update processes the agent's latest action and updates state:
// - Mark steps complete
// - Create new branches from discoveries
// - Transition phases when criteria are met
func (e *Engine) Update(action *AgentAction, result *tools.Result) error { ... }

// IsExhausted returns true when all viable paths are explored.
func (e *Engine) IsExhausted() bool { ... }

// SuggestNext returns the highest-priority pending action for the agent.
func (e *Engine) SuggestNext() string { ... }
```

### 4.4 Charm TUI

Built on Bubble Tea (TUI framework), Lip Gloss (styling), and Glamour (markdown rendering). **Not a fork of Crush** — we use the libraries directly.

**Layout:**

```
┌─ StrikeClaw ──────────────────────────────────────────────────┐
│ Mission: target-2026-02-25  │  Phase: web_analysis  │  $0.47  │
│ Model: claude-sonnet-4-5    │  Tools: 14 (3 MCP)    │  34m    │
├───────────────────────────────────────────────────────────────┤
│                                                               │
│  [planning → claude-sonnet-4-5]                               │
│  Based on recon findings, api.target.com has a GraphQL        │
│  endpoint at /graphql. Running introspection query...         │
│                                                               │
│  ▶ exec: curl -s -X POST https://api.target.com/graphql      │
│    -H 'Content-Type: application/json'                        │
│    -d '{"query":"{__schema{types{name}}}"}'                   │
│                                                               │
│  [output_parsing → llama-3.1-8b]                              │
│  Introspection returned 47 types. Notable: User, AdminConfig, │
│  FileUpload, InternalAPI. The AdminConfig type has mutations   │
│  for updateSystemSetting and resetUserPassword.               │
│                                                               │
│  [analysis → claude-sonnet-4-5]                               │
│  HIGH PRIORITY: AdminConfig.resetUserPassword mutation is      │
│  accessible without admin role check. Testing IDOR...         │
│                                                               │
├───────────────────────────────────────────────────────────────┤
│ Branches: BRANCH-001 GraphQL (active) │ BRANCH-002 nginx CVE │
│ Findings: 1 HIGH │ 2 MED │ 3 INFO    │ Phase: 2/5 complete   │
├───────────────────────────────────────────────────────────────┤
│ > Type a message or press [q]uit [p]ause [r]eport [b]ranches │
└───────────────────────────────────────────────────────────────┘
```

**Key components:**

```go
// internal/tui/app.go

import (
    tea "github.com/charmbracelet/bubbletea/v2"
    "github.com/charmbracelet/lipgloss/v2"
    "github.com/charmbracelet/glamour"
)

// App is the top-level Bubble Tea model.
type App struct {
    agent      *agent.Core
    chatView   *views.ChatView      // Scrollable message stream
    statusBar  *views.StatusBar     // Model, phase, cost, time
    missionBar *views.MissionBar    // Branches, findings summary
    inputField textinput.Model      // User input at bottom
    width      int
    height     int
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "q", "ctrl+c":
            return a, tea.Quit
        case "p":
            a.agent.TogglePause()
        case "r":
            a.agent.GenerateReport()
        case "b":
            a.chatView.ShowBranches()
        }
    case AgentOutputMsg:
        // Streaming output from agent loop — append to chat view
        a.chatView.Append(msg)
    case ModelSwitchMsg:
        // Router selected a different model — update status bar
        a.statusBar.SetModel(msg.Model, msg.Tier)
    case WorkflowUpdateMsg:
        // Phase/branch changed — update mission bar
        a.missionBar.Update(msg.State)
    case FindingMsg:
        // New finding confirmed — flash notification
        a.missionBar.AddFinding(msg.Finding)
    }
    ...
}

// internal/tui/styles.go

var (
    // Theme: dark background with muted accent colors
    HeaderStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("#7D56F4")).
        Background(lipgloss.Color("#1a1a2e"))

    ToolCallStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#50fa7b")).
        PaddingLeft(2).
        SetString("▶")

    TierHeavyStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#ff79c6"))

    TierLightStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#8be9fd"))

    FindingCritical = lipgloss.NewStyle().
        Bold(true).
        Foreground(lipgloss.Color("#ff5555"))

    FindingHigh = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#ffb86c"))
)
```

**Glamour for markdown rendering:**

The agent's responses and findings are rendered as markdown in the chat view using Glamour, giving you proper syntax highlighting, tables, and formatting without any extra work.

---

## 5. Config Schema (Complete)

```json
{
  "$schema": "https://strikeclaw.dev/config.json",

  "agents": {
    "defaults": {
      "workspace": "~/.strikeclaw/workspace",
      "max_tokens": 8192,
      "temperature": 0.1,
      "max_tool_iterations": 50,
      "restrict_to_workspace": false
    }
  },

  "routing": {
    "enabled": true,
    "classifier": "rule_based",
    "tiers": {
      "heavy": {
        "provider": "anthropic",
        "model": "claude-sonnet-4-5-20250929",
        "use_for": ["planning", "analysis", "exploitation", "report_writing"]
      },
      "medium": {
        "provider": "lm_studio",
        "model": "qwen3-32b",
        "use_for": ["tool_selection", "code_review", "js_analysis"]
      },
      "light": {
        "provider": "nvidia_nim",
        "model": "meta/llama-3.1-8b-instruct",
        "use_for": ["output_parsing", "summarization", "formatting"]
      }
    },
    "default_tier": "heavy",
    "fallback": {
      "provider": "openrouter",
      "model": "anthropic/claude-sonnet-4-5"
    }
  },

  "providers": {
    "anthropic": {
      "api_key_env": "ANTHROPIC_API_KEY"
    },
    "openai": {
      "api_key_env": "OPENAI_API_KEY"
    },
    "openrouter": {
      "api_key_env": "OPENROUTER_API_KEY",
      "api_base": "https://openrouter.ai/api/v1"
    },
    "lm_studio": {
      "api_base": "http://localhost:1234/v1"
    },
    "nvidia_nim": {
      "api_key_env": "NIM_API_KEY",
      "api_base": "https://integrate.api.nvidia.com/v1",
      "extra_body": {
        "_comment": "For NIM thinking models that need chat_template_kwargs",
        "chat_template_kwargs": {"enable_thinking": true}
      }
    },
    "azure_openai": {
      "api_key_env": "AZURE_OPENAI_API_KEY",
      "api_base": "https://your-resource.openai.azure.com",
      "api_version": "2024-12-01-preview"
    },
    "bedrock": {
      "region": "us-east-1",
      "access_key_env": "AWS_ACCESS_KEY_ID",
      "secret_key_env": "AWS_SECRET_ACCESS_KEY"
    }
  },

  "mcp": {
    "servers": {
      "example_stdio": {
        "transport": "stdio",
        "command": "npx",
        "args": ["-y", "some-mcp-server"],
        "env": {}
      },
      "example_http": {
        "transport": "http",
        "url": "https://mcp.example.com/sse"
      }
    }
  },

  "channels": {
    "discord": {
      "enabled": true,
      "token_env": "DISCORD_BOT_TOKEN",
      "allow_from": ["YOUR_USER_ID"]
    },
    "telegram": {
      "enabled": false,
      "token_env": "TELEGRAM_BOT_TOKEN",
      "allow_from": []
    }
  },

  "tui": {
    "theme": "dark",
    "compact_mode": false,
    "show_cost": true,
    "show_model_tier": true
  }
}
```

---

## 6. CLI Commands

```bash
# Inherited from picoclaw
strikeclaw onboard                          # Initialize workspace
strikeclaw agent                            # Interactive agent (old TUI)
strikeclaw agent -m "message"               # One-shot message
strikeclaw gateway                          # Start messaging gateway
strikeclaw status                           # Show config and connectivity

# New: TUI mode
strikeclaw tui                              # Launch Charm TUI (default mode)
strikeclaw tui --mission web-target-01      # Resume a specific mission

# New: Mission management
strikeclaw mission create --workflow security_pentest --target example.com
strikeclaw mission list                     # List active/completed missions
strikeclaw mission resume web-target-01     # Resume mission in TUI
strikeclaw mission report web-target-01     # Generate findings report

# New: Workflow management
strikeclaw workflow list                    # List available workflow definitions
strikeclaw workflow validate my_workflow.md # Validate a workflow definition

# New: MCP management
strikeclaw mcp list                         # List configured MCP servers
strikeclaw mcp test nmap                    # Test connectivity to an MCP server
strikeclaw mcp tools                        # List all discovered MCP tools

# New: Provider management
strikeclaw provider list                    # List configured providers + connectivity
strikeclaw provider test                    # Send test prompt to each tier
```

---

## 7. Development Plan

### Phase 0: Fork & Foundation (Week 1)

**Goal:** Clean fork that builds, runs, and passes existing picoclaw tests.

- [ ] Fork picoclaw, rename to strikeclaw
- [ ] Update module paths, binary name, workspace defaults
- [ ] Verify all existing functionality works: agent mode, Discord gateway, sessions
- [ ] Add `go.mod` dependencies: `bubbletea/v2`, `lipgloss/v2`, `glamour`, `bubbles/v2`
- [ ] Scaffold new package directories: `internal/llm/`, `internal/mcp/`, `internal/workflow/`, `internal/tui/`
- [ ] Write provider interface (`Provider`) with no implementations yet
- [ ] **Deliverable:** `strikeclaw agent -m "hello"` works with picoclaw's existing LLM client

### Phase 1: Multi-Provider LLM (Week 2)

**Goal:** Multiple LLM providers, manually selected via config.

- [ ] Implement `AnthropicProvider` (native Messages API with tool_use support)
- [ ] Implement `OpenAICompatProvider` (covers LM Studio, NIM, OpenRouter, Azure, Bedrock)
- [ ] Provider-specific quirks: NIM `extra_body`, Azure `api-version`, Bedrock signing
- [ ] Config loading for `providers` section
- [ ] Provider health check (`strikeclaw provider test`)
- [ ] Swap picoclaw's single LLM client for provider abstraction in agent loop
- [ ] **Deliverable:** `strikeclaw agent` works with any configured provider. Can switch between Anthropic and LM Studio via config.

### Phase 2: Task-Based Routing (Week 3)

**Goal:** Agent automatically selects model tier per turn.

- [ ] Implement `Router` with rule-based `ClassifyTask`
- [ ] Config loading for `routing` section (tiers, use_for, fallback)
- [ ] Hook router into agent loop: before each LLM call, classify → select tier → dispatch
- [ ] Add tier/model info to session logs
- [ ] Cost tracking per tier (input/output tokens × model pricing)
- [ ] `strikeclaw provider list` shows tier assignments
- [ ] **Deliverable:** Same session uses different models. Heavy model for planning, light model for parsing large tool output. Cost logged per session.

### Phase 3: MCP Client (Week 4)

**Goal:** Load and invoke tools from MCP servers.

- [ ] Implement MCP client with stdio transport (covers most MCP servers)
- [ ] Implement `tools/list` and `tools/call` JSON-RPC methods
- [ ] `mcp_bridge.go`: register MCP tools into agent's tool registry
- [ ] Config loading for `mcp.servers` section
- [ ] `strikeclaw mcp list`, `strikeclaw mcp test`, `strikeclaw mcp tools` commands
- [ ] Add HTTP and SSE transports
- [ ] **Deliverable:** Agent can discover and invoke MCP tools alongside built-in tools. `strikeclaw mcp tools` lists all available tools from all servers.

### Phase 4: Workflow Engine (Week 5)

**Goal:** Agent follows user-defined methodologies with state tracking.

- [ ] Workflow definition parser (markdown → phase/step/branch structures)
- [ ] Mission state manager (read/write `state.json` + `MISSION.md`)
- [ ] `WorkflowEngine.InjectContext()` — add phase awareness to agent prompts
- [ ] `WorkflowEngine.Update()` — track progress after each action
- [ ] Branch creation from discovery triggers
- [ ] Phase transition when completion criteria met
- [ ] `strikeclaw mission create/list/resume/report` commands
- [ ] `strikeclaw workflow list/validate` commands
- [ ] Include example workflow definition: `workflows/example.md`
- [ ] **Deliverable:** Agent follows a multi-phase workflow. State persists across restarts. Branches tracked in mission files.

### Phase 5: Charm TUI (Week 6-7)

**Goal:** Polished terminal interface.

- [ ] `App` model with Bubble Tea: chat view, status bar, mission bar, input field
- [ ] Lip Gloss theme (dark mode, tier color coding, finding severity colors)
- [ ] Glamour markdown rendering for agent output
- [ ] Streaming output — tokens appear as they arrive
- [ ] Status bar: current model + tier, workflow phase, cost, elapsed time
- [ ] Mission bar: open branches, findings count by severity, phase progress
- [ ] Keyboard shortcuts: `q`uit, `p`ause, `r`eport, `b`ranches
- [ ] Tool execution display: command shown with `▶` prefix, collapsible output
- [ ] Model switch indicator: when router changes tier, show `[tier → model]` tag
- [ ] `strikeclaw tui` as default interactive mode
- [ ] **Deliverable:** Full TUI that looks professional. All agent activity visible. Mission state at a glance.

### Phase 6: Integration Testing & Polish (Week 8)

**Goal:** Battle-tested against real workflows.

- [ ] Test: security pentest workflow against DVWA/HackTheBox
- [ ] Test: CI/CD workflow (lint → test → deploy) to prove domain-agnostic
- [ ] Test: multi-provider routing under real load (local + API mixed)
- [ ] Test: MCP tools alongside built-in tools in same session
- [ ] Test: Discord gateway with mission status notifications
- [ ] Session resume after kill -9 (checkpoint integrity)
- [ ] Documentation: README, config reference, workflow definition guide
- [ ] **Deliverable:** v0.1.0 release. Stable enough for internal company use.

---

## 8. Dependencies

### Go Modules (New)

```
github.com/charmbracelet/bubbletea/v2      # TUI framework
github.com/charmbracelet/lipgloss/v2       # Terminal styling
github.com/charmbracelet/glamour           # Markdown rendering
github.com/charmbracelet/bubbles/v2        # TUI components (viewport, textinput, spinner)
```

### Inherited from Picoclaw

All existing picoclaw dependencies remain. No removals.

### External (Runtime)

MCP servers are external processes. Users install them separately (e.g., `npx -y gc-nmap-mcp`). StrikeClaw only needs the MCP client, not the servers.

---

## 9. Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Picoclaw upstream diverges significantly | Medium | Track upstream releases. Cherry-pick relevant fixes. Our changes are in new packages, minimizing merge conflicts. |
| Go MCP client doesn't exist as a library | Medium | Implement minimal JSON-RPC over stdio/HTTP. MCP protocol is simple — tools/list + tools/call. Consider `github.com/mark3labs/mcp-go` if available. |
| Rule-based task classification is too crude | Low | Start rule-based. Add LLM-assisted classification later (ask light model "is this planning or parsing?" — costs <0.01¢/call). |
| Charm v2 breaking changes | Low | Pin dependency versions. Charm v2 is GA as of Feb 2026. |
| Scope creep into multi-agent, web UI, etc. | High | **This doc is the scope guard.** Section 2 defines the six objectives. Everything else waits for v2. |

---

## 10. What NOT to Build (v1)

Explicit list to prevent scope creep:

- ❌ Multi-agent orchestration (single agent is enough for v1)
- ❌ Web dashboard or browser UI
- ❌ Fine-tuning or model training integration
- ❌ RAG/vector database (file-based context is sufficient)
- ❌ Browser automation (Playwright, etc.)
- ❌ Custom MCP server implementations (we're a client only)
- ❌ Billing/metering system (cost tracking in logs is enough)
- ❌ Plugin marketplace or skill store
- ❌ OAuth/SSO for multi-user access
