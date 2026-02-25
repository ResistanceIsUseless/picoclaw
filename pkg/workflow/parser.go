package workflow

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parser parses workflow definitions from markdown files
type Parser struct{}

// NewParser creates a new workflow parser
func NewParser() *Parser {
	return &Parser{}
}

// ParseFile parses a workflow definition from a markdown file
func (p *Parser) ParseFile(path string) (*Workflow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}

	return p.Parse(string(data))
}

// Parse parses a workflow definition from markdown content
func (p *Parser) Parse(content string) (*Workflow, error) {
	// Split into frontmatter and body
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid workflow format: missing YAML frontmatter")
	}

	// Parse frontmatter
	var metadata struct {
		Name        string   `yaml:"name"`
		Description string   `yaml:"description"`
		Phases      []string `yaml:"phases"`
	}

	if err := yaml.Unmarshal([]byte(parts[1]), &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse frontmatter: %w", err)
	}

	// Parse body
	workflow := &Workflow{
		Name:        metadata.Name,
		Description: metadata.Description,
		Phases:      make([]Phase, 0),
	}

	phases, err := p.parseBody(parts[2])
	if err != nil {
		return nil, fmt.Errorf("failed to parse workflow body: %w", err)
	}

	workflow.Phases = phases
	return workflow, nil
}

// parseBody parses the markdown body into phases
func (p *Parser) parseBody(body string) ([]Phase, error) {
	phases := make([]Phase, 0)
	var currentPhase *Phase
	var currentSection string

	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// Phase header: ## Phase: <name>
		if strings.HasPrefix(trimmed, "## Phase:") {
			if currentPhase != nil {
				phases = append(phases, *currentPhase)
			}
			phaseName := strings.TrimSpace(strings.TrimPrefix(trimmed, "## Phase:"))
			currentPhase = &Phase{
				Name:     phaseName,
				Steps:    make([]Step, 0),
				Branches: make([]Branch, 0),
			}
			currentSection = ""
			continue
		}

		if currentPhase == nil {
			continue
		}

		// Section headers: ### <section>
		if strings.HasPrefix(trimmed, "###") {
			sectionName := strings.TrimSpace(strings.TrimPrefix(trimmed, "###"))
			currentSection = strings.ToLower(sectionName)
			continue
		}

		// Parse content based on current section
		switch currentSection {
		case "steps":
			if strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "*") {
				step := p.parseStep(trimmed)
				if step != nil {
					currentPhase.Steps = append(currentPhase.Steps, *step)
				}
			}

		case "completion criteria", "completion":
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
				// Accumulate completion description
				if currentPhase.Completion.Description != "" {
					currentPhase.Completion.Description += " "
				}
				currentPhase.Completion.Description += trimmed

				// Determine completion type
				if strings.Contains(strings.ToLower(trimmed), "all") && strings.Contains(strings.ToLower(trimmed), "required") {
					currentPhase.Completion.Type = CompletionAllRequired
				} else if strings.Contains(strings.ToLower(trimmed), "branch") {
					currentPhase.Completion.Type = CompletionAnyBranch
				} else {
					currentPhase.Completion.Type = CompletionCustom
				}
			}

		case "branches":
			if strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "*") {
				branch := p.parseBranch(trimmed)
				if branch != nil {
					currentPhase.Branches = append(currentPhase.Branches, *branch)
				}
			}
		}
	}

	// Add last phase
	if currentPhase != nil {
		phases = append(phases, *currentPhase)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning workflow: %w", err)
	}

	return phases, nil
}

// parseStep parses a step line
// Format: "- step_id: Description (required)"
// Or: "- Description"
func (p *Parser) parseStep(line string) *Step {
	// Remove list marker
	line = strings.TrimPrefix(line, "-")
	line = strings.TrimPrefix(line, "*")
	line = strings.TrimSpace(line)

	if line == "" {
		return nil
	}

	step := &Step{
		Required: strings.Contains(strings.ToLower(line), "(required)"),
	}

	// Remove "(required)" marker
	line = strings.ReplaceAll(line, "(required)", "")
	line = strings.ReplaceAll(line, "(Required)", "")
	line = strings.TrimSpace(line)

	// Check for ID:Description format
	if strings.Contains(line, ":") {
		parts := strings.SplitN(line, ":", 2)
		step.ID = strings.TrimSpace(parts[0])
		step.Name = strings.TrimSpace(parts[1])
		step.Description = step.Name
	} else {
		// Generate ID from name
		step.Name = line
		step.Description = line
		step.ID = strings.ToLower(strings.ReplaceAll(step.Name, " ", "_"))
	}

	return step
}

// parseBranch parses a branch line
// Format: "- condition → description"
// Or: "- condition: description"
func (p *Parser) parseBranch(line string) *Branch {
	// Remove list marker
	line = strings.TrimPrefix(line, "-")
	line = strings.TrimPrefix(line, "*")
	line = strings.TrimSpace(line)

	if line == "" {
		return nil
	}

	var condition, description string

	// Check for arrow format
	if strings.Contains(line, "→") {
		parts := strings.SplitN(line, "→", 2)
		condition = strings.TrimSpace(parts[0])
		description = strings.TrimSpace(parts[1])
	} else if strings.Contains(line, ":") {
		parts := strings.SplitN(line, ":", 2)
		condition = strings.TrimSpace(parts[0])
		description = strings.TrimSpace(parts[1])
	} else {
		condition = line
		description = line
	}

	return &Branch{
		Condition:   condition,
		Description: description,
	}
}

// LoadWorkflow loads a workflow from the workspace
func LoadWorkflow(workspace, name string) (*Workflow, error) {
	parser := NewParser()

	// Try various locations
	locations := []string{
		filepath.Join(workspace, "workflows", name+".md"),
		filepath.Join(workspace, "workflows", name),
		filepath.Join(workspace, name+".md"),
		filepath.Join(workspace, name),
	}

	for _, path := range locations {
		if _, err := os.Stat(path); err == nil {
			return parser.ParseFile(path)
		}
	}

	return nil, fmt.Errorf("workflow not found: %s", name)
}
