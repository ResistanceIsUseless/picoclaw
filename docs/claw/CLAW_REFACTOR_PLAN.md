# CLAW Refactor Plan

## Overview
Transforming picoclaw into CLAW - a phase-isolated, graph-based autonomous security agent platform.

## What to Keep from Picoclaw

### ✅ Keep As-Is
- `pkg/providers/` - Anthropic & Ollama clients (core abstraction)
- `pkg/logger/` - Logging infrastructure
- `pkg/config/` - Configuration loading (extend for pipeline.yaml)
- `pkg/utils/` - General utilities
- `pkg/mcp/` - MCP client (extend with injection filter and RAG)

### 🔄 Refactor & Extend
- `pkg/agent/` - Refactor for phase-scoped execution
  - Keep: Basic agent loop structure
  - Replace: Context building (flat → phase-scoped)
  - Remove: Global state, chat memory

- `pkg/tools/` - Extend with tier system
  - Keep: Tool interface, registry structure
  - Add: Tier enforcement (-1, 0, 1, 2, 3)
  - Integrate: Our new filter system

- `pkg/routing/` - Extend for multi-model phase routing
  - Keep: Model selection logic
  - Add: Phase-specific routing rules

### ❌ Strip (Not Needed for CLAW)
- `pkg/channels/` - Chat platform integrations (Discord, Telegram, Slack)
- `pkg/cron/` - Scheduled tasks
- `pkg/heartbeat/` - Periodic prompts
- `pkg/voice/` - Voice interface
- `pkg/tui/` - Terminal UI
- `pkg/skills/` - Skill system (replaced by pipeline phases)
- `pkg/workflow/` - Old workflow system (replaced by DAG orchestrator)
- `pkg/session/` - Chat session management (replaced by phase isolation)
- `pkg/state/` - Old state management (replaced by Blackboard)
- `pkg/bus/` - Event bus (replaced by Blackboard pub/sub)
- `pkg/devices/` - Device management
- `pkg/auth/` - OAuth (not needed initially)
- `pkg/migrate/` - Migration tool
- `pkg/health/` - Health checks

## New Packages to Create

### Core CLAW Architecture
```
pkg/
  orchestrator/          # Phase lifecycle, DAG execution, approval gates
    orchestrator.go
    dag.go
    router.go

  blackboard/            # Typed artifact store with pub/sub
    blackboard.go
    persist.go
    pubsub.go

  graph/                 # Knowledge graph for exploration
    graph.go             # Core graph structure
    entity.go            # Entity type registry
    frontier.go          # Frontier computation
    interest.go          # Interest scoring
    mutations.go         # Graph mutation structs

  artifacts/             # All typed artifact definitions
    types.go             # Core artifact types
    schema.go            # JSON schema validation
    web.go               # Web/cloud artifacts
    network.go           # Network artifacts
    source.go            # Source code artifacts
    firmware.go          # Firmware artifacts
    binary.go            # Binary artifacts

  registry/              # Enhanced tool registry
    registry.go          # Tool definitions
    tiers.go             # Tier enforcement
    parsers.go           # Output parser assignments

  compression/           # 3-layer output compression
    layer1.go            # Structural parsers
    layer2.go            # LLM summarization
    layer3.go            # ChainAST compaction
    chainast.go          # ChainAST implementation

  episodic/              # Episodic memory (learning from failures)
    store.go             # SQLite + sqlite-vec store
    retriever.go         # Cosine similarity retrieval

  approval/              # Human approval gates
    approval.go          # Approval interface
    cli.go               # CLI approval
    telegram.go          # Telegram approval (future)

  pipeline/              # Phase definitions
    phases/
      recon.go
      network.go
      source.go
      firmware.go
      binary.go
      exploit.go
      report.go
```

### Enhanced Existing Packages
```
pkg/
  mcp/                   # Extend existing MCP package
    filter.go            # Injection defense (NEW)
    toolrag.go           # MCP-RAG tool discovery (NEW)

  agent/                 # Refactor existing agent
    context.go           # Phase-scoped context builder (REPLACE)
    state.go             # DAG state renderer (NEW)
    contract.go          # Phase contract validation (NEW)
    compressor.go        # Compression pipeline caller (NEW)
```

## Implementation Strategy

### Phase 1: Foundation (Current Sprint)
Focus on core infrastructure that everything else depends on.

**Priority Order:**
1. Create Blackboard (artifact storage + pub/sub)
2. Define typed artifacts (all domains)
3. Implement Tool Registry with tiers
4. Refactor agent context builder (phase-scoped)
5. Implement Layer 1 compression (structural parsers)

**Why this order:**
- Blackboard is the system of record - everything writes to it
- Artifacts are the contracts - define them early
- Tool registry gates all execution - must be solid
- Context builder determines what models see - critical
- Compression keeps context manageable - needed immediately

### Phase 2: Knowledge Graph (Week 2)
Build the exploration model.

1. Graph core (nodes, edges, properties)
2. Entity type registry
3. Graph mutations from tool output
4. Frontier computation
5. Interest scoring (web domain first)

### Phase 3: Orchestrator (Week 2-3)
Build the execution engine.

1. DAG loader (pipeline.yaml)
2. Phase lifecycle management
3. DAGState renderer
4. PhaseContract validation
5. Approval gates (CLI only)

### Phase 4: Web Pipeline (Week 3-4)
First complete pipeline end-to-end.

1. Web phase chain (recon → enum → scan → exploit → report)
2. Web-specific parsers and mutations
3. Model routing (Ollama for extraction, Claude for reasoning)
4. End-to-end test on lab target

### Phase 5+: Additional Pipelines
Network, source code, firmware, binary - one at a time.

## Migration Path

### Step 1: Create Parallel Structure
- Don't delete anything initially
- Create new packages alongside old
- New cmd/claw binary alongside cmd/picoclaw

### Step 2: Port Incrementally
- Move provider code first (minimal changes)
- Refactor agent code with phase isolation
- Test each component in isolation

### Step 3: Integration Testing
- Web pipeline end-to-end
- Network pipeline end-to-end
- Cross-domain artifact flow

### Step 4: Cleanup
- Remove unused picoclaw packages
- Archive old code for reference
- Update documentation

## File Organization

```
picoclaw/                          # Current picoclaw (untouched)
  pkg/
  cmd/picoclaw/

claw/                              # New CLAW structure (symlink or git branch)
  pkg/
    orchestrator/
    blackboard/
    graph/
    artifacts/
    compression/
    episodic/
    approval/
    pipeline/

    # Ported from picoclaw
    providers/
    logger/
    config/
    utils/
    mcp/

    # Refactored from picoclaw
    agent/
    tools/
    routing/

  cmd/claw/
  config/
    pipeline.yaml

  docs/
    architecture.md
    phase-contracts.md
    knowledge-graph.md
```

## Critical Design Decisions to Make

### Decision 1: In-Place Refactor vs Clean Fork
**Option A (Recommended):** Create `cmd/claw` alongside `cmd/picoclaw`, share packages
- Pros: Can reference old code, gradual migration, keep picoclaw working
- Cons: Directory structure gets complex

**Option B:** Git branch and major refactor
- Pros: Clean slate, clear separation
- Cons: Harder to reference old code, breaks existing picoclaw

**Decision:** Option A - create `cmd/claw` with new orchestrator, gradually port packages.

### Decision 2: Graph Store Backend
**Options:**
- A: In-memory with JSON serialization (simple, fast)
- B: SQLite with custom schema (queryable, persistent)
- C: Embedded graph DB like Cayley (powerful, complex)

**Decision:** Start with A (in-memory + JSON), migrate to B if needed.

### Decision 3: Artifact Serialization
**Options:**
- A: JSON only (simple, portable)
- B: Protocol Buffers (efficient, typed)
- C: Custom binary format (optimal, complex)

**Decision:** A (JSON) - matches existing picoclaw patterns, easy debugging.

## Next Actions

1. ✅ Create this plan document
2. Create `pkg/blackboard/` package
3. Create `pkg/artifacts/types.go` with core artifact structs
4. Create `pkg/registry/tiers.go` with tier system
5. Refactor `pkg/agent/context.go` for phase isolation
6. Implement first Layer 1 parsers (nmap, subfinder, httpx)

## Testing Strategy

### Unit Tests
- Blackboard pub/sub mechanics
- Graph frontier computation
- Tier enforcement
- Artifact validation
- Parser correctness

### Integration Tests
- Phase → tool → compression → artifact flow
- Graph mutation → frontier update
- Cross-domain artifact sharing
- Resume-on-failure

### End-to-End Tests
- Full web pipeline on DVWA
- Full network pipeline on lab network
- Source code pipeline on vulnerable repo
- Multi-domain pipeline with artifact crossover

## Success Criteria

### Phase 1 Complete When:
- Blackboard stores and retrieves typed artifacts
- Tool registry enforces tier restrictions
- Agent uses phase-scoped context (no global state)
- At least 3 Layer 1 parsers working (nmap, subfinder, httpx)
- Simple test: operator prompt → recon phase → SubdomainList artifact

### Phase 2 Complete When:
- Knowledge graph stores entities and relationships
- Graph mutations created from tool output
- Frontier computation identifies unexplored entities
- Interest scoring ranks exploration priorities
- Simple test: subdomain → graph → frontier shows ports to scan

### Phase 3 Complete When:
- Orchestrator loads pipeline.yaml
- Phase transitions based on artifact availability
- DAGState prevents wrong-order tool calls
- PhaseContract enforces completion requirements
- Simple test: full web chain executes autonomously

## Risk Mitigation

### Risk: Scope Creep
**Mitigation:** Strict adherence to phase boundaries. Each phase must be fully functional before moving to next.

### Risk: Context Window Overrun
**Mitigation:** Implement compression early (Phase 1). Test with small models (Ollama) to catch issues fast.

### Risk: Graph Complexity
**Mitigation:** Start simple (in-memory), prove concept with web domain only, then expand.

### Risk: MCP Reliability
**Mitigation:** Tool execution errors become graph properties (tried/failed), not fatal.

### Risk: Agent Hallucination
**Mitigation:** DAGState enforcement prevents hallucination structurally. Test extensively with phase contracts.

## Timeline Estimate

- **Phase 1 (Foundation):** 2 weeks
- **Phase 2 (Knowledge Graph):** 1 week
- **Phase 3 (Orchestrator):** 1 week
- **Phase 4 (Web Pipeline):** 1-2 weeks
- **Phase 5+ (Other Pipelines):** 1 week each

**Total for MVP (Web pipeline working):** ~6 weeks
**Total for full platform:** ~12 weeks

## Open Questions

1. Should we use existing output filter system or replace with Layer 1 parsers?
   - **Answer:** Use both - filters for context management, parsers for graph mutations

2. How to handle partial tool failures in phase execution?
   - **Answer:** Mark as graph property, continue if non-critical, escalate if required tool

3. Should cross-domain artifacts trigger new phases automatically?
   - **Answer:** Yes via Blackboard pub/sub - phases subscribe to artifact types

4. How to handle operator interruption mid-phase?
   - **Answer:** Blackboard persistence + phase checkpointing

5. Initial model routing defaults?
   - **Answer:** Ollama for extraction/classification, Claude for reasoning/exploitation
