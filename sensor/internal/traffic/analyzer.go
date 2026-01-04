package traffic

import (
	"sort"
	"sync"

	"github.com/asset_discovery/sensor/internal/capture"
	"github.com/asset_discovery/sensor/internal/output"
	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
)

// Analyzer aggregates traffic statistics
type Analyzer struct {
	mu sync.RWMutex

	// Protocol counts
	protocols map[string]int64

	// Port counts (key: "protocol:port")
	ports map[string]int64

	// Traffic by IP
	talkers map[string]*talkerStats

	// DNS domains
	domains map[string]*domainStats

	// Destinations (external IPs)
	destinations map[string]*destStats

	// Local subnet for determining "external"
	localPrefix string
}

type talkerStats struct {
	BytesSent       int64
	BytesReceived   int64
	PacketsSent     int64
	PacketsReceived int64
}

type domainStats struct {
	QueryCount  int64
	QueryingIPs map[string]bool
}

type destStats struct {
	ConnectionCount int64
	BytesTotal      int64
}

// NewAnalyzer creates a new traffic analyzer
func NewAnalyzer(localIP string) *Analyzer {
	// Extract prefix for local/external determination
	prefix := ""
	if len(localIP) > 0 {
		// Simple heuristic: use first two octets for /16 matching
		parts := splitIP(localIP)
		if len(parts) >= 2 {
			prefix = parts[0] + "." + parts[1]
		}
	}

	return &Analyzer{
		protocols:    make(map[string]int64),
		ports:        make(map[string]int64),
		talkers:      make(map[string]*talkerStats),
		domains:      make(map[string]*domainStats),
		destinations: make(map[string]*destStats),
		localPrefix:  prefix,
	}
}

// ProcessPacket analyzes a single packet
func (a *Analyzer) ProcessPacket(packet gopacket.Packet) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Count protocol
	proto := capture.GetProtocol(packet)
	a.protocols[proto]++

	// Get IPs
	srcIP, dstIP := capture.ExtractIPs(packet)
	packetSize := capture.GetPacketSize(packet)

	// Track talkers
	if srcIP != "" {
		if _, ok := a.talkers[srcIP]; !ok {
			a.talkers[srcIP] = &talkerStats{}
		}
		a.talkers[srcIP].PacketsSent++
		a.talkers[srcIP].BytesSent += int64(packetSize)
	}
	if dstIP != "" {
		if _, ok := a.talkers[dstIP]; !ok {
			a.talkers[dstIP] = &talkerStats{}
		}
		a.talkers[dstIP].PacketsReceived++
		a.talkers[dstIP].BytesReceived += int64(packetSize)
	}

	// Track ports
	_, dstPort, protocol := capture.ExtractPorts(packet)
	if dstPort > 0 {
		key := protocol + ":" + itoa(int(dstPort))
		a.ports[key]++
	}

	// Track external destinations
	if dstIP != "" && !a.isLocal(dstIP) {
		if _, ok := a.destinations[dstIP]; !ok {
			a.destinations[dstIP] = &destStats{}
		}
		a.destinations[dstIP].ConnectionCount++
		a.destinations[dstIP].BytesTotal += int64(packetSize)
	}

	// Parse DNS
	a.parseDNS(packet, srcIP)
}

// parseDNS extracts DNS query information
func (a *Analyzer) parseDNS(packet gopacket.Packet, srcIP string) {
	dnsLayer := packet.Layer(layers.LayerTypeDNS)
	if dnsLayer == nil {
		return
	}

	dns := dnsLayer.(*layers.DNS)

	// Process queries
	for _, q := range dns.Questions {
		domain := string(q.Name)
		if domain == "" {
			continue
		}

		if _, ok := a.domains[domain]; !ok {
			a.domains[domain] = &domainStats{
				QueryingIPs: make(map[string]bool),
			}
		}
		a.domains[domain].QueryCount++
		if srcIP != "" {
			a.domains[domain].QueryingIPs[srcIP] = true
		}
	}
}

// isLocal checks if an IP is on the local network
func (a *Analyzer) isLocal(ip string) bool {
	if a.localPrefix == "" {
		return false
	}
	parts := splitIP(ip)
	if len(parts) >= 2 {
		return parts[0]+"."+parts[1] == a.localPrefix
	}
	return false
}

// GetResults returns aggregated traffic statistics
func (a *Analyzer) GetResults() output.TrafficInfo {
	a.mu.RLock()
	defer a.mu.RUnlock()

	result := output.TrafficInfo{
		ProtocolCounts: make(map[string]int64),
	}

	// Copy protocol counts
	for k, v := range a.protocols {
		result.ProtocolCounts[k] = v
	}

	// Get top ports
	result.TopPorts = a.getTopPorts(20)

	// Get top talkers
	result.TopTalkers = a.getTopTalkers(20)

	// Get DNS domains
	result.DNSDomains = a.getDNSDomains(50)

	// Get destinations
	result.Destinations = a.getDestinations(20)

	return result
}

func (a *Analyzer) getTopPorts(limit int) []output.PortCount {
	type portEntry struct {
		key   string
		count int64
	}

	entries := make([]portEntry, 0, len(a.ports))
	for k, v := range a.ports {
		entries = append(entries, portEntry{k, v})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].count > entries[j].count
	})

	if len(entries) > limit {
		entries = entries[:limit]
	}

	result := make([]output.PortCount, 0, len(entries))
	for _, e := range entries {
		proto, port := parsePortKey(e.key)
		result = append(result, output.PortCount{
			Port:     port,
			Protocol: proto,
			Count:    e.count,
		})
	}
	return result
}

func (a *Analyzer) getTopTalkers(limit int) []output.TalkerInfo {
	type talkerEntry struct {
		ip    string
		stats *talkerStats
		total int64
	}

	entries := make([]talkerEntry, 0, len(a.talkers))
	for ip, stats := range a.talkers {
		total := stats.BytesSent + stats.BytesReceived
		entries = append(entries, talkerEntry{ip, stats, total})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].total > entries[j].total
	})

	if len(entries) > limit {
		entries = entries[:limit]
	}

	result := make([]output.TalkerInfo, 0, len(entries))
	for _, e := range entries {
		result = append(result, output.TalkerInfo{
			IP:              e.ip,
			BytesSent:       e.stats.BytesSent,
			BytesReceived:   e.stats.BytesReceived,
			PacketsSent:     e.stats.PacketsSent,
			PacketsReceived: e.stats.PacketsReceived,
		})
	}
	return result
}

func (a *Analyzer) getDNSDomains(limit int) []output.DNSDomainInfo {
	type domainEntry struct {
		domain string
		stats  *domainStats
	}

	entries := make([]domainEntry, 0, len(a.domains))
	for d, s := range a.domains {
		entries = append(entries, domainEntry{d, s})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].stats.QueryCount > entries[j].stats.QueryCount
	})

	if len(entries) > limit {
		entries = entries[:limit]
	}

	result := make([]output.DNSDomainInfo, 0, len(entries))
	for _, e := range entries {
		ips := make([]string, 0, len(e.stats.QueryingIPs))
		for ip := range e.stats.QueryingIPs {
			ips = append(ips, ip)
		}
		result = append(result, output.DNSDomainInfo{
			Domain:      e.domain,
			QueryCount:  e.stats.QueryCount,
			QueryingIPs: ips,
		})
	}
	return result
}

func (a *Analyzer) getDestinations(limit int) []output.DestinationInfo {
	type destEntry struct {
		address string
		stats   *destStats
	}

	entries := make([]destEntry, 0, len(a.destinations))
	for addr, s := range a.destinations {
		entries = append(entries, destEntry{addr, s})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].stats.BytesTotal > entries[j].stats.BytesTotal
	})

	if len(entries) > limit {
		entries = entries[:limit]
	}

	result := make([]output.DestinationInfo, 0, len(entries))
	for _, e := range entries {
		result = append(result, output.DestinationInfo{
			Address:         e.address,
			ConnectionCount: e.stats.ConnectionCount,
			BytesTotal:      e.stats.BytesTotal,
		})
	}
	return result
}

// Helper functions
func splitIP(ip string) []string {
	result := make([]string, 0, 4)
	current := ""
	for _, c := range ip {
		if c == '.' {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
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

func parsePortKey(key string) (string, int) {
	proto := ""
	port := 0
	inPort := false
	for _, c := range key {
		if c == ':' {
			inPort = true
			continue
		}
		if inPort {
			port = port*10 + int(c-'0')
		} else {
			proto += string(c)
		}
	}
	return proto, port
}
