# Workflow Engine Guide

## Overview

The workflow engine tracks multi-phase security assessments, guiding the agent through structured methodologies while maintaining flexibility to adapt based on discoveries.

## Key Features

- **Phase-based tracking**: Break assessments into logical phases (recon, enumeration, analysis, etc.)
- **Step completion**: Track progress through required and optional steps
- **Branch creation**: Dynamically create investigation branches when interesting findings emerge
- **Finding management**: Record security findings with severity levels and evidence
- **State persistence**: Resume missions from saved state files

## Workflow Definition Format

Workflows are defined in Markdown files with YAML frontmatter:

```markdown
---
name: workflow-name
description: Brief description
phases: [phase1, phase2, phase3]
---

## Phase: phase1

### Steps

- step_id: Step description (required)
- optional_step: Optional step description

### Completion Criteria

Description of when this phase is complete. Keywords like "all required"
trigger automatic completion detection.

### Branches

- condition → Description of what this branch investigates
- web_found → Deep web application analysis
```

## Using Workflows

### Starting a Mission

Workflows are loaded programmatically via the agent API:

```go
// Load workflow and start mission
agent := registry.GetDefaultAgent()
err := agent.LoadWorkflow("network-scan", "192.168.1.0/24")
```

### Workflow Context in System Prompt

Once loaded, the workflow engine injects context into the agent's system prompt:

```
# Active Mission Context

**Workflow**: network-scan
**Target**: 192.168.1.0/24
**Started**: 2026-02-25 15:30:00

## Current Phase: discovery

### Steps:
- ✓ ping_sweep (required)
- ○ port_scan (required)
- ○ service_detection (required)

### Completion: All discovered hosts have been scanned for services

### Possible Branches:
- **web_service_found**: Enumerate web applications
- **smb_discovered**: Test SMB shares and authentication
```

### Agent Tools

The agent has workflow management tools available:

#### `workflow_step_complete`
Mark a step as complete:
```json
{
  "step_id": "ping_sweep"
}
```

#### `workflow_create_branch`
Create an investigation branch:
```json
{
  "condition": "web_service_found",
  "description": "Found HTTP server on 192.168.1.50:80, investigating web application"
}
```

#### `workflow_complete_branch`
Mark a branch as complete:
```json
{
  "condition": "web_service_found"
}
```

#### `workflow_add_finding`
Record a security finding:
```json
{
  "title": "Default Credentials on Admin Panel",
  "description": "The admin panel at 192.168.1.50/admin accepts default credentials admin:admin",
  "severity": "high",
  "evidence": "Successfully logged in with credentials admin:admin. Session token: abc123..."
}
```

#### `workflow_advance_phase`
Move to the next phase (only when completion criteria met):
```json
{}
```

## Mission State Files

Mission state is automatically saved to `{workspace}/missions/{target}_state.json`:

```json
{
  "workflow_name": "network-scan",
  "target": "192.168.1.0/24",
  "start_time": "2026-02-25T15:30:00Z",
  "current_phase": 1,
  "phase_history": [...],
  "active_branches": [...],
  "findings": [...]
}
```

### Resuming Missions

Load existing mission state:

```go
engine, err := workflow.LoadEngine(wf, "path/to/state.json", workspace)
```

## Workflow Examples

### Example 1: Network Scan

See `examples/workflows/network-scan.md` for a complete internal network reconnaissance workflow with 5 phases.

### Example 2: Web Application Assessment

```markdown
---
name: web-app-assessment
description: Web application security testing
phases: [reconnaissance, mapping, testing, exploitation, reporting]
---

## Phase: reconnaissance

### Steps
- subdomain_enumeration: Find all subdomains
- technology_detection: Identify frameworks and libraries
- endpoint_discovery: Map all accessible endpoints

### Completion Criteria
All subdomains discovered and technologies identified.

### Branches
- javascript_found → JavaScript analysis
- api_discovered → API testing
```

## Integration with METHODOLOGY.md

The workflow engine complements the existing `METHODOLOGY.md`:

- **METHODOLOGY.md**: Provides detailed guidance and principles (always loaded in system prompt)
- **Workflow files**: Provide structured tracking for specific mission types

The agent should:
1. Follow the detailed guidance from METHODOLOGY.md
2. Use workflow tools to track progress through the structure
3. Adapt and branch based on discoveries

## Best Practices

### Workflow Design

- **Keep phases broad**: 3-5 phases per workflow, not 20 micro-phases
- **Make steps actionable**: Each step should be something the agent can complete and mark done
- **Use branches for discoveries**: Don't try to predict all possibilities upfront
- **Write clear completion criteria**: The agent needs to know when to advance

### Agent Guidance

The workflow system is **guidance, not a script**:
- Agent should adapt based on findings
- Steps can be done out of order if it makes sense
- Agent should create branches for unexpected discoveries
- The methodology principles (from METHODOLOGY.md) take precedence over rigid following of steps

### Finding Management

Record findings throughout the assessment:
- **Critical/High**: Immediate security risks (RCE, auth bypass, data exposure)
- **Medium**: Configuration issues, information leakage
- **Low**: Potential issues requiring validation
- **Info**: Notable observations for context

## Architecture

The workflow engine is implemented in `pkg/workflow/`:

- `types.go`: Workflow, Phase, Step, MissionState, Finding types
- `engine.go`: State management, context generation, phase tracking
- `parser.go`: Markdown workflow definition parser

Workflow tools are in `pkg/tools/workflow.go`.

Context injection happens via `pkg/agent/context.go` using a callback pattern to avoid tight coupling.

## Future Enhancements

Potential future additions:

- Workflow templates with parameter substitution
- Parallel branch execution tracking
- Time-based reminders for long-running branches
- Workflow metrics and statistics
- Visual workflow progress display (TUI - Phase 4)

## Related Documentation

- [METHODOLOGY.md](METHODOLOGY.md) - Security research methodology and principles
- [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md) - StrikeClaw architecture roadmap
- [TIER_ROUTING_GUIDE.md](TIER_ROUTING_GUIDE.md) - Cost-optimized model routing
