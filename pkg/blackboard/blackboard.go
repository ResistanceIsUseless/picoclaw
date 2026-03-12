package blackboard

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ResistanceIsUseless/picoclaw/pkg/logger"
)

// Artifact represents a typed data structure produced by a phase
type Artifact interface {
	// Type returns the artifact type identifier (e.g., "SubdomainList", "PortScanResult")
	Type() string

	// Validate ensures the artifact conforms to its schema
	Validate() error

	// GetMetadata returns creation time, phase, and other tracking info
	GetMetadata() ArtifactMetadata
}

// ArtifactMetadata tracks provenance and lifecycle info
type ArtifactMetadata struct {
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	Phase     string    `json:"phase"`
	Version   string    `json:"version"`
	Domain    string    `json:"domain"` // web, network, source, firmware, binary
}

// ArtifactEnvelope wraps an artifact with metadata for storage
type ArtifactEnvelope struct {
	Metadata ArtifactMetadata `json:"metadata"`
	Data     json.RawMessage  `json:"data"`
}

// Subscriber is called when an artifact of subscribed type is published
type Subscriber func(ctx context.Context, artifact Artifact)

// Blackboard is a concurrent-safe typed artifact store with pub/sub
// It serves as the system of record for all phase outputs
type Blackboard struct {
	mu          sync.RWMutex
	artifacts   map[string][]ArtifactEnvelope // type -> list of artifacts
	subscribers map[string][]Subscriber       // type -> list of subscribers
	persister   Persister                     // disk persistence
}

// New creates a new Blackboard with optional persistence
func New(persister Persister) *Blackboard {
	return &Blackboard{
		artifacts:   make(map[string][]ArtifactEnvelope),
		subscribers: make(map[string][]Subscriber),
		persister:   persister,
	}
}

// Publish adds an artifact to the blackboard and notifies subscribers
func (b *Blackboard) Publish(ctx context.Context, artifact Artifact) error {
	// Validate artifact before accepting
	if err := artifact.Validate(); err != nil {
		return fmt.Errorf("artifact validation failed: %w", err)
	}

	metadata := artifact.GetMetadata()
	artifactType := artifact.Type()

	// Marshal artifact data
	data, err := json.Marshal(artifact)
	if err != nil {
		return fmt.Errorf("failed to marshal artifact: %w", err)
	}

	envelope := ArtifactEnvelope{
		Metadata: metadata,
		Data:     data,
	}

	// Store artifact
	b.mu.Lock()
	b.artifacts[artifactType] = append(b.artifacts[artifactType], envelope)

	// Get copy of subscribers before unlocking
	var subscribers []Subscriber
	if subs, exists := b.subscribers[artifactType]; exists {
		subscribers = make([]Subscriber, len(subs))
		copy(subscribers, subs)
	}
	b.mu.Unlock()

	logger.InfoCF("blackboard", "Artifact published",
		map[string]any{
			"type":   artifactType,
			"phase":  metadata.Phase,
			"domain": metadata.Domain,
		})

	// Persist if persister configured
	if b.persister != nil {
		if err := b.persister.Persist(envelope); err != nil {
			logger.ErrorCF("blackboard", "Failed to persist artifact",
				map[string]any{
					"type":  artifactType,
					"error": err.Error(),
				})
			// Don't fail the publish operation on persistence error
		}
	}

	// Notify subscribers asynchronously
	for _, sub := range subscribers {
		go func(subscriber Subscriber) {
			defer func() {
				if r := recover(); r != nil {
					logger.ErrorCF("blackboard", "Subscriber panic",
						map[string]any{
							"type":  artifactType,
							"panic": r,
						})
				}
			}()
			subscriber(ctx, artifact)
		}(sub)
	}

	return nil
}

// Get retrieves all artifacts of a given type
func (b *Blackboard) Get(artifactType string) ([]ArtifactEnvelope, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	artifacts, exists := b.artifacts[artifactType]
	if !exists {
		return nil, fmt.Errorf("no artifacts of type %q found", artifactType)
	}

	// Return a copy to prevent external modification
	result := make([]ArtifactEnvelope, len(artifacts))
	copy(result, artifacts)

	return result, nil
}

// GetByType retrieves all artifacts of a given type.
func (b *Blackboard) GetByType(artifactType string) ([]ArtifactEnvelope, error) {
	return b.Get(artifactType)
}

// GetAll retrieves all artifacts across all types.
func (b *Blackboard) GetAll() ([]ArtifactEnvelope, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make([]ArtifactEnvelope, 0)
	for _, artifacts := range b.artifacts {
		result = append(result, artifacts...)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no artifacts found")
	}
	return result, nil
}

// GetLatest retrieves the most recent artifact of a given type
func (b *Blackboard) GetLatest(artifactType string) (*ArtifactEnvelope, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	artifacts, exists := b.artifacts[artifactType]
	if !exists || len(artifacts) == 0 {
		return nil, fmt.Errorf("no artifacts of type %q found", artifactType)
	}

	// Return latest (last in list)
	latest := artifacts[len(artifacts)-1]
	return &latest, nil
}

// GetByPhase retrieves all artifacts produced by a specific phase
func (b *Blackboard) GetByPhase(phase string) ([]ArtifactEnvelope, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var result []ArtifactEnvelope
	for _, artifacts := range b.artifacts {
		for _, artifact := range artifacts {
			if artifact.Metadata.Phase == phase {
				result = append(result, artifact)
			}
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no artifacts from phase %q found", phase)
	}

	return result, nil
}

// GetByDomain retrieves all artifacts from a specific domain
func (b *Blackboard) GetByDomain(domain string) ([]ArtifactEnvelope, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var result []ArtifactEnvelope
	for _, artifacts := range b.artifacts {
		for _, artifact := range artifacts {
			if artifact.Metadata.Domain == domain {
				result = append(result, artifact)
			}
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no artifacts from domain %q found", domain)
	}

	return result, nil
}

// Subscribe registers a callback for artifacts of a specific type
// The subscriber will be called asynchronously when artifacts are published
func (b *Blackboard) Subscribe(artifactType string, subscriber Subscriber) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers[artifactType] = append(b.subscribers[artifactType], subscriber)

	logger.DebugCF("blackboard", "Subscriber registered",
		map[string]any{
			"type": artifactType,
		})
}

// Unsubscribe removes all subscribers for a given type
func (b *Blackboard) Unsubscribe(artifactType string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.subscribers, artifactType)

	logger.DebugCF("blackboard", "Subscribers removed",
		map[string]any{
			"type": artifactType,
		})
}

// HasType checks if any artifacts of the given type exist
func (b *Blackboard) HasType(artifactType string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()

	artifacts, exists := b.artifacts[artifactType]
	return exists && len(artifacts) > 0
}

// Count returns the number of artifacts of a given type
func (b *Blackboard) Count(artifactType string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if artifacts, exists := b.artifacts[artifactType]; exists {
		return len(artifacts)
	}
	return 0
}

// Clear removes all artifacts (used for testing, not normal operation)
func (b *Blackboard) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.artifacts = make(map[string][]ArtifactEnvelope)

	logger.WarnCF("blackboard", "Blackboard cleared", nil)
}

// Snapshot returns a complete snapshot of all artifacts for persistence
func (b *Blackboard) Snapshot() map[string][]ArtifactEnvelope {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Deep copy
	snapshot := make(map[string][]ArtifactEnvelope)
	for artifactType, artifacts := range b.artifacts {
		snapshot[artifactType] = make([]ArtifactEnvelope, len(artifacts))
		copy(snapshot[artifactType], artifacts)
	}

	return snapshot
}

// Restore loads artifacts from a snapshot (used for resume-on-failure)
func (b *Blackboard) Restore(snapshot map[string][]ArtifactEnvelope) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.artifacts = snapshot

	logger.InfoCF("blackboard", "Blackboard restored from snapshot",
		map[string]any{
			"types": len(snapshot),
		})
}

// Summary returns a human-readable summary of blackboard contents
func (b *Blackboard) Summary() string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.artifacts) == 0 {
		return "Blackboard is empty"
	}

	summary := fmt.Sprintf("Blackboard contains %d artifact types:\n", len(b.artifacts))
	for artifactType, artifacts := range b.artifacts {
		if len(artifacts) > 0 {
			latest := artifacts[len(artifacts)-1]
			summary += fmt.Sprintf("  - %s: %d artifacts (latest from phase %q at %s)\n",
				artifactType,
				len(artifacts),
				latest.Metadata.Phase,
				latest.Metadata.CreatedAt.Format(time.RFC3339))
		}
	}

	return summary
}

// GetSummary returns a summary suitable for the Commander agent
// Alias for Summary() for Commander orchestrator compatibility
func (b *Blackboard) GetSummary() string {
	return b.Summary()
}

// GetDetailedSummary returns a more detailed summary with artifact contents
func (b *Blackboard) GetDetailedSummary() string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if len(b.artifacts) == 0 {
		return "No discoveries yet."
	}

	var summary strings.Builder
	for artifactType, artifacts := range b.artifacts {
		summary.WriteString(fmt.Sprintf("## %s (%d)\n\n", artifactType, len(artifacts)))

		// Show latest few artifacts
		start := 0
		if len(artifacts) > 3 {
			start = len(artifacts) - 3
		}

		for i := start; i < len(artifacts); i++ {
			env := artifacts[i]
			summary.WriteString(fmt.Sprintf("- Phase: %s, Domain: %s, Created: %s\n",
				env.Metadata.Phase, env.Metadata.Domain, env.Metadata.CreatedAt.Format(time.RFC3339)))

			// Show a snippet of the data
			if len(env.Data) > 0 && len(env.Data) < 200 {
				summary.WriteString(fmt.Sprintf("  Data: %s\n", string(env.Data)))
			} else if len(env.Data) > 0 {
				summary.WriteString(fmt.Sprintf("  Data: %s... (%d bytes)\n", string(env.Data[:100]), len(env.Data)))
			}
		}
		summary.WriteString("\n")
	}

	return summary.String()
}

// RecordUserObjective records the user's assessment objective
func (b *Blackboard) RecordUserObjective(objective string) error {
	// Create a simple artifact directly
	data := map[string]string{
		"objective": objective,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal objective: %w", err)
	}

	env := ArtifactEnvelope{
		Metadata: ArtifactMetadata{
			Type:      "UserObjective",
			CreatedAt: time.Now(),
			Phase:     "input",
			Domain:    "meta",
		},
		Data: jsonData,
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.artifacts["UserObjective"] == nil {
		b.artifacts["UserObjective"] = []ArtifactEnvelope{}
	}
	b.artifacts["UserObjective"] = append(b.artifacts["UserObjective"], env)

	return nil
}

// RecordSpecialistOutput records output from a specialist agent
func (b *Blackboard) RecordSpecialistOutput(agentName string, output string) error {
	data := map[string]string{
		"agent":  agentName,
		"output": output,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal specialist output: %w", err)
	}

	env := ArtifactEnvelope{
		Metadata: ArtifactMetadata{
			Type:      "SpecialistOutput",
			CreatedAt: time.Now(),
			Phase:     agentName,
			Domain:    "meta",
		},
		Data: jsonData,
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.artifacts["SpecialistOutput"] == nil {
		b.artifacts["SpecialistOutput"] = []ArtifactEnvelope{}
	}
	b.artifacts["SpecialistOutput"] = append(b.artifacts["SpecialistOutput"], env)

	return nil
}

// RecordError records an error from agent execution
func (b *Blackboard) RecordError(agentName string, errorMsg string) error {
	data := map[string]string{
		"agent": agentName,
		"error": errorMsg,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal error: %w", err)
	}

	env := ArtifactEnvelope{
		Metadata: ArtifactMetadata{
			Type:      "ExecutionError",
			CreatedAt: time.Now(),
			Phase:     agentName,
			Domain:    "meta",
		},
		Data: jsonData,
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.artifacts["ExecutionError"] == nil {
		b.artifacts["ExecutionError"] = []ArtifactEnvelope{}
	}
	b.artifacts["ExecutionError"] = append(b.artifacts["ExecutionError"], env)

	return nil
}
