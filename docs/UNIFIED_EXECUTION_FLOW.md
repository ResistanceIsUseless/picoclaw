# Unified Execution Flow

This note maps the current split architecture to a single consistent user flow.

## Desired User Flow

1. User launches PicoClaw.
2. System checks configuration and tool readiness.
3. If anything required is missing, a clean TUI-driven setup/preflight flow collects it.
4. User gives the task once.
5. Agent selects the right execution mode automatically.
6. Agent uses tool knowledge, methodology, and recursive tool use until completion.
7. Results are reduced into structured state, not raw transcript bloat.

## Current Split Points

### 1. Startup and configuration

Current files:
- `cmd/picoclaw/main.go:69`
- `cmd/picoclaw/internal/onboard/wizard.go:205`

Current behavior:
- First-run config is handled before normal command execution.
- Onboarding is separate from task execution.
- Tool-specific readiness is not part of the same runtime flow.

Gap:
- Config setup exists, but operational tool readiness does not live in the same path.

### 2. User interaction entrypoints

Current files:
- `cmd/picoclaw/internal/agent/helpers.go:23`
- `cmd/picoclaw/internal/agent/helpers.go:201`
- `cmd/picoclaw/internal/claw/command.go:53`

Current behavior:
- `agent` command runs the flexible agent loop.
- `claw` command runs structured assessments separately.
- TUI wraps the agent path only.

Gap:
- There are multiple top-level execution paths instead of one runtime choosing the right strategy.

### 3. Agent execution core

Current files:
- `pkg/agent/loop.go:563`
- `pkg/routing/tier_router.go:356`

Current behavior:
- The main agent loop already supports recursive tool use.
- Tier routing and supervision live here.
- This is the best current place for dynamic multi-model execution.

Gap:
- It does not use the blackboard/orchestrator artifact path as its default execution substrate.

### 4. Structured execution core

Current files:
- `pkg/integration/claw_adapter.go:37`
- `pkg/orchestrator/orchestrator.go:374`
- `pkg/orchestrator/commander.go:64`

Current behavior:
- CLAW has its own adapter, orchestrator, blackboard, and graph path.
- Pipeline mode is rigid and structured.
- Commander mode is more dynamic, but separate from the main agent loop.

Gap:
- This is a second execution engine instead of a mode inside one engine.

### 5. Shared state

Current files:
- `pkg/blackboard/blackboard.go:44`

Current behavior:
- Blackboard is already the right place for structured intermediate state.

Gap:
- The main agent loop still mostly depends on message history, while CLAW depends on artifacts.

### 6. Tool execution and filtering

Current files:
- `pkg/tools/registry.go:86`
- `pkg/tools/registry.go:146`
- `pkg/tools/filters/filter.go:134`

Current behavior:
- Tool execution is centralized enough to be reusable.
- Some output filters exist.

Gap:
- Preflight and readiness checks are not part of the same tool runtime.
- Output reduction is inconsistent between normal agent mode and CLAW mode.

## Recommended Single Flow

## A. One entrypoint runtime

Target:
- Make `agent` and TUI the primary user-facing runtime.
- Treat CLAW pipeline and Commander as execution strategies selected by the runtime, not separate user mental models.

Implementation home:
- `cmd/picoclaw/internal/agent/helpers.go:23`
- new runtime coordinator package, likely `pkg/runtime/` or `pkg/execution/`

Responsibility:
- Accept user intent
- inspect config, tools, workflow/methodology hints
- choose execution mode:
  - direct agent loop
  - structured pipeline
  - commander-style recursive execution

## B. Add a preflight layer before tool execution

Target:
- Before any task starts, resolve what is needed for success.

This layer should check:
- provider/model configuration
- tool availability on system
- tool-specific config or flags
- auth/secrets required for selected tools
- workspace/output directories

Implementation home:
- new preflight manager near `pkg/tools/registry.go`
- interactive UI hooks in `pkg/tui/`
- reuse onboarding logic from `cmd/picoclaw/internal/onboard/wizard.go`

Important design rule:
- onboarding and runtime preflight should share components
- do not maintain one setup wizard for models and a completely separate flow for tool readiness

## C. Make the blackboard the shared execution memory

Target:
- All significant tool results should become structured artifacts.

Implementation home:
- `pkg/blackboard/blackboard.go`
- `pkg/orchestrator/orchestrator.go:544`
- main agent path should adopt the same publish flow after tool execution

Desired behavior:
- tool runs
- output reduced via parser/filter/profile
- artifact published to blackboard
- graph/frontier updated if applicable
- model sees artifact summary, not raw output

This is the main place where the two worlds should merge.

## D. Keep one recursive execution loop

Target:
- One recursive tool-using loop should drive execution until complete.

Best current base:
- `pkg/agent/loop.go:563`

What to unify into it:
- structured phase contracts from CLAW
- blackboard-backed state
- frontier/tool recommendations
- tier routing and supervision

Meaning:
- the agent loop should become the universal executor
- CLAW should become a planning/state layer, not a separate runtime

## E. Convert methodologies and pipelines into planning inputs

Target:
- Methodologies should guide execution, not force users into separate commands.

Current files:
- `pkg/orchestrator/pipeline.go:237`
- `pkg/workflow/`

Recommended future role:
- workflows/pipelines become task plans or contracts
- runtime selects them automatically when useful
- user can still force a mode, but default flow should not require that knowledge

## F. Introduce tool profiles

Target:
- Tooling remains dynamic, but output handling becomes consistent.

Profiles should define:
- readiness requirements
- preferred invocation mode
- deterministic parser, if available
- fallback summarizer/parser model
- relevance rules for what enters context

Examples:
- `port-scan` for `nmap`, `masscan`, `naabu`
- `crawl` for `katana`, `gospider`, `hakrawler`
- `web-probe` for `httpx`
- `vuln-scan` for `nuclei`

Implementation home:
- alongside `pkg/tools/registry.go`
- likely a new `pkg/tools/profiles/` package

## Prompt scope note

- workflow context should bias toward the immediate actionable step, not a full long-range checklist
- supporting context should stay limited to current phase progress, completion criteria, active branches, and recent findings
- deeper future work belongs in methodology state, not the active prompt focus
- CLAW phase context should be assembled by priority, and lower-priority sections should drop first under token pressure while preserving core sections

## Recommended Convergence Plan

### Phase 1: unify entrypoint and preflight

- keep `picoclaw agent` and TUI as the main interactive path
- add runtime preflight for models, tools, and secrets
- reuse onboarding components instead of duplicating prompts

### Phase 2: unify tool result handling

- after every tool run, send output through one reduction pipeline
- publish structured artifacts to blackboard from both agent mode and CLAW mode
- stop sending large raw output directly into message history by default

### Phase 3: unify execution engine

- make the main agent loop the single recursive execution kernel
- use CLAW contracts, graph, and frontier as optional structured state within that loop
- keep Commander as a planning/routing strategy, not a fully separate orchestrator

### Phase 4: unify mode selection

- runtime chooses direct, workflow, pipeline-like, or commander-like behavior automatically
- users can still override, but default behavior should be one mental model

## Final Target Architecture

User flow:

1. open PicoClaw
2. runtime checks readiness
3. TUI asks only for missing config or tool prerequisites
4. user gives objective
5. runtime selects strategy and model tiers
6. recursive loop executes tools
7. outputs become artifacts
8. blackboard/graph/frontier update shared state
9. reasoning continues until completion
10. final response/report produced from structured state

## Short Version

- keep one user entrypoint
- keep one recursive execution loop
- make blackboard the shared memory layer
- make preflight part of runtime, not a separate setup world
- make methodologies guide the loop, not fork the product
- make tool profiles decide relevance before context grows
