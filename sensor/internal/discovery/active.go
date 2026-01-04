package discovery

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/asset_discovery/sensor/internal/oui"
	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/gopacket/gopacket/pcap"
)

// ActiveDiscovery performs active network scanning
type ActiveDiscovery struct {
	registry  *DeviceRegistry
	oui       *oui.Lookup
	ifaceName string
	localIP   net.IP
	localMAC  net.HardwareAddr
	subnet    *net.IPNet
}

// NewActiveDiscovery creates a new active discovery instance
func NewActiveDiscovery(registry *DeviceRegistry, ouiLookup *oui.Lookup, ifaceName string, localIP net.IP, localMAC net.HardwareAddr, subnet *net.IPNet) *ActiveDiscovery {
	return &ActiveDiscovery{
		registry:  registry,
		oui:       ouiLookup,
		ifaceName: ifaceName,
		localIP:   localIP,
		localMAC:  localMAC,
		subnet:    subnet,
	}
}

// Run performs active discovery
func (a *ActiveDiscovery) Run(ctx context.Context) error {
	if a.subnet == nil {
		return fmt.Errorf("no subnet configured for active discovery")
	}

	// ARP sweep
	if err := a.arpSweep(ctx); err != nil {
		return fmt.Errorf("ARP sweep failed: %w", err)
	}

	return nil
}

// arpSweep sends ARP requests to all IPs in the subnet
func (a *ActiveDiscovery) arpSweep(ctx context.Context) error {
	// Open handle for sending
	handle, err := pcap.OpenLive(a.ifaceName, 65535, true, pcap.BlockForever)
	if err != nil {
		return fmt.Errorf("failed to open interface: %w", err)
	}
	defer handle.Close()

	// Set BPF filter to capture ARP replies
	if err := handle.SetBPFFilter("arp"); err != nil {
		return fmt.Errorf("failed to set BPF filter: %w", err)
	}

	// Start listening for responses
	var wg sync.WaitGroup
	responseChan := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		a.listenARPReplies(ctx, handle, responseChan)
	}()

	// Calculate IP range
	ips := a.enumerateSubnet()

	// Send ARP requests
	for _, ip := range ips {
		select {
		case <-ctx.Done():
			close(responseChan)
			wg.Wait()
			return nil
		default:
		}

		if err := a.sendARPRequest(handle, ip); err != nil {
			// Log but continue
			continue
		}

		// Small delay to avoid flooding
		time.Sleep(10 * time.Millisecond)
	}

	// Wait for responses
	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
	}

	close(responseChan)
	wg.Wait()

	return nil
}

// listenARPReplies listens for ARP reply packets
func (a *ActiveDiscovery) listenARPReplies(ctx context.Context, handle *pcap.Handle, done <-chan struct{}) {
	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case packet, ok := <-packetSource.Packets():
			if !ok {
				return
			}

			arpLayer := packet.Layer(layers.LayerTypeARP)
			if arpLayer == nil {
				continue
			}

			arp := arpLayer.(*layers.ARP)
			if arp.Operation != layers.ARPReply {
				continue
			}

			// Extract sender info
			srcMAC := net.HardwareAddr(arp.SourceHwAddress).String()
			srcIP := net.IP(arp.SourceProtAddress).String()

			if srcMAC != "" && srcIP != "" {
				device := a.registry.GetOrCreate(srcMAC)
				device.AddIP(srcIP)
				device.DiscoverySource = "active-arp"
				if device.Vendor == "" {
					device.Vendor = a.oui.GetVendor(srcMAC)
				}
			}
		}
	}
}

// sendARPRequest sends an ARP request for the given IP
func (a *ActiveDiscovery) sendARPRequest(handle *pcap.Handle, targetIP net.IP) error {
	// Build Ethernet frame
	eth := layers.Ethernet{
		SrcMAC:       a.localMAC,
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}, // Broadcast
		EthernetType: layers.EthernetTypeARP,
	}

	// Build ARP request
	arp := layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     6,
		ProtAddressSize:   4,
		Operation:         layers.ARPRequest,
		SourceHwAddress:   []byte(a.localMAC),
		SourceProtAddress: []byte(a.localIP.To4()),
		DstHwAddress:      []byte{0, 0, 0, 0, 0, 0},
		DstProtAddress:    []byte(targetIP.To4()),
	}

	// Serialize
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	if err := gopacket.SerializeLayers(buf, opts, &eth, &arp); err != nil {
		return err
	}

	return handle.WritePacketData(buf.Bytes())
}

// enumerateSubnet returns all IPs in the subnet
func (a *ActiveDiscovery) enumerateSubnet() []net.IP {
	var ips []net.IP

	// Get network address
	network := a.subnet.IP.Mask(a.subnet.Mask)

	// Calculate number of hosts
	ones, bits := a.subnet.Mask.Size()
	hostBits := bits - ones

	// Limit to reasonable size (max 1024 hosts)
	maxHosts := 1 << hostBits
	if maxHosts > 1024 {
		maxHosts = 1024
	}

	// Enumerate IPs
	for i := 1; i < maxHosts-1; i++ { // Skip network and broadcast
		ip := make(net.IP, 4)
		copy(ip, network.To4())

		// Add offset
		ip[3] += byte(i & 0xff)
		ip[2] += byte((i >> 8) & 0xff)
		ip[1] += byte((i >> 16) & 0xff)
		ip[0] += byte((i >> 24) & 0xff)

		// Skip our own IP
		if ip.Equal(a.localIP) {
			continue
		}

		ips = append(ips, ip)
	}

	return ips
}
