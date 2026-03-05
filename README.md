# CLAW — Context-as-Artifacts, LLM-Advised Workflow

<div align="center">

  **Autonomous Security Assessment Orchestration System**

  [![Go 1.25.7+](https://img.shields.io/badge/Go-1.25.7+-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev)
  [![License: MIT](https://img.shields.io/badge/license-MIT-green)](LICENSE)
  [![Architecture](https://img.shields.io/badge/Arch-x86__64%2C%20ARM64-blue)](#)

  *Phase-isolated pipeline execution • Contract-driven phases • Knowledge graph exploration*

</div>

---

## What is CLAW?

CLAW is an **autonomous security assessment orchestrator** that uses LLMs to intelligently execute security tools, parse their outputs, build a knowledge graph of discovered entities, and explore frontiers until phase contracts are satisfied.

### Key Innovations

- **Phase Isolation** — Each pipeline phase gets a clean context with only relevant artifacts. No prompt pollution.
- **Contract-Driven** — Phases complete when contracts are satisfied (required tools executed, artifacts produced, minimum iterations met).
- **Knowledge Graph** — Discovered entities (domains, IPs, services, vulnerabilities) form a graph. Frontiers (unknown properties) guide exploration.
- **Blackboard Architecture** — Artifacts (SubdomainList, PortScanResult, VulnerabilityReport) are versioned and shared between phases via pub/sub.
- **Layer 1 + Layer 2 Parsers** — Tools have structural parsers (regex, JSON) that create typed artifacts. LLMs validate and enrich as needed.
- **Multi-Model Routing** — Cost-effective tier-based routing: fast models for planning, powerful models for analysis, specialized models for parsing.
- **44+ Security Tools** — Auto-discovered tools (subfinder, nmap, nuclei, httpx, etc.) with declarative metadata and parser definitions.

---

## Quick Start

### Installation

```bash
# Clone repository
git clone https://github.com/ResistanceIsUseless/picoclaw.git
cd picoclaw

# Build CLAW
make build

# Or build specific binary
go build -o build/picoclaw cmd/picoclaw/main.go
go build -o build/test-claw cmd/test-claw/main.go
```

### First-Run Setup

CLAW now features a Crush-inspired interactive setup wizard:

```bash
./build/picoclaw agent
```

On first run, you'll be greeted with:

```
╔═══════════════════════════════════════════╗
║   Welcome to CLAW Security Assistant      ║
╚═══════════════════════════════════════════╝

No configuration found. Let's get you set up!

Step 1: Choose Your Provider
─────────────────────────────

We recommend starting with one of these providers:

  [1] Anthropic Claude    (Best for security analysis)
      Free tier: No | Get key: https://console.anthropic.com

  [2] OpenRouter          (Access 100+ models)
      Free tier: Limited | Get key: https://openrouter.ai/keys

  [3] OpenAI GPT          (Popular, widely supported)
      Free tier: No | Get key: https://platform.openai.com

  [4] Local LM Studio     (Privacy-first, offline)
      Free: Yes | Setup: https://lmstudio.ai

Your choice [1-4]: 1
```

The wizard will:
1. Detect API keys from environment variables (or let you enter manually)
2. Help you select appropriate models for security assessment
3. Test the connection
4. Optionally configure multi-model routing
5. Save configuration to `~/.picoclaw/config.json`

**Manual Configuration:**

If you prefer, create `~/.picoclaw/config.json`:

```json
{
  "model_list": [
    {
      "model_name": "claude-sonnet",
      "model": "anthropic/claude-sonnet-4.6",
      "api_key": "sk-ant-..."
    }
  ],
  "agents": {
    "defaults": {
      "model_name": "claude-sonnet",
      "max_tokens": 8192,
      "temperature": 0.7
    }
  }
}
```

Or use environment variables:

```bash
export ANTHROPIC_API_KEY=sk-ant-...
export OPENROUTER_API_KEY=sk-or-v1-...
```

---

## Usage

### Security Assessment (CLAW Mode)

Run a security assessment against a target:

```bash
# Quick reconnaissance
./build/test-claw -target example.com -pipeline web_quick

# Full web assessment
./build/test-claw -target example.com -pipeline web_full

# With Web UI
./build/test-claw -target example.com -pipeline web_quick -webui :8080
```

**Available Pipelines:**
- `web_quick` — Fast reconnaissance (subdomains, HTTP probing)
- `web_full` — Comprehensive web assessment (recon + vulnerability scanning + exploitation)

**Web UI:**

CLAW includes a real-time web dashboard:

```bash
# Start backend with Web UI
./build/test-claw -target example.com -webui :8080

# In another terminal, start frontend
cd web
npm install
npm run dev
```

Open [http://localhost:5173](http://localhost:5173) to view:
- **Pipeline View** — Real-time phase progress, tool execution log, artifact counts
- **Graph View** — D3.js force-directed knowledge graph with frontier highlighting
- **Tools View** — Browse 44+ security tools organized by tier

### Agent Mode (Interactive Chat)

Use PicoClaw's interactive agent mode:

```bash
# One-shot message
picoclaw agent -m "What can you help me with?"

# Interactive chat
picoclaw agent

# With terminal UI
picoclaw agent --tui
```

---

## Architecture

### Pipeline Execution Flow

```
Target Defined
    ↓
Operator publishes OperatorTarget artifact to blackboard
    ↓
┌─────────────────────────────────────────┐
│ Phase 1: Reconnaissance                 │
│ ─────────────────────────────────────   │
│ Objective: Discover subdomains          │
│ Contract:                                │
│   ✓ Required tools: [subfinder]         │
│   ✓ Required artifacts: [SubdomainList] │
│   ✓ Min iterations: 1                   │
│                                          │
│ LLM calls tools → Parser creates         │
│ SubdomainList artifact → Graph updated  │
│ → Contract satisfied → Phase complete   │
└─────────────────────────────────────────┘
    ↓
┌─────────────────────────────────────────┐
│ Phase 2: Port Scanning                  │
│ ─────────────────────────────────────   │
│ Objective: Identify open ports          │
│ Context: Receives SubdomainList from    │
│          previous phase                 │
│ Contract:                                │
│   ✓ Required tools: [nmap]              │
│   ✓ Required artifacts: [PortScanResult]│
│   ✓ Min iterations: 1                   │
└─────────────────────────────────────────┘
    ↓
... (more phases)
```

### Key Components

**1. Orchestrator** (`pkg/orchestrator/`)
- Executes pipelines phase-by-phase
- Builds phase-scoped prompts with relevant context
- Enforces phase contracts
- Limits iterations per phase

**2. Blackboard** (`pkg/blackboard/`)
- Versioned artifact storage
- Pub/sub for artifact updates
- Persistence to disk (JSONL)

**3. Knowledge Graph** (`pkg/graph/`)
- Property graph (nodes = entities, edges = relationships)
- Frontier computation (unknown properties to explore)
- Neo4j-style Cypher-like queries

**4. Tool Registry** (`pkg/registry/`)
- Auto-discovery of security tools (PATH scanning)
- Tool metadata (name, params, output type)
- Layer 1 parsers (structural regex/JSON → typed artifacts)

**5. Parsers** (`pkg/parsers/`)
- Layer 1: Structural parsing (regex, JSON, XML)
- Layer 2: LLM validation and enrichment
- 20+ parser definitions for common security tools

**6. Phase Contracts** (`pkg/contracts/`)
- Define phase success criteria
- Validate tool execution and artifact production
- Support custom validation functions

---

## Project Structure

```
picoclaw/
├── cmd/
│   ├── picoclaw/          # Interactive agent CLI
│   └── test-claw/         # CLAW orchestrator test harness
├── pkg/
│   ├── agent/             # Agent loop and context
│   ├── blackboard/        # Artifact storage
│   ├── config/            # Configuration and validation
│   ├── contracts/         # Phase contract validation
│   ├── graph/             # Knowledge graph
│   ├── orchestrator/      # Pipeline orchestration
│   ├── parsers/           # Tool output parsers
│   ├── providers/         # LLM provider integrations
│   ├── registry/          # Tool discovery and registration
│   ├── routing/           # Multi-model routing
│   ├── session/           # Conversation persistence
│   ├── tools/             # Built-in tools
│   └── webui/             # Web UI backend (REST + WebSocket)
├── web/                   # React + TypeScript Web UI
│   ├── src/
│   │   ├── components/
│   │   │   ├── Pipeline/  # Real-time pipeline view
│   │   │   ├── Graph/     # D3.js graph visualization
│   │   │   └── Tools/     # Tool registry browser
│   │   ├── api/           # API client
│   │   └── hooks/         # WebSocket hooks
│   └── package.json
├── docs/
│   ├── claw/              # CLAW-specific documentation
│   ├── webui/             # Web UI documentation
│   ├── templates/         # Workspace templates
│   ├── METHODOLOGY.md     # CLAW methodology overview
│   ├── TIER_ROUTING_GUIDE.md  # Multi-model routing guide
│   └── STRIKECLAW_*.md    # StrikeClaw fork documentation
├── examples/              # Example workflows and scenarios
├── pipelines/             # Pipeline definitions
│   ├── web_quick.json
│   └── web_full.json
└── tools/                 # Tool metadata and parsers
    ├── subfinder.json
    ├── nmap.json
    └── ... (44+ tools)
```

---

## Configuration

### Model Configuration (Simple)

```json
{
  "model_list": [
    {
      "model_name": "claude-sonnet",
      "model": "anthropic/claude-sonnet-4.6",
      "api_key": "sk-ant-..."
    }
  ],
  "agents": {
    "defaults": {
      "model_name": "claude-sonnet"
    }
  }
}
```

### Multi-Model Routing (Advanced)

```json
{
  "model_list": [
    {
      "model_name": "gpt-4o-mini",
      "model": "openai/gpt-4o-mini",
      "api_key": "sk-..."
    },
    {
      "model_name": "claude-sonnet",
      "model": "anthropic/claude-sonnet-4.6",
      "api_key": "sk-ant-..."
    },
    {
      "model_name": "gemini-flash",
      "model": "google/gemini-2.0-flash-exp",
      "api_key": "..."
    }
  ],
  "routing": {
    "enabled": true,
    "default_tier": "analysis",
    "tiers": {
      "planning": {
        "model_name": "gpt-4o-mini",
        "use_for": ["task_breakdown", "planning"]
      },
      "analysis": {
        "model_name": "claude-sonnet",
        "use_for": ["security_analysis", "reasoning"]
      },
      "parsing": {
        "model_name": "gemini-flash",
        "use_for": ["tool_output_parsing"]
      }
    }
  }
}
```

**Cost Savings:**
- Planning tier (fast, cheap): $0.15/M tokens
- Analysis tier (powerful): $3/M tokens
- Parsing tier (structured): $0.075/M tokens

Estimated 80-95% cost reduction vs. using premium model for everything.

---

## Supported Providers

CLAW supports 17+ LLM providers:

| Provider | Models | API Base | Notes |
|----------|--------|----------|-------|
| **Anthropic** | Claude Sonnet 4.6, Opus 4.6, Haiku 4.5 | `https://api.anthropic.com/v1` | Recommended for security |
| **OpenRouter** | 100+ models (Claude, GPT, DeepSeek, etc.) | `https://openrouter.ai/api/v1` | Single API for all models |
| **OpenAI** | GPT-4.5-turbo, GPT-4o-mini, o1-preview | `https://api.openai.com/v1` | Reliable, fast |
| **DeepSeek** | DeepSeek-V3 | `https://api.deepseek.com/v1` | Excellent value ($0.14/M) |
| **Google** | Gemini 2.0 Flash, Gemini 1.5 Pro | `https://generativelanguage.googleapis.com/v1beta` | Strong structured output |
| **Groq** | Llama 3.3 70B, Mixtral 8x22B | `https://api.groq.com/openai/v1` | Fast inference |
| **LM Studio** | Any GGUF model | `http://localhost:1234/v1` | Local, privacy-first |
| **Ollama** | Any GGUF model | `http://localhost:11434/v1` | Local, Docker-friendly |

See [docs/TIER_ROUTING_GUIDE.md](docs/TIER_ROUTING_GUIDE.md) for model recommendations.

---

## Tools

CLAW auto-discovers 44+ security tools from your PATH:

### Reconnaissance
- **subfinder**, **amass**, **assetfinder** — Subdomain enumeration
- **httpx**, **httprobe** — HTTP probing
- **dnsx**, **massdns** — DNS resolution

### Scanning
- **nmap** — Port scanning
- **masscan** — Fast port scanning
- **rustscan** — Fast port scanner

### Web Assessment
- **nuclei** — Vulnerability scanning with templates
- **ffuf**, **gobuster**, **dirsearch** — Directory fuzzing
- **arjun**, **paramspider** — Parameter discovery
- **dalfox** — XSS scanning
- **sqlmap** — SQL injection
- **wpscan** — WordPress scanning

### Exploitation
- **metasploit** (msfconsole) — Exploitation framework
- **searchsploit** — Exploit database search

### Analysis
- **wafw00f** — WAF detection
- **whatweb** — Technology detection
- **sslscan**, **sslyze** — SSL/TLS analysis

And many more! Each tool has:
- Metadata (description, params, output type)
- Layer 1 parser (structural regex/JSON)
- Example commands and expected output

---

## Pipelines

Pipelines define multi-phase assessment workflows:

### web_quick (Fast Reconnaissance)

```json
{
  "name": "web_quick",
  "description": "Quick web reconnaissance",
  "phases": [
    {
      "name": "recon",
      "objective": "Discover subdomains",
      "max_iterations": 3,
      "contract": {
        "required_tools": ["subfinder"],
        "required_artifacts": ["SubdomainList"],
        "minimum_iterations": 1
      }
    },
    {
      "name": "http_probe",
      "objective": "Probe discovered subdomains for HTTP services",
      "max_iterations": 2,
      "contract": {
        "required_tools": ["httpx"],
        "required_artifacts": ["WebFindings"],
        "minimum_iterations": 1
      }
    }
  ]
}
```

### web_full (Comprehensive Assessment)

Includes additional phases:
- Port scanning (nmap)
- Vulnerability scanning (nuclei)
- Directory fuzzing (ffuf)
- Parameter discovery (arjun)
- Exploitation attempts (metasploit)

Create custom pipelines by defining phase objectives and contracts.

---

## Web UI

CLAW includes a modern web dashboard built with React + TypeScript + D3.js:

### Features

**Pipeline View:**
- Real-time phase progress with iteration counts
- Tool execution log (last 50 events)
- Contract status (required tools, artifacts, iterations)
- Artifact and graph node counts

**Graph View:**
- Force-directed D3.js visualization
- Color-coded nodes by entity type (domain, IP, service, vulnerability)
- Frontier nodes highlighted (unknown properties to explore)
- Click nodes to inspect properties
- Search and filter

**Tools View:**
- Browse 44+ security tools
- Organized by tier (reconnaissance, scanning, exploitation)
- Tool descriptions and metadata

### Running

```bash
# Start backend
./build/test-claw -target example.com -webui :8080

# Start frontend
cd web
npm install
npm run dev
```

Open [http://localhost:5173](http://localhost:5173)

See [docs/webui/](docs/webui/) for more details.

---

## Development

### Building

```bash
# Build all binaries
make build

# Build specific binary
make build-picoclaw
make build-test-claw

# Run tests
make test

# Run linter
make lint

# Format code
make fmt
```

### Testing

```bash
# Run unit tests
go test ./...

# Run specific test
go test -run TestPhaseContract ./pkg/contracts/

# Run with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. -benchmem ./...
```

### Code Quality

```bash
# Pre-commit checks (format, vet, test)
make check

# Static analysis
make vet

# Security scan
make security
```

---

## Documentation

- **[docs/METHODOLOGY.md](docs/METHODOLOGY.md)** — CLAW methodology overview
- **[docs/TIER_ROUTING_GUIDE.md](docs/TIER_ROUTING_GUIDE.md)** — Multi-model routing guide
- **[docs/WORKFLOW_GUIDE.md](docs/WORKFLOW_GUIDE.md)** — Workflow engine documentation
- **[docs/claw/](docs/claw/)** — CLAW-specific technical docs
- **[docs/webui/](docs/webui/)** — Web UI setup and usage
- **[examples/](examples/)** — Example workflows and scenarios

---

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Areas of Interest

- **Tool Parsers** — Add Layer 1 parsers for new security tools
- **Pipeline Definitions** — Create new assessment workflows
- **LLM Provider Support** — Add new provider integrations
- **Web UI Features** — Enhance dashboard with new views
- **Documentation** — Improve docs, add examples

---

## Credits

CLAW is built on top of [PicoClaw](https://github.com/sipeed/picoclaw), which was inspired by [nanobot](https://github.com/HKUDS/nanobot).

**Key Libraries:**
- [Chi](https://github.com/go-chi/chi) — HTTP router
- [Gorilla WebSocket](https://github.com/gorilla/websocket) — WebSocket support
- [Cobra](https://github.com/spf13/cobra) — CLI framework
- [React](https://react.dev/) — UI library
- [D3.js](https://d3js.org/) — Graph visualization
- [TailwindCSS](https://tailwindcss.com/) — Styling

---

## License

MIT License — see [LICENSE](LICENSE) for details.

---

## Status

**Current Version:** CLAW v0.2.0 (March 2026)

**Features:**
- ✅ Phase-isolated pipeline execution
- ✅ Contract-driven phases with DAG state tracking
- ✅ Knowledge graph with frontier exploration
- ✅ Blackboard artifact store
- ✅ 44+ security tools with Layer 1 + Layer 2 parsers
- ✅ Multi-model routing support
- ✅ Real-time Web UI (React + WebSocket + D3.js)
- ✅ Interactive setup wizard (Crush-inspired UX)
- ✅ Config validation and model recommendations
- 🚧 Mid-session model switching (planned)
- 🚧 Direct prompt handling `picoclaw "scan example.com"` (planned)
- 🚧 Simplified commands `picoclaw scan <target>` (planned)

**Recent Updates:**
- **March 2026** — Interactive setup wizard with provider selection, API key detection, and multi-model routing
- **March 2026** — Config validation system with weak model warnings
- **March 2026** — Complete Web UI with Pipeline, Graph, and Tools views
- **March 2026** — Real-time WebSocket event streaming
- **March 2026** — D3.js force-directed graph visualization with frontier highlighting

---

<div align="center">

  **[Documentation](docs/)** • **[Examples](examples/)** • **[Contributing](CONTRIBUTING.md)** • **[License](LICENSE)**

  Made with ⚡ for autonomous security assessment

</div>
