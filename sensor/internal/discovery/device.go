package discovery

import (
	"sync"
	"time"

	"github.com/asset_discovery/sensor/internal/output"
)

// Device represents a discovered network device
type Device struct {
	MAC             string
	IPs             map[string]bool // Set of IP addresses
	Vendor          string
	Hostname        string
	OSGuess         string
	Confidence      float64
	SignalsUsed     []output.Signal
	DiscoverySource string // "passive", "active-arp", "active-mdns", etc.
	FirstSeen       time.Time
	LastSeen        time.Time
}

// NewDevice creates a new device with initial values
func NewDevice(mac string) *Device {
	now := time.Now()
	return &Device{
		MAC:             mac,
		IPs:             make(map[string]bool),
		DiscoverySource: "passive",
		FirstSeen:       now,
		LastSeen:        now,
	}
}

// AddIP adds an IP address to the device
func (d *Device) AddIP(ip string) {
	d.IPs[ip] = true
	d.LastSeen = time.Now()
}

// GetIPs returns a slice of all IP addresses
func (d *Device) GetIPs() []string {
	result := make([]string, 0, len(d.IPs))
	for ip := range d.IPs {
		result = append(result, ip)
	}
	return result
}

// ToInfo converts the device to output format
func (d *Device) ToInfo() output.DeviceInfo {
	signals := make([]string, 0, len(d.SignalsUsed))
	for _, s := range d.SignalsUsed {
		signals = append(signals, s.Type+":"+s.Detail)
	}

	return output.DeviceInfo{
		MAC:             d.MAC,
		IPs:             d.GetIPs(),
		Vendor:          d.Vendor,
		Hostname:        d.Hostname,
		OSGuess:         d.OSGuess,
		Confidence:      d.Confidence,
		SignalsUsed:     signals,
		DiscoverySource: d.DiscoverySource,
		FirstSeen:       d.FirstSeen,
		LastSeen:        d.LastSeen,
	}
}

// DeviceRegistry tracks all discovered devices
type DeviceRegistry struct {
	mu      sync.RWMutex
	devices map[string]*Device // keyed by MAC
}

// NewDeviceRegistry creates a new device registry
func NewDeviceRegistry() *DeviceRegistry {
	return &DeviceRegistry{
		devices: make(map[string]*Device),
	}
}

// GetOrCreate returns an existing device or creates a new one
func (r *DeviceRegistry) GetOrCreate(mac string) *Device {
	r.mu.Lock()
	defer r.mu.Unlock()

	if device, ok := r.devices[mac]; ok {
		device.LastSeen = time.Now()
		return device
	}

	device := NewDevice(mac)
	r.devices[mac] = device
	return device
}

// Get returns a device by MAC address, or nil if not found
func (r *DeviceRegistry) Get(mac string) *Device {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.devices[mac]
}

// Update updates a device's properties
func (r *DeviceRegistry) Update(mac string, fn func(*Device)) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if device, ok := r.devices[mac]; ok {
		fn(device)
		device.LastSeen = time.Now()
	}
}

// All returns all devices as a slice
func (r *DeviceRegistry) All() []*Device {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Device, 0, len(r.devices))
	for _, d := range r.devices {
		result = append(result, d)
	}
	return result
}

// Count returns the number of devices
func (r *DeviceRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.devices)
}

// ToInfoSlice converts all devices to output format
func (r *DeviceRegistry) ToInfoSlice() []output.DeviceInfo {
	devices := r.All()
	result := make([]output.DeviceInfo, 0, len(devices))
	for _, d := range devices {
		result = append(result, d.ToInfo())
	}
	return result
}
