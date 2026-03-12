package profiles

import "path/filepath"

const (
	ProfileSubdomainEnum = "subdomain-enum"
	ProfilePortScan      = "port-scan"
	ProfileCrawl         = "crawl"
	ProfileWebProbe      = "web-probe"
	ProfileVulnScan      = "vuln-scan"
	ProfileFuzz          = "fuzz"
	ProfileCodeAnalysis  = "code-analysis"
)

type ToolProfile struct {
	Name         string
	ArtifactTool string
}

var profileTools = map[string][]string{
	ProfileSubdomainEnum: {"subfinder", "amass", "assetfinder", "chaos", "shuffledns", "puredns", "dnsx"},
	ProfilePortScan:      {"nmap", "masscan", "naabu", "rustscan"},
	ProfileCrawl:         {"katana", "gospider", "hakrawler"},
	ProfileWebProbe:      {"httpx", "whatweb", "wappalyzer", "webanalyze"},
	ProfileVulnScan:      {"nuclei", "nikto", "wpscan"},
	ProfileFuzz:          {"ffuf", "wfuzz", "gobuster", "feroxbuster", "dirsearch", "fuzzparam", "kxss"},
	ProfileCodeAnalysis:  {"semgrep", "bandit", "gosec", "eslint", "sonarqube"},
}

var toolProfiles = map[string]ToolProfile{
	"subfinder":   {Name: ProfileSubdomainEnum, ArtifactTool: "subfinder"},
	"amass":       {Name: ProfileSubdomainEnum, ArtifactTool: "amass"},
	"nmap":        {Name: ProfilePortScan, ArtifactTool: "nmap"},
	"httpx":       {Name: ProfileWebProbe, ArtifactTool: "httpx"},
	"nuclei":      {Name: ProfileVulnScan, ArtifactTool: "nuclei"},
	"katana":      {Name: ProfileCrawl},
	"gospider":    {Name: ProfileCrawl},
	"hakrawler":   {Name: ProfileCrawl},
	"masscan":     {Name: ProfilePortScan},
	"naabu":       {Name: ProfilePortScan},
	"rustscan":    {Name: ProfilePortScan},
	"whatweb":     {Name: ProfileWebProbe},
	"wappalyzer":  {Name: ProfileWebProbe},
	"webanalyze":  {Name: ProfileWebProbe},
	"nikto":       {Name: ProfileVulnScan},
	"wpscan":      {Name: ProfileVulnScan},
	"ffuf":        {Name: ProfileFuzz},
	"wfuzz":       {Name: ProfileFuzz},
	"gobuster":    {Name: ProfileFuzz},
	"feroxbuster": {Name: ProfileFuzz},
	"dirsearch":   {Name: ProfileFuzz},
	"fuzzparam":   {Name: ProfileFuzz},
	"kxss":        {Name: ProfileFuzz},
	"semgrep":     {Name: ProfileCodeAnalysis},
	"bandit":      {Name: ProfileCodeAnalysis},
	"gosec":       {Name: ProfileCodeAnalysis},
	"eslint":      {Name: ProfileCodeAnalysis},
	"sonarqube":   {Name: ProfileCodeAnalysis},
}

func ResolveToolProfile(toolName string) (ToolProfile, bool) {
	profile, ok := toolProfiles[filepath.Base(toolName)]
	return profile, ok
}

func ToolsForProfile(profileName string) []string {
	return append([]string(nil), profileTools[profileName]...)
}
