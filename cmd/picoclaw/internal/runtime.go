package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ResistanceIsUseless/picoclaw/pkg/agent"
	"github.com/ResistanceIsUseless/picoclaw/pkg/bus"
	"github.com/ResistanceIsUseless/picoclaw/pkg/config"
	"github.com/ResistanceIsUseless/picoclaw/pkg/graph"
	"github.com/ResistanceIsUseless/picoclaw/pkg/integration"
	"github.com/ResistanceIsUseless/picoclaw/pkg/orchestrator"
	"github.com/ResistanceIsUseless/picoclaw/pkg/providers"
	"github.com/ResistanceIsUseless/picoclaw/pkg/registry"
	"github.com/ResistanceIsUseless/picoclaw/pkg/tools"
	"github.com/ResistanceIsUseless/picoclaw/pkg/tools/profiles"
	"github.com/ResistanceIsUseless/picoclaw/pkg/webui"
	"github.com/ResistanceIsUseless/picoclaw/pkg/workflow"
)

type AgentRuntime struct {
	Config           *config.Config
	Provider         providers.LLMProvider
	ModelID          string
	Bus              *bus.MessageBus
	AgentLoop        *agent.AgentLoop
	ProfileReadiness *ProfileReadiness
	WebUIServer      *webui.Server
	WebUIURL         string
}

type PipelinePreflight struct {
	MissingRequired []string
	MissingOptional []string
}

func (r *AgentRuntime) StartEmbeddedWebUI(addr string) (string, error) {
	if r.WebUIURL != "" {
		return r.WebUIURL, nil
	}

	server := webui.NewServer(nil, r.AgentLoop.GetBlackboard(), nil, nil)
	url, err := server.StartBackground(addr)
	if err != nil {
		return "", err
	}

	r.WebUIServer = server
	r.WebUIURL = url
	return url, nil
}

func StartEmbeddedCLAWWebUI(addr string, adapter *integration.CLAWAdapter) (string, *webui.Server, error) {
	if adapter == nil {
		return "", nil, fmt.Errorf("claw adapter is nil")
	}

	orch := adapter.GetOrchestrator()
	var g *graph.Graph
	if orch != nil {
		g = orch.GetGraph()
	}
	server := webui.NewServer(orch, adapter.GetBlackboard(), g, adapter.GetToolRegistry())
	if orch != nil {
		orch.SetEventEmitter(server.GetEventEmitter())
	}
	url, err := server.StartBackground(addr)
	if err != nil {
		return "", nil, err
	}
	return url, server, nil
}

func OpenBrowser(url string) error {
	if url == "" {
		return fmt.Errorf("web ui url is empty")
	}

	cmd := exec.Command("open", url)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to open browser: %w", err)
	}
	return nil
}

type ProfileReadiness struct {
	ReadyProfiles   []string
	MissingProfiles []string
	ProfileTools    map[string][]string
}

type WorkflowProfileAssessment struct {
	RequiredProfiles []string
	MissingProfiles  []string
}

type ProfileGuidance struct {
	Profile      string
	Summary      string
	InstallHints []string
}

type PreflightSummary struct {
	Scope            string
	RequiredProfiles []string
	MissingProfiles  []string
	Guidance         string
}

func BootstrapAgentRuntime(modelOverride string) (*AgentRuntime, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("error loading config: %w", err)
	}

	if modelOverride != "" {
		cfg.Agents.Defaults.ModelName = modelOverride
	}

	provider, modelID, err := providers.CreateProvider(cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating provider: %w", err)
	}

	if modelID != "" {
		cfg.Agents.Defaults.ModelName = modelID
	}

	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider)
	profileReadiness := CollectProfileReadiness()

	return &AgentRuntime{
		Config:           cfg,
		Provider:         provider,
		ModelID:          modelID,
		Bus:              msgBus,
		AgentLoop:        agentLoop,
		ProfileReadiness: profileReadiness,
	}, nil
}

func ResolveCLAWPersistenceDir(cfg *config.Config) string {
	if cfg != nil && cfg.Agents.Defaults.CLAWMode != nil && cfg.Agents.Defaults.CLAWMode.PersistenceDir != "" {
		return expandUserPath(cfg.Agents.Defaults.CLAWMode.PersistenceDir)
	}
	return expandUserPath("~/.picoclaw/blackboard")
}

func BuildCLAWAdapter(cfg *config.Config, provider providers.LLMProvider, execRegistry *tools.ToolRegistry, pipeline string) (*integration.CLAWAdapter, error) {
	adapterCfg := &integration.CLAWConfig{
		Enabled:        true,
		Pipeline:       pipeline,
		PersistenceDir: ResolveCLAWPersistenceDir(cfg),
		ExecRegistry:   execRegistry,
	}

	return integration.NewCLAWAdapter(adapterCfg, provider)
}

func PreflightCLAWPipeline(pipeline string) (*PipelinePreflight, error) {
	p, err := orchestrator.GetPredefinedPipeline(pipeline)
	if err != nil {
		return nil, err
	}

	missingRequired := make(map[string]bool)
	missingOptional := make(map[string]bool)

	for _, phase := range p.Phases {
		required := make(map[string]bool)
		for _, tool := range phase.RequiredTools {
			required[tool] = true
			if _, err := registry.GetToolPath(tool); err != nil {
				missingRequired[tool] = true
			}
		}

		for _, profileName := range phase.RequiredProfiles {
			profileHasTool := false
			for _, tool := range orchestratorToolsForProfile(phase, profileName) {
				required[tool] = true
				if _, err := registry.GetToolPath(tool); err == nil {
					profileHasTool = true
				}
			}
			if !profileHasTool {
				missingRequired[profileName+" profile"] = true
			}
		}

		for _, tool := range phase.ResolvedTools() {
			if required[tool] {
				continue
			}
			if _, err := registry.GetToolPath(tool); err != nil {
				missingOptional[tool] = true
			}
		}
	}

	result := &PipelinePreflight{
		MissingRequired: mapKeysSorted(missingRequired),
		MissingOptional: mapKeysSorted(missingOptional),
	}

	return result, nil
}

func orchestratorToolsForProfile(phase *orchestrator.PhaseDefinition, profileName string) []string {
	tools := make([]string, 0)
	for _, tool := range phase.ResolvedTools() {
		profile, ok := profiles.ResolveToolProfile(tool)
		if ok && profile.Name == profileName {
			tools = append(tools, tool)
		}
	}
	return tools
}

func (p *PipelinePreflight) HasBlockingIssues() bool {
	return p != nil && len(p.MissingRequired) > 0
}

func (p *PipelinePreflight) BlockingMessage(pipeline string) string {
	if p == nil || len(p.MissingRequired) == 0 {
		return ""
	}

	return fmt.Sprintf(
		"pipeline %q is missing required tools: %s",
		pipeline,
		strings.Join(p.MissingRequired, ", "),
	)
}

func CollectProfileReadiness() *ProfileReadiness {
	knownProfiles := []string{
		profiles.ProfileSubdomainEnum,
		profiles.ProfilePortScan,
		profiles.ProfileCrawl,
		profiles.ProfileWebProbe,
		profiles.ProfileVulnScan,
		profiles.ProfileFuzz,
		profiles.ProfileCodeAnalysis,
	}

	result := &ProfileReadiness{
		ReadyProfiles:   make([]string, 0),
		MissingProfiles: make([]string, 0),
		ProfileTools:    make(map[string][]string),
	}

	for _, profileName := range knownProfiles {
		available := make([]string, 0)
		for _, tool := range profiles.ToolsForProfile(profileName) {
			if _, err := registry.GetToolPath(tool); err == nil {
				available = append(available, tool)
			}
		}
		if len(available) > 0 {
			result.ReadyProfiles = append(result.ReadyProfiles, profileName)
			result.ProfileTools[profileName] = available
		} else {
			result.MissingProfiles = append(result.MissingProfiles, profileName)
		}
	}

	sort.Strings(result.ReadyProfiles)
	sort.Strings(result.MissingProfiles)
	return result
}

func GetProfileGuidance(profileName string) ProfileGuidance {
	switch profileName {
	case profiles.ProfileSubdomainEnum:
		return ProfileGuidance{
			Profile: profileName,
			Summary: "Install at least one subdomain enumeration tool.",
			InstallHints: []string{
				"Try one of: subfinder, amass, assetfinder, dnsx",
				"Example: `go install github.com/projectdiscovery/subfinder/v2/cmd/subfinder@latest`",
			},
		}
	case profiles.ProfilePortScan:
		return ProfileGuidance{
			Profile: profileName,
			Summary: "Install at least one port scanning tool.",
			InstallHints: []string{
				"Try one of: nmap, naabu, masscan, rustscan",
				"Example: `brew install nmap` or `go install github.com/projectdiscovery/naabu/v2/cmd/naabu@latest`",
			},
		}
	case profiles.ProfileCrawl:
		return ProfileGuidance{
			Profile: profileName,
			Summary: "Install at least one crawler for endpoint discovery.",
			InstallHints: []string{
				"Try one of: katana, gospider, hakrawler",
				"Example: `go install github.com/projectdiscovery/katana/cmd/katana@latest`",
			},
		}
	case profiles.ProfileWebProbe:
		return ProfileGuidance{
			Profile: profileName,
			Summary: "Install at least one web probing/fingerprinting tool.",
			InstallHints: []string{
				"Try one of: httpx, whatweb, webanalyze",
				"Example: `go install github.com/projectdiscovery/httpx/cmd/httpx@latest`",
			},
		}
	case profiles.ProfileVulnScan:
		return ProfileGuidance{
			Profile: profileName,
			Summary: "Install at least one vulnerability scanning tool.",
			InstallHints: []string{
				"Try one of: nuclei, nikto, wpscan",
				"Example: `go install github.com/projectdiscovery/nuclei/v3/cmd/nuclei@latest`",
			},
		}
	case profiles.ProfileFuzz:
		return ProfileGuidance{
			Profile: profileName,
			Summary: "Install at least one content or parameter fuzzing tool.",
			InstallHints: []string{
				"Try one of: ffuf, gobuster, feroxbuster, wfuzz",
				"Example: `go install github.com/ffuf/ffuf/v2@latest` or `brew install gobuster`",
			},
		}
	case profiles.ProfileCodeAnalysis:
		return ProfileGuidance{
			Profile: profileName,
			Summary: "Install at least one static/code analysis tool.",
			InstallHints: []string{
				"Try one of: semgrep, gosec, bandit, eslint",
				"Example: `brew install semgrep` or `go install github.com/securego/gosec/v2/cmd/gosec@latest`",
			},
		}
	default:
		return ProfileGuidance{
			Profile: profileName,
			Summary: "Install at least one tool that satisfies this capability.",
			InstallHints: []string{
				fmt.Sprintf("Known candidates: %s", strings.Join(profiles.ToolsForProfile(profileName), ", ")),
			},
		}
	}
}

func FormatProfileGuidance(profileNames []string) string {
	if len(profileNames) == 0 {
		return ""
	}

	var sb strings.Builder
	for _, profileName := range profileNames {
		guidance := GetProfileGuidance(profileName)
		fmt.Fprintf(&sb, "- %s: %s\n", guidance.Profile, guidance.Summary)
		for _, hint := range guidance.InstallHints {
			fmt.Fprintf(&sb, "  %s\n", hint)
		}
	}
	return strings.TrimSpace(sb.String())
}

func BuildPreflightSummary(scope string, requiredProfiles []string, readiness *ProfileReadiness) *PreflightSummary {
	if readiness == nil {
		return nil
	}

	missing := make([]string, 0)
	readySet := make(map[string]bool)
	for _, profileName := range readiness.ReadyProfiles {
		readySet[profileName] = true
	}

	if len(requiredProfiles) == 0 {
		requiredProfiles = append([]string{}, readiness.MissingProfiles...)
	}

	for _, profileName := range requiredProfiles {
		if !readySet[profileName] {
			missing = append(missing, profileName)
		}
	}

	sort.Strings(requiredProfiles)
	sort.Strings(missing)

	return &PreflightSummary{
		Scope:            scope,
		RequiredProfiles: requiredProfiles,
		MissingProfiles:  missing,
		Guidance:         FormatProfileGuidance(missing),
	}
}

func (p *PreflightSummary) HasGaps() bool {
	return p != nil && len(p.MissingProfiles) > 0
}

func (p *PreflightSummary) Message(prefix string) string {
	if p == nil || len(p.MissingProfiles) == 0 {
		return ""
	}
	if prefix == "" {
		prefix = "Capability gaps"
	}
	if p.Guidance == "" {
		return fmt.Sprintf("%s: %s", prefix, strings.Join(p.MissingProfiles, ", "))
	}
	return fmt.Sprintf("%s: %s\n%s", prefix, strings.Join(p.MissingProfiles, ", "), p.Guidance)
}

func AssessWorkflowProfileReadiness(workflowName, workspace string, readiness *ProfileReadiness) (*WorkflowProfileAssessment, error) {
	wf, err := workflow.LoadWorkflow(workspace, workflowName)
	if err != nil {
		return nil, err
	}

	required := inferWorkflowProfiles(wf)
	readySet := make(map[string]bool)
	if readiness != nil {
		for _, profileName := range readiness.ReadyProfiles {
			readySet[profileName] = true
		}
	}

	missing := make([]string, 0)
	for _, profileName := range required {
		if !readySet[profileName] {
			missing = append(missing, profileName)
		}
	}

	return &WorkflowProfileAssessment{
		RequiredProfiles: required,
		MissingProfiles:  missing,
	}, nil
}

func inferWorkflowProfiles(wf *workflow.Workflow) []string {
	if wf == nil {
		return nil
	}

	profileMatchers := map[string][]string{
		profiles.ProfileSubdomainEnum: {"subdomain", "dns", "amass", "subfinder", "assetfinder", "dnsx", "crtsh", "recon"},
		profiles.ProfilePortScan:      {"port scan", "service detection", "nmap", "masscan", "naabu", "rustscan", "live hosts"},
		profiles.ProfileCrawl:         {"crawl", "crawler", "katana", "gospider", "hakrawler", "endpoint discovery"},
		profiles.ProfileWebProbe:      {"httpx", "headers", "technology", "web service", "curl -si", "status codes", "fingerprint"},
		profiles.ProfileVulnScan:      {"nuclei", "nikto", "wpscan", "vulnerability scan", "cve", "sqlmap", "dalfox", "ssrf", "ssti", "command injection"},
		profiles.ProfileFuzz:          {"fuzz", "ffuf", "gobuster", "feroxbuster", "wfuzz", "wordlist", "bruteforce"},
		profiles.ProfileCodeAnalysis:  {"semgrep", "static analysis", "code review", "grep patterns", "dependency check", "gosec", "bandit", "eslint"},
	}

	textParts := []string{wf.Name, wf.Description}
	for _, phase := range wf.Phases {
		textParts = append(textParts, phase.Name, phase.Completion.Description)
		for _, step := range phase.Steps {
			textParts = append(textParts, step.ID, step.Name, step.Description)
		}
		for _, branch := range phase.Branches {
			textParts = append(textParts, branch.Condition, branch.Description)
		}
	}
	corpus := strings.ToLower(strings.Join(textParts, " "))

	required := make([]string, 0)
	for profileName, needles := range profileMatchers {
		for _, needle := range needles {
			if strings.Contains(corpus, strings.ToLower(needle)) {
				required = append(required, profileName)
				break
			}
		}
	}

	sort.Strings(required)
	return required
}

func mapKeysSorted(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func expandUserPath(path string) string {
	if path == "" {
		return path
	}

	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return home
			}
			return filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}

	return os.ExpandEnv(path)
}
