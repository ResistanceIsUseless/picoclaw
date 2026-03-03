package blackboard

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ResistanceIsUseless/picoclaw/pkg/logger"
)

// Persister handles disk persistence of artifacts
type Persister interface {
	// Persist saves a single artifact to disk
	Persist(envelope ArtifactEnvelope) error

	// Load retrieves all persisted artifacts
	Load() (map[string][]ArtifactEnvelope, error)

	// Clear removes all persisted data
	Clear() error
}

// FilePersister implements disk-based persistence using JSON files
type FilePersister struct {
	baseDir string
}

// NewFilePersister creates a new file-based persister
func NewFilePersister(baseDir string) (*FilePersister, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create persistence directory: %w", err)
	}

	return &FilePersister{
		baseDir: baseDir,
	}, nil
}

// Persist saves an artifact to disk as a JSON file
func (f *FilePersister) Persist(envelope ArtifactEnvelope) error {
	// Create type-specific directory
	typeDir := filepath.Join(f.baseDir, envelope.Metadata.Type)
	if err := os.MkdirAll(typeDir, 0755); err != nil {
		return fmt.Errorf("failed to create type directory: %w", err)
	}

	// Generate filename with timestamp
	filename := fmt.Sprintf("%s-%s.json",
		envelope.Metadata.Phase,
		envelope.Metadata.CreatedAt.Format("20060102-150405"))
	filepath := filepath.Join(typeDir, filename)

	// Marshal envelope to JSON
	data, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal artifact: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write artifact file: %w", err)
	}

	logger.DebugCF("blackboard", "Artifact persisted",
		map[string]any{
			"type": envelope.Metadata.Type,
			"path": filepath,
		})

	return nil
}

// Load retrieves all persisted artifacts from disk
func (f *FilePersister) Load() (map[string][]ArtifactEnvelope, error) {
	artifacts := make(map[string][]ArtifactEnvelope)

	// Check if base directory exists
	if _, err := os.Stat(f.baseDir); os.IsNotExist(err) {
		logger.DebugCF("blackboard", "No persisted data found", nil)
		return artifacts, nil
	}

	// Read type directories
	typeDirs, err := os.ReadDir(f.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read persistence directory: %w", err)
	}

	for _, typeDir := range typeDirs {
		if !typeDir.IsDir() {
			continue
		}

		artifactType := typeDir.Name()
		typeArtifacts, err := f.loadType(artifactType)
		if err != nil {
			logger.WarnCF("blackboard", "Failed to load artifact type",
				map[string]any{
					"type":  artifactType,
					"error": err.Error(),
				})
			continue
		}

		artifacts[artifactType] = typeArtifacts
	}

	logger.InfoCF("blackboard", "Loaded persisted artifacts",
		map[string]any{
			"types": len(artifacts),
		})

	return artifacts, nil
}

// loadType loads all artifacts of a specific type
func (f *FilePersister) loadType(artifactType string) ([]ArtifactEnvelope, error) {
	typeDir := filepath.Join(f.baseDir, artifactType)
	files, err := os.ReadDir(typeDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read type directory: %w", err)
	}

	var artifacts []ArtifactEnvelope
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(typeDir, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			logger.WarnCF("blackboard", "Failed to read artifact file",
				map[string]any{
					"path":  filePath,
					"error": err.Error(),
				})
			continue
		}

		var envelope ArtifactEnvelope
		if err := json.Unmarshal(data, &envelope); err != nil {
			logger.WarnCF("blackboard", "Failed to unmarshal artifact",
				map[string]any{
					"path":  filePath,
					"error": err.Error(),
				})
			continue
		}

		artifacts = append(artifacts, envelope)
	}

	// Sort by creation time
	for i := 0; i < len(artifacts)-1; i++ {
		for j := i + 1; j < len(artifacts); j++ {
			if artifacts[i].Metadata.CreatedAt.After(artifacts[j].Metadata.CreatedAt) {
				artifacts[i], artifacts[j] = artifacts[j], artifacts[i]
			}
		}
	}

	return artifacts, nil
}

// Clear removes all persisted artifacts
func (f *FilePersister) Clear() error {
	if err := os.RemoveAll(f.baseDir); err != nil {
		return fmt.Errorf("failed to clear persistence directory: %w", err)
	}

	logger.WarnCF("blackboard", "Cleared all persisted artifacts", nil)
	return nil
}

// Archive moves all current artifacts to an archive directory
func (f *FilePersister) Archive() error {
	timestamp := time.Now().Format("20060102-150405")
	archiveDir := filepath.Join(filepath.Dir(f.baseDir), fmt.Sprintf("blackboard-archive-%s", timestamp))

	if err := os.Rename(f.baseDir, archiveDir); err != nil {
		return fmt.Errorf("failed to archive persistence directory: %w", err)
	}

	// Recreate base directory
	if err := os.MkdirAll(f.baseDir, 0755); err != nil {
		return fmt.Errorf("failed to recreate persistence directory: %w", err)
	}

	logger.InfoCF("blackboard", "Archived persisted artifacts",
		map[string]any{
			"archive_dir": archiveDir,
		})

	return nil
}

// NullPersister implements Persister but does nothing (for testing or disabled persistence)
type NullPersister struct{}

func (n *NullPersister) Persist(envelope ArtifactEnvelope) error {
	return nil
}

func (n *NullPersister) Load() (map[string][]ArtifactEnvelope, error) {
	return make(map[string][]ArtifactEnvelope), nil
}

func (n *NullPersister) Clear() error {
	return nil
}
