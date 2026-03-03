# MCP Integration Strategy for StrikeClaw

## Priority MCPs for Security Testing

### 1. GhidraMCP (CRITICAL for Reverse Engineering)
**Repository:** https://github.com/LaurieWired/GhidraMCP

**Why Essential:**
- Maintains Ghidra project state across analysis sessions
- Enables iterative binary analysis (disassemble → analyze → query → repeat)
- Agent can ask questions about functions, strings, cross-references
- Perfect for CTF binary exploitation and malware analysis

**Use Cases:**
- Binary CTF challenges (reverse engineering)
- Vulnerability discovery in compiled code
- Malware analysis workflows
- Finding exploit primitives (buffer overflows, format strings)

**Integration Pattern:**
```go
// pkg/mcp/ghidra_client.go
type GhidraClient struct {
    connection *mcp.Connection
    projectPath string
    currentBinary string
}

func (g *GhidraClient) AnalyzeBinary(path string) error {
    // 1. Import binary to Ghidra project
    // 2. Run auto-analysis
    // 3. Return when ready for queries
}

func (g *GhidraClient) QueryFunctions(filter string) ([]Function, error) {
    // Get function list matching criteria
}

func (g *GhidraClient) Decompile(address string) (string, error) {
    // Get decompiled C code for function
}
```

**Agent Workflow:**
```
1. Agent receives binary file
2. Load into Ghidra via MCP
3. Query: "Find functions that call dangerous functions (strcpy, system, etc.)"
4. Decompile suspicious functions
5. Analyze for vulnerabilities
6. Generate exploit PoC
```

---

### 2. Burp Suite MCP (HIGH Priority for Web Testing)
**Repository:** https://github.com/PortSwigger/mcp-server

**Why Valuable:**
- Maintains HTTP history and session state
- Proxy intercepts for manual review
- Automated scanning with context
- Extension ecosystem (Active Scan++, Collaborator)

**Use Cases:**
- Web application penetration testing
- Session management attacks
- Complex authentication workflows
- Out-of-band vulnerability detection (via Collaborator)

**Integration Pattern:**
```go
// pkg/mcp/burp_client.go
type BurpClient struct {
    connection *mcp.Connection
    projectFile string
}

func (b *BurpClient) StartProxy(target string) error {
    // Configure Burp proxy for target
}

func (b *BurpClient) GetSiteMap() ([]HTTPRequest, error) {
    // Retrieve all discovered endpoints
}

func (b *BurpClient) RunActiveScan(urls []string) (*ScanResults, error) {
    // Run Burp Scanner on specific endpoints
}

func (b *BurpClient) GetIssues() ([]Issue, error) {
    // Retrieve all findings
}
```

**Agent Workflow:**
```
1. Agent identifies web target
2. Configure Burp proxy via MCP
3. Run web crawler through proxy (captures all traffic)
4. Analyze site map for interesting endpoints
5. Run active scanner on high-value targets
6. Query for findings
7. Manually verify via Burp UI if needed
```

---

### 3. Semgrep MCP (MEDIUM Priority - Overlap with Filters)
**Repository:** https://github.com/semgrep/semgrep/tree/develop/cli/src/semgrep/mcp

**Why Useful (but not critical):**
- Official Semgrep server with rule management
- Real-time rule updates
- Registry access for community rules

**Consideration:**
- You already have code analysis filters
- For simple scans, CLI + filter is sufficient
- MCP valuable if you need:
  - Custom rule development during analysis
  - Interactive rule refinement
  - Registry browsing

**Recommendation:** Start with CLI integration, add MCP later if needed.

---

### 4. Kali MCP (LOW Priority - Redundant)
**Repository:** https://www.kali.org/tools/mcp-kali-server/

**Why NOT Recommended:**
- Just wraps existing CLI tools (nmap, metasploit, etc.)
- You already have better direct integration
- Adds unnecessary abstraction layer
- Your filtering system handles output better

**Alternative:** Direct tool invocation with your filter system is superior.

---

## Recommended MCP Architecture

### Configuration (`config/config.json`)
```json
{
  "mcp": {
    "enabled": true,
    "servers": {
      "ghidra": {
        "enabled": true,
        "url": "http://localhost:9001",
        "transport": "stdio",
        "binary": "/path/to/ghidra-mcp-server",
        "auto_start": true
      },
      "burp": {
        "enabled": true,
        "url": "http://localhost:9002",
        "api_key": "${BURP_API_KEY}",
        "project_dir": "~/.picoclaw/burp-projects"
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

### MCP Client Manager (`pkg/mcp/manager.go`)
```go
type MCPManager struct {
    servers map[string]*MCPServerConfig
    connections map[string]*mcp.Connection
    filterRegistry *filters.FilterRegistry
}

func (m *MCPManager) ConnectServer(name string) error {
    // Establish connection to MCP server
    // Handle stdio vs HTTP transport
}

func (m *MCPManager) CallTool(server, tool string, args map[string]any) (*ToolResult, error) {
    // 1. Route to appropriate server
    // 2. Call MCP tool
    // 3. Apply filter if registered
    // 4. Return filtered result
}

func (m *MCPManager) AutoStart(server string) error {
    // Launch MCP server as subprocess if auto_start enabled
}
```

### Tool Registration with MCP Awareness
```go
// pkg/tools/registry.go enhancement
func (r *ToolRegistry) RegisterMCPTools(manager *MCPManager) error {
    // Query each MCP server for available tools
    for serverName, conn := range manager.GetConnections() {
        tools, err := conn.ListTools()
        if err != nil {
            continue
        }

        for _, tool := range tools {
            // Wrap MCP tool as native Tool interface
            mcpTool := &MCPToolWrapper{
                server: serverName,
                tool: tool,
                manager: manager,
            }
            r.Register(mcpTool)
        }
    }
}
```

---

## Decision Matrix: When to Use Which Integration

| Scenario | Solution | Reason |
|----------|----------|--------|
| Simple CLI tool (nmap, subfinder) | **Native + Filter** | Fast, no dependencies, your filters handle output better |
| Stateful tool (Ghidra, IDA) | **MCP** | Complex state, iterative analysis required |
| Browser automation | **Native (Playwright/Selenium)** | Better control, easier debugging |
| Interactive proxy (Burp) | **MCP** | Maintains session state, extension ecosystem |
| Code analysis (Semgrep) | **Native + Filter** first, **MCP** if advanced features needed | Your filters handle output well |
| Web crawling | **Native + Filter** | Your web crawler filter is excellent |
| Secret scanning (TruffleHog) | **Native + Filter** or **MCP** | Both work; MCP if you want verification streaming |

---

## Implementation Priority

### Phase 1: Core MCP Infrastructure (Sprint 1)
- [ ] Create `pkg/mcp/` package
- [ ] Implement `MCPManager` with stdio/HTTP transports
- [ ] Add MCP tool wrapper implementing `Tool` interface
- [ ] Integrate with existing `ToolRegistry`
- [ ] Add filtering middleware for MCP tools

### Phase 2: GhidraMCP Integration (Sprint 2)
- [ ] Set up GhidraMCP server
- [ ] Implement `GhidraClient` with common operations
- [ ] Add to AGENTS.md with reverse engineering guidance
- [ ] Test with CTF binary challenges
- [ ] Document workflow for binary analysis

### Phase 3: Burp Suite MCP Integration (Sprint 3)
- [ ] Set up Burp Suite MCP server
- [ ] Implement `BurpClient` for proxy/scanner control
- [ ] Integrate with web application workflows
- [ ] Add to AGENTS.md for web testing
- [ ] Test with vulnerable web apps (DVWA, WebGoat)

### Phase 4: Your Recon MCP Enhancement (Sprint 4)
- [ ] Add more tools to recon_mcp_server.py
- [ ] Implement streaming output for long-running scans
- [ ] Add progress callbacks
- [ ] Optimize filter integration

---

## Example: Autonomous Binary CTF Workflow

```
User: "Solve the binary challenge at ./challenge"

Agent Loop:
1. Detect file type: ELF 64-bit binary
2. Load into Ghidra via MCP:
   ghidra.analyze_binary("./challenge")

3. Query for dangerous functions:
   funcs = ghidra.find_functions_calling(["strcpy", "gets", "system"])

4. Decompile suspicious function:
   code = ghidra.decompile(funcs[0].address)

5. Analyze decompiled code with LLM:
   "This function has a buffer overflow in strcpy at line 15"

6. Find offset via dynamic analysis:
   run_tool("gdb", ["./challenge"], input="pattern_create 200")

7. Generate exploit:
   exploit = generate_exploit(overflow_offset=112, target_function=0x401234)

8. Test exploit:
   run_tool("python", ["exploit.py"])

9. Extract flag:
   "flag{ghidra_mcp_ftw}"
```

---

## Key Advantages of MCP for Your Use Case

1. **Stateful Analysis:** Ghidra/Burp maintain context across queries
2. **Async Operations:** Long-running scans don't block agent
3. **Streaming Results:** Progressive output for large datasets
4. **Tool Reuse:** Multiple agents can share same MCP servers
5. **Official Support:** Vendor-maintained servers (Semgrep, Burp)
6. **Language Flexibility:** Python tools, Java tools, Go agent - all connected

---

## Potential Issues to Watch

1. **Latency:** MCP adds network round-trip (use stdio for local tools)
2. **Complexity:** More moving parts to debug
3. **State Management:** MCP servers need cleanup between tasks
4. **Resource Usage:** Multiple MCP servers consume memory
5. **Dependency Hell:** Agent now depends on external servers

---

## Recommendation

**Start with:**
1. ✅ Your recon MCP (already built, excellent)
2. ✅ Native tools + filters (working great)
3. 🚀 Add GhidraMCP (critical for reverse engineering/CTF)
4. 🚀 Add Burp MCP (valuable for web testing)

**Skip for now:**
- ❌ Kali MCP (redundant with your direct integration)
- ⏸️ Semgrep MCP (use CLI first, MCP only if needed)

Your filtering system is already handling CLI tool output beautifully. MCPs shine when you need **stateful, interactive workflows** - which is exactly what Ghidra and Burp provide for reverse engineering and web testing.
