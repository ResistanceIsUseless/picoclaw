# StrikeClaw Usage Guide

Complete guide to using the StrikeClaw security assessment system with PicoClaw.

## Quick Start

### Basic Interactive Mode

```bash
# Traditional CLI mode
picoclaw agent

# Terminal UI mode (recommended)
picoclaw agent --tui
```

### Workflow-Guided Assessment

```bash
# Run network scan with TUI
picoclaw agent --tui --workflow network-scan --target 192.168.1.0/24

# Single-shot scan (non-interactive)
picoclaw agent --workflow network-scan --target 192.168.1.0/24 \
  -m "Begin discovery phase, scan the network for live hosts"
```

## Features

### 1. Tier-Based Model Routing

StrikeClaw automatically routes LLM calls to appropriate model tiers for cost optimization:

- **Heavy Tier** (Claude Sonnet): Strategic planning, deep analysis, final reporting
- **Medium Tier** (Codestral local): Tool selection, code review, JavaScript analysis
- **Light Tier** (Nemotron local): Output parsing, triage, summarization

**Cost savings:** 80-95% compared to using premium models for everything.

### 2. Workflow Engine

Workflows provide structured guidance for multi-phase security assessments:

**Available workflows:**
- `network-scan`: 5-phase internal network reconnaissance

**Workflow features:**
- Phase-based tracking (discovery → enumeration → analysis → validation → reporting)
- Step completion checkboxes
- Adaptive branch creation on discoveries
- Finding management with severity levels
- State persistence for resume capability

### 3. Terminal UI

Beautiful, functional TUI for interactive sessions:

**Layout:**
- **Status bar** (top): Current model, tier, session cost
- **Chat area** (center): Conversation with markdown rendering
- **Mission panel** (right): Workflow state (toggle with Ctrl+M)
- **Input bar** (bottom): Your commands

**Keyboard shortcuts:**
- `Ctrl+C` or `Esc`: Exit
- `Ctrl+M`: Toggle mission panel
- `Tab`: Switch focus (chat ↔ input)
- `↑/↓`: Scroll chat history
- `PgUp/PgDn`: Fast scroll
- `Home/End`: Jump to top/bottom

## Command Reference

### Agent Command

```bash
picoclaw agent [flags]
```

**Flags:**

| Flag | Short | Type | Description |
|------|-------|------|-------------|
| `--tui` | | bool | Use terminal UI (interactive only) |
| `--workflow` | `-w` | string | Load workflow (e.g., 'network-scan') |
| `--target` | `-t` | string | Target for workflow (required with --workflow) |
| `--message` | `-m` | string | Single message (non-interactive) |
| `--session` | `-s` | string | Session key (default: "cli:default") |
| `--model` | | string | Override default model |
| `--debug` | `-d` | bool | Enable debug logging |

## Configuration

### Config File Location

`~/.picoclaw/config.json`

### Tier Routing Configuration

See [TIER_ROUTING_GUIDE.md](TIER_ROUTING_GUIDE.md) for complete configuration details.

**Minimal example:**

```json
{
  "routing": {
    "enabled": true,
    "default_tier": "heavy",
    "tiers": {
      "heavy": {
        "model_name": "claude-sonnet-4",
        "use_for": ["planning", "analysis"],
        "cost_per_m": {"input": 3.0, "output": 15.0}
      },
      "light": {
        "model_name": "nemotron-nano-local",
        "use_for": ["parsing", "summary"],
        "cost_per_m": {"input": 0.0, "output": 0.0}
      }
    }
  },
  "model_list": [
    {
      "model_name": "claude-sonnet-4",
      "model": "anthropic/claude-sonnet-4-20250514",
      "api_key": "${ANTHROPIC_API_KEY}"
    },
    {
      "model_name": "nemotron-nano-local",
      "model": "openai/nvidia-nemotron-3-nano",
      "api_base": "http://localhost:1234/v1"
    }
  ]
}
```

### Environment Variables

```bash
# Required for heavy tier
export ANTHROPIC_API_KEY="sk-ant-..."

# Required for local models
export LM_STUDIO_BASE_URL="http://localhost:1234/v1"

# Optional
export OPENROUTER_API_KEY="sk-or-..."
export NVIDIA_API_KEY="nvapi-..."
```

## Workflows

### Creating Custom Workflows

Workflows are markdown files with YAML frontmatter:

**Location:** `~/.picoclaw/workspace/workflows/your-workflow.md`

**Format:**

```markdown
---
name: your-workflow
description: Brief description
phases: [phase1, phase2, phase3]
---

## Phase: phase1

### Steps

- step_id: Step description (required)
- optional_step: Optional step

### Completion Criteria

Description of when this phase is complete.

### Branches

- condition → What this branch investigates
- web_found → Deep web analysis
```

See [WORKFLOW_GUIDE.md](WORKFLOW_GUIDE.md) and [network-scan.md](../examples/workflows/network-scan.md) for details.

### Workflow Tools

When a workflow is loaded, the agent has access to workflow management tools:

**Available tools:**
- `workflow_step_complete`: Mark steps done
- `workflow_create_branch`: Create investigation branch
- `workflow_complete_branch`: Close branch
- `workflow_add_finding`: Record security finding
- `workflow_advance_phase`: Move to next phase

The agent uses these automatically based on the methodology guidance.

## Usage Examples

### Example 1: Internal Network Scan with TUI

```bash
# Prerequisites
# - LM Studio running with local models
# - Anthropic API key configured

# Start assessment
picoclaw agent --tui \
  --workflow network-scan \
  --target 192.168.1.0/24

# In the TUI:
# 1. Agent starts in discovery phase
# 2. Runs ping sweep and port scans (tier routing: light tier for parsing)
# 3. Identifies services (heavy tier for analysis)
# 4. Creates branches for web services, SMB shares, etc.
# 5. Advances through enumeration → analysis → validation → reporting
# 6. Mission panel shows progress in real-time
# 7. Cost tracker shows ~$0.30 total (vs $3-5 without tier routing)
```

### Example 2: Single-Target Deep Dive

```bash
# Assess single host without workflow
picoclaw agent --tui -m "Perform a comprehensive security assessment of 192.168.1.50"

# Agent will:
# - Run port scan
# - Enumerate services
# - Test for common vulnerabilities
# - Check for misconfigurations
# - Report findings

# All without rigid workflow structure
```

### Example 3: Web Application Assessment

```bash
# Create custom web-app workflow first (see WORKFLOW_GUIDE.md)

picoclaw agent --tui \
  --workflow web-app \
  --target https://example.com

# Agent follows web app methodology:
# - Reconnaissance (subdomains, tech stack)
# - Crawling and mapping
# - JavaScript analysis
# - API discovery
# - Vulnerability testing
```

### Example 4: Batch Scanning (Non-Interactive)

```bash
# Scan multiple targets in sequence
for target in $(cat targets.txt); do
  echo "Scanning $target..."
  picoclaw agent \
    --workflow network-scan \
    --target "$target" \
    -m "Complete the network scan mission" \
    > "reports/${target}.txt" 2>&1
done
```

## Understanding Output

### Mission State Files

Workflow state is saved to: `~/.picoclaw/workspace/missions/{target}_state.json`

**Contains:**
- Workflow name and target
- Current phase and phase history
- Completed steps
- Active investigation branches
- All findings with evidence
- Metadata

**Use case:** Resume interrupted assessments, generate reports later

### Cost Tracking

Session costs are tracked in real-time:

**In TUI:** Status bar shows running total
**In logs:** Cost breakdown by tier and model

**Example cost report:**

```
Session Cost Report
==================
Total: $0.28

By Tier:
  heavy: $0.28 (4 calls)
  light: $0.00 (15 calls)

By Model:
  claude-sonnet-4: $0.28
  nemotron-nano-local: $0.00
```

### Findings Format

Findings are JSON objects with:

```json
{
  "id": "uuid",
  "title": "Default Credentials on Admin Panel",
  "description": "The admin panel accepts default credentials...",
  "severity": "high",
  "phase": "analysis",
  "created_at": "2026-02-25T16:30:00Z",
  "evidence": "Login successful with admin:admin...",
  "metadata": {}
}
```

**Severity levels:** critical, high, medium, low, informational

## Best Practices

### 1. Workflow Selection

- Use workflows for **methodical, multi-phase** assessments
- Skip workflows for **quick spot checks** or **specific testing**
- Create custom workflows for **repetitive tasks**

### 2. Cost Optimization

- Always use tier routing (enable in config)
- Run local models (LM Studio) for medium/light tiers
- Monitor costs in real-time via TUI
- Heavy tier should be <30% of total calls

### 3. Tool Usage

- Let agent choose tools (it knows nmap, nuclei, etc.)
- Agent will install missing tools via `install()` function
- Validate findings manually before reporting
- Use workflow branches to track parallel investigations

### 4. Finding Management

- Record findings as you discover them (don't batch at end)
- Include evidence (tool output, screenshots, logs)
- Use appropriate severity levels
- Add remediation guidance in description

### 5. State Management

- Mission state auto-saves every action
- Resume with: `picoclaw agent --workflow <name> --target <target>`
- State files are human-readable JSON
- Delete state file to restart mission fresh

## Troubleshooting

### TUI Not Starting

**Issue:** `picoclaw agent --tui` fails or shows garbled output

**Solutions:**
- Ensure terminal supports colors: `echo $TERM`
- Update terminal emulator
- Try without TUI: `picoclaw agent`

### Workflow Not Found

**Issue:** `failed to load workflow 'network-scan'`

**Solutions:**
- Check workflow file exists: `ls ~/.picoclaw/workspace/workflows/`
- Try with full path: `--workflow ~/.picoclaw/workspace/workflows/network-scan.md`
- Copy example: `cp examples/workflows/network-scan.md ~/.picoclaw/workspace/workflows/`

### High Costs Despite Tier Routing

**Issue:** Session costs higher than expected

**Solutions:**
- Verify routing enabled: `jq '.routing.enabled' ~/.picoclaw/config.json`
- Check tier assignments in logs: look for "Routing to tier" messages
- Ensure LM Studio running: `curl http://localhost:1234/v1/models`
- Review cost report: heavy tier should be <30% of calls

### Model Connection Errors

**Issue:** `failed to connect to model`

**Solutions:**
- Check API key: `echo $ANTHROPIC_API_KEY`
- Verify LM Studio: `curl http://localhost:1234/v1/models`
- Test connectivity: `ping api.anthropic.com`
- Check config model names match: `jq '.model_list' ~/.picoclaw/config.json`

## Advanced Topics

### Custom Methodology Integration

Place methodology files in `~/.picoclaw/workspace/METHODOLOGY.md`

The agent will load this as system prompt context alongside workflows.

### MCP Server Integration (Future)

Phase 2 (deferred) will add MCP tool servers:
- `mcp-kali-server`: Kali Linux security tools
- Custom security tool servers
- Network via MCP or direct to agent (TBD)

### Multi-Target Campaigns

Use mission state files to track campaign progress:

```bash
# Script to manage campaign
for target in $(cat campaign-targets.txt); do
  state_file="missions/${target}_state.json"

  if [ -f "$state_file" ]; then
    echo "Resuming $target..."
  else
    echo "Starting $target..."
  fi

  picoclaw agent \
    --workflow network-scan \
    --target "$target" \
    -m "Continue mission" \
    | tee "logs/${target}.log"
done
```

## Getting Help

- **Documentation:** See docs/ directory
- **Examples:** See examples/workflows/
- **Issues:** https://github.com/sipeed/picoclaw/issues
- **Architecture:** See STRIKECLAW_ARCHITECTURE.md

## Related Documentation

- [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md) - Development roadmap
- [TIER_ROUTING_GUIDE.md](TIER_ROUTING_GUIDE.md) - Cost optimization details
- [WORKFLOW_GUIDE.md](WORKFLOW_GUIDE.md) - Workflow system reference
- [METHODOLOGY.md](METHODOLOGY.md) - Security research methodology
