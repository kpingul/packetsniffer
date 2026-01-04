//go:build linux

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
		return &LinuxDetector{}
	}
}

// LinuxDetector handles Linux-specific detection
type LinuxDetector struct{}

func (d *LinuxDetector) GetOSInfo() OSInfo {
	version := "unknown"

	// Try to get version from /etc/os-release
	out, err := exec.Command("cat", "/etc/os-release").Output()
	if err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				version = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
				break
			}
		}
	}

	// Fallback to uname
	if version == "unknown" {
		out, err = exec.Command("uname", "-r").Output()
		if err == nil {
			version = strings.TrimSpace(string(out))
		}
	}

	return OSInfo{
		Name:    "Linux",
		Version: version,
		Arch:    runtime.GOARCH,
	}
}

func (d *LinuxDetector) CheckPrerequisites() error {
	// Check for root or CAP_NET_RAW
	if os.Geteuid() != 0 {
		// Check if we have CAP_NET_RAW capability
		// This is a simplified check - in production you'd use a capabilities library
		out, err := exec.Command("getcap", os.Args[0]).Output()
		if err != nil || !strings.Contains(string(out), "cap_net_raw") {
			return errors.New("requires root privileges or CAP_NET_RAW capability")
		}
	}
	return nil
}

func (d *LinuxDetector) GetGuidance() string {
	return `Linux Prerequisites:
  1. Run with sudo: sudo ./sensor
  2. Or grant capability: sudo setcap cap_net_raw+ep ./sensor

Note: Raw packet capture requires either root privileges or
the CAP_NET_RAW capability on the binary.`
}
