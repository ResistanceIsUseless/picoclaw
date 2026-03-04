package parsers

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/ResistanceIsUseless/picoclaw/pkg/artifacts"
	"github.com/ResistanceIsUseless/picoclaw/pkg/blackboard"
)

// HTTPXResult represents a single line of httpx JSON output
type HTTPXResult struct {
	URL           string   `json:"url"`
	StatusCode    int      `json:"status_code"`
	ContentLength int      `json:"content_length"`
	ContentType   string   `json:"content_type"`
	Title         string   `json:"title"`
	Host          string   `json:"host"`
	Port          string   `json:"port"`
	Scheme        string   `json:"scheme"`
	Webserver     string   `json:"webserver"`
	ResponseTime  string   `json:"response_time"`
	Tech          []string `json:"tech"`
	Method        string   `json:"method"`
	IP            string   `json:"ip"`
	CDN           string   `json:"cdn"`
	CDNName       string   `json:"cdn_name"`
	A             []string `json:"a"`
	CNAME         []string `json:"cname"`
	TLS           *struct {
		Host          string   `json:"host"`
		Port          string   `json:"port"`
		Version       string   `json:"version"`
		Cipher        string   `json:"cipher"`
		TLSConnection string   `json:"tls_connection"`
		SubjectDN     string   `json:"subject_dn"`
		IssuerDN      string   `json:"issuer_dn"`
		NotBefore     string   `json:"not_before"`
		NotAfter      string   `json:"not_after"`
		SubjectAN     []string `json:"subject_an"`
	} `json:"tls"`
	Headers map[string]string `json:"header"`
}

// ParseHTTPXOutput parses httpx JSON output into WebFindings artifact
// httpx outputs one JSON object per line when using -json flag
func ParseHTTPXOutput(toolName string, output []byte, phase string) (*artifacts.WebFindings, error) {
	endpoints := make([]artifacts.Endpoint, 0)
	technologies := make([]artifacts.Technology, 0)
	techMap := make(map[string]bool) // dedupe technologies
	statusCodeDist := make(map[int]int)
	contentTypeDist := make(map[string]int)
	uniqueHosts := make(map[string]bool)
	uniquePaths := make(map[string]bool)

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var result HTTPXResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			// Skip malformed lines
			continue
		}

		// Parse response time
		var responseTime time.Duration
		if result.ResponseTime != "" {
			// httpx outputs like "123ms" or "1.5s"
			if strings.HasSuffix(result.ResponseTime, "ms") {
				ms, _ := strconv.ParseFloat(strings.TrimSuffix(result.ResponseTime, "ms"), 64)
				responseTime = time.Duration(ms * float64(time.Millisecond))
			} else if strings.HasSuffix(result.ResponseTime, "s") {
				s, _ := strconv.ParseFloat(strings.TrimSuffix(result.ResponseTime, "s"), 64)
				responseTime = time.Duration(s * float64(time.Second))
			}
		}

		// Build endpoint
		endpoint := artifacts.Endpoint{
			URL:           result.URL,
			Method:        result.Method,
			StatusCode:    result.StatusCode,
			ContentType:   result.ContentType,
			ContentLength: result.ContentLength,
			ResponseTime:  responseTime,
			Headers:       result.Headers,
			Title:         result.Title,
			DiscoveredAt:  time.Now(),
			Source:        toolName,
		}
		endpoints = append(endpoints, endpoint)

		// Track stats
		statusCodeDist[result.StatusCode]++
		if result.ContentType != "" {
			contentTypeDist[result.ContentType]++
		}
		if result.Host != "" {
			uniqueHosts[result.Host] = true
		}
		// Extract path from URL
		if idx := strings.Index(result.URL, "://"); idx != -1 {
			rest := result.URL[idx+3:]
			if idx2 := strings.Index(rest, "/"); idx2 != -1 {
				uniquePaths[rest[idx2:]] = true
			} else {
				uniquePaths["/"] = true
			}
		}

		// Extract technologies
		for _, tech := range result.Tech {
			if !techMap[tech] {
				techMap[tech] = true
				technologies = append(technologies, artifacts.Technology{
					Name:       tech,
					Categories: []string{"detected"},
					Confidence: 80, // httpx detection is fairly reliable
					Evidence:   []string{"httpx detection"},
				})
			}
		}

		// Add webserver as technology if present
		if result.Webserver != "" && !techMap[result.Webserver] {
			techMap[result.Webserver] = true
			technologies = append(technologies, artifacts.Technology{
				Name:       result.Webserver,
				Categories: []string{"web-server"},
				Confidence: 90,
				Evidence:   []string{"Server header"},
			})
		}

		// Add CDN as technology if present
		if result.CDNName != "" && !techMap[result.CDNName] {
			techMap[result.CDNName] = true
			technologies = append(technologies, artifacts.Technology{
				Name:       result.CDNName,
				Categories: []string{"cdn"},
				Confidence: 95,
				Evidence:   []string{"CDN detection"},
			})
		}
	}

	return &artifacts.WebFindings{
		Metadata: blackboard.ArtifactMetadata{
			Type:      "WebFindings",
			CreatedAt: time.Now(),
			Phase:     phase,
			Version:   "1.0",
			Domain:    "web",
		},
		Endpoints:    endpoints,
		Parameters:   make([]artifacts.Parameter, 0),    // httpx doesn't extract parameters
		Technologies: technologies,
		Findings:     make([]artifacts.WebFinding, 0),   // httpx doesn't report vulns
		Crawled: artifacts.CrawlStats{
			TotalURLs:           len(endpoints),
			UniqueHosts:         len(uniqueHosts),
			UniquePaths:         len(uniquePaths),
			StatusCodeDist:      statusCodeDist,
			ContentTypeDist:     contentTypeDist,
			CrawlDuration:       0, // not tracked by httpx
		},
	}, nil
}

// ParseHTTPXToServiceFingerprint converts httpx output to ServiceFingerprint
// This is an alternative artifact type focused on service identification
func ParseHTTPXToServiceFingerprint(toolName string, output []byte, phase string) (*artifacts.ServiceFingerprint, error) {
	services := make([]artifacts.IdentifiedService, 0)

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var result HTTPXResult
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			continue
		}

		// Parse port
		port := 443 // default HTTPS
		if result.Port != "" {
			if p, err := strconv.Atoi(result.Port); err == nil {
				port = p
			}
		} else if result.Scheme == "http" {
			port = 80
		}

		service := artifacts.IdentifiedService{
			Host:         result.Host,
			Port:         port,
			Protocol:     result.Scheme,
			Service:      "http",
			Version:      "", // httpx doesn't provide detailed version
			Banner:       result.Webserver,
			Headers:      result.Headers,
			Metadata:     make(map[string]string),
			DiscoveredAt: time.Now(),
		}

		// Add metadata
		if result.Title != "" {
			service.Metadata["title"] = result.Title
		}
		if result.CDNName != "" {
			service.Metadata["cdn"] = result.CDNName
		}
		if len(result.Tech) > 0 {
			service.Metadata["technologies"] = strings.Join(result.Tech, ", ")
		}

		// Add TLS info if present
		if result.TLS != nil {
			service.TLS = &artifacts.TLSInfo{
				Version: result.TLS.Version,
				Cipher:  result.TLS.Cipher,
				Certificate: artifacts.Certificate{
					Subject:         result.TLS.SubjectDN,
					Issuer:          result.TLS.IssuerDN,
					SubjectAltNames: result.TLS.SubjectAN,
					// Note: NotBefore/NotAfter parsing would require date format handling
				},
			}
		}

		services = append(services, service)
	}

	return &artifacts.ServiceFingerprint{
		Metadata: blackboard.ArtifactMetadata{
			Type:      "ServiceFingerprint",
			CreatedAt: time.Now(),
			Phase:     phase,
			Version:   "1.0",
			Domain:    "web",
		},
		Services: services,
		Total:    len(services),
	}, nil
}
