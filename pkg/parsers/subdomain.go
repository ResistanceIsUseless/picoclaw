package parsers

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"time"

	"github.com/ResistanceIsUseless/picoclaw/pkg/artifacts"
	"github.com/ResistanceIsUseless/picoclaw/pkg/blackboard"
)

// ParseSubfinderOutput parses subfinder JSON output into SubdomainList artifact
func ParseSubfinderOutput(toolName string, output []byte, baseDomain string, phase string) (*artifacts.SubdomainList, error) {
	subdomains := make([]artifacts.Subdomain, 0)
	sources := make(map[string]int)

	// Subfinder outputs one subdomain per line (plain text mode)
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		subdomains = append(subdomains, artifacts.Subdomain{
			Name:         line,
			Source:       toolName,
			Verified:     true, // subfinder verifies DNS resolution
			DiscoveredAt: time.Now(),
		})
		sources[toolName]++
	}

	return &artifacts.SubdomainList{
		Metadata: blackboard.ArtifactMetadata{
			Type:      "SubdomainList",
			CreatedAt: time.Now(),
			Phase:     phase,
			Version:   "1.0",
			Domain:    "web",
		},
		BaseDomain: baseDomain,
		Subdomains: subdomains,
		Sources:    sources,
		Total:      len(subdomains),
	}, nil
}

// ParseAmassOutput parses amass JSON output into SubdomainList artifact
func ParseAmassOutput(toolName string, output []byte, baseDomain string, phase string) (*artifacts.SubdomainList, error) {
	subdomains := make([]artifacts.Subdomain, 0)
	sources := make(map[string]int)

	// Amass can output JSON format (-json flag)
	var amassResults []struct {
		Name   string   `json:"name"`
		Domain string   `json:"domain"`
		Addrs  []string `json:"addresses"`
		Tag    string   `json:"tag"`
		Source string   `json:"source"`
	}

	// Try JSON parsing first
	if err := json.Unmarshal(output, &amassResults); err == nil {
		for _, result := range amassResults {
			subdomains = append(subdomains, artifacts.Subdomain{
				Name:         result.Name,
				IPs:          result.Addrs,
				Source:       result.Source,
				Verified:     len(result.Addrs) > 0,
				DiscoveredAt: time.Now(),
			})
			sources[result.Source]++
		}
	} else {
		// Fall back to plain text parsing (one subdomain per line)
		scanner := bufio.NewScanner(bytes.NewReader(output))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}

			subdomains = append(subdomains, artifacts.Subdomain{
				Name:         line,
				Source:       toolName,
				Verified:     true,
				DiscoveredAt: time.Now(),
			})
			sources[toolName]++
		}
	}

	return &artifacts.SubdomainList{
		Metadata: blackboard.ArtifactMetadata{
			Type:      "SubdomainList",
			CreatedAt: time.Now(),
			Phase:     phase,
			Version:   "1.0",
			Domain:    "web",
		},
		BaseDomain: baseDomain,
		Subdomains: subdomains,
		Sources:    sources,
		Total:      len(subdomains),
	}, nil
}

// MergeSubdomainLists combines multiple SubdomainList artifacts, deduplicating subdomains
func MergeSubdomainLists(lists ...*artifacts.SubdomainList) *artifacts.SubdomainList {
	if len(lists) == 0 {
		return nil
	}

	seen := make(map[string]bool)
	merged := &artifacts.SubdomainList{
		Metadata:   lists[0].Metadata,
		BaseDomain: lists[0].BaseDomain,
		Subdomains: make([]artifacts.Subdomain, 0),
		Sources:    make(map[string]int),
	}

	for _, list := range lists {
		for _, subdomain := range list.Subdomains {
			if !seen[subdomain.Name] {
				seen[subdomain.Name] = true
				merged.Subdomains = append(merged.Subdomains, subdomain)
			}
		}

		// Merge source counts
		for source, count := range list.Sources {
			merged.Sources[source] += count
		}
	}

	merged.Total = len(merged.Subdomains)
	merged.Metadata.CreatedAt = time.Now()

	return merged
}
