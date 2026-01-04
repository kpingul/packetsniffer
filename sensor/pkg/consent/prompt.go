package consent

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const consentBanner = `
================================================================================
                    LOCAL NETWORK VISIBILITY SENSOR
================================================================================

This tool captures network traffic on your local network to discover
devices and analyze traffic patterns.

WHAT THIS TOOL DOES:
  - Captures packets on your selected network interface
  - Identifies devices by MAC address and IP
  - Determines device vendors (via OUI lookup) and operating systems
  - Analyzes traffic protocols, ports, and DNS queries
  - Generates summary reports (no payload storage by default)

REQUIREMENTS:
  - Administrator/root privileges are required for packet capture
  - On Windows: Npcap must be installed (https://npcap.com)
  - On macOS/Linux: Run with sudo

PRIVACY & LEGAL NOTICE:
  - Only use this tool on networks you own or have explicit authorization
    to monitor. Unauthorized network monitoring may violate laws.
  - By default, this tool only collects metadata (no packet payloads).
  - OS detection uses best-effort heuristics with confidence scores.

================================================================================
`

// CheckAndPromptConsent checks if user has previously consented,
// and if not, prompts for consent. Returns error if user declines.
func CheckAndPromptConsent() error {
	consentPath := getConsentFilePath()

	// Check if consent file exists
	if _, err := os.Stat(consentPath); err == nil {
		return nil // Already consented
	}

	// Display banner and prompt
	fmt.Print(consentBanner)

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Do you acknowledge and wish to continue? [y/N]: ")

	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		return fmt.Errorf("user declined authorization")
	}

	// Save consent
	if err := saveConsent(consentPath); err != nil {
		// Non-fatal - we can continue even if we can't save
		fmt.Printf("Warning: could not save consent file: %v\n", err)
	}

	return nil
}

// ResetConsent removes the consent file, requiring re-acknowledgement
func ResetConsent() error {
	consentPath := getConsentFilePath()
	err := os.Remove(consentPath)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func getConsentFilePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".network-sensor-consent")
}

func saveConsent(path string) error {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	content := fmt.Sprintf("Consented at: %s\n", time.Now().Format(time.RFC3339))
	return os.WriteFile(path, []byte(content), 0644)
}
