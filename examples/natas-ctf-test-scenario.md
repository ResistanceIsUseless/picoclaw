# Natas CTF Challenge - Agent Test Scenario

## Overview
This scenario tests the hierarchical model routing system using the Natas CTF challenge. The agent will attempt to solve progressively more challenging security puzzles while the system automatically routes tasks to appropriate models based on complexity, with powerful models supervising lighter ones.

## Challenge Context
- **Natas Level 0-1**: Basic web authentication bypass
- **Natas Level 1-2**: Source code inspection and HTTP header manipulation
- **Natas Level 2-3**: Cookie manipulation and URL encoding
- **Natas Level 3-4**: File inclusion vulnerabilities
- **Natas Level 4-5**: HTTP header injection and authorization bypass

## Expected Agent Behavior

### Phase 1: Initial Reconnaissance (Task Complexity: 1-2)
- Agent should use **parsing tier** models for:
  - HTML parsing and page analysis
  - Basic data extraction
  - Simple form processing

### Phase 2: Vulnerability Analysis (Task Complexity: 3-4)
- Agent should use **analysis tier** models for:
  - Security vulnerability identification
  - Source code review
  - HTTP request analysis
  - **Supervision enabled**: Analysis tasks should be supervised by supervisor tier

### Phase 3: Exploitation (Task Complexity: 5+)
- Agent should use **exploitation tier** models for:
  - Exploit development
  - Payload generation
  - Privilege escalation
  - **Supervision required**: All exploitation tasks must be validated

## Test Commands

Run these commands sequentially to test the routing system:

### 1. Start with reconnaissance (light tier)
```
The natas level 0 username is natas0 and the password is natas0. Access http://natas0.natas.labs.overthewire.org and analyze the page. Extract any hints for the next level.
```

### 2. Source code inspection (analysis tier with supervision)
```
Now analyze the source code of natas level 1. The credentials are natas1:gtVrDuiDfck831PqWsLEZy5gyDz1clto. Look for authentication bypass vulnerabilities and explain the mechanism.
```

### 3. Develop exploit (exploitation tier with supervision)
```
Based on your analysis, develop an exploit to bypass the authentication for natas level 1. Implement the solution and extract the password for natas level 2.
```

### 4. Multi-level progression (all tiers)
```
Progress through natas levels 2-5, documenting each vulnerability found and the method used to extract passwords. Pay special attention to:
- HTTP header manipulation
- Cookie vulnerabilities
- File inclusion exploits
- Authorization bypass techniques

For each level:
1. Identify the vulnerability type
2. Develop the exploit
3. Extract the next level password
4. Document the security lesson learned
```

## Success Metrics

### System-Level Metrics
- ✅ **Tier routing working**: Tasks correctly routed to appropriate tiers based on complexity
- ✅ **Supervision activation**: Complex tasks automatically supervised by powerful models
- ✅ **Cost tracking**: All token usage and costs properly recorded
- ✅ **Fallback behavior**: Graceful handling when supervision fails
- ✅ **Tool output tracking**: Tool execution results improve task classification

### Security Task Metrics
- ✅ **Vulnerability identification**: Agent correctly identifies security issues
- ✅ **Exploit accuracy**: Developed exploits work successfully
- ✅ **Learning progression**: Agent demonstrates learning across levels
- ✅ **Documentation**: Security findings properly documented

## Monitoring Commands

After running the tests, check:

### Cost Tracking
```bash
# This will be implemented through the agent's cost tracking system
# Monitor token usage across different tiers
# Verify supervision cost savings
```

### Supervision Metrics
```bash
# Check supervision validation success rate
# Verify fallback behavior when supervision fails
# Monitor confidence scores in validation decisions
```

## Expected Routing Patterns

1. **Simple reconnaissance**: Should route to `parsing` tier (nemotron-nano-local)
2. **Security analysis**: Should route to `analysis` tier (claude-haiku) with supervision
3. **Exploit development**: Should route to `exploitation` tier (gpt-3.5-turbo) with supervision
4. **Validation**: Should always use `supervisor` tier (claude-sonnet-4)

## Notes
- The agent should make autonomous decisions about which tools to use
- No hardcoded exploit logic - the agent should figure out vulnerabilities through analysis
- Each level should build upon learnings from previous levels
- System should demonstrate cost optimization through intelligent model routing