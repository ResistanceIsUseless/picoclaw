package artifacts

import (
	"fmt"
	"time"

	"github.com/ResistanceIsUseless/picoclaw/pkg/blackboard"
)

// Web/Cloud domain artifacts

// SubdomainList contains discovered subdomains from reconnaissance
type SubdomainList struct {
	Metadata blackboard.ArtifactMetadata `json:"metadata"`

	BaseDomain string     `json:"base_domain"`
	Subdomains []Subdomain `json:"subdomains"`
	Sources    map[string]int `json:"sources"` // tool -> count of findings
	Total      int        `json:"total"`
}

type Subdomain struct {
	Name        string   `json:"name"`
	IPs         []string `json:"ips"`
	Source      string   `json:"source"` // which tool found it
	Verified    bool     `json:"verified"` // DNS resolution successful
	DiscoveredAt time.Time `json:"discovered_at"`
}

func (s *SubdomainList) Type() string { return "SubdomainList" }

func (s *SubdomainList) Validate() error {
	if s.BaseDomain == "" {
		return fmt.Errorf("base_domain cannot be empty")
	}
	if s.Subdomains == nil {
		return fmt.Errorf("subdomains list cannot be nil")
	}
	return nil
}

func (s *SubdomainList) GetMetadata() blackboard.ArtifactMetadata { return s.Metadata }

// PortScanResult contains results from port scanning phase
type PortScanResult struct {
	Metadata blackboard.ArtifactMetadata `json:"metadata"`

	Hosts       []ScannedHost  `json:"hosts"`
	TotalHosts  int            `json:"total_hosts"`
	TotalPorts  int            `json:"total_ports"`
	ScanDuration time.Duration `json:"scan_duration"`
	Scanner     string         `json:"scanner"` // nmap, masscan, etc.
}

type ScannedHost struct {
	Hostname    string       `json:"hostname"`
	IP          string       `json:"ip"`
	Ports       []OpenPort   `json:"ports"`
	OS          OSFingerprint `json:"os,omitempty"`
	Status      string       `json:"status"` // up, down, filtered
	ScannedAt   time.Time    `json:"scanned_at"`
}

type OpenPort struct {
	Port        int            `json:"port"`
	Protocol    string         `json:"protocol"` // tcp, udp
	State       string         `json:"state"` // open, closed, filtered
	Service     string         `json:"service"` // http, ssh, etc.
	Version     string         `json:"version,omitempty"` // service version
	Product     string         `json:"product,omitempty"` // e.g., "Apache httpd"
	ExtraInfo   string         `json:"extra_info,omitempty"`
	Banner      string         `json:"banner,omitempty"`
	Script      map[string]string `json:"script,omitempty"` // nmap script output
}

type OSFingerprint struct {
	Name        string  `json:"name"`
	Accuracy    int     `json:"accuracy"` // 0-100
	OSClass     string  `json:"os_class"`
	OSFamily    string  `json:"os_family"`
	OSGeneration string `json:"os_generation,omitempty"`
}

func (p *PortScanResult) Type() string { return "PortScanResult" }

func (p *PortScanResult) Validate() error {
	if p.Hosts == nil {
		return fmt.Errorf("hosts list cannot be nil")
	}
	if p.Scanner == "" {
		return fmt.Errorf("scanner cannot be empty")
	}
	return nil
}

func (p *PortScanResult) GetMetadata() blackboard.ArtifactMetadata { return p.Metadata }

// ServiceFingerprint contains detailed service identification results
type ServiceFingerprint struct {
	Metadata blackboard.ArtifactMetadata `json:"metadata"`

	Services []IdentifiedService `json:"services"`
	Total    int                 `json:"total"`
}

type IdentifiedService struct {
	Host        string            `json:"host"`
	Port        int               `json:"port"`
	Protocol    string            `json:"protocol"`
	Service     string            `json:"service"`
	Version     string            `json:"version"`
	CPE         []string          `json:"cpe,omitempty"` // Common Platform Enumeration
	Banner      string            `json:"banner,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"` // HTTP headers if applicable
	TLS         *TLSInfo          `json:"tls,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"` // additional findings
	DiscoveredAt time.Time        `json:"discovered_at"`
}

type TLSInfo struct {
	Version     string   `json:"version"`
	Cipher      string   `json:"cipher"`
	Certificate Certificate `json:"certificate"`
	Vulnerabilities []string `json:"vulnerabilities,omitempty"` // Heartbleed, etc.
}

type Certificate struct {
	Subject         string    `json:"subject"`
	Issuer          string    `json:"issuer"`
	NotBefore       time.Time `json:"not_before"`
	NotAfter        time.Time `json:"not_after"`
	SerialNumber    string    `json:"serial_number"`
	SignatureAlg    string    `json:"signature_alg"`
	SubjectAltNames []string  `json:"subject_alt_names,omitempty"`
	IsExpired       bool      `json:"is_expired"`
	IsSelfSigned    bool      `json:"is_self_signed"`
}

func (s *ServiceFingerprint) Type() string { return "ServiceFingerprint" }

func (s *ServiceFingerprint) Validate() error {
	if s.Services == nil {
		return fmt.Errorf("services list cannot be nil")
	}
	return nil
}

func (s *ServiceFingerprint) GetMetadata() blackboard.ArtifactMetadata { return s.Metadata }

// WebFindings contains web application security findings
type WebFindings struct {
	Metadata blackboard.ArtifactMetadata `json:"metadata"`

	Endpoints    []Endpoint        `json:"endpoints"`
	Parameters   []Parameter       `json:"parameters"`
	Technologies []Technology      `json:"technologies"`
	Findings     []WebFinding      `json:"findings"`
	Crawled      CrawlStats        `json:"crawled"`
}

type Endpoint struct {
	URL          string            `json:"url"`
	Method       string            `json:"method"` // GET, POST, etc.
	StatusCode   int               `json:"status_code"`
	ContentType  string            `json:"content_type"`
	ContentLength int              `json:"content_length"`
	ResponseTime time.Duration     `json:"response_time"`
	Headers      map[string]string `json:"headers,omitempty"`
	Title        string            `json:"title,omitempty"`
	Redirect     string            `json:"redirect,omitempty"`
	DiscoveredAt time.Time         `json:"discovered_at"`
	Source       string            `json:"source"` // crawler, fuzzer, etc.
}

type Parameter struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"` // query, body, header, cookie
	URLs        []string `json:"urls"` // where this parameter appears
	SampleValues []string `json:"sample_values,omitempty"`
	Interesting bool     `json:"interesting"` // id, user, admin, etc.
}

type Technology struct {
	Name        string   `json:"name"`
	Version     string   `json:"version,omitempty"`
	Categories  []string `json:"categories"` // web server, cms, framework, etc.
	Confidence  int      `json:"confidence"` // 0-100
	Evidence    []string `json:"evidence"` // how it was detected
}

type WebFinding struct {
	ID          string    `json:"id"`
	Type        string    `json:"type"` // XSS, SQLi, SSRF, etc.
	Severity    string    `json:"severity"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	Parameter   string    `json:"parameter,omitempty"`
	Payload     string    `json:"payload,omitempty"`
	Evidence    string    `json:"evidence"`
	Impact      string    `json:"impact"`
	Remediation string    `json:"remediation"`
	References  []string  `json:"references"`
	Confidence  string    `json:"confidence"` // CONFIRMED, HIGH, MEDIUM, LOW
	CVSS        float64   `json:"cvss,omitempty"`
	CWE         []string  `json:"cwe,omitempty"`
	OWASP       []string  `json:"owasp,omitempty"`
	Tool        string    `json:"tool"` // which tool found it
	Timestamp   time.Time `json:"timestamp"`
}

type CrawlStats struct {
	TotalURLs        int            `json:"total_urls"`
	UniqueHosts      int            `json:"unique_hosts"`
	UniquePaths      int            `json:"unique_paths"`
	StatusCodeDist   map[int]int    `json:"status_code_distribution"`
	ContentTypeDist  map[string]int `json:"content_type_distribution"`
	CrawlDuration    time.Duration  `json:"crawl_duration"`
}

func (w *WebFindings) Type() string { return "WebFindings" }

func (w *WebFindings) Validate() error {
	if w.Endpoints == nil {
		return fmt.Errorf("endpoints list cannot be nil")
	}
	if w.Findings == nil {
		return fmt.Errorf("findings list cannot be nil")
	}
	return nil
}

func (w *WebFindings) GetMetadata() blackboard.ArtifactMetadata { return w.Metadata }

// CloudFindings contains cloud-specific misconfigurations and findings
type CloudFindings struct {
	Metadata blackboard.ArtifactMetadata `json:"metadata"`

	Provider string         `json:"provider"` // aws, azure, gcp, etc.
	Resources []CloudResource `json:"resources"`
	Findings  []CloudFinding  `json:"findings"`
}

type CloudResource struct {
	ID           string            `json:"id"`
	Type         string            `json:"type"` // s3, ec2, lambda, etc.
	Name         string            `json:"name"`
	Region       string            `json:"region"`
	Tags         map[string]string `json:"tags,omitempty"`
	Public       bool              `json:"public"`
	Encrypted    bool              `json:"encrypted"`
	Metadata     map[string]string `json:"metadata"`
	DiscoveredAt time.Time         `json:"discovered_at"`
}

type CloudFinding struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"` // misconfiguration, exposure, etc.
	Severity     string    `json:"severity"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	ResourceID   string    `json:"resource_id"`
	ResourceType string    `json:"resource_type"`
	Impact       string    `json:"impact"`
	Remediation  string    `json:"remediation"`
	Compliance   []string  `json:"compliance,omitempty"` // CIS, NIST, etc.
	Confidence   string    `json:"confidence"`
	Timestamp    time.Time `json:"timestamp"`
}

func (c *CloudFindings) Type() string { return "CloudFindings" }

func (c *CloudFindings) Validate() error {
	if c.Provider == "" {
		return fmt.Errorf("provider cannot be empty")
	}
	if c.Resources == nil {
		return fmt.Errorf("resources list cannot be nil")
	}
	return nil
}

func (c *CloudFindings) GetMetadata() blackboard.ArtifactMetadata { return c.Metadata }
