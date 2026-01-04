package platform

import (
	"fmt"
	"os"
	"runtime"
)

// OSInfo contains information about the host operating system
type OSInfo struct {
	Name    string // "darwin", "windows", "linux"
	Version string
	Arch    string
}

// Detector interface for OS-specific detection
type Detector interface {
	GetOSInfo() OSInfo
	CheckPrerequisites() error
	GetGuidance() string
}

// NewDetector returns the appropriate detector for the current OS
// This function is defined in platform-specific files
var NewDetector = newGenericDetector

func newGenericDetector() Detector {
	return &GenericDetector{}
}

// GetHostname returns the system hostname
func GetHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

// GenericDetector for unsupported platforms
type GenericDetector struct{}

func (g *GenericDetector) GetOSInfo() OSInfo {
	return OSInfo{
		Name:    runtime.GOOS,
		Version: "unknown",
		Arch:    runtime.GOARCH,
	}
}

func (g *GenericDetector) CheckPrerequisites() error {
	return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
}

func (g *GenericDetector) GetGuidance() string {
	return fmt.Sprintf("Platform %s is not fully supported. Packet capture may not work.", runtime.GOOS)
}
