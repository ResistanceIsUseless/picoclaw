# Multi-Model And Context Reference

This note describes the current behavior in code, not the intended future design.

## Executive View

- The main interactive agent supports multi-model routing through the tier router.
- The CLAW pipeline orchestrator uses a blackboard and graph for state, but currently runs on a single provider/model per orchestrator instance.
- The Commander orchestrator is multi-agent in structure, but not truly multi-model yet; Commander and specialists currently use the same provider default model.
- The blackboard is not a model router. It is the typed artifact store and shared state layer.

## 1. Main Agent Loop

Code:
- `pkg/agent/loop.go:612`
- `pkg/routing/tier_router.go:356`
- `pkg/routing/tier_router.go:420`
- `pkg/config/config.go:703`

Current behavior:

- The main agent loop checks whether tier routing is enabled.
- If enabled, it classifies the current turn into a task type such as `planning`, `analysis`, `parsing`, or `summary`.
- That task type is routed to a configured tier from `routing.tiers`.
- If supervision is required, a worker model handles the task first and a supervisor model validates or corrects the output.

What this means:

- This is the primary place where PicoClaw currently uses multiple models intentionally.
- Multi-model behavior here is task-based, not tool-family-based.
- Cost tracking is attached to this path.

## 2. Blackboard

Code:
- `pkg/blackboard/blackboard.go:44`
- `pkg/integration/claw_adapter.go:191`
- `pkg/orchestrator/orchestrator.go:588`

Current behavior:

- The blackboard stores typed artifacts produced during assessment.
- Artifacts are published by phases or adapters.
- The blackboard supports retrieval by type, phase, and domain, plus pub/sub.

What this means:

- The blackboard is shared memory/state for CLAW.
- It does not choose models.
- It helps reduce context by storing structured artifacts outside the live prompt.

## 3. CLAW Pipeline Orchestrator

Code:
- `pkg/integration/claw_adapter.go:111`
- `pkg/orchestrator/orchestrator.go:86`
- `pkg/orchestrator/orchestrator.go:374`
- `pkg/orchestrator/orchestrator.go:562`

Current behavior:

- Pipeline mode creates one orchestrator and calls `SetProvider(provider)` once.
- During phase execution, the orchestrator calls `o.provider.Chat(...)` with that one provider/model.
- Tool output is reduced through a two-layer parsing path:
  - Layer 1: structural parser if the tool has one
  - Layer 2: `LLMParser` fallback if structural parsing is unavailable or fails
- Parsed artifacts are published to the blackboard and may update the graph.

What this means:

- The pipeline orchestrator is model-assisted, but not multi-model in practice.
- It already has the best context-reduction architecture in the repo: parse raw output into artifacts before carrying results forward.
- It does not currently route different phases or parsing tasks to different models.

## 4. CLAW Commander Orchestrator

Code:
- `pkg/integration/claw_adapter.go:64`
- `pkg/orchestrator/commander.go:22`
- `pkg/orchestrator/commander.go:124`
- `pkg/orchestrator/commander.go:219`

Current behavior:

- Commander mode is hierarchical in workflow shape:
  - Commander inspects blackboard state
  - Commander routes work to a specialist
  - Specialist uses tools and records results back to the blackboard
- Both Commander and specialists currently call `co.provider.Chat(..., co.provider.GetDefaultModel(), ...)`.
- Specialists get dynamic access to tools in the execution registry.
- When available, Commander now reuses the shared main-agent execution registry instead of always creating a separate minimal tool set.

What this means:

- Commander mode is multi-agent, not truly multi-model yet.
- The routing unit is specialist role, not model tier.
- The blackboard summary is the main context bridge between cycles.

## 5. Context Reduction Paths

### Main agent path

Code:
- `pkg/tools/registry.go:146`
- `pkg/tools/filters/filter.go:134`
- `pkg/agent/loop.go:860`
- `pkg/agent/loop.go:906`

Current behavior:

- Tool output is filtered only if a matching output filter is registered.
- If no filter exists, raw `ForLLM` output is added to the conversation.
- Session summarization happens later when history gets large.

Implication:

- This path is dynamic, but relevance selection is still incomplete.
- It is easy for unoptimized tools to leak too much raw output into context.

### CLAW/orchestrator path

Code:
- `pkg/parsers/nmap.go:109`
- `pkg/parsers/httpx.go:49`
- `pkg/parsers/nuclei.go:48`
- `pkg/parsers/subdomain.go:14`
- `pkg/parsers/llm_parser.go:34`

Current behavior:

- Known tools can be parsed into typed artifacts.
- Unknown or unsupported tool output can fall back to LLM parsing.
- The orchestrator carries forward artifact summaries and graph state instead of raw output when possible.

Implication:

- This is the stronger foundation for smart context handling.
- It should eventually become the common pattern for more of the system.

## 6. Where Dynamic Tooling Exists Today

Code:
- `pkg/tools/registry.go:213`
- `pkg/orchestrator/commander.go:389`
- `pkg/graph/frontier.go:281`

Current behavior:

- The main agent exposes tools dynamically from the tool registry.
- Commander specialists also receive tools dynamically from the execution registry.
- The frontier can recommend tools based on missing graph properties.

Caveat:

- Some CLAW pipeline definitions and phase contracts still hardcode specific tool names like `subfinder`, `nmap`, `httpx`, and `nuclei`.
- So the system is only partially capability-driven today.

## 7. Confirmed Current-State Conclusions

- Multi-model routing is real in the main agent loop.
- Supervision is real in the main agent loop, though tests currently indicate issues in that path.
- Blackboard and orchestrator are about structured state and execution flow, not model selection.
- The pipeline orchestrator is single-model today, but has better output-to-artifact reduction than the main loop.
- Commander is multi-agent in design, but single-model in current implementation.

## 8. Practical Memory Aids

If you need to remember the system quickly:

- `agent loop` = where multi-model routing exists now
- `blackboard` = shared typed memory, not routing
- `pipeline orchestrator` = single-model, strong parsing/artifact path
- `commander` = multi-agent workflow, still same model by default
