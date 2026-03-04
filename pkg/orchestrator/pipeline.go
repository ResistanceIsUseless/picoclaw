package orchestrator

import (
	"fmt"
)

// Pipeline defines the complete assessment workflow
type Pipeline struct {
	Name        string
	Description string
	Phases      []*PhaseDefinition
	Domain      string // web, network, source, firmware, binary
}

// PhaseDefinition defines a single phase in the pipeline
type PhaseDefinition struct {
	Name              string
	Objective         string
	Tools             []string          // Available tools for this phase
	RequiredTools     []string          // Tools that MUST be executed
	RequiredArtifacts []string          // Artifact types that MUST be produced
	Dependencies      map[string][]string // Tool dependencies (tool -> required tools)
	DependsOn         []string          // Phase dependencies
	MinIterations     int
	MaxIterations     int
	TokenBudget       int
}

// NewPipeline creates a new pipeline
func NewPipeline(name, description, domain string) *Pipeline {
	return &Pipeline{
		Name:        name,
		Description: description,
		Domain:      domain,
		Phases:      make([]*PhaseDefinition, 0),
	}
}

// AddPhase adds a phase to the pipeline
func (p *Pipeline) AddPhase(phase *PhaseDefinition) *Pipeline {
	p.Phases = append(p.Phases, phase)
	return p
}

// Validate checks if the pipeline is valid
func (p *Pipeline) Validate() error {
	if p.Name == "" {
		return fmt.Errorf("pipeline name cannot be empty")
	}

	if len(p.Phases) == 0 {
		return fmt.Errorf("pipeline must have at least one phase")
	}

	// Check for duplicate phase names
	seen := make(map[string]bool)
	for _, phase := range p.Phases {
		if seen[phase.Name] {
			return fmt.Errorf("duplicate phase name: %q", phase.Name)
		}
		seen[phase.Name] = true
	}

	// Validate each phase
	for _, phase := range p.Phases {
		if err := phase.Validate(); err != nil {
			return fmt.Errorf("phase %q invalid: %w", phase.Name, err)
		}
	}

	// Validate phase dependencies
	phaseNames := make(map[string]bool)
	for _, phase := range p.Phases {
		phaseNames[phase.Name] = true
	}

	for _, phase := range p.Phases {
		for _, dep := range phase.DependsOn {
			if !phaseNames[dep] {
				return fmt.Errorf("phase %q depends on unknown phase %q", phase.Name, dep)
			}
		}
	}

	// Check for circular dependencies
	if err := p.checkCircularDependencies(); err != nil {
		return err
	}

	return nil
}

// Validate checks if a phase definition is valid
func (pd *PhaseDefinition) Validate() error {
	if pd.Name == "" {
		return fmt.Errorf("phase name cannot be empty")
	}

	if pd.Objective == "" {
		return fmt.Errorf("phase objective cannot be empty")
	}

	if len(pd.Tools) == 0 {
		return fmt.Errorf("phase must have at least one tool")
	}

	if pd.MinIterations < 1 {
		return fmt.Errorf("min_iterations must be >= 1")
	}

	if pd.MaxIterations < pd.MinIterations {
		return fmt.Errorf("max_iterations must be >= min_iterations")
	}

	// Validate that required tools are in available tools
	toolSet := make(map[string]bool)
	for _, tool := range pd.Tools {
		toolSet[tool] = true
	}

	for _, required := range pd.RequiredTools {
		if !toolSet[required] {
			return fmt.Errorf("required tool %q not in available tools", required)
		}
	}

	return nil
}

// checkCircularDependencies detects circular phase dependencies
func (p *Pipeline) checkCircularDependencies() error {
	// Build adjacency list
	graph := make(map[string][]string)
	for _, phase := range p.Phases {
		graph[phase.Name] = phase.DependsOn
	}

	// Check for cycles using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(string) bool
	hasCycle = func(node string) bool {
		visited[node] = true
		recStack[node] = true

		for _, neighbor := range graph[node] {
			if !visited[neighbor] {
				if hasCycle(neighbor) {
					return true
				}
			} else if recStack[neighbor] {
				return true
			}
		}

		recStack[node] = false
		return false
	}

	for _, phase := range p.Phases {
		if !visited[phase.Name] {
			if hasCycle(phase.Name) {
				return fmt.Errorf("circular dependency detected involving phase %q", phase.Name)
			}
		}
	}

	return nil
}

// GetPhase retrieves a phase definition by name
func (p *Pipeline) GetPhase(name string) (*PhaseDefinition, error) {
	for _, phase := range p.Phases {
		if phase.Name == name {
			return phase, nil
		}
	}
	return nil, fmt.Errorf("phase %q not found", name)
}

// TopologicalSort returns phases in dependency order
func (p *Pipeline) TopologicalSort() ([]*PhaseDefinition, error) {
	// Build in-degree map
	inDegree := make(map[string]int)
	for _, phase := range p.Phases {
		if _, exists := inDegree[phase.Name]; !exists {
			inDegree[phase.Name] = 0
		}
		for _, dep := range phase.DependsOn {
			inDegree[dep] = 0 // Ensure dependency exists in map
		}
	}

	// Count dependencies
	for _, phase := range p.Phases {
		for range phase.DependsOn {
			inDegree[phase.Name]++
		}
	}

	// Queue phases with no dependencies
	queue := make([]*PhaseDefinition, 0)
	for _, phase := range p.Phases {
		if inDegree[phase.Name] == 0 {
			queue = append(queue, phase)
		}
	}

	// Process queue
	result := make([]*PhaseDefinition, 0, len(p.Phases))
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// Reduce in-degree for dependent phases
		for _, phase := range p.Phases {
			for _, dep := range phase.DependsOn {
				if dep == current.Name {
					inDegree[phase.Name]--
					if inDegree[phase.Name] == 0 {
						queue = append(queue, phase)
					}
				}
			}
		}
	}

	if len(result) != len(p.Phases) {
		return nil, fmt.Errorf("circular dependency detected in pipeline")
	}

	return result, nil
}

// PredefinedPipelines contains standard assessment pipelines
var PredefinedPipelines = map[string]*Pipeline{
	"web_full": NewPipeline("web_full", "Complete web application assessment", "web").
		AddPhase(&PhaseDefinition{
			Name:              "recon",
			Objective:         "Discover subdomains and infrastructure",
			Tools:             []string{"subfinder", "amass", "crtsh"},
			RequiredTools:     []string{"subfinder"},
			RequiredArtifacts: []string{"SubdomainList"},
			MinIterations:     1,
			MaxIterations:     5,
			TokenBudget:       10000,
		}).
		AddPhase(&PhaseDefinition{
			Name:              "port_scan",
			Objective:         "Identify open ports and services",
			Tools:             []string{"nmap", "masscan"},
			RequiredTools:     []string{"nmap"},
			RequiredArtifacts: []string{"PortScanResult"},
			DependsOn:         []string{"recon"},
			MinIterations:     1,
			MaxIterations:     3,
			TokenBudget:       8000,
		}).
		AddPhase(&PhaseDefinition{
			Name:              "service_discovery",
			Objective:         "Fingerprint services and technologies",
			Tools:             []string{"httpx", "whatweb", "wappalyzer"},
			RequiredTools:     []string{"httpx"},
			RequiredArtifacts: []string{"ServiceFingerprint"},
			DependsOn:         []string{"port_scan"},
			MinIterations:     1,
			MaxIterations:     5,
			TokenBudget:       12000,
		}).
		AddPhase(&PhaseDefinition{
			Name:              "vulnerability_scan",
			Objective:         "Identify vulnerabilities in discovered services",
			Tools:             []string{"nuclei", "nikto", "wpscan"},
			RequiredTools:     []string{"nuclei"},
			RequiredArtifacts: []string{"VulnerabilityList"},
			DependsOn:         []string{"service_discovery"},
			MinIterations:     1,
			MaxIterations:     10,
			TokenBudget:       15000,
		}),

	"web_quick": NewPipeline("web_quick", "Quick web application assessment", "web").
		AddPhase(&PhaseDefinition{
			Name:              "recon",
			Objective:         "Discover subdomains",
			Tools:             []string{"subfinder"},
			RequiredTools:     []string{"subfinder"},
			RequiredArtifacts: []string{"SubdomainList"},
			MinIterations:     1,
			MaxIterations:     3,
			TokenBudget:       5000,
		}).
		AddPhase(&PhaseDefinition{
			Name:              "quick_scan",
			Objective:         "Quick vulnerability scan",
			Tools:             []string{"httpx", "nuclei"},
			RequiredTools:     []string{"httpx", "nuclei"},
			RequiredArtifacts: []string{"WebFindings"},
			DependsOn:         []string{"recon"},
			MinIterations:     1,
			MaxIterations:     5,
			TokenBudget:       8000,
		}),
}

// GetPredefinedPipeline retrieves a predefined pipeline
func GetPredefinedPipeline(name string) (*Pipeline, error) {
	pipeline, exists := PredefinedPipelines[name]
	if !exists {
		return nil, fmt.Errorf("predefined pipeline %q not found", name)
	}
	return pipeline, nil
}
