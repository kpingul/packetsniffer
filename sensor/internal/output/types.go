package output

import "time"

// Summary is the main output structure written as JSON
type Summary struct {
	Sensor  SensorInfo    `json:"sensor"`
	Capture CaptureInfo   `json:"capture"`
	Devices []DeviceInfo  `json:"devices"`
	Traffic TrafficInfo   `json:"traffic"`
}

// SensorInfo contains information about the sensor machine
type SensorInfo struct {
	OS        string `json:"os"`
	Hostname  string `json:"hostname"`
	Interface string `json:"interface"`
	LocalIP   string `json:"localIP"`
}

// CaptureInfo contains capture session metadata
type CaptureInfo struct {
	StartTime   time.Time `json:"startTime"`
	Duration    int       `json:"duration"` // seconds
	PacketCount int64     `json:"packetCount"`
}

// DeviceInfo contains information about a discovered device
type DeviceInfo struct {
	MAC             string   `json:"mac"`
	IPs             []string `json:"ips"`
	Vendor          string   `json:"vendor,omitempty"`
	Hostname        string   `json:"hostname,omitempty"`
	OSGuess         string   `json:"osGuess,omitempty"`
	Confidence      float64  `json:"confidence,omitempty"`
	SignalsUsed     []string `json:"signalsUsed,omitempty"`
	DiscoverySource string   `json:"discoverySource"` // "passive", "active-arp", etc.
	FirstSeen       time.Time `json:"firstSeen"`
	LastSeen        time.Time `json:"lastSeen"`
}

// TrafficInfo contains aggregated traffic statistics
type TrafficInfo struct {
	ProtocolCounts map[string]int64  `json:"protocolCounts"`
	TopPorts       []PortCount       `json:"topPorts"`
	TopTalkers     []TalkerInfo      `json:"topTalkers"`
	DNSDomains     []DNSDomainInfo   `json:"dnsDomains"`
	Destinations   []DestinationInfo `json:"destinations"`
}

// PortCount represents a port usage count
type PortCount struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"` // "TCP" or "UDP"
	Count    int64  `json:"count"`
}

// TalkerInfo represents a top talker by traffic volume
type TalkerInfo struct {
	IP              string `json:"ip"`
	BytesSent       int64  `json:"bytesSent"`
	BytesReceived   int64  `json:"bytesReceived"`
	PacketsSent     int64  `json:"packetsSent"`
	PacketsReceived int64  `json:"packetsReceived"`
}

// DNSDomainInfo represents a queried DNS domain
type DNSDomainInfo struct {
	Domain      string   `json:"domain"`
	QueryCount  int64    `json:"queryCount"`
	QueryingIPs []string `json:"queryingIPs,omitempty"`
}

// DestinationInfo represents an external destination
type DestinationInfo struct {
	Address         string `json:"address"` // IP or domain
	ConnectionCount int64  `json:"connectionCount"`
	BytesTotal      int64  `json:"bytesTotal"`
}

// Signal represents an OS fingerprinting signal
type Signal struct {
	Type   string  `json:"type"`   // "mDNS", "LLMNR", "NBNS", "DHCP", "TTL"
	Detail string  `json:"detail"` // Specific observation
	Weight float64 `json:"weight"`
	OS     string  `json:"os"` // Implied OS
}
