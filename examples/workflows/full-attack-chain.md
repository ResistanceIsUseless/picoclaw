---
name: full-attack-chain
description: Full security research methodology — recon, web analysis, API testing, vuln scanning, validation and reporting
phases: [recon, web-analysis, api-testing, vuln-scanning, validation, reporting]
autonomous: true
---

# Security Research Methodology

**AUTONOMOUS EXECUTION**: Execute all steps without waiting for user confirmation. Use tools immediately.

## Core Principles

- **Every step informs the next.** Don't run tools in a fixed sequence. Read the output, think about what it tells you, then decide what to do next.
- **Go deeper, not wider.** Finding one interesting endpoint and fully testing it is worth more than superficially scanning a hundred.
- **Validate everything.** Never report a raw tool output as a finding. Confirm it manually or with a second tool.
- **Install what you need.** If the right tool isn't available, use exec to `pip install` or `go install` it. Don't settle for a worse tool just because it's already there.
- **Track what you've found.** Use workflow_add_finding for confirmed findings. Use workflow_step_complete as you finish each step. Use workflow_create_branch when you discover something that needs deeper investigation.

## Adaptive Behaviors

Follow these patterns whenever the condition is met, regardless of which phase you're in:

- **When you find something unexpected:** Stop and investigate before continuing planned steps. An unexpected 500 error, a different server header, or an unusual redirect often leads to the best findings.
- **When a tool fails:** Don't just report the failure. Try a different tool, different flags, or a different approach. If nmap can't scan a port, try netcat. If nuclei misses something, try manual testing.
- **When you find a technology:** Immediately think about what vulnerabilities are common for that specific technology and version. Check CVE databases.
- **When you find credentials or tokens:** Don't stop at reporting them. Check if they're valid. Check what access they grant. Check if they're reused elsewhere.
- **When you're not finding anything:** Change your approach. Try different wordlists, scan different port ranges, look at the target from a different angle.
- **When output is large:** Don't dump it all to context. Filter, grep, or summarize it first. Pull out the interesting bits and discard the noise.

---

## Phase: recon

**Goal**: Understand the target's attack surface before touching anything.

**Action**: Run these commands NOW using the exec tool. Do not simulate output.

### Steps

- subdomain_enum: **USE exec** with `subfinder -d TARGET -silent` or `amass enum -passive -d TARGET` to enumerate subdomains. Parse output into a list of discovered domains. (required)
- dns_resolution: **USE exec** with `dnsx -l subdomains.txt -a -resp-only` or `nmap -sn TARGET` to resolve discovered hosts and identify which are live. (required)
- port_scan: **USE exec** with `nmap -sS -sV -T4 --open TARGET` for service detection on discovered hosts. For large scopes, use `masscan -p1-65535 --rate=1000 TARGET` first, then `nmap -sV` on open ports. (required)
- http_probing: **USE exec** with `httpx -l hosts.txt -sc -title -tech-detect -follow-redirects` to find web services, status codes, titles, and technologies. If httpx not available, use `curl -sI` on each host:port combination. (required)
- non_standard_ports: **USE exec** to scan non-standard ports: `nmap -p 8080,8443,8888,9090,3000,4443,5000,7443,8000,8081,8181,8444,9443,10000 TARGET`. Don't stop at the first layer. (required)

### Completion Criteria

All discovered hosts have been port-scanned with service detection. All web services HTTP-probed with technology identification.

### Branches

- web_service_found → Move to web-analysis phase. **USE exec** with `curl -sI http://HOST:PORT` to grab headers first.
- api_endpoint_found → Move to api-testing phase. Check for /api/docs, /swagger.json, /graphql.
- ssh_found → **USE exec** with `nmap --script ssh2-enum-algos,ssh-auth-methods -p PORT HOST`. Check for weak algorithms and password auth.
- ftp_found → **USE exec** with `ftp HOST` to test anonymous login. Check for writable directories.
- smtp_found → **USE exec** with `nmap --script smtp-commands,smtp-enum-users -p 25 HOST`. Test open relay.
- database_found → **USE exec** with `nmap --script mysql-info,mysql-enum -p PORT HOST` or `nmap --script pgsql-info -p PORT HOST`. Test default credentials.
- smb_discovered → **USE exec** with `smbclient -L //HOST -N` and `enum4linux -a HOST`. Check for null sessions and writable shares.
- dns_found → **USE exec** with `dig axfr @HOST DOMAIN` for zone transfers. `dnsx -ptr` for reverse DNS.
- snmp_found → **USE exec** with `snmpwalk -v2c -c public HOST` to enumerate community strings.
- cloud_infra_found → Check S3 bucket permissions, Azure blob access, GCP misconfigurations.
- subdomain_takeover → If CNAME points to unregistered third-party service, **USE exec** with `curl -sI` to confirm and flag as finding.

---

## Phase: web-analysis

**Goal**: Map the application thoroughly before testing for vulnerabilities.

**Action**: For each web service found in recon, run targeted analysis NOW.

### Steps

- crawl_site: **USE exec** with `katana -u URL -d 3 -jc -kf -ef css,png,jpg,gif,svg,woff,ttf` to crawl with JavaScript parsing. If katana not available, use `gospider -s URL -d 2 --other-source` or `hakrawler -url URL -depth 2`. Save output for review. (required)
- review_headers: **USE exec** with `curl -sI URL` on multiple pages. Look at `Server`, `X-Powered-By`, `Content-Security-Policy`, `X-Frame-Options`, `Strict-Transport-Security`. Note differences across pages — different headers often mean different backend services. (required)
- identify_technology: From headers, cookies, and response patterns, identify the full technology stack. **USE exec** with `whatweb URL` or parse httpx tech-detect output. Note frameworks, languages, servers, CDNs, WAFs. (required)
- javascript_analysis: For every JavaScript file discovered by the crawler: **USE exec** with `curl -s JS_URL -o /tmp/target.js` to download it, then `grep -oP '["'"'"'][a-zA-Z0-9_/.-]*api[a-zA-Z0-9_/.-]*["'"'"']' /tmp/target.js` to extract API endpoints. Also grep for: `fetch(`, `axios`, `XMLHttpRequest`, hardcoded tokens/keys (`apikey`, `secret`, `token`, `password`), internal hostnames, WebSocket URLs (`wss://`, `ws://`), hidden routes, and debug/admin paths. Feed discovered endpoints back into testing. (required)
- form_analysis: Identify all forms — login, search, contact, file upload, registration. **USE exec** with `curl -s URL` and grep for `<form`, `<input`, `type="hidden"`, `type="file"`. Note hidden fields (session tokens, CSRF tokens, internal IDs). Note GET vs POST. Check file upload endpoints for type validation. (required)
- auth_analysis: Identify the authentication mechanism — cookie-based, JWT, OAuth, API key. **USE exec** to examine: cookie flags (Secure, HttpOnly, SameSite), token format (decode JWTs with `echo TOKEN | base64 -d`), session expiration, logout behavior. Look for password reset flows and registration endpoints even on invite-only apps. (required)
- directory_bruteforce: **USE exec** with `feroxbuster -u URL -w /usr/share/wordlists/dirb/common.txt --no-recursion -t 20` or `ffuf -u URL/FUZZ -w wordlist -mc 200,301,302,403`. Include technology-specific wordlists when the stack is known.

### Completion Criteria

Full site crawled, technology stack identified, JavaScript files analyzed for endpoints and secrets, all forms and auth mechanisms documented.

### Branches

- api_discovered_in_js → Create branch. Extract all API endpoints from JavaScript analysis and feed to api-testing phase.
- credentials_in_js → **CRITICAL**. Use workflow_add_finding immediately. Then **USE exec** to test if credentials are valid.
- admin_panel_found → **USE exec** with `curl -s ADMIN_URL` to probe. Test default credentials. Check for auth bypass.
- file_upload_found → Test file type restrictions. Try uploading shells with double extensions (.php.jpg), null bytes, content-type manipulation.
- websocket_found → **USE exec** with `curl --include --no-buffer -H "Connection: Upgrade" -H "Upgrade: websocket" URL` to test WebSocket security.
- subdomain_variation → If dev/staging/test environments found, prioritize these — they often have weaker security.

---

## Phase: api-testing

**Goal**: Test any APIs found during web analysis or recon.

### Steps

- api_discovery: **USE exec** to check for documentation endpoints: `curl -s URL/api/docs`, `curl -s URL/swagger.json`, `curl -s URL/openapi.json`, `curl -s URL/.well-known/openid-configuration`, `curl -s URL/graphql?query={__schema{types{name}}}`. Map all discovered endpoints. (required)
- graphql_introspection: If GraphQL detected, **USE exec** with `curl -s -X POST URL/graphql -H 'Content-Type: application/json' -d '{"query":"{__schema{queryType{name}mutationType{name}types{name kind fields{name type{name}}}}}"}'`. Identify all queries, mutations, subscriptions. Look for sensitive operations (user management, file access, admin functions). Check query depth/complexity limits. (required if GraphQL found)
- rest_enumeration: For REST APIs, **USE exec** to enumerate endpoints from JavaScript analysis and crawl data. Test each endpoint with `curl -X GET`, `curl -X POST`, `curl -X PUT`, `curl -X DELETE`, `curl -X PATCH`. Test IDOR on any endpoint accepting IDs — increment/decrement the ID, try other users' IDs. (required if REST API found)
- auth_bypass: **USE exec** to test: requests without auth headers, expired/manipulated tokens, JWT algorithm confusion (change alg to "none"), API key in URL vs header, OAuth redirect manipulation. (required)
- authorization_testing: Test horizontal privilege escalation — can user A access user B's resources? **USE exec** to replay requests with different user tokens/sessions. Test vertical privilege escalation — can a regular user access admin endpoints? (required)
- mass_assignment: **USE exec** to send extra fields in POST/PUT requests (e.g., `"role":"admin"`, `"isAdmin":true`, `"price":0`). Check if the API blindly accepts and persists unexpected fields.
- input_validation: **USE exec** to test every parameter with: SQL injection payloads, XSS payloads, command injection, path traversal (../), SSRF (internal URLs), large values, null bytes, special characters. Check error messages for internal details.
- cors_check: **USE exec** with `curl -s -H "Origin: https://evil.com" -I URL` and check `Access-Control-Allow-Origin` and `Access-Control-Allow-Credentials`. Test with null origin.
- rate_limiting: **USE exec** to send rapid requests and check for rate limiting. Test if rate limits apply per-endpoint, per-user, or per-IP.

### Completion Criteria

All API endpoints tested for authentication bypass, authorization flaws, injection, and mass assignment.

### Branches

- idor_found → Document with workflow_add_finding. Test scope — what data is accessible? Can it be chained?
- auth_bypass_confirmed → **CRITICAL**. Document immediately. Test what access it grants.
- graphql_mutation_exposed → Test each sensitive mutation for authorization. Check for batching attacks.
- verbose_errors → Extract internal details (stack traces, file paths, database info) and use them to inform further testing.

---

## Phase: vuln-scanning

**Goal**: Use automated scanners, but only after you understand the target. Don't just blast everything.

### Steps

- targeted_nuclei: **USE exec** with `nuclei -u URL -t TEMPLATE_PATH` using templates relevant to discovered technologies. Don't run all templates blindly — select based on what you found: `-tags cve,misconfig` for general, `-tags wordpress` for WP, `-tags apache` for Apache, etc. (required)
- sqli_testing: For any input that likely touches a database (search, login, ID parameters), **USE exec** with `sqlmap -u "URL?param=value" --batch --level=3 --risk=2` or manual testing with `' OR 1=1--`, `' UNION SELECT NULL--`, time-based: `' AND SLEEP(5)--`. (required)
- xss_testing: For any reflected or stored input, **USE exec** with `dalfox url URL -b YOUR_CALLBACK` or manual payloads: `<script>alert(1)</script>`, `"><img src=x onerror=alert(1)>`, event handlers in attributes. Test both reflected and stored contexts. (required)
- ssrf_testing: For any URL parameter, **USE exec** with payloads pointing to internal services: `http://127.0.0.1`, `http://169.254.169.254/latest/meta-data/` (AWS metadata), `http://[::1]`, DNS rebinding if applicable.
- path_traversal: For file-related endpoints, **USE exec** with `../../../etc/passwd`, `....//....//....//etc/passwd`, null byte injection `%00`, encoding variations `%2e%2e%2f`.
- command_injection: For any parameter that might reach a shell, **USE exec** with `` `id` ``, `$(id)`, `; id`, `| id`, `%0aid`. Test blind injection with time delays: `; sleep 5`.
- ssti_testing: For template injection, **USE exec** with `{{7*7}}`, `${7*7}`, `<%= 7*7 %>`. If math resolves, escalate to RCE payloads.

### Completion Criteria

All discovered input vectors tested for injection. Nuclei run with targeted templates. All scanner findings reviewed (not just reported raw).

### Branches

- sqli_confirmed → Escalate with sqlmap. Dump database schema. Check for stacked queries and file read/write.
- xss_confirmed → Determine context (reflected/stored/DOM). Test for session hijacking, cookie theft, phishing.
- ssrf_confirmed → Map internal network. Check cloud metadata endpoints. Test for port scanning through SSRF.
- rce_confirmed → **CRITICAL**. Document immediately with full reproduction steps. Do NOT escalate further without authorization.

---

## Phase: validation

**Goal**: Confirm findings and eliminate false positives.

### Steps

- reproduce_findings: For each potential finding, **USE exec** to re-run the specific request that triggers it. Confirm it's reproducible and not a one-time fluke. (required)
- eliminate_false_positives: Review each scanner finding critically. WAF-blocked responses, generic error pages, and informational headers are NOT findings. Remove anything that isn't a real security issue. (required)
- assess_impact: For each confirmed finding, determine: Can it be exploited remotely? Does it require authentication? Does it expose sensitive data? Can it be chained with other findings for greater impact? What's the worst case? (required)
- assign_severity: Rate each finding — Critical (RCE, auth bypass, data breach), High (SQLi, stored XSS, IDOR with sensitive data), Medium (reflected XSS, info disclosure, weak crypto), Low (missing headers, verbose errors), Info (technology disclosure, minor misconfig). (required)

### Completion Criteria

All findings confirmed reproducible, false positives removed, impact and severity assessed for each.

---

## Phase: reporting

**Goal**: Produce an actionable security assessment report.

### Steps

- write_findings: **USE write_file** to create a findings report in workspace. For each finding include: title, severity (Critical/High/Medium/Low/Info), affected endpoint or component, description of the vulnerability, steps to reproduce (exact commands and requests), evidence (response snippets, screenshots description), recommended remediation, CVSS score if applicable. (required)
- write_executive_summary: **USE write_file** to create executive summary: scope of assessment, methodology used, total findings by severity, overall risk assessment, top 3 risks, and recommended remediation priorities. (required)
- remediation_guidance: For each finding, provide specific remediation — not just "fix the vulnerability" but the actual code change, configuration update, or patch to apply. (required)

### Completion Criteria

Complete report with detailed findings, reproduction steps, evidence, remediation guidance, and executive summary written to workspace.
