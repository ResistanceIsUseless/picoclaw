package filters

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// PortScanFilter filters port scan output to extract key information
type PortScanFilter struct {
	*BaseFilter
	commonPorts     map[int]string
	interestingOnly bool
}

// NewPortScanFilter creates a new port scan output filter
func NewPortScanFilter(outputDir string) *PortScanFilter {
	return &PortScanFilter{
		BaseFilter:      NewBaseFilter("port_scan", outputDir),
		interestingOnly: true,
		commonPorts: map[int]string{
			21:    "FTP",
			22:    "SSH",
			23:    "Telnet",
			25:    "SMTP",
			53:    "DNS",
			80:    "HTTP",
			110:   "POP3",
			143:   "IMAP",
			443:   "HTTPS",
			445:   "SMB",
			3306:  "MySQL",
			3389:  "RDP",
			5432:  "PostgreSQL",
			5900:  "VNC",
			6379:  "Redis",
			8080:  "HTTP-Alt",
			8443:  "HTTPS-Alt",
			27017: "MongoDB",
		},
	}
}

// PortInfo represents information about an open port
type PortInfo struct {
	Port           int               `json:"port"`
	Protocol       string            `json:"protocol"`
	Service        string            `json:"service"`
	Version        string            `json:"version"`
	IsUncommon     bool              `json:"is_uncommon"`
	VulnSignatures []VulnSignature   `json:"vuln_signatures"`
}

// VulnSignature represents a potential vulnerability signature
type VulnSignature struct {
	Name        string   `json:"name"`
	Severity    string   `json:"severity"`
	CVEs        []string `json:"cves"`
	Description string   `json:"description"`
}

// FilteredPortScan represents the filtered port scan results
type FilteredPortScan struct {
	TotalPorts         int               `json:"total_ports"`
	OpenPorts          []PortInfo        `json:"open_ports"`
	UncommonPorts      []PortInfo        `json:"uncommon_ports"`
	VersionedServices  []PortInfo        `json:"versioned_services"`
	VulnerableServices []PortInfo        `json:"vulnerable_services"`
	Summary            string            `json:"summary"`
}

func (psf *PortScanFilter) Filter(toolName string, output []byte) (string, string, error) {
	// Save full output
	fullPath, err := psf.SaveFullOutput(toolName, output)
	if err != nil {
		return "", "", err
	}

	// Parse scan output
	filtered := psf.analyzePortScanOutput(output)

	// Generate summary
	summary := psf.generateSummary(filtered)

	return summary, fullPath, nil
}

func (psf *PortScanFilter) analyzePortScanOutput(output []byte) *FilteredPortScan {
	result := &FilteredPortScan{
		OpenPorts:          make([]PortInfo, 0),
		UncommonPorts:      make([]PortInfo, 0),
		VersionedServices:  make([]PortInfo, 0),
		VulnerableServices: make([]PortInfo, 0),
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		// Try JSON format first (nmap -oJ, masscan --output-format json)
		var jsonData map[string]interface{}
		if err := json.Unmarshal([]byte(line), &jsonData); err == nil {
			psf.parseJSONPort(jsonData, result)
			continue
		}

		// Parse nmap text format: "22/tcp open ssh OpenSSH 7.4 (protocol 2.0)"
		if portInfo := psf.parseNmapLine(line); portInfo != nil {
			psf.addPortInfo(result, portInfo)
		}
	}

	result.TotalPorts = len(result.OpenPorts)
	return result
}

func (psf *PortScanFilter) parseJSONPort(data map[string]interface{}, result *FilteredPortScan) {
	// Handle different JSON formats from various scanners
	if ports, ok := data["ports"].([]interface{}); ok {
		for _, p := range ports {
			if portMap, ok := p.(map[string]interface{}); ok {
				portInfo := psf.extractPortInfo(portMap)
				if portInfo != nil {
					psf.addPortInfo(result, portInfo)
				}
			}
		}
	}
}

func (psf *PortScanFilter) parseNmapLine(line string) *PortInfo {
	// Regex for nmap output: PORT STATE SERVICE VERSION
	nmapRegex := regexp.MustCompile(`(\d+)/(tcp|udp)\s+open\s+(\S+)(?:\s+(.+))?`)
	matches := nmapRegex.FindStringSubmatch(line)

	if len(matches) >= 4 {
		port, _ := strconv.Atoi(matches[1])
		version := ""
		if len(matches) > 4 {
			version = strings.TrimSpace(matches[4])
		}

		return &PortInfo{
			Port:     port,
			Protocol: matches[2],
			Service:  matches[3],
			Version:  version,
		}
	}

	return nil
}

func (psf *PortScanFilter) extractPortInfo(data map[string]interface{}) *PortInfo {
	portInfo := &PortInfo{}

	if port, ok := data["port"].(float64); ok {
		portInfo.Port = int(port)
	} else if port, ok := data["portid"].(float64); ok {
		portInfo.Port = int(port)
	}

	if protocol, ok := data["protocol"].(string); ok {
		portInfo.Protocol = protocol
	}

	if service, ok := data["service"].(string); ok {
		portInfo.Service = service
	} else if serviceName, ok := data["name"].(string); ok {
		portInfo.Service = serviceName
	}

	if version, ok := data["version"].(string); ok {
		portInfo.Version = version
	} else if product, ok := data["product"].(string); ok {
		portInfo.Version = product
	}

	return portInfo
}

func (psf *PortScanFilter) addPortInfo(result *FilteredPortScan, portInfo *PortInfo) {
	// Check if uncommon
	if _, exists := psf.commonPorts[portInfo.Port]; !exists {
		portInfo.IsUncommon = true
		result.UncommonPorts = append(result.UncommonPorts, *portInfo)
	}

	// Check if versioned
	if portInfo.Version != "" {
		result.VersionedServices = append(result.VersionedServices, *portInfo)

		// Check for known vulnerabilities
		if vulns := psf.checkVulnerabilities(portInfo); len(vulns) > 0 {
			portInfo.VulnSignatures = vulns
			result.VulnerableServices = append(result.VulnerableServices, *portInfo)
		}
	}

	result.OpenPorts = append(result.OpenPorts, *portInfo)
}

func (psf *PortScanFilter) checkVulnerabilities(portInfo *PortInfo) []VulnSignature {
	vulns := make([]VulnSignature, 0)

	versionLower := strings.ToLower(portInfo.Version)

	// Known vulnerable versions (simplified - in production use CVE database)
	vulnerablePatterns := map[string]VulnSignature{
		"openssh 7.2":    {Name: "OpenSSH User Enumeration", Severity: "MEDIUM", CVEs: []string{"CVE-2016-10009"}},
		"openssh 7.4":    {Name: "OpenSSH sftp-server Permissions", Severity: "LOW", CVEs: []string{"CVE-2017-15906"}},
		"apache 2.4.49":  {Name: "Apache Path Traversal", Severity: "CRITICAL", CVEs: []string{"CVE-2021-41773"}},
		"apache 2.4.50":  {Name: "Apache Path Traversal", Severity: "CRITICAL", CVEs: []string{"CVE-2021-42013"}},
		"proftpd 1.3.3c": {Name: "ProFTPd Backdoor", Severity: "CRITICAL", CVEs: []string{"CVE-2010-4221"}},
		"vsftpd 2.3.4":   {Name: "VSFTPD Backdoor", Severity: "CRITICAL", CVEs: []string{"CVE-2011-2523"}},
		"mysql 5.5":      {Name: "MySQL Multiple Vulnerabilities", Severity: "HIGH", CVEs: []string{"CVE-2016-6662"}},
	}

	for pattern, vuln := range vulnerablePatterns {
		if strings.Contains(versionLower, pattern) {
			vulns = append(vulns, vuln)
		}
	}

	// Generic checks
	if strings.Contains(versionLower, "telnet") {
		vulns = append(vulns, VulnSignature{
			Name:        "Unencrypted Telnet",
			Severity:    "HIGH",
			Description: "Telnet transmits credentials in plaintext",
		})
	}

	if strings.Contains(versionLower, "ftp") && !strings.Contains(versionLower, "ftps") {
		vulns = append(vulns, VulnSignature{
			Name:        "Unencrypted FTP",
			Severity:    "MEDIUM",
			Description: "FTP may transmit credentials in plaintext",
		})
	}

	return vulns
}

func (psf *PortScanFilter) generateSummary(filtered *FilteredPortScan) string {
	var summary strings.Builder

	summary.WriteString(fmt.Sprintf("Port Scan Results Summary:\n"))
	summary.WriteString(fmt.Sprintf("- Total open ports: %d\n", filtered.TotalPorts))

	if len(filtered.OpenPorts) > 0 {
		summary.WriteString("\nOpen Ports:\n")
		for i, port := range filtered.OpenPorts {
			if i >= 20 {
				summary.WriteString(fmt.Sprintf("  ... and %d more\n", len(filtered.OpenPorts)-20))
				break
			}
			serviceName := port.Service
			if name, exists := psf.commonPorts[port.Port]; exists && serviceName == "" {
				serviceName = name
			}
			summary.WriteString(fmt.Sprintf("  - %d/%s: %s", port.Port, port.Protocol, serviceName))
			if port.Version != "" {
				summary.WriteString(fmt.Sprintf(" (%s)", port.Version))
			}
			summary.WriteString("\n")
		}
	}

	if len(filtered.UncommonPorts) > 0 {
		summary.WriteString(fmt.Sprintf("\n🔍 Uncommon Ports (%d):\n", len(filtered.UncommonPorts)))
		for i, port := range filtered.UncommonPorts {
			if i >= 10 {
				summary.WriteString(fmt.Sprintf("  ... and %d more\n", len(filtered.UncommonPorts)-10))
				break
			}
			summary.WriteString(fmt.Sprintf("  - %d/%s: %s %s\n", port.Port, port.Protocol, port.Service, port.Version))
		}
	}

	if len(filtered.VulnerableServices) > 0 {
		summary.WriteString(fmt.Sprintf("\n⚠️  Potentially Vulnerable Services (%d):\n", len(filtered.VulnerableServices)))
		for i, port := range filtered.VulnerableServices {
			if i >= 10 {
				summary.WriteString(fmt.Sprintf("  ... and %d more\n", len(filtered.VulnerableServices)-10))
				break
			}
			summary.WriteString(fmt.Sprintf("  - %d/%s: %s %s\n", port.Port, port.Protocol, port.Service, port.Version))
			for _, vuln := range port.VulnSignatures {
				summary.WriteString(fmt.Sprintf("    → [%s] %s", vuln.Severity, vuln.Name))
				if len(vuln.CVEs) > 0 {
					summary.WriteString(fmt.Sprintf(" (%s)", strings.Join(vuln.CVEs, ", ")))
				}
				summary.WriteString("\n")
			}
		}
	}

	// Recommendations
	if len(filtered.VulnerableServices) > 0 || len(filtered.UncommonPorts) > 0 {
		summary.WriteString("\n🎯 Recommended Next Steps:\n")
		if len(filtered.VulnerableServices) > 0 {
			summary.WriteString("  - Prioritize exploitation of vulnerable services\n")
			summary.WriteString("  - Search for public exploits for identified CVEs\n")
		}
		if len(filtered.UncommonPorts) > 0 {
			summary.WriteString("  - Investigate uncommon ports for custom services\n")
		}
		if len(filtered.VersionedServices) > 0 {
			summary.WriteString("  - Cross-reference service versions with vulnerability databases\n")
		}
	}

	return psf.TruncateSummary(summary.String())
}
