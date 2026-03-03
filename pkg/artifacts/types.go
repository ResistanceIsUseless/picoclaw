package artifacts

import (
	"fmt"
	"time"

	"github.com/ResistanceIsUseless/picoclaw/pkg/blackboard"
)

// Common artifact types used across all domains

// OperatorTarget is the initial input from the operator
type OperatorTarget struct {
	Metadata blackboard.ArtifactMetadata `json:"metadata"`

	// Target specification
	Target      string   `json:"target"`       // domain, IP, CIDR, or file path
	TargetType  string   `json:"target_type"`  // web, network, source, firmware, binary
	Scope       []string `json:"scope"`        // additional scope rules
	Exclusions  []string `json:"exclusions"`   // out-of-scope targets
	Description string   `json:"description"`  // optional operator notes
}

func (o *OperatorTarget) Type() string { return "OperatorTarget" }

func (o *OperatorTarget) Validate() error {
	if o.Target == "" {
		return fmt.Errorf("target cannot be empty")
	}
	if o.TargetType == "" {
		return fmt.Errorf("target_type cannot be empty")
	}
	return nil
}

func (o *OperatorTarget) GetMetadata() blackboard.ArtifactMetadata { return o.Metadata }

// NewOperatorTarget creates a new operator target artifact
func NewOperatorTarget(target, targetType string, phase string) *OperatorTarget {
	return &OperatorTarget{
		Metadata: blackboard.ArtifactMetadata{
			Type:      "OperatorTarget",
			CreatedAt: time.Now(),
			Phase:     phase,
			Version:   "1.0",
			Domain:    targetType,
		},
		Target:     target,
		TargetType: targetType,
	}
}

// PipelineSummary aggregates all phase results
type PipelineSummary struct {
	Metadata blackboard.ArtifactMetadata `json:"metadata"`

	StartTime    time.Time         `json:"start_time"`
	EndTime      time.Time         `json:"end_time"`
	Duration     time.Duration     `json:"duration"`
	PhasesRun    []string          `json:"phases_run"`
	ArtifactTally map[string]int    `json:"artifact_tally"` // type -> count
	FindingsSummary map[string]int  `json:"findings_summary"` // severity -> count
	ToolsExecuted []string          `json:"tools_executed"`
	Escalations   []string          `json:"escalations"` // phases that escalated
}

func (p *PipelineSummary) Type() string { return "PipelineSummary" }

func (p *PipelineSummary) Validate() error {
	if p.StartTime.IsZero() {
		return fmt.Errorf("start_time cannot be zero")
	}
	return nil
}

func (p *PipelineSummary) GetMetadata() blackboard.ArtifactMetadata { return p.Metadata }

// VulnerabilityList aggregates all confirmed vulnerabilities across domains
type VulnerabilityList struct {
	Metadata blackboard.ArtifactMetadata `json:"metadata"`

	Vulnerabilities []Vulnerability `json:"vulnerabilities"`
	Summary         VulnSummary     `json:"summary"`
}

type Vulnerability struct {
	ID          string    `json:"id"`            // unique identifier
	Title       string    `json:"title"`
	Severity    string    `json:"severity"`      // CRITICAL, HIGH, MEDIUM, LOW, INFO
	CVE         []string  `json:"cve"`           // associated CVEs if applicable
	CWE         []string  `json:"cwe"`           // CWE classifications
	OWASP       []string  `json:"owasp"`         // OWASP categories
	Description string    `json:"description"`
	Impact      string    `json:"impact"`
	Affected    []string  `json:"affected"`      // affected hosts/files/functions
	Evidence    []string  `json:"evidence"`      // proof of vulnerability
	Remediation string    `json:"remediation"`
	References  []string  `json:"references"`
	Confidence  string    `json:"confidence"`    // CONFIRMED, HIGH, MEDIUM, LOW
	Domain      string    `json:"domain"`        // web, network, source, firmware, binary
	DiscoveredAt time.Time `json:"discovered_at"`
	DiscoveredBy string   `json:"discovered_by"` // which phase/tool found it
}

type VulnSummary struct {
	Total    int            `json:"total"`
	BySeverity map[string]int `json:"by_severity"`
	ByDomain  map[string]int `json:"by_domain"`
	Confirmed int            `json:"confirmed"` // high confidence findings
}

func (v *VulnerabilityList) Type() string { return "VulnerabilityList" }

func (v *VulnerabilityList) Validate() error {
	if v.Vulnerabilities == nil {
		return fmt.Errorf("vulnerabilities list cannot be nil")
	}
	return nil
}

func (v *VulnerabilityList) GetMetadata() blackboard.ArtifactMetadata { return v.Metadata }

// ExploitResult contains the result of exploit generation/execution
type ExploitResult struct {
	Metadata blackboard.ArtifactMetadata `json:"metadata"`

	VulnerabilityID string    `json:"vulnerability_id"` // references Vulnerability.ID
	Status          string    `json:"status"`            // SUCCESS, FAILED, PARTIAL
	ExploitType     string    `json:"exploit_type"`      // PoC, Full, Metasploit, Manual
	Code            string    `json:"code"`              // exploit code/script
	Instructions    string    `json:"instructions"`      // how to run the exploit
	Output          string    `json:"output"`            // execution output if tested
	Verified        bool      `json:"verified"`          // was the exploit actually tested?
	Limitations     []string  `json:"limitations"`       // known limitations
	Timestamp       time.Time `json:"timestamp"`
}

func (e *ExploitResult) Type() string { return "ExploitResult" }

func (e *ExploitResult) Validate() error {
	if e.VulnerabilityID == "" {
		return fmt.Errorf("vulnerability_id cannot be empty")
	}
	if e.Status == "" {
		return fmt.Errorf("status cannot be empty")
	}
	return nil
}

func (e *ExploitResult) GetMetadata() blackboard.ArtifactMetadata { return e.Metadata }

// FinalReport is the end product of the entire pipeline
type FinalReport struct {
	Metadata blackboard.ArtifactMetadata `json:"metadata"`

	ExecutiveSummary string              `json:"executive_summary"`
	Target           OperatorTarget      `json:"target"`
	Pipeline         PipelineSummary     `json:"pipeline"`
	Vulnerabilities  VulnerabilityList   `json:"vulnerabilities"`
	Exploits         []ExploitResult     `json:"exploits"`
	ReportFormat     string              `json:"report_format"` // markdown, json, html
	ReportContent    string              `json:"report_content"` // rendered report
	GeneratedAt      time.Time           `json:"generated_at"`
}

func (f *FinalReport) Type() string { return "FinalReport" }

func (f *FinalReport) Validate() error {
	if f.ExecutiveSummary == "" {
		return fmt.Errorf("executive_summary cannot be empty")
	}
	if f.ReportContent == "" {
		return fmt.Errorf("report_content cannot be empty")
	}
	return nil
}

func (f *FinalReport) GetMetadata() blackboard.ArtifactMetadata { return f.Metadata }
