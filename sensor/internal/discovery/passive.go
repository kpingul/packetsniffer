package discovery

import (
	"github.com/asset_discovery/sensor/internal/capture"
	"github.com/asset_discovery/sensor/internal/oui"
	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
)

// PassiveDiscovery discovers devices from observed traffic
type PassiveDiscovery struct {
	registry *DeviceRegistry
	oui      *oui.Lookup
}

// NewPassiveDiscovery creates a new passive discovery instance
func NewPassiveDiscovery(registry *DeviceRegistry, ouiLookup *oui.Lookup) *PassiveDiscovery {
	return &PassiveDiscovery{
		registry: registry,
		oui:      ouiLookup,
	}
}

// ProcessPacket extracts device information from a packet
func (p *PassiveDiscovery) ProcessPacket(packet gopacket.Packet) {
	// Extract MACs
	srcMAC, _ := capture.ExtractMACs(packet)
	if srcMAC == "" || srcMAC == "ff:ff:ff:ff:ff:ff" {
		return
	}

	// Skip broadcast/multicast MACs
	if isBroadcastOrMulticast(srcMAC) {
		return
	}

	// Get or create device
	device := p.registry.GetOrCreate(srcMAC)

	// Add vendor if not set
	if device.Vendor == "" {
		device.Vendor = p.oui.GetVendor(srcMAC)
	}

	// Extract and add IP
	srcIP, _ := capture.ExtractIPs(packet)
	if srcIP != "" && !isBroadcastIP(srcIP) {
		device.AddIP(srcIP)
	}

	// Try to extract hostname from various protocols
	p.extractHostname(packet, device)

	// Process ARP for additional IP-MAC mappings
	p.processARP(packet)

	// Process DHCP for hostname and IP info
	p.processDHCP(packet)
}

// extractHostname tries to extract hostname from various protocols
func (p *PassiveDiscovery) extractHostname(packet gopacket.Packet, device *Device) {
	// Try mDNS
	if hostname := p.extractMDNSHostname(packet); hostname != "" && device.Hostname == "" {
		device.Hostname = hostname
	}

	// Try NBNS
	if hostname := p.extractNBNSHostname(packet); hostname != "" && device.Hostname == "" {
		device.Hostname = hostname
	}
}

// extractMDNSHostname extracts hostname from mDNS packets
func (p *PassiveDiscovery) extractMDNSHostname(packet gopacket.Packet) string {
	dnsLayer := packet.Layer(layers.LayerTypeDNS)
	if dnsLayer == nil {
		return ""
	}

	_, dstPort, proto := capture.ExtractPorts(packet)
	if proto != "UDP" || dstPort != 5353 {
		return ""
	}

	dns := dnsLayer.(*layers.DNS)

	// Look for hostname in answers
	for _, answer := range dns.Answers {
		name := string(answer.Name)
		if len(name) > 6 && name[len(name)-6:] == ".local" {
			// Strip .local suffix and any service type
			hostname := name[:len(name)-6]
			// Remove any service prefix like "_tcp." etc.
			for i := len(hostname) - 1; i >= 0; i-- {
				if hostname[i] == '.' {
					return hostname[:i]
				}
			}
			return hostname
		}
	}

	return ""
}

// extractNBNSHostname extracts hostname from NBNS packets
func (p *PassiveDiscovery) extractNBNSHostname(packet gopacket.Packet) string {
	// NBNS uses UDP port 137
	_, dstPort, proto := capture.ExtractPorts(packet)
	if proto != "UDP" || (dstPort != 137 && dstPort != 138) {
		return ""
	}

	// NBNS name decoding is complex - simplified version
	// Full implementation would decode the NetBIOS name from the payload
	return ""
}

// processARP extracts IP-MAC mappings from ARP packets
func (p *PassiveDiscovery) processARP(packet gopacket.Packet) {
	arpLayer := packet.Layer(layers.LayerTypeARP)
	if arpLayer == nil {
		return
	}

	arp := arpLayer.(*layers.ARP)

	// Process sender
	if arp.Operation == layers.ARPReply || arp.Operation == layers.ARPRequest {
		srcMAC := formatMAC(arp.SourceHwAddress)
		srcIP := formatIP(arp.SourceProtAddress)

		if srcMAC != "" && srcIP != "" && !isBroadcastOrMulticast(srcMAC) {
			device := p.registry.GetOrCreate(srcMAC)
			device.AddIP(srcIP)
			if device.Vendor == "" {
				device.Vendor = p.oui.GetVendor(srcMAC)
			}
		}
	}
}

// processDHCP extracts information from DHCP packets
func (p *PassiveDiscovery) processDHCP(packet gopacket.Packet) {
	// DHCP uses UDP 67/68
	_, dstPort, proto := capture.ExtractPorts(packet)
	if proto != "UDP" || (dstPort != 67 && dstPort != 68) {
		return
	}

	// Parse DHCP layer (gopacket has DHCPv4 support)
	dhcpLayer := packet.Layer(layers.LayerTypeDHCPv4)
	if dhcpLayer == nil {
		return
	}

	dhcp := dhcpLayer.(*layers.DHCPv4)
	srcMAC := formatMAC(dhcp.ClientHWAddr)

	if srcMAC == "" || isBroadcastOrMulticast(srcMAC) {
		return
	}

	device := p.registry.GetOrCreate(srcMAC)

	// Get hostname from DHCP options
	for _, opt := range dhcp.Options {
		if opt.Type == layers.DHCPOptHostname {
			if device.Hostname == "" {
				device.Hostname = string(opt.Data)
			}
		}
	}

	// Add client IP if assigned
	if dhcp.YourClientIP != nil && !dhcp.YourClientIP.IsUnspecified() {
		device.AddIP(dhcp.YourClientIP.String())
	}
}

// Helper functions
func formatMAC(addr []byte) string {
	if len(addr) != 6 {
		return ""
	}
	return formatHex(addr[0]) + ":" + formatHex(addr[1]) + ":" + formatHex(addr[2]) + ":" +
		formatHex(addr[3]) + ":" + formatHex(addr[4]) + ":" + formatHex(addr[5])
}

func formatHex(b byte) string {
	const hex = "0123456789abcdef"
	return string(hex[b>>4]) + string(hex[b&0xf])
}

func formatIP(addr []byte) string {
	if len(addr) != 4 {
		return ""
	}
	return itoa(int(addr[0])) + "." + itoa(int(addr[1])) + "." + itoa(int(addr[2])) + "." + itoa(int(addr[3]))
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}

func isBroadcastOrMulticast(mac string) bool {
	if len(mac) < 2 {
		return false
	}
	// Check first byte for multicast bit
	firstByte := mac[0:2]
	if firstByte == "ff" || firstByte == "FF" {
		return true
	}
	// Check multicast bit (LSB of first byte)
	if len(mac) >= 2 {
		c := mac[1]
		if c >= 'a' && c <= 'f' {
			c = c - 32 // to uppercase
		}
		if c == '1' || c == '3' || c == '5' || c == '7' || c == '9' || c == 'B' || c == 'D' || c == 'F' {
			return true
		}
	}
	return false
}

func isBroadcastIP(ip string) bool {
	return ip == "255.255.255.255" || ip == "0.0.0.0"
}
