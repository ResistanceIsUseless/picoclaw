# Implementation Summary: Output Filtering & MCP Integration

## Completed: Output Filtering System

### Problem Solved
Your agent was being overwhelmed with massive tool outputs (web crawls with 1000s of URLs, port scans with verbose output, fuzzing results with 10K+ requests). This filled the context window with noise and made it difficult for the LLM to identify important findings.

### Solution Implemented
A comprehensive **filtering system** that:
1. **Automatically detects** when output exceeds threshold (10KB default)
2. **Saves full output** to disk for later reference
3. **Sends concise summaries** to the LLM with only relevant information
4. **Provides actionable recommendations** based on findings

### Files Created

#### Core Infrastructure
- `pkg/tools/filters/filter.go` - Base filter interface and registry
- `pkg/tools/filters/web_crawler.go` - Web crawling output filter
- `pkg/tools/filters/port_scan.go` - Port scan output filter
- `pkg/tools/filters/fuzzing.go` - Fuzzing output filter
- `pkg/tools/filters/code_analysis.go` - Code analysis output filter

#### Integration
- Modified `pkg/tools/registry.go` to apply filters automatically during tool execution

### How It Works

**Before filtering:**
```
nmap output: 50KB of detailed port information
→ Sent to LLM: Entire 50KB (wastes context)
```

**After filtering:**
```
nmap output: 50KB of detailed port information
→ Filter applied: Extract key findings
→ Sent to LLM: 2KB summary with:
  - Open ports (22/ssh, 80/http, 443/https)
  - Uncommon ports (8000/node.js)
  - Vulnerable services (OpenSSH 7.2 - CVE-2016-10009)
  - Recommendations (test SSH auth bypass, investigate Node.js debug)
→ Full output saved: ~/.picoclaw/workspace/filtered-output/nmap-20260302-143052.txt
```

### Auto-Registered Filters

The system automatically filters these tools:
- **Web crawlers:** crawl, spider, katana, gospider, hakrawler
- **Port scanners:** nmap, masscan, naabu, rustscan
- **Fuzzers:** ffuf, wfuzz, gobuster, feroxbuster, dirsearch
- **Code analysis:** semgrep, bandit, gosec, eslint, sonarqube

### Example Filter Output

**Web Crawl Filter:**
```
Web Crawl Results Summary:
- Total URLs crawled: 1247
- Unique endpoints: 89
- Parameters found: 34

⚠️  Sensitive Files Found (3):
  - https://target.com/.git/config
  - https://target.com/backup.sql
  - https://target.com/.env.example

Interesting Parameters (34):
  - id, user, admin, token, key, session...

🎯 Recommended Next Steps:
  - Investigate sensitive file access
  - Test parameters for injection vulnerabilities
```

**Port Scan Filter:**
```
Port Scan Results Summary:
- Total open ports: 15

⚠️  Potentially Vulnerable Services (2):
  - 22/tcp: OpenSSH 7.2
    → [MEDIUM] OpenSSH User Enumeration (CVE-2016-10009)
  - 80/tcp: Apache 2.4.49
    → [CRITICAL] Apache Path Traversal (CVE-2021-41773)

🎯 Recommended Next Steps:
  - Prioritize exploitation of vulnerable services
  - Search for public exploits for identified CVEs
```

---

## Completed: MCP Infrastructure

### Problem Addressed
You needed a way to integrate **stateful, complex tools** like Ghidra and Burp Suite that require:
- Session management
- Interactive workflows
- Back-and-forth communication
- Process isolation

### Solution Implemented
A complete **MCP (Model Context Protocol) manager** that:
1. Connects to multiple MCP servers (stdio, HTTP, SSE)
2. Discovers tools dynamically
3. Wraps MCP tools as native Tool interface
4. Applies filtering to MCP tool output
5. Auto-starts stdio servers

### Files Created

#### Core MCP Infrastructure
- `pkg/mcp/manager.go` - Central MCP server manager
- `pkg/mcp/http_connection.go` - HTTP/REST transport
- `pkg/mcp/stdio_connection.go` - Stdio subprocess transport
- `pkg/mcp/sse_connection.go` - Server-Sent Events transport (skeleton)
- `pkg/mcp/tool_wrapper.go` - MCP tool wrapper for native integration

#### Documentation
- `docs/mcp-integration-strategy.md` - Complete MCP integration guide

### Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Agent Loop                          │
└───────────────┬─────────────────────────────────────────────┘
                │
                ▼
┌─────────────────────────────────────────────────────────────┐
│                      Tool Registry                          │
│  ┌──────────────────┐         ┌──────────────────────┐     │
│  │  Native Tools    │         │   MCP Tool Wrappers  │     │
│  │  - shell         │         │   - ghidra_*         │     │
│  │  - filesystem    │         │   - burp_*           │     │
│  │  - web_search    │         │   - recon_*          │     │
│  └──────────────────┘         └──────────────────────┘     │
└───────────────┬─────────────────────────┬───────────────────┘
                │                         │
                ▼                         ▼
┌───────────────────────────┐  ┌─────────────────────────────┐
│   Filter Registry         │  │      MCP Manager            │
│   - Auto-filter >10KB     │  │   ┌─────────────────────┐   │
│   - Save full output      │  │   │  Stdio Connection   │   │
│   - Return summaries      │  │   │  (GhidraMCP)        │   │
└───────────────────────────┘  │   └─────────────────────┘   │
                               │   ┌─────────────────────┐   │
                               │   │  HTTP Connection    │   │
                               │   │  (Burp Suite)       │   │
                               │   └─────────────────────┘   │
                               │   ┌─────────────────────┐   │
                               │   │  HTTP Connection    │   │
                               │   │  (Your Recon MCP)   │   │
                               │   └─────────────────────┘   │
                               └─────────────────────────────┘
```

### Configuration Format

```json
{
  "mcp": {
    "enabled": true,
    "servers": {
      "ghidra": {
        "enabled": true,
        "transport": "stdio",
        "binary": "/path/to/ghidra-mcp-server",
        "auto_start": true
      },
      "burp": {
        "enabled": true,
        "transport": "http",
        "url": "http://localhost:9002",
        "api_key": "${BURP_API_KEY}"
      },
      "recon": {
        "enabled": true,
        "transport": "http",
        "url": "http://localhost:8000"
      }
    }
  }
}
```

### Usage Example

```go
// Initialize MCP manager
filterRegistry := filters.NewFilterRegistry("~/.picoclaw/filtered-output")
mcpManager := mcp.NewMCPManager(filterRegistry)

// Register servers
mcpManager.RegisterServer(&mcp.MCPServerConfig{
    Name:      "ghidra",
    Enabled:   true,
    Transport: mcp.TransportStdio,
    Binary:    "/usr/local/bin/ghidra-mcp",
    AutoStart: true,
})

// Connect to all servers
mcpManager.ConnectAll(ctx)

// Discover and register MCP tools
mcp.RegisterMCPToolsInRegistry(ctx, toolRegistry, mcpManager)

// Now agent can use MCP tools like native tools
result := toolRegistry.Execute(ctx, "ghidra_analyze_binary", map[string]any{
    "path": "./challenge",
})
```

---

## MCP Integration Recommendations

### Priority 1: GhidraMCP (Reverse Engineering)
**Why:** Essential for CTF binary challenges and vulnerability research
**Use cases:**
- Disassemble and decompile binaries
- Query functions, strings, cross-references
- Find dangerous function calls (strcpy, system, etc.)
- Analyze exploit primitives

**Implementation:**
```go
// pkg/mcp/ghidra_client.go
type GhidraClient struct {
    manager *MCPManager
}

func (g *GhidraClient) AnalyzeBinary(path string) error
func (g *GhidraClient) QueryFunctions(filter string) ([]Function, error)
func (g *GhidraClient) Decompile(address string) (string, error)
func (g *GhidraClient) FindVulnerabilities() ([]Vulnerability, error)
```

### Priority 2: Burp Suite MCP (Web Testing)
**Why:** Stateful web app testing with proxy and scanner
**Use cases:**
- Maintain HTTP history
- Run active scans
- Use Collaborator for out-of-band detection
- Manual verification via UI

**Implementation:**
```go
// pkg/mcp/burp_client.go
type BurpClient struct {
    manager *MCPManager
}

func (b *BurpClient) StartProxy(target string) error
func (b *BurpClient) GetSiteMap() ([]HTTPRequest, error)
func (b *BurpClient) RunActiveScan(urls []string) (*ScanResults, error)
func (b *BurpClient) GetIssues() ([]Issue, error)
```

### Priority 3: Your Recon MCP (Already Built!)
**Keep using:** Your existing recon MCP server
**Enhancement:** Add more tools and streaming support

### Skip: Kali MCP
**Reason:** Your direct CLI integration + filtering is superior

### Maybe Later: Semgrep MCP
**Reason:** CLI + code analysis filter works well for now

---

## Decision Matrix: When to Use MCP

| Tool Type | Use MCP? | Reason |
|-----------|----------|--------|
| **Stateful (Ghidra, Burp)** | ✅ YES | Need persistent sessions |
| **Interactive (IDA, debuggers)** | ✅ YES | Back-and-forth queries |
| **Heavy (Large frameworks)** | ✅ YES | Avoid bundling in binary |
| **Simple CLI (nmap, subfinder)** | ❌ NO | Direct + filter is faster |
| **Fire-and-forget (one-shot)** | ❌ NO | Native integration sufficient |
| **Vendor-maintained** | ✅ YES | Automatic updates |

---

## Next Steps

### Immediate (Sprint 1)
1. ✅ **Output filtering** - DONE
2. ✅ **MCP infrastructure** - DONE
3. 📝 **Test filters with real tool output**
4. 🔧 **Integrate MCP manager into agent initialization**

### Short-term (Sprint 2-3)
5. 🚀 **Add GhidraMCP** - Critical for reverse engineering
6. 🚀 **Add Burp MCP** - Important for web testing
7. 📚 **Update AGENTS.md** with filtering/MCP guidance

### Medium-term (Sprint 4+)
8. 🤖 **Agent depth improvements** (supervision, reflection)
9. 💾 **Efficient context management** (hierarchical summaries)
10. 🔧 **Tool format standardization** (for model compatibility)
11. 🧠 **Autonomous tool learning** (discovery and installation)

---

## Key Benefits Achieved

### 1. Context Efficiency
- **Before:** 50KB nmap output fills context
- **After:** 2KB summary with key findings

### 2. Agent Focus
- **Before:** LLM distracted by noise
- **After:** LLM sees only relevant information

### 3. Actionable Intelligence
- **Before:** Raw data dumps
- **After:** Recommendations and next steps

### 4. Flexibility
- **Native tools:** Fast, simple, filtered
- **MCP tools:** Stateful, interactive, filtered
- **Best of both worlds**

### 5. Scalability
- Add new filters without changing agent code
- Register new MCP servers dynamically
- Tool ecosystem grows organically

---

## Testing Next

To validate the filtering system:
1. Run real reconnaissance tools
2. Capture their output
3. Apply filters
4. Verify summaries are accurate and concise
5. Test with agent to ensure context usage improves

To validate MCP integration:
1. Set up GhidraMCP server
2. Connect picoclaw to it
3. Analyze a CTF binary
4. Verify agent can query and explore effectively
5. Measure performance impact
