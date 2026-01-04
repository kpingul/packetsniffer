//go:build windows

package platform

import (
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

func init() {
	NewDetector = func() Detector {
		return &WindowsDetector{}
	}
}

// WindowsDetector handles Windows-specific detection
type WindowsDetector struct{}

func (w *WindowsDetector) GetOSInfo() OSInfo {
	version := "unknown"
	out, err := exec.Command("cmd", "/c", "ver").Output()
	if err == nil {
		version = strings.TrimSpace(string(out))
	}

	return OSInfo{
		Name:    "Windows",
		Version: version,
		Arch:    runtime.GOARCH,
	}
}

func (w *WindowsDetector) CheckPrerequisites() error {
	// Check for Npcap/WinPcap installation
	if !w.hasNpcap() {
		return errors.New("Npcap is not installed - download from https://npcap.com")
	}

	// Check for Administrator privileges
	if !w.isAdmin() {
		return errors.New("requires Administrator privileges")
	}

	return nil
}

func (w *WindowsDetector) hasNpcap() bool {
	// Check for Npcap DLL
	_, err := syscall.LoadDLL("wpcap.dll")
	return err == nil
}

func (w *WindowsDetector) isAdmin() bool {
	var sid *windows.SID
	err := windows.AllocateAndInitializeSid(
		&windows.SECURITY_NT_AUTHORITY,
		2,
		windows.SECURITY_BUILTIN_DOMAIN_RID,
		windows.DOMAIN_ALIAS_RID_ADMINS,
		0, 0, 0, 0, 0, 0,
		&sid)
	if err != nil {
		return false
	}
	defer windows.FreeSid(sid)

	token := windows.Token(0)
	member, err := token.IsMember(sid)
	if err != nil {
		return false
	}
	return member
}

func (w *WindowsDetector) GetGuidance() string {
	return `Windows Prerequisites:
  1. Install Npcap from https://npcap.com
     - During installation, enable "WinPcap API-compatible Mode"
  2. Run Command Prompt or PowerShell as Administrator
  3. Run: sensor.exe

Note: Windows requires Npcap for raw packet capture and
Administrator privileges to access network interfaces.`
}
