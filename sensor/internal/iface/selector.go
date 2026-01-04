package iface

import (
	"fmt"
	"net"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/gopacket/gopacket/pcap"
)

// Selector handles network interface enumeration and selection
type Selector struct {
	virtualPatterns []*regexp.Regexp
}

// NewSelector creates a new interface selector
func NewSelector() *Selector {
	patterns := []string{
		`^lo\d*$`,            // Loopback
		`^docker\d*$`,        // Docker
		`^veth.*`,            // Virtual ethernet (Docker, etc.)
		`^br-.*`,             // Docker bridge
		`^virbr\d*$`,         // libvirt bridge
		`^vmnet\d*$`,         // VMware
		`^vboxnet\d*$`,       // VirtualBox
		`^utun\d*$`,          // macOS tunnels
		`^awdl\d*$`,          // Apple Wireless Direct Link
		`^llw\d*$`,           // Low latency WLAN
		`^bridge\d*$`,        // Bridge interfaces
		`^Loopback.*`,        // Windows loopback
		`^isatap.*`,          // Windows ISATAP
		`^Teredo.*`,          // Windows Teredo
		`.*Hyper-V.*`,        // Hyper-V
		`.*Virtual.*Adapter`, // Generic virtual
	}

	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		if re, err := regexp.Compile(p); err == nil {
			compiled = append(compiled, re)
		}
	}

	return &Selector{virtualPatterns: compiled}
}

// ListInterfaces returns all available capture-capable interfaces
func (s *Selector) ListInterfaces() ([]InterfaceInfo, error) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate interfaces: %w", err)
	}

	// Get system interface info for UP status
	sysIfaces, _ := net.Interfaces()
	sysIfaceMap := make(map[string]net.Interface)
	for _, i := range sysIfaces {
		sysIfaceMap[i.Name] = i
	}

	result := make([]InterfaceInfo, 0, len(devices))
	for _, dev := range devices {
		info := InterfaceInfo{
			Name:        dev.Name,
			Description: dev.Description,
		}

		// Get IPs
		for _, addr := range dev.Addresses {
			if addr.IP != nil {
				info.IPs = append(info.IPs, addr.IP.String())
			}
		}

		// Check system interface for additional info
		if sysIface, ok := sysIfaceMap[dev.Name]; ok {
			info.MAC = sysIface.HardwareAddr.String()
			info.IsUp = sysIface.Flags&net.FlagUp != 0
			info.IsLoopback = sysIface.Flags&net.FlagLoopback != 0
		} else {
			// Fallback: assume up if has addresses
			info.IsUp = len(dev.Addresses) > 0
			info.IsLoopback = s.isLoopbackName(dev.Name)
		}

		info.IsVirtual = s.isVirtualInterface(dev.Name)
		info.Score = s.scoreInterface(info)

		result = append(result, info)
	}

	// Sort by score descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].Score > result[j].Score
	})

	return result, nil
}

// AutoSelect returns the best interface for capture
func (s *Selector) AutoSelect() (*InterfaceInfo, error) {
	ifaces, err := s.ListInterfaces()
	if err != nil {
		return nil, err
	}

	// Filter to only usable interfaces
	var candidates []InterfaceInfo
	for _, iface := range ifaces {
		if iface.Score > 0 {
			candidates = append(candidates, iface)
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no suitable network interface found")
	}

	return &candidates[0], nil
}

// GetInterfaceByName returns a specific interface by name
func (s *Selector) GetInterfaceByName(name string) (*InterfaceInfo, error) {
	ifaces, err := s.ListInterfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range ifaces {
		if iface.Name == name {
			return &iface, nil
		}
	}

	return nil, fmt.Errorf("interface %q not found", name)
}

// scoreInterface assigns a score to an interface for auto-selection
func (s *Selector) scoreInterface(info InterfaceInfo) int {
	score := 0

	// Exclude loopback
	if info.IsLoopback {
		return 0
	}

	// Exclude virtual interfaces
	if info.IsVirtual {
		return 0
	}

	// Must be UP
	if !info.IsUp {
		return 0
	}

	// Prefer interfaces with RFC1918 addresses
	for _, ipStr := range info.IPs {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}
		if IsRFC1918(ip) {
			score += 100 // Strong preference for private IPs
		} else if ip.To4() != nil {
			score += 20 // Some points for any IPv4
		}
	}

	// Bonus for common physical interface names
	if s.isPhysicalName(info.Name) {
		score += 50
	}

	// Small bonus for having a description
	if info.Description != "" {
		score += 5
	}

	return score
}

// isLoopbackName checks if the interface name indicates loopback
func (s *Selector) isLoopbackName(name string) bool {
	lower := strings.ToLower(name)
	return lower == "lo" || lower == "lo0" || strings.Contains(lower, "loopback")
}

// isVirtualInterface checks if the interface appears to be virtual
func (s *Selector) isVirtualInterface(name string) bool {
	for _, re := range s.virtualPatterns {
		if re.MatchString(name) {
			return true
		}
	}
	return false
}

// isPhysicalName checks if the interface name suggests a physical interface
func (s *Selector) isPhysicalName(name string) bool {
	// Common physical interface name patterns by OS
	switch runtime.GOOS {
	case "darwin":
		// en0, en1, etc. are typically physical on macOS
		if matched, _ := regexp.MatchString(`^en\d+$`, name); matched {
			return true
		}
	case "linux":
		// eth*, wlan*, enp*, wlp* are typically physical on Linux
		patterns := []string{`^eth\d+$`, `^wlan\d+$`, `^enp\d+s\d+.*`, `^wlp\d+s\d+.*`}
		for _, p := range patterns {
			if matched, _ := regexp.MatchString(p, name); matched {
				return true
			}
		}
	case "windows":
		// On Windows, look for "Ethernet" or "Wi-Fi" in description
		lower := strings.ToLower(name)
		if strings.Contains(lower, "ethernet") || strings.Contains(lower, "wi-fi") {
			return true
		}
	}
	return false
}

// FormatInterfaceList formats interfaces for CLI display
func FormatInterfaceList(ifaces []InterfaceInfo) string {
	var sb strings.Builder
	sb.WriteString("Available Network Interfaces:\n")
	sb.WriteString(strings.Repeat("-", 70) + "\n")

	for i, iface := range ifaces {
		status := "DOWN"
		if iface.IsUp {
			status = "UP"
		}

		flags := ""
		if iface.IsLoopback {
			flags += " [loopback]"
		}
		if iface.IsVirtual {
			flags += " [virtual]"
		}

		sb.WriteString(fmt.Sprintf("%2d. %s (%s)%s\n", i+1, iface.Name, status, flags))

		if iface.Description != "" {
			sb.WriteString(fmt.Sprintf("    Description: %s\n", iface.Description))
		}
		if iface.MAC != "" {
			sb.WriteString(fmt.Sprintf("    MAC: %s\n", iface.MAC))
		}
		if len(iface.IPs) > 0 {
			sb.WriteString(fmt.Sprintf("    IPs: %s\n", strings.Join(iface.IPs, ", ")))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
