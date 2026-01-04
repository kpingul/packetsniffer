package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Generator creates summary JSON files
type Generator struct {
	outputDir string
}

// NewGenerator creates a new summary generator
func NewGenerator(outputDir string) *Generator {
	return &Generator{
		outputDir: outputDir,
	}
}

// Generate creates and writes a summary file
func (g *Generator) Generate(summary *Summary) (string, error) {
	// Ensure output directory exists
	if err := os.MkdirAll(g.outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate filename with timestamp
	timestamp := summary.Capture.StartTime.Format("20060102_150405")
	filename := fmt.Sprintf("summary_%s.json", timestamp)
	filepath := filepath.Join(g.outputDir, filename)

	// Marshal to JSON with indentation
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal summary: %w", err)
	}

	// Write file
	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write summary file: %w", err)
	}

	return filepath, nil
}

// NewSummary creates a new summary with sensor info
func NewSummary(sensorOS, hostname, ifaceName, localIP string) *Summary {
	return &Summary{
		Sensor: SensorInfo{
			OS:        sensorOS,
			Hostname:  hostname,
			Interface: ifaceName,
			LocalIP:   localIP,
		},
		Capture: CaptureInfo{
			StartTime: time.Now(),
		},
		Devices: make([]DeviceInfo, 0),
		Traffic: TrafficInfo{
			ProtocolCounts: make(map[string]int64),
			TopPorts:       make([]PortCount, 0),
			TopTalkers:     make([]TalkerInfo, 0),
			DNSDomains:     make([]DNSDomainInfo, 0),
			Destinations:   make([]DestinationInfo, 0),
		},
	}
}

// SetCaptureInfo sets capture metadata
func (s *Summary) SetCaptureInfo(startTime time.Time, duration int, packetCount int64) {
	s.Capture.StartTime = startTime
	s.Capture.Duration = duration
	s.Capture.PacketCount = packetCount
}

// SetDevices sets the devices list
func (s *Summary) SetDevices(devices []DeviceInfo) {
	s.Devices = devices
}

// SetTraffic sets the traffic statistics
func (s *Summary) SetTraffic(traffic TrafficInfo) {
	s.Traffic = traffic
}

// PrettyPrint returns a formatted string representation
func (s *Summary) PrettyPrint() string {
	result := fmt.Sprintf(`
Capture Summary
===============
Sensor:     %s (%s)
Interface:  %s (%s)
Duration:   %d seconds
Packets:    %d
Devices:    %d discovered

Top Protocols:
`,
		s.Sensor.Hostname,
		s.Sensor.OS,
		s.Sensor.Interface,
		s.Sensor.LocalIP,
		s.Capture.Duration,
		s.Capture.PacketCount,
		len(s.Devices),
	)

	// Add protocol counts
	for proto, count := range s.Traffic.ProtocolCounts {
		result += fmt.Sprintf("  %s: %d\n", proto, count)
	}

	// Add top ports
	if len(s.Traffic.TopPorts) > 0 {
		result += "\nTop Ports:\n"
		limit := 5
		if len(s.Traffic.TopPorts) < limit {
			limit = len(s.Traffic.TopPorts)
		}
		for i := 0; i < limit; i++ {
			p := s.Traffic.TopPorts[i]
			result += fmt.Sprintf("  %s/%d: %d\n", p.Protocol, p.Port, p.Count)
		}
	}

	// Add device summary
	if len(s.Devices) > 0 {
		result += "\nDiscovered Devices:\n"
		limit := 10
		if len(s.Devices) < limit {
			limit = len(s.Devices)
		}
		for i := 0; i < limit; i++ {
			d := s.Devices[i]
			osInfo := "Unknown"
			if d.OSGuess != "" {
				osInfo = fmt.Sprintf("%s (%.0f%%)", d.OSGuess, d.Confidence*100)
			}
			ips := ""
			if len(d.IPs) > 0 {
				ips = d.IPs[0]
				if len(d.IPs) > 1 {
					ips += fmt.Sprintf(" (+%d more)", len(d.IPs)-1)
				}
			}
			result += fmt.Sprintf("  %s | %s | %s | %s\n", d.MAC, ips, d.Vendor, osInfo)
		}
		if len(s.Devices) > limit {
			result += fmt.Sprintf("  ... and %d more devices\n", len(s.Devices)-limit)
		}
	}

	return result
}
