# Tier Routing Guide

## Overview

Tier routing enables cost-optimized LLM usage by automatically routing different types of tasks to appropriate model tiers:

- **Heavy Tier**: Strategic planning, deep analysis, reporting (expensive, powerful models)
- **Medium Tier**: Code review, tool selection (local models or mid-tier APIs)
- **Light Tier**: Parsing, summarization, triage (fast local models)

This approach can reduce costs by 80-95% compared to using premium models for everything, while maintaining quality where it matters.

## Quick Start

### 1. Setup Environment

```bash
# Set API keys
export ANTHROPIC_API_KEY="sk-ant-..."
export OPENROUTER_API_KEY="sk-or-..."  # Optional
export NVIDIA_API_KEY="nvapi-..."     # Optional

# LM Studio (for local models)
export LM_STUDIO_BASE_URL="http://localhost:1234/v1"
```

### 2. Start LM Studio

1. Download and install [LM Studio](https://lmstudio.ai/)
2. Load your local models:
   - **codestral-22b-v0.1-8bit** (medium tier - code/tool selection)
   - **nvidia-nemotron-3-nano** (light tier - parsing/triage)
   - Or: **glm-4.7-flash** (alternative for both tiers)
3. Start the local server on port 1234

### 3. Configure PicoClaw

```bash
# Copy example config
cp config/config.tier-routing.example.json ~/.picoclaw/config.json

# Edit with your preferences
nano ~/.picoclaw/config.json
```

### 4. Test It

```bash
# Build
make build

# Run with tier routing
./build/picoclaw agent -m "Scan my network 192.168.1.0/24 and identify web services"

# Watch the logs to see model switches:
# [INFO] Routing to tier tier=heavy model=claude-sonnet-4 task=planning
# [INFO] Routing to tier tier=light model=nemotron-nano-local task=parsing
# [INFO] Routing to tier tier=heavy model=claude-sonnet-4 task=analysis
```

## Configuration

### Minimal Configuration

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
        "model_name": "local-model",
        "use_for": ["parsing"],
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
      "model_name": "local-model",
      "model": "lmstudio/your-model-name",
      "api_base": "http://localhost:1234/v1"
    }
  ]
}
```

### Full Configuration

See [`config/config.tier-routing.example.json`](config/config.tier-routing.example.json) for a complete example with:
- 3 tiers (heavy/medium/light)
- Multiple models (Anthropic, OpenRouter, NVIDIA, LM Studio)
- Full cost tracking configuration

## Task Classification

The tier router automatically classifies tasks based on context:

| Task Type | Triggers | Tier | Use Case |
|-----------|----------|------|----------|
| `planning` | Session start, phase change | Heavy | Initial strategy, replanning |
| `analysis` | "analyze", "examine" keywords | Heavy | Deep reasoning about findings |
| `exploitation` | "test", "exploit", "vulnerability" | Heavy | Security testing decisions |
| `report_writing` | Report request | Heavy | Final documentation |
| `tool_selection` | "which tool", "what command" | Medium | Choosing tools to run |
| `code_review` | "code", "review" keywords | Medium | Analyzing code/configs |
| `js_analysis` | "javascript", "js file" | Medium | JavaScript analysis |
| `parsing` | Large tool output (2K-10K chars) | Light | Extracting data from output |
| `summary` | Very large output (>10K chars) | Light | Summarizing large results |
| `triage` | Quick decisions | Light | Fast filtering/sorting |

## Cost Tracking

The tier router tracks costs in real-time:

```bash
# During execution, costs accumulate per session
# At session end, view the cost report:

Session Cost Report
==================
Session: agent:main
Duration: 5m 23s
Total Cost: $0.24

By Tier:
--------
  heavy:
    Calls: 4
    Input tokens: 8234
    Output tokens: 1456
    Cost: $0.24
    Avg latency: 2.3s

  light:
    Calls: 12
    Input tokens: 15430
    Output tokens: 8234
    Cost: $0.00
    Avg latency: 340ms

By Model:
---------
  claude-sonnet-4:
    Calls: 4
    Input tokens: 8234
    Output tokens: 1456
    Cost: $0.24
    Avg latency: 2.3s

  nemotron-nano-local:
    Calls: 12
    Input tokens: 15430
    Output tokens: 8234
    Cost: $0.00
    Avg latency: 340ms
```

## Expected Cost Savings

### Example: Internal Network Scan

**Without tier routing** (all Claude Sonnet):
- 15-20 LLM calls
- ~100K tokens total
- Cost: **$2.00 - $4.00**

**With tier routing**:
- 4 heavy tier calls (planning + analysis): $0.20 - $0.30
- 12-16 light tier calls (parsing): $0.00
- Cost: **$0.20 - $0.30** (85-90% savings)

### Example: Web Application Security Assessment

**Without tier routing** (all Claude Sonnet):
- 50-80 LLM calls
- ~500K tokens total
- Cost: **$10.00 - $15.00**

**With tier routing**:
- 8-12 heavy tier calls: $1.50 - $2.50
- 40-70 light tier calls: $0.00
- Cost: **$1.50 - $2.50** (80-85% savings)

## Model Recommendations

### Heavy Tier (Strategic Work)

**Option 1: Claude Sonnet 4 (Recommended)**
- Excellent reasoning
- Good at security analysis
- Cost: $3/$15 per 1M tokens
- Use for: Planning, analysis, reporting

**Option 2: Kimi K2.5 (Budget)**
- Good reasoning, cheaper
- Via OpenRouter
- Cost: ~$2/$10 per 1M tokens
- Use for: Budget-conscious assessments

**Option 3: Claude Opus 4 (Critical)**
- Best reasoning, expensive
- Use sparingly
- Cost: $15/$75 per 1M tokens
- Use for: Critical decisions only

### Medium Tier (Tool/Code Work)

**Option 1: Codestral 22B (Recommended)**
- Good at code analysis
- Run locally via LM Studio
- Cost: $0
- Use for: JavaScript analysis, code review

**Option 2: GLM-4.7-Flash**
- Fast, decent quality
- Run locally via LM Studio
- Cost: $0
- Use for: Tool selection, quick analysis

### Light Tier (Parsing/Triage)

**Option 1: Nemotron Nano (Recommended)**
- Ultra-fast
- Run locally via LM Studio
- Cost: $0
- Use for: Parsing tool output, triage

**Option 2: GLM-4.7-Flash**
- Can serve both medium and light
- Run locally via LM Studio
- Cost: $0
- Use for: All local tasks

## Troubleshooting

### Tier routing not working

1. Check config:
   ```bash
   cat ~/.picoclaw/config.json | jq '.routing.enabled'
   # Should return: true
   ```

2. Check logs for routing messages:
   ```bash
   # Look for: "Tier routing enabled"
   # Look for: "Routing to tier tier=heavy model=..."
   ```

3. Verify models are loaded:
   ```bash
   # Test LM Studio
   curl http://localhost:1234/v1/models

   # Test Anthropic
   curl -H "x-api-key: $ANTHROPIC_API_KEY" https://api.anthropic.com/v1/messages
   ```

### High costs despite tier routing

1. Check tier assignments:
   - Are tasks being classified correctly?
   - Look at logs: what tier is being used?

2. Adjust classification:
   - Edit `pkg/routing/tier_router.go`
   - Modify `ClassifyTask()` rules

3. Review heavy tier usage:
   - Should be <30% of calls for most workflows
   - If >50%, classification needs tuning

### Local models not loading

1. LM Studio not running:
   ```bash
   # Check if server is up
   curl http://localhost:1234/v1/models
   ```

2. Model not loaded in LM Studio:
   - Open LM Studio
   - Go to "Local Server" tab
   - Load your model
   - Start server

3. Wrong model name in config:
   - Model name in config must match LM Studio
   - Check LM Studio logs for actual model path

## Advanced Usage

### Custom Task Types

Add new task types in `pkg/routing/tier_router.go`:

```go
const (
    // ... existing types ...
    TaskNetworkEnum TaskType = "network_enum"  // Your custom type
)
```

Update classification in `ClassifyTask()`:

```go
if strings.Contains(userLower, "nmap") || strings.Contains(userLower, "scan") {
    return TaskNetworkEnum
}
```

Add to tier config:

```json
{
  "medium": {
    "use_for": ["tool_selection", "network_enum"]
  }
}
```

### Dynamic Tier Selection

Future enhancement: route based on:
- Token budget remaining
- Response latency requirements
- Time of day (API rate limits)
- Session cost so far

### Hybrid Approach

Run both API and local models:
- Heavy tier: Claude Sonnet (API)
- Medium tier: Codestral (local) with Claude fallback (API)
- Light tier: Nemotron (local) always

## Integration with Workflow Engine

When the workflow engine is implemented (Phase 3), tier routing will integrate:

1. **Phase detection**: Automatically route heavy tier during phase transitions
2. **Branch creation**: Use heavy tier when creating new workflow branches
3. **Validation**: Use heavy tier for final validation before reporting
4. **Execution**: Use light/medium tiers for most tool execution

See `METHODOLOGY.md` for the workflow design.

## Performance Metrics

Track these metrics to optimize routing:

1. **Cost per assessment**: Target <$0.50 for internal network scans
2. **Heavy tier percentage**: Target <30% of total calls
3. **Latency**: Heavy tier 2-4s, light tier <500ms
4. **Quality**: Findings should match all-API approach

## Next Steps

1. Test with your internal network
2. Review cost report after first assessment
3. Tune classification rules based on results
4. Add custom task types as needed
5. Integrate with workflow engine (Phase 3)

## Support

- Issues: https://github.com/sipeed/picoclaw/issues
- Implementation plan: `IMPLEMENTATION_PLAN.md`
- Architecture: `STRIKECLAW_ARCHITECTURE.md`
