---
name: network-scan
description: Internal network reconnaissance and service discovery
phases: [discovery, enumeration, analysis, validation, reporting]
---

# Network Scan Workflow

This workflow guides the agent through a systematic internal network reconnaissance mission.

## Phase: discovery

### Steps

- ping_sweep: Discover live hosts on the network (required)
- port_scan: Scan discovered hosts for open ports (required)
- service_detection: Identify services running on open ports (required)

### Completion Criteria

All discovered hosts have been scanned for services.

### Branches

- web_service_found → Enumerate web applications
- smb_discovered → Test SMB shares and authentication
- database_found → Check for default credentials
- ssh_found → Enumerate SSH version and authentication methods
- dns_found → Query DNS for zone transfers and records

## Phase: enumeration

### Steps

- technology_identification: Identify software versions and technologies
- banner_grabbing: Collect service banners for version info
- vulnerability_lookup: Check identified versions against CVE databases

### Completion Criteria

All services have been enumerated and versions identified.

### Branches

- web_app_discovered → Deep web application analysis
- api_endpoint_found → API security testing
- outdated_service_found → Exploit research

## Phase: analysis

### Steps

- vulnerability_assessment: Analyze discovered services for known vulnerabilities (required)
- configuration_review: Check for misconfigurations and weak settings
- credential_testing: Test for default and common credentials

### Completion Criteria

All services have been analyzed for security issues.

### Branches

- critical_vuln_found → Immediate exploitation research
- weak_credentials_found → Privilege escalation paths
- misconfiguration_found → Impact assessment

## Phase: validation

### Steps

- finding_validation: Confirm all findings are accurate and reproducible (required)
- false_positive_elimination: Remove any false positives
- impact_assessment: Determine the severity and impact of findings (required)

### Completion Criteria

All findings have been validated and assessed.

## Phase: reporting

### Steps

- finding_documentation: Document all validated findings with evidence (required)
- remediation_guidance: Provide fix recommendations for each finding (required)
- summary_creation: Create executive summary of mission results (required)

### Completion Criteria

Complete mission report has been generated.
