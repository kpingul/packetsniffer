package capture

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"
)

// PacketHandler is called for each captured packet
type PacketHandler func(packet gopacket.Packet)

// Engine manages packet capture
type Engine struct {
	handle       *pcap.Handle
	ifaceName    string
	snapLen      int32
	promisc      bool
	timeout      time.Duration
	packetCount  atomic.Int64
	handlers     []PacketHandler
	handlerMutex sync.RWMutex
}

// Config holds capture configuration
type Config struct {
	InterfaceName string
	SnapLen       int32         // Snapshot length (default 1600)
	Promiscuous   bool          // Promiscuous mode (default true)
	Timeout       time.Duration // Read timeout
}

// DefaultConfig returns sensible default configuration
func DefaultConfig(ifaceName string) Config {
	return Config{
		InterfaceName: ifaceName,
		SnapLen:       1600, // Enough for most headers
		Promiscuous:   true,
		Timeout:       pcap.BlockForever,
	}
}

// NewEngine creates a new capture engine
func NewEngine(cfg Config) (*Engine, error) {
	return &Engine{
		ifaceName: cfg.InterfaceName,
		snapLen:   cfg.SnapLen,
		promisc:   cfg.Promiscuous,
		timeout:   cfg.Timeout,
		handlers:  make([]PacketHandler, 0),
	}, nil
}

// AddHandler adds a packet handler
func (e *Engine) AddHandler(h PacketHandler) {
	e.handlerMutex.Lock()
	defer e.handlerMutex.Unlock()
	e.handlers = append(e.handlers, h)
}

// Start begins packet capture for the specified duration
func (e *Engine) Start(ctx context.Context, duration time.Duration) error {
	// Open the capture handle
	handle, err := pcap.OpenLive(e.ifaceName, e.snapLen, e.promisc, e.timeout)
	if err != nil {
		return fmt.Errorf("failed to open interface %s: %w", e.ifaceName, err)
	}
	e.handle = handle
	defer func() {
		handle.Close()
		e.handle = nil
	}()

	// Create packet source
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	packetSource.DecodeOptions.Lazy = true
	packetSource.DecodeOptions.NoCopy = true

	// Create timeout context
	captureCtx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()

	// Capture loop
	for {
		select {
		case <-captureCtx.Done():
			return nil
		case packet, ok := <-packetSource.Packets():
			if !ok {
				return nil
			}
			e.packetCount.Add(1)
			e.dispatchPacket(packet)
		}
	}
}

// dispatchPacket sends the packet to all handlers
func (e *Engine) dispatchPacket(packet gopacket.Packet) {
	e.handlerMutex.RLock()
	defer e.handlerMutex.RUnlock()
	for _, h := range e.handlers {
		h(packet)
	}
}

// PacketCount returns the number of packets captured
func (e *Engine) PacketCount() int64 {
	return e.packetCount.Load()
}

// ExtractMACs extracts source and destination MAC addresses from a packet
func ExtractMACs(packet gopacket.Packet) (srcMAC, dstMAC string) {
	if ethLayer := packet.Layer(layers.LayerTypeEthernet); ethLayer != nil {
		eth := ethLayer.(*layers.Ethernet)
		return eth.SrcMAC.String(), eth.DstMAC.String()
	}
	return "", ""
}

// ExtractIPs extracts source and destination IP addresses from a packet
func ExtractIPs(packet gopacket.Packet) (srcIP, dstIP string) {
	if ipv4Layer := packet.Layer(layers.LayerTypeIPv4); ipv4Layer != nil {
		ip := ipv4Layer.(*layers.IPv4)
		return ip.SrcIP.String(), ip.DstIP.String()
	}
	if ipv6Layer := packet.Layer(layers.LayerTypeIPv6); ipv6Layer != nil {
		ip := ipv6Layer.(*layers.IPv6)
		return ip.SrcIP.String(), ip.DstIP.String()
	}
	return "", ""
}

// ExtractPorts extracts source and destination ports from TCP/UDP packets
func ExtractPorts(packet gopacket.Packet) (srcPort, dstPort uint16, protocol string) {
	if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
		tcp := tcpLayer.(*layers.TCP)
		return uint16(tcp.SrcPort), uint16(tcp.DstPort), "TCP"
	}
	if udpLayer := packet.Layer(layers.LayerTypeUDP); udpLayer != nil {
		udp := udpLayer.(*layers.UDP)
		return uint16(udp.SrcPort), uint16(udp.DstPort), "UDP"
	}
	return 0, 0, ""
}

// GetProtocol returns the transport/network protocol name
func GetProtocol(packet gopacket.Packet) string {
	if packet.Layer(layers.LayerTypeTCP) != nil {
		return "TCP"
	}
	if packet.Layer(layers.LayerTypeUDP) != nil {
		return "UDP"
	}
	if packet.Layer(layers.LayerTypeICMPv4) != nil {
		return "ICMP"
	}
	if packet.Layer(layers.LayerTypeICMPv6) != nil {
		return "ICMPv6"
	}
	if packet.Layer(layers.LayerTypeARP) != nil {
		return "ARP"
	}
	if packet.Layer(layers.LayerTypeIPv6) != nil {
		return "IPv6"
	}
	if packet.Layer(layers.LayerTypeIPv4) != nil {
		return "IPv4"
	}
	return "Other"
}

// GetPacketSize returns the total packet size in bytes
func GetPacketSize(packet gopacket.Packet) int {
	return len(packet.Data())
}

// GetTTL extracts the TTL from an IP packet
func GetTTL(packet gopacket.Packet) int {
	if ipv4Layer := packet.Layer(layers.LayerTypeIPv4); ipv4Layer != nil {
		ip := ipv4Layer.(*layers.IPv4)
		return int(ip.TTL)
	}
	if ipv6Layer := packet.Layer(layers.LayerTypeIPv6); ipv6Layer != nil {
		ip := ipv6Layer.(*layers.IPv6)
		return int(ip.HopLimit)
	}
	return 0
}
