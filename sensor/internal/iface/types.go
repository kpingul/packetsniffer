package iface

import "net"

// InterfaceInfo contains information about a network interface
type InterfaceInfo struct {
	Name        string   // Interface name (e.g., "en0", "eth0", "Wi-Fi")
	Description string   // Human-readable description
	MAC         string   // Hardware MAC address
	IPs         []string // Assigned IP addresses
	IsUp        bool     // Whether interface is up
	IsLoopback  bool     // Whether interface is loopback
	IsVirtual   bool     // Whether interface appears to be virtual
	Score       int      // Selection score (higher = better candidate)
}

// ScoredInterface pairs an interface with its selection score
type ScoredInterface struct {
	Info  InterfaceInfo
	Score int
}

// IsRFC1918 checks if an IP address is in RFC1918 private address space
func IsRFC1918(ip net.IP) bool {
	ip = ip.To4()
	if ip == nil {
		return false
	}

	// 10.0.0.0/8
	if ip[0] == 10 {
		return true
	}
	// 172.16.0.0/12
	if ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31 {
		return true
	}
	// 192.168.0.0/16
	if ip[0] == 192 && ip[1] == 168 {
		return true
	}
	return false
}

// GetLocalSubnet returns the local subnet for an interface based on its IP
func GetLocalSubnet(ip net.IP, mask net.IPMask) *net.IPNet {
	if ip == nil || mask == nil {
		return nil
	}
	return &net.IPNet{
		IP:   ip.Mask(mask),
		Mask: mask,
	}
}
