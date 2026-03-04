package parsers

import (
	"encoding/xml"
	"strings"
	"time"

	"github.com/ResistanceIsUseless/picoclaw/pkg/artifacts"
	"github.com/ResistanceIsUseless/picoclaw/pkg/blackboard"
)

// NmapRun represents the root element of nmap XML output
type NmapRun struct {
	XMLName xml.Name   `xml:"nmaprun"`
	Hosts   []NmapHost `xml:"host"`
	RunStats NmapRunStats `xml:"runstats"`
}

type NmapRunStats struct {
	Finished NmapFinished `xml:"finished"`
}

type NmapFinished struct {
	Time    string  `xml:"time,attr"`
	Elapsed float64 `xml:"elapsed,attr"`
}

type NmapHost struct {
	Status    NmapStatus    `xml:"status"`
	Addresses []NmapAddress `xml:"address"`
	Hostnames NmapHostnames `xml:"hostnames"`
	Ports     NmapPorts     `xml:"ports"`
	OS        NmapOS        `xml:"os"`
}

type NmapStatus struct {
	State  string `xml:"state,attr"`
	Reason string `xml:"reason,attr"`
}

type NmapAddress struct {
	Addr     string `xml:"addr,attr"`
	AddrType string `xml:"addrtype,attr"` // ipv4, ipv6, mac
}

type NmapHostnames struct {
	Hostnames []NmapHostname `xml:"hostname"`
}

type NmapHostname struct {
	Name string `xml:"name,attr"`
	Type string `xml:"type,attr"`
}

type NmapPorts struct {
	Ports []NmapPort `xml:"port"`
}

type NmapPort struct {
	Protocol string         `xml:"protocol,attr"`
	PortID   int            `xml:"portid,attr"`
	State    NmapPortState  `xml:"state"`
	Service  NmapService    `xml:"service"`
	Scripts  []NmapScript   `xml:"script"`
}

type NmapPortState struct {
	State  string `xml:"state,attr"` // open, closed, filtered
	Reason string `xml:"reason,attr"`
}

type NmapService struct {
	Name       string `xml:"name,attr"`
	Product    string `xml:"product,attr"`
	Version    string `xml:"version,attr"`
	ExtraInfo  string `xml:"extrainfo,attr"`
	Method     string `xml:"method,attr"`
	Conf       string `xml:"conf,attr"`
	CPE        []string `xml:"cpe"`
	Tunnel     string `xml:"tunnel,attr"`
	OSType     string `xml:"ostype,attr"`
	DeviceType string `xml:"devicetype,attr"`
}

type NmapOS struct {
	OSMatches []NmapOSMatch `xml:"osmatch"`
	OSClasses []NmapOSClass `xml:"osclass"`
}

type NmapOSMatch struct {
	Name     string `xml:"name,attr"`
	Accuracy int    `xml:"accuracy,attr"`
	Line     string `xml:"line,attr"`
}

type NmapOSClass struct {
	Type     string `xml:"type,attr"`
	Vendor   string `xml:"vendor,attr"`
	OSFamily string `xml:"osfamily,attr"`
	OSGen    string `xml:"osgen,attr"`
	Accuracy int    `xml:"accuracy,attr"`
}

type NmapScript struct {
	ID     string `xml:"id,attr"`
	Output string `xml:"output,attr"`
}

// ParseNmapXMLOutput parses nmap XML output into PortScanResult artifact
// nmap outputs XML when using -oX flag
func ParseNmapXMLOutput(toolName string, output []byte, phase string) (*artifacts.PortScanResult, error) {
	var nmapRun NmapRun
	if err := xml.Unmarshal(output, &nmapRun); err != nil {
		return nil, err
	}

	hosts := make([]artifacts.ScannedHost, 0)
	totalPorts := 0

	for _, nmapHost := range nmapRun.Hosts {
		// Extract hostname
		hostname := ""
		if len(nmapHost.Hostnames.Hostnames) > 0 {
			hostname = nmapHost.Hostnames.Hostnames[0].Name
		}

		// Extract IP address
		ip := ""
		for _, addr := range nmapHost.Addresses {
			if addr.AddrType == "ipv4" {
				ip = addr.Addr
				break
			}
		}
		if ip == "" && len(nmapHost.Addresses) > 0 {
			ip = nmapHost.Addresses[0].Addr
		}

		// Parse ports
		ports := make([]artifacts.OpenPort, 0)
		for _, nmapPort := range nmapHost.Ports.Ports {
			// Build script output map
			scriptOutput := make(map[string]string)
			for _, script := range nmapPort.Scripts {
				scriptOutput[script.ID] = script.Output
			}

			// Build banner from service info
			banner := nmapPort.Service.Product
			if nmapPort.Service.Version != "" {
				banner += " " + nmapPort.Service.Version
			}
			if nmapPort.Service.ExtraInfo != "" {
				banner += " (" + nmapPort.Service.ExtraInfo + ")"
			}

			port := artifacts.OpenPort{
				Port:      nmapPort.PortID,
				Protocol:  nmapPort.Protocol,
				State:     nmapPort.State.State,
				Service:   nmapPort.Service.Name,
				Version:   nmapPort.Service.Version,
				Product:   nmapPort.Service.Product,
				ExtraInfo: nmapPort.Service.ExtraInfo,
				Banner:    strings.TrimSpace(banner),
				Script:    scriptOutput,
			}

			ports = append(ports, port)
			if nmapPort.State.State == "open" {
				totalPorts++
			}
		}

		// Parse OS detection
		var osFingerprint artifacts.OSFingerprint
		if len(nmapHost.OS.OSMatches) > 0 {
			bestMatch := nmapHost.OS.OSMatches[0]
			osFingerprint.Name = bestMatch.Name
			osFingerprint.Accuracy = bestMatch.Accuracy
		}
		if len(nmapHost.OS.OSClasses) > 0 {
			bestClass := nmapHost.OS.OSClasses[0]
			osFingerprint.OSClass = bestClass.Type
			osFingerprint.OSFamily = bestClass.OSFamily
			osFingerprint.OSGeneration = bestClass.OSGen
			if osFingerprint.Accuracy == 0 {
				osFingerprint.Accuracy = bestClass.Accuracy
			}
		}

		host := artifacts.ScannedHost{
			Hostname:  hostname,
			IP:        ip,
			Ports:     ports,
			OS:        osFingerprint,
			Status:    nmapHost.Status.State,
			ScannedAt: time.Now(),
		}

		hosts = append(hosts, host)
	}

	// Parse scan duration
	scanDuration := time.Duration(0)
	if nmapRun.RunStats.Finished.Elapsed > 0 {
		scanDuration = time.Duration(nmapRun.RunStats.Finished.Elapsed * float64(time.Second))
	}

	return &artifacts.PortScanResult{
		Metadata: blackboard.ArtifactMetadata{
			Type:      "PortScanResult",
			CreatedAt: time.Now(),
			Phase:     phase,
			Version:   "1.0",
			Domain:    "web",
		},
		Hosts:        hosts,
		TotalHosts:   len(hosts),
		TotalPorts:   totalPorts,
		ScanDuration: scanDuration,
		Scanner:      toolName,
	}, nil
}
