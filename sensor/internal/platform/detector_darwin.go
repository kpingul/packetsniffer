//go:build darwin

package platform

import (
	"errors"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func init() {
	NewDetector = func() Detector {
		return &DarwinDetector{}
	}
}

// DarwinDetector handles macOS-specific detection
type DarwinDetector struct{}

func (d *DarwinDetector) GetOSInfo() OSInfo {
	version := "unknown"
	out, err := exec.Command("sw_vers", "-productVersion").Output()
	if err == nil {
		version = strings.TrimSpace(string(out))
	}

	return OSInfo{
		Name:    "macOS",
		Version: version,
		Arch:    runtime.GOARCH,
	}
}

func (d *DarwinDetector) CheckPrerequisites() error {
	// Check if running with sufficient privileges
	if os.Geteuid() != 0 {
		return errors.New("packet capture requires root privileges - run with sudo")
	}
	return nil
}

func (d *DarwinDetector) GetGuidance() string {
	return `macOS Prerequisites:
  1. Run with sudo: sudo ./sensor
  2. Or run the binary as root

Note: On macOS, raw packet capture requires elevated privileges.
The sensor needs access to the network interface in promiscuous mode.`
}
