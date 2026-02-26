---
name: code-review
description: Comprehensive source code security review with static analysis, manual review, and optional fuzzing
phases: [reconnaissance, static-analysis, manual-review, validation, reporting]
autonomous: true
---

# Source Code Security Review Workflow

**AUTONOMOUS EXECUTION**: Execute all steps without waiting for user confirmation. Use tools immediately.

This workflow guides the agent through a systematic source code security assessment using automated tools and manual analysis.

## Phase: reconnaissance

**Action**: If target is already a local directory path, skip cloning and proceed directly to analysis.

### Steps

- examine_structure: Examine directory structure (ls -la, find . -type f -name "*.go" | wc -l) (required)
- identify_stack: Identify programming languages and frameworks (required)
- map_dependencies: Extract and analyze dependencies (go.mod, package.json, etc.) (required)
- architecture_mapping: Map entry points (grep for main functions, HTTP handlers) (required)
- identify_attack_surface: List input vectors (CLI args, HTTP endpoints, file ops) and dangerous sinks (exec.Command, SQL, file I/O) (required)

### Completion Criteria

All entry points identified, technology stack documented, and attack surface mapped.

### Branches

- vulnerable_deps_found → Immediate security updates needed
- secrets_detected → Secret rotation required
- high_risk_patterns → Deep dive into specific vulnerability class
- legacy_code_found → Extra scrutiny for old unmaintained code

## Phase: static-analysis

**Action**: Execute these tools NOW. Do not wait for permission.

### Steps

- run_semgrep: EXECUTE `semgrep --config=auto --severity ERROR --severity WARNING --json .` and parse results (required)
- grep_patterns: EXECUTE grep patterns for command injection, SQL injection, secrets, weak crypto (required)
- parse_semgrep_results: Parse semgrep JSON output, extract findings, assess severity (required)
- secret_scanning: `grep -rn "api.*key.*=\|password.*=\|secret.*=" --include="*.go"` (required)
- dependency_check: Check go.mod for known vulnerable dependencies (required)
- analyze_findings: For each finding, read the surrounding code to validate if it's a true positive (required)

### Completion Criteria

All automated security scanning tools have run and results are triaged.

### Branches

- sql_injection_found → SQL injection deep dive
- command_injection_found → Command injection analysis
- xss_found → Cross-site scripting review
- crypto_issues_found → Cryptography audit
- secrets_in_repo → Secret remediation plan
- container_vulns_found → Container security hardening

## Phase: manual-review

### Steps

- review_authentication: Audit authentication mechanisms (required)
- review_authorization: Check authorization and access control (required)
- review_input_validation: Examine input validation and sanitization (required)
- review_crypto: Audit cryptographic implementations
- review_business_logic: Check for business logic flaws (required)
- review_deserialization: Check for unsafe deserialization
- review_xxe: Review XML parsing for XXE vulnerabilities
- review_ssrf: Check for server-side request forgery
- review_file_operations: Audit file upload and path traversal

### Completion Criteria

All critical code paths have been manually reviewed and validated.

### Branches

- auth_bypass_possible → Authentication bypass investigation
- idor_found → Insecure direct object reference audit
- privilege_escalation → Privilege escalation analysis
- injection_pattern_found → Systematic injection vulnerability review
- race_condition_detected → Concurrency vulnerability analysis

## Phase: validation

### Steps

- validate_critical_findings: Confirm all critical findings are reproducible (required)
- api_fuzzing: Fuzz API endpoints with unexpected inputs
- binary_fuzzing: Fuzz binary parsers and input handlers
- dynamic_testing: Run application with instrumentation to detect issues
- false_positive_elimination: Remove false positives from report (required)
- exploit_development: Create proof-of-concept exploits for critical issues
- impact_assessment: Determine severity and business impact (required)

### Completion Criteria

All findings validated, false positives removed, and exploitability confirmed.

## Phase: reporting

### Steps

- finding_documentation: Document all validated findings with evidence (required)
- severity_ranking: Rank findings by CVSS score and business impact (required)
- remediation_guidance: Provide specific fix recommendations with code examples (required)
- quick_wins: Identify easy-to-fix high-impact issues (required)
- executive_summary: Create executive summary for stakeholders (required)
- technical_report: Generate detailed technical report with POCs (required)

### Completion Criteria

Complete security assessment report delivered with prioritized remediation plan.
