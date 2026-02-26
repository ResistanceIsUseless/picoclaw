<div align="center">
  <h1>StrikeClaw</h1>
  <h3>Methodology-driven AI agent framework with multi-model routing, workflow engine, and local model support</h3>

  <p>
    <img src="https://img.shields.io/badge/Go-1.25.7+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go">
    <img src="https://img.shields.io/badge/license-MIT-green" alt="License">
    <img src="https://img.shields.io/badge/Arch-x86__64%2C%20ARM64%2C%20RISC--V-blue" alt="Hardware">
  </p>
</div>

---

StrikeClaw is a fork of [PicoClaw](https://github.com/sipeed/picoclaw) extended into a methodology-driven agent framework. It adds structured workflows, multi-model routing for cost optimization, a Charm-based terminal UI, and robust local model support — while keeping the original's lightweight footprint.

## Key Features

**Workflow Engine** — Define multi-phase methodologies in Markdown. The agent follows structured phases, tracks step completion, creates investigation branches on discoveries, and records findings with severity levels. State persists to disk for resume.

**Tier-Based Model Routing** — Route tasks to different models by complexity. Heavy tasks (planning, analysis) go to Claude/GPT; light tasks (parsing, formatting) go to small local models. 80-95% cost reduction vs. premium-only.

**Local Model Support** — Run the agent entirely on local models via LM Studio, Ollama, or any OpenAI-compatible endpoint. A fallback text parser handles models that emit `<functioncall>` tags instead of structured API tool calls (codestral, qwen, mistral, etc.).

**Terminal UI** — Charm-based TUI (Bubble Tea + Lip Gloss + Glamour) with a chat view, mission progress panel, real-time cost tracking, and streaming output.

**Autonomous Execution** — The agent executes tools directly without asking for permission. System prompt rules enforce actual tool usage — no fabricated output, no simulated commands.

**19 Built-in Tools** — File operations, shell execution, web search/fetch, messaging, scheduled tasks (cron), subagent spawning, I2C/SPI hardware access, skill discovery/installation, and 5 workflow tracking tools.

## Quick Start

### Install with Go

```bash
go install github.com/ResistanceIsUseless/picoclaw/cmd/picoclaw@latest
```

Requires Go 1.25.7+. The binary is placed in your `$GOBIN` (default `~/go/bin`).

### Install from source

```bash
git clone https://github.com/ResistanceIsUseless/picoclaw.git
cd picoclaw
make deps && make build
make install  # installs to ~/.local/bin
```

### Configure

```bash
picoclaw onboard
```

Edit `~/.picoclaw/config.json`:

```json
{
  "agents": {
    "defaults": {
      "model_name": "claude-sonnet",
      "max_tokens": 8192,
      "temperature": 0.7,
      "max_tool_iterations": 40
    }
  },
  "model_list": [
    {
      "model_name": "claude-sonnet",
      "model": "openrouter/anthropic/claude-sonnet-4",
      "api_key": "sk-or-v1-your-key",
      "api_base": "https://openrouter.ai/api/v1"
    }
  ]
}
```

### Chat

```bash
# One-shot
picoclaw agent -m "What services are running on my network?"

# Interactive
picoclaw agent

# With TUI
picoclaw agent --tui
```

## Workflows

Workflows are Markdown files that define multi-phase methodologies. The agent receives the current phase, steps, and completion criteria in its system prompt and uses workflow tools to track progress.

### Running a workflow

```bash
picoclaw agent -w network-scan -t 192.168.0.0/24 -m "do a network scan of 192.168.0.0/24"
```

The `-w` flag loads the workflow definition, `-t` sets the target. The workflow engine injects phase context into the system prompt, and the agent follows the methodology autonomously.

### Example: Network Scan Workflow

The built-in `network-scan` workflow defines 5 phases:

| Phase | Steps | Purpose |
|-------|-------|---------|
| **Discovery** | Ping sweep, port scan, service detection | Find live hosts and open ports |
| **Enumeration** | Technology ID, banner grabbing, CVE lookup | Identify software versions |
| **Analysis** | Vulnerability assessment, config review, credential testing | Find security issues |
| **Validation** | Finding validation, false positive elimination, impact assessment | Confirm findings |
| **Reporting** | Documentation, remediation guidance, executive summary | Produce final report |

Each phase has completion criteria and branches that trigger on discoveries (e.g., `web_service_found`, `smb_discovered`, `database_found`).

### Writing custom workflows

Create a Markdown file in `~/.picoclaw/workspace/workflows/`:

```markdown
---
name: my-workflow
description: Description of the workflow
phases: [recon, analysis, reporting]
---

## Phase: recon

### Steps
- step_one: Description of first step (required)
- step_two: Description of second step

### Completion Criteria
All required steps have been completed.

### Branches
- interesting_finding → Investigate deeper
```

### Workflow tools

The agent uses these tools during workflow execution:

| Tool | Purpose |
|------|---------|
| `workflow_step_complete` | Mark a step as done |
| `workflow_create_branch` | Create an investigation branch |
| `workflow_complete_branch` | Close a branch |
| `workflow_add_finding` | Record a finding with severity + evidence |
| `workflow_advance_phase` | Move to the next phase |

## Local Model Support

StrikeClaw works with local models via LM Studio, Ollama, or any OpenAI-compatible server.

### LM Studio setup

1. Load a model in LM Studio (codestral-22b, qwen3-8b, etc.)
2. Start the local server (default: `http://localhost:1234/v1`)
3. Configure:

```json
{
  "agents": {
    "defaults": {
      "provider": "openai",
      "model_name": "lmstudio",
      "max_tokens": 4096
    }
  },
  "model_list": [
    {
      "model_name": "lmstudio",
      "model": "openai/codestral-22b-v0.1",
      "api_key": "lm-studio",
      "api_base": "http://localhost:1234/v1"
    }
  ]
}
```

### Fallback tool call parser

Many local models don't use OpenAI's structured `tool_calls` field. Instead they emit text like:

```
<functioncall>{"name": "exec", "arguments": {"command": "nmap -sV 192.168.0.0/24"}}</functioncall>
```

StrikeClaw automatically detects and parses these text-formatted tool calls using brace-counting JSON extraction. This enables tool-calling with models that would otherwise be incompatible.

### Context window considerations

The system prompt + 19 tool definitions requires ~4,200 tokens. Recommendations:

| Context Window | Capability |
|---|---|
| **8K** | 1-2 simple tool calls |
| **16K** | Basic tasks, 5-8 tool calls |
| **32K+** | Complex multi-step workflows |

If your local model has a small default context (LM Studio defaults to 4096), increase it in the model settings.

## Tier-Based Model Routing

Route tasks to different models based on complexity to reduce costs by 80-95%.

```json
{
  "routing": {
    "enabled": true,
    "default_tier": "medium",
    "tiers": [
      {
        "name": "heavy",
        "models": ["claude-sonnet"],
        "cost_per_million_input": 3.0,
        "cost_per_million_output": 15.0
      },
      {
        "name": "medium",
        "models": ["codestral"],
        "cost_per_million_input": 0.3,
        "cost_per_million_output": 0.9
      },
      {
        "name": "light",
        "models": ["lmstudio"],
        "cost_per_million_input": 0.0,
        "cost_per_million_output": 0.0
      }
    ]
  }
}
```

Task classification is rule-based (zero extra LLM calls). Planning and analysis go to heavy tier; parsing and formatting go to light tier.

## Configuration Reference

### Agent defaults

```json
{
  "agents": {
    "defaults": {
      "workspace": "~/.picoclaw/workspace",
      "restrict_to_workspace": false,
      "provider": "openai",
      "model_name": "claude-sonnet",
      "max_tokens": 8192,
      "context_window": 128000,
      "temperature": 0.7,
      "max_tool_iterations": 40
    }
  }
}
```

| Field | Default | Description |
|-------|---------|-------------|
| `model_name` | — | Model name from `model_list` |
| `max_tokens` | 8192 | Max output tokens per response |
| `context_window` | 128000 | Model's input context window (for summarization threshold) |
| `max_tool_iterations` | 40 | Max tool call rounds per message |
| `restrict_to_workspace` | `true` | Sandbox file/exec access to workspace |

### Model list

The `model_list` array defines available models. Format: `vendor/model-name`.

```json
{
  "model_list": [
    {
      "model_name": "claude-sonnet",
      "model": "anthropic/claude-sonnet-4",
      "api_key": "sk-ant-..."
    },
    {
      "model_name": "lmstudio",
      "model": "openai/codestral-22b-v0.1",
      "api_key": "lm-studio",
      "api_base": "http://localhost:1234/v1"
    }
  ]
}
```

Supported vendor prefixes: `openai/`, `anthropic/`, `openrouter/`, `zhipu/`, `deepseek/`, `gemini/`, `groq/`, `ollama/`, `nvidia/`, `moonshot/`, `cerebras/`, `vllm/`

### Exec tool timeout

Default command timeout is 5 minutes. Configure for long-running tasks:

```json
{
  "tools": {
    "exec": {
      "timeout_seconds": 600
    }
  }
}
```

### Named agents

Define multiple agents with different configs for different tasks:

```json
{
  "agents": {
    "defaults": {
      "model_name": "claude-sonnet",
      "max_tokens": 8192
    },
    "security": {
      "workspace": "/path/to/security-workspace",
      "model_name": "claude-sonnet",
      "temperature": 0.3,
      "restrict_to_workspace": false
    },
    "local": {
      "provider": "openai",
      "model_name": "lmstudio",
      "max_tokens": 4096
    }
  }
}
```

## Chat Channels

Connect the agent to messaging platforms. All channels support `allow_from` access control.

| Channel | Complexity |
|---------|-----------|
| Telegram | Easy (bot token) |
| Discord | Easy (bot token + intents) |
| Slack | Easy (bot + app tokens) |
| QQ | Easy (AppID + AppSecret) |
| DingTalk | Medium (client credentials) |
| LINE | Medium (credentials + webhook) |
| WeCom | Medium (CorpID + webhook) |

Start the gateway to serve all enabled channels:

```bash
picoclaw gateway
```

## Workspace Layout

```
~/.picoclaw/workspace/
├── sessions/          # Conversation history
├── memory/            # Long-term memory (MEMORY.md)
├── missions/          # Workflow mission state (JSON)
├── workflows/         # Workflow definitions (Markdown)
├── state/             # Persistent state
├── cron/              # Scheduled jobs
├── skills/            # Installed skills
├── AGENTS.md          # Agent behavior guide
├── IDENTITY.md        # Agent identity
├── SOUL.md            # Agent personality
├── USER.md            # User preferences
└── HEARTBEAT.md       # Periodic task prompts
```

## Security Sandbox

When `restrict_to_workspace: true` (default), file and exec tools are sandboxed to the workspace directory. Additionally, dangerous commands are always blocked regardless of sandbox settings:

- `rm -rf`, `format`, `mkfs`, `dd if=` — destructive operations
- `shutdown`, `reboot`, `poweroff` — system control
- Fork bombs and direct disk writes

## CLI Reference

| Command | Description |
|---------|-------------|
| `picoclaw onboard` | Initialize config and workspace |
| `picoclaw agent -m "..."` | One-shot message |
| `picoclaw agent` | Interactive chat |
| `picoclaw agent --tui` | Terminal UI mode |
| `picoclaw agent -w NAME -t TARGET` | Run with workflow |
| `picoclaw gateway` | Start messaging gateway |
| `picoclaw config` | List models and test connections |
| `picoclaw config discover` | Interactive provider model discovery |
| `picoclaw status` | Show agent status |
| `picoclaw cron list` | List scheduled jobs |
| `picoclaw skills list` | List installed skills |

## Build & Development

```bash
make build          # Build for current platform
make build-all      # Cross-compile for linux/darwin/windows
make test           # Run all tests
make check          # deps + fmt + vet + test
make lint           # Full linter
make install        # Install to ~/.local/bin
```

## Architecture

```
cmd/picoclaw/          CLI entrypoint (cobra)
pkg/agent/             Agent loop (Think/Act/Observe), context builder, registry
pkg/providers/         LLM provider abstraction
  ├── anthropic/       Native Anthropic SDK
  ├── openai_compat/   OpenAI-compatible (LM Studio, Ollama, OpenRouter, etc.)
  └── protocoltypes/   Shared message/tool types
pkg/routing/           Tier-based model routing
pkg/workflow/          Workflow engine, parser, state management
pkg/tools/             Tool registry + implementations
pkg/session/           Conversation persistence
pkg/channels/          Messaging platforms (Discord, Telegram, Slack, etc.)
pkg/skills/            Skill discovery and installation
pkg/tui/               Charm-based terminal UI
```

## Credits

Forked from [PicoClaw](https://github.com/sipeed/picoclaw), originally inspired by [nanobot](https://github.com/HKUDS/nanobot). Licensed under MIT.
