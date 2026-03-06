Architecture Refactor Guide: Picoclaw 2.0
Objective: Evolve from static, linear tool execution into a dynamic, intent-driven AI security agent utilizing Just-In-Time Context and a Hierarchical Multi-Agent System (MAS).
Phase 1: The "Keep vs. Replace" Audit
Before writing new code, reorganize your existing codebase.
✅ KEEP (Your Foundation):
The Blackboard / Knowledge Graph (pkg/blackboard): This is your "Shared State." It is perfect. It ensures the LLMs don't have to memorize raw data.
Layer 1 & 2 Parsers (pkg/parsers): This is your Semantic Distillation layer. Keep routing verbose tool outputs (like Katana) to local/cheap models to extract structured JSON.
Tool Wrappers (pkg/tools): Your sandboxed execution functions for Nmap, Subfinder, etc.
Multi-Model Router (pkg/models): Keep using cheap models for parsing and frontier models for reasoning.
❌ RIPPED OUT (The Bottlenecks):
Static JSON Pipelines (e.g., web_quick.json): Delete these. The agent will now decide the execution order dynamically.
Rigid Phase Contracts: Stop hardcoding "Phase 1 must complete before Phase 2." The orchestrator will now evaluate the Blackboard to decide if a phase is complete.
Monolithic Prompts: Remove any code that loads the entire METHODOLOGY.md into the context window at once.
Phase 2: Dynamic Context Scoping (Slicing the Methodology)
To fix your context window bloat, you must break your METHODOLOGY.md into isolated prompt templates. The agent should only see the rules for the task it is currently doing.
Create a new directory (e.g., pkg/prompts/) and split your methodology into these separate files:
commander_prompt.txt: Focuses only on the "Core Principles" and "Adaptive Behaviors" from your methodology. Instructs the model to act as the router.
recon_prompt.txt: Contains only Phase 1 (Subdomain enum, DNS, port scanning, HTTP probing).
web_analysis_prompt.txt: Contains only Phase 2 (Crawling, JS analysis, Forms, Auth).
api_testing_prompt.txt: Contains only Phase 3 (Discovery, Mapping, Security Testing).
vuln_validation_prompt.txt: Contains only Phase 4 & 5 (Targeted scanning, manual validation).
Rule: An active LLM context window should NEVER contain more than one of these files at a time.
Phase 3: Building the Hierarchical State Machine
This is the core refactor. You need to implement a Cyclical Graph (using a framework like LangGraph, or custom graph logic).
1. Define the Global State Object
Create a global state object that passes between the nodes. It should contain:
user_objective (The plain English prompt: e.g., "Find bugs on example.com")
blackboard_summary (A lightweight summary of what is currently known)
current_errors (Any recent tool failures)
task_queue (Next immediate actions)
2. Build the Nodes (The Agents)
Each node is a Python/Go function that wraps an LLM call.
Node A: Master Orchestrator (Commander Agent)
Input: User prompt + Blackboard Summary.
Prompt injected: commander_prompt.txt.
Job: Look at the state and decide who should work next.
Logic: "I see a domain but no open ports. Route to Recon." OR "I see a /graphql endpoint in the blackboard. Route to API Tester."
Node B: Reconnaissance Agent
Input: Target IP/Domain.
Prompt injected: recon_prompt.txt.
Job: Decide which tools to run (Nmap, Subfinder). Call pkg/tools. Send output to pkg/parsers. Update the Blackboard.
Node C: Web / API Analysis Agent
Input: URL + Specific objective.
Prompt injected: web_analysis_prompt.txt or api_testing_prompt.txt.
Job: Run Katana, extract JS, look for secrets. Update Blackboard.
3. Define the Edges (The Routing Logic)
In a static pipeline, Step A goes to Step B. In an autonomous graph, everything routes back to the Commander.
START → Master Orchestrator
Master Orchestrator → Recon Agent (conditional)
Recon Agent → Master Orchestrator (ALWAYS loops back when done)
Master Orchestrator → API Testing Agent (conditional based on new Blackboard data)
API Testing Agent → Master Orchestrator
Master Orchestrator → END (when objective is complete)
Phase 4: The Execution Flow (How a Plain English Prompt Works Now)
Here is how picoclaw will work once you implement this refactor:
User Input: > picoclaw "Do a deep dive on target.com, I think they have broken access controls."
Node 1 (Orchestrator): The Orchestrator wakes up. It reads the prompt. It checks the Blackboard (empty). It decides to route to the Recon Agent.
Node 2 (Recon): Gets loaded with the recon_prompt.txt methodology. It runs Subfinder and Nmap.
Distillation (Under the hood): Nmap returns 5,000 lines of XML. Your Layer 1 parser (SLM) reads it, extracts "Ports 80, 443 are open", and saves only that to the Blackboard.
Node 1 (Orchestrator): The Orchestrator wakes up again. It sees the updated Blackboard. It sees Port 443 is open. It routes to the Web Analysis Agent.
Node 3 (Web Analysis): Gets loaded with web_analysis_prompt.txt. It runs Katana. Katana finds a weird /api/v2/admin endpoint. The Layer 1 parser adds this to the Blackboard.
Node 1 (Orchestrator): Wakes up. Sees the admin API. Evaluates the user's original prompt ("broken access controls"). Routes to the API Testing Agent specifically to attack that endpoint.
Phase 5: Implementation Checklist

Step 1: Delete or archive the JSON pipeline processing logic.

Step 2: Break METHODOLOGY.md into the 5 discrete text files listed in Phase 2.

Step 3: Implement the Routing logic. If using Python, pip install langgraph and set up a StateGraph. If using Go, build a simple state machine switch statement where functions return the string name of the next node.

Step 4: Build the "Commander" LLM prompt. Tell it: "You are the orchestrator. You do not run tools. You look at the Blackboard, and you decide which agent to invoke next:[RECON, WEB_ANALYSIS, API_TESTING, REPORTING]."

Step 5: Connect your existing pkg/parsers to update the Blackboard before returning control to the Commander.
Why this fixes your problems:
No context window filling up: The Orchestrator never sees the methodology. The Web Agent never sees the Nmap results. They only see exactly what they need.
No rigid automation: If the target is just an IP address with an FTP server, the Orchestrator will skip the Web and API agents entirely because it dynamically evaluates the situation.
Cost Effective: You are still using your excellent Layer 1/Layer 2 parsers to compress data cheaply before the expensive Orchestrator model has to look at it.