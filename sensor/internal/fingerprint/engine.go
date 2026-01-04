package fingerprint

import (
	"sync"

	"github.com/asset_discovery/sensor/internal/capture"
	"github.com/asset_discovery/sensor/internal/discovery"
	"github.com/asset_discovery/sensor/internal/output"
	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
)

// Engine coordinates OS fingerprinting from various signals
type Engine struct {
	registry *discovery.DeviceRegistry
	signals  map[string][]output.Signal // keyed by MAC
	mu       sync.RWMutex
}

// NewEngine creates a new fingerprinting engine
func NewEngine(registry *discovery.DeviceRegistry) *Engine {
	return &Engine{
		registry: registry,
		signals:  make(map[string][]output.Signal),
	}
}

// ProcessPacket analyzes a packet for OS fingerprinting signals
func (e *Engine) ProcessPacket(packet gopacket.Packet) {
	srcMAC, _ := capture.ExtractMACs(packet)
	if srcMAC == "" {
		return
	}

	// Check for mDNS (Apple/Linux indicator)
	if signal := e.checkMDNS(packet); signal != nil {
		e.addSignal(srcMAC, *signal)
	}

	// Check for LLMNR (Windows indicator)
	if signal := e.checkLLMNR(packet); signal != nil {
		e.addSignal(srcMAC, *signal)
	}

	// Check for NBNS (Windows indicator)
	if signal := e.checkNBNS(packet); signal != nil {
		e.addSignal(srcMAC, *signal)
	}

	// Check TTL for hints
	if signal := e.checkTTL(packet); signal != nil {
		e.addSignal(srcMAC, *signal)
	}
}

// addSignal adds a fingerprinting signal for a device
func (e *Engine) addSignal(mac string, signal output.Signal) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Avoid duplicate signals
	for _, existing := range e.signals[mac] {
		if existing.Type == signal.Type && existing.Detail == signal.Detail {
			return
		}
	}

	e.signals[mac] = append(e.signals[mac], signal)
}

// checkMDNS detects mDNS traffic (typically Apple devices)
func (e *Engine) checkMDNS(packet gopacket.Packet) *output.Signal {
	_, dstPort, proto := capture.ExtractPorts(packet)
	if proto != "UDP" || dstPort != 5353 {
		return nil
	}

	dnsLayer := packet.Layer(layers.LayerTypeDNS)
	if dnsLayer == nil {
		return nil
	}

	dns := dnsLayer.(*layers.DNS)

	// Check for Apple-specific service types
	for _, q := range dns.Questions {
		name := string(q.Name)

		// iOS/Apple specific services
		if contains(name, "_apple-mobdev2._tcp") {
			return &output.Signal{
				Type:   "mDNS",
				Detail: "_apple-mobdev2._tcp",
				Weight: 0.9,
				OS:     "iOS",
			}
		}
		if contains(name, "_airplay._tcp") {
			return &output.Signal{
				Type:   "mDNS",
				Detail: "_airplay._tcp",
				Weight: 0.85,
				OS:     "macOS",
			}
		}
		if contains(name, "_companion-link._tcp") {
			return &output.Signal{
				Type:   "mDNS",
				Detail: "_companion-link._tcp",
				Weight: 0.85,
				OS:     "iOS",
			}
		}
		if contains(name, "_homekit._tcp") {
			return &output.Signal{
				Type:   "mDNS",
				Detail: "_homekit._tcp",
				Weight: 0.8,
				OS:     "macOS",
			}
		}
		if contains(name, "_rdlink._tcp") {
			return &output.Signal{
				Type:   "mDNS",
				Detail: "_rdlink._tcp",
				Weight: 0.85,
				OS:     "macOS",
			}
		}
		if contains(name, "_smb._tcp") {
			// SMB can be any OS, but common on Windows/macOS
			return &output.Signal{
				Type:   "mDNS",
				Detail: "_smb._tcp",
				Weight: 0.3,
				OS:     "Unknown",
			}
		}
	}

	for _, a := range dns.Answers {
		name := string(a.Name)
		if contains(name, "_apple") {
			return &output.Signal{
				Type:   "mDNS",
				Detail: "apple-service",
				Weight: 0.7,
				OS:     "macOS",
			}
		}
	}

	// Generic mDNS - often Apple but not definitive
	return &output.Signal{
		Type:   "mDNS",
		Detail: "generic",
		Weight: 0.5,
		OS:     "macOS",
	}
}

// checkLLMNR detects LLMNR traffic (Windows)
func (e *Engine) checkLLMNR(packet gopacket.Packet) *output.Signal {
	_, dstPort, proto := capture.ExtractPorts(packet)
	if proto != "UDP" || dstPort != 5355 {
		return nil
	}

	// LLMNR is Windows-specific
	return &output.Signal{
		Type:   "LLMNR",
		Detail: "query",
		Weight: 0.8,
		OS:     "Windows",
	}
}

// checkNBNS detects NetBIOS Name Service traffic (Windows)
func (e *Engine) checkNBNS(packet gopacket.Packet) *output.Signal {
	_, dstPort, proto := capture.ExtractPorts(packet)
	if proto != "UDP" || (dstPort != 137 && dstPort != 138) {
		return nil
	}

	// NBNS is Windows-specific (or Samba)
	return &output.Signal{
		Type:   "NBNS",
		Detail: "query",
		Weight: 0.75,
		OS:     "Windows",
	}
}

// checkTTL uses initial TTL values as hints
func (e *Engine) checkTTL(packet gopacket.Packet) *output.Signal {
	ttl := capture.GetTTL(packet)
	if ttl == 0 {
		return nil
	}

	// Common initial TTL values:
	// Windows: 128
	// Linux/macOS: 64
	// Some routers: 255

	// Only use TTL if it's close to a known initial value
	if ttl >= 125 && ttl <= 128 {
		return &output.Signal{
			Type:   "TTL",
			Detail: "128",
			Weight: 0.3,
			OS:     "Windows",
		}
	}
	if ttl >= 61 && ttl <= 64 {
		return &output.Signal{
			Type:   "TTL",
			Detail: "64",
			Weight: 0.3,
			OS:     "Linux",
		}
	}

	return nil
}

// ApplyFingerprints applies accumulated signals to devices
func (e *Engine) ApplyFingerprints() {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for mac, signals := range e.signals {
		device := e.registry.Get(mac)
		if device == nil {
			continue
		}

		guess := e.calculateGuess(signals)
		device.OSGuess = guess.OS
		device.Confidence = guess.Confidence
		device.SignalsUsed = signals
	}
}

// OSGuess represents an OS determination with confidence
type OSGuess struct {
	OS         string
	Confidence float64
}

// calculateGuess determines the most likely OS from signals
func (e *Engine) calculateGuess(signals []output.Signal) OSGuess {
	if len(signals) == 0 {
		return OSGuess{OS: "Unknown", Confidence: 0}
	}

	// Aggregate weights by OS
	scores := make(map[string]float64)
	for _, sig := range signals {
		if sig.OS != "" && sig.OS != "Unknown" {
			scores[sig.OS] += sig.Weight
		}
	}

	if len(scores) == 0 {
		return OSGuess{OS: "Unknown", Confidence: 0}
	}

	// Find winner
	var bestOS string
	var bestScore float64
	var total float64

	for os, score := range scores {
		total += score
		if score > bestScore {
			bestScore = score
			bestOS = os
		}
	}

	// Confidence is winner's share of total
	confidence := bestScore / total
	if confidence > 0.95 {
		confidence = 0.95 // Cap at 95%
	}

	return OSGuess{
		OS:         bestOS,
		Confidence: confidence,
	}
}

// GetSignals returns all signals for a MAC address
func (e *Engine) GetSignals(mac string) []output.Signal {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.signals[mac]
}

// Helper function
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
