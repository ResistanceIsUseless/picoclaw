---
name: code-review
description: Comprehensive source code security review with static analysis, manual review, and optional fuzzing
phases: [reconnaissance, static-analysis, manual-review, validation, reporting]
---

# Source Code Security Review Workflow

This workflow guides the agent through a systematic source code security assessment using automated tools and manual analysis.

## Phase: reconnaissance

### Steps

- clone_repository: Clone and examine repository structure (required)
- identify_stack: Identify programming languages and frameworks (required)
- map_dependencies: Extract and analyze dependencies (required)
- architecture_mapping: Map entry points and data flows (required)
- identify_attack_surface: List input vectors and dangerous sinks (required)

### Completion Criteria

All entry points identified, technology stack documented, and attack surface mapped.

### Branches

- vulnerable_deps_found → Immediate security updates needed
- secrets_detected → Secret rotation required
- high_risk_patterns → Deep dive into specific vulnerability class
- legacy_code_found → Extra scrutiny for old unmaintained code

## Phase: static-analysis

### Steps

- run_codeql: Execute CodeQL security queries for the language (required)
- run_semgrep: Run Semgrep with security rulesets (required)
- language_linters: Run language-specific security linters (required)
- secret_scanning: Scan for hardcoded secrets and credentials (required)
- dependency_check: Check for known vulnerable dependencies (required)
- container_scanning: Scan Dockerfiles and container images
- iac_scanning: Scan infrastructure-as-code if present

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
