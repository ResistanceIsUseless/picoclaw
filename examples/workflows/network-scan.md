---
name: network-scan
description: Network reconnaissance, service enumeration, and vulnerability assessment
phases: [discovery, enumeration, analysis, validation, reporting]
autonomous: true
---

# Network Scan Workflow

**AUTONOMOUS EXECUTION**: Execute all steps without waiting for user confirmation. Use tools immediately.

## Phase: discovery

**Action**: Run these commands NOW using the exec tool. Do not simulate output.

### Steps

- ping_sweep: **USE exec** with `nmap -sn TARGET` to discover live hosts. Parse output to count hosts up. (required)
- port_scan: **USE exec** with `nmap -sS -T4 --open -p- TARGET` for full port scan on discovered hosts. If that's too slow, use `nmap -sS -T4 --open --top-ports 1000 TARGET` instead. (required)
- service_detection: **USE exec** with `nmap -sV -T4 --open TARGET` to identify service versions on open ports. Parse the output — extract each host, port, service, and version. (required)

### Completion Criteria

All discovered hosts have been scanned and services identified with version info.

### Branches

- web_service_found → **USE exec** with `curl -sI http://HOST:PORT` to grab headers, then `nikto -h http://HOST:PORT` or `gobuster dir -u http://HOST:PORT -w /usr/share/wordlists/dirb/common.txt`
- smb_discovered → **USE exec** with `smbclient -L //HOST -N` and `enum4linux -a HOST`
- database_found → **USE exec** with `nmap --script mysql-info,mysql-enum -p PORT HOST` or `nmap --script pgsql-info -p PORT HOST`
- ssh_found → **USE exec** with `nmap --script ssh2-enum-algos,ssh-auth-methods -p PORT HOST`
- dns_found → **USE exec** with `dig axfr @HOST DOMAIN` and `nmap --script dns-zone-transfer -p 53 HOST`
- snmp_found → **USE exec** with `snmpwalk -v2c -c public HOST`

## Phase: enumeration

**Action**: For each service found in discovery, run targeted enumeration NOW.

### Steps

- technology_identification: **USE exec** with `nmap -sV --version-intensity 5 -p PORTS HOST` and `curl -sI` on web services to identify exact software versions (required)
- banner_grabbing: **USE exec** with `nmap --script banner -p PORTS HOST` or `echo '' | nc -w3 HOST PORT` to collect service banners (required)
- vulnerability_lookup: For each identified service+version, **USE exec** with `searchsploit SERVICE VERSION` or `nmap --script vulners -sV -p PORT HOST` to check for known CVEs (required)

### Completion Criteria

All services have version info and have been checked against CVE databases.

### Branches

- web_app_discovered → **USE exec** with `nikto -h URL`, `gobuster dir -u URL -w wordlist`, `curl` to probe endpoints
- api_endpoint_found → **USE exec** with `curl -X OPTIONS URL`, test common API paths with `curl`
- outdated_service_found → **USE exec** with `searchsploit SERVICE VERSION` for exploit research

## Phase: analysis

### Steps

- vulnerability_assessment: For each CVE or weakness found, **USE exec** to verify exploitability. Use `nmap --script` NSE scripts, `curl` for web vulns, `searchsploit -m EXPLOIT_ID` to examine exploit code. (required)
- configuration_review: **USE exec** to check for misconfigurations: anonymous FTP (`ftp HOST`), open proxies (`curl -x http://HOST:PORT`), directory listing (`curl URL`), default pages. (required)
- credential_testing: **USE exec** with `hydra -l admin -P /usr/share/wordlists/rockyou.txt HOST SERVICE` or `nmap --script http-default-accounts` for default credential checks. (required)

### Completion Criteria

All services analyzed for vulnerabilities, misconfigurations, and weak credentials.

### Branches

- critical_vuln_found → Document immediately with workflow_add_finding, severity: critical
- weak_credentials_found → Test lateral movement paths
- misconfiguration_found → Assess data exposure impact

## Phase: validation

### Steps

- finding_validation: Re-run key commands to confirm each finding is reproducible. **USE exec** to re-test. (required)
- false_positive_elimination: Review tool output carefully. Remove findings that are informational-only or scanner artifacts. (required)
- impact_assessment: For each confirmed finding, determine: Can it be exploited remotely? Does it expose sensitive data? Can it lead to further compromise? (required)

### Completion Criteria

All findings confirmed reproducible, false positives removed, impact rated.

## Phase: reporting

### Steps

- finding_documentation: **USE write_file** to create a report in workspace with: finding title, severity (critical/high/medium/low/info), affected hosts, evidence (exact command + output), and impact description. (required)
- remediation_guidance: For each finding, provide specific fix: patch version, config change, or mitigation. (required)
- summary_creation: **USE write_file** to create executive summary: scope, methodology, finding counts by severity, top 3 risks, and recommended priorities. (required)

### Completion Criteria

Complete report with findings, evidence, remediation, and executive summary written to workspace.
