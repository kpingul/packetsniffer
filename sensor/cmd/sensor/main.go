package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asset_discovery/sensor/internal/capture"
	"github.com/asset_discovery/sensor/internal/discovery"
	"github.com/asset_discovery/sensor/internal/fingerprint"
	"github.com/asset_discovery/sensor/internal/iface"
	"github.com/asset_discovery/sensor/internal/oui"
	"github.com/asset_discovery/sensor/internal/output"
	"github.com/asset_discovery/sensor/internal/platform"
	"github.com/asset_discovery/sensor/internal/traffic"
	"github.com/asset_discovery/sensor/pkg/consent"
	"github.com/fatih/color"
	"github.com/gopacket/gopacket"
	"github.com/spf13/cobra"
)

var (
	// CLI flags
	listIfaces bool
	ifaceName  string
	autoIface  bool
	duration   int
	activeMode bool
	outputDir  string
	skipCheck  bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "sensor",
		Short: "Local Network Visibility Sensor",
		Long: `A network visibility tool for authorized sysadmins to discover
devices on the local network and analyze traffic patterns.

This tool captures network traffic passively (and optionally actively)
to build an inventory of devices and their behaviors.`,
		RunE: runSensor,
	}

	// Flags
	rootCmd.Flags().BoolVar(&listIfaces, "list-ifaces", false, "List available network interfaces and exit")
	rootCmd.Flags().StringVar(&ifaceName, "iface", "", "Network interface to capture on")
	rootCmd.Flags().BoolVar(&autoIface, "auto-iface", true, "Automatically select the best interface")
	rootCmd.Flags().IntVar(&duration, "duration", 30, "Capture duration in seconds (30 or 60)")
	rootCmd.Flags().BoolVar(&activeMode, "active", false, "Enable active discovery (ARP sweep)")
	rootCmd.Flags().StringVar(&outputDir, "output", ".", "Output directory for summary files")
	rootCmd.Flags().BoolVar(&skipCheck, "skip-prereq", false, "Skip prerequisite checks")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runSensor(cmd *cobra.Command, args []string) error {
	// Create interface selector
	selector := iface.NewSelector()

	// Handle --list-ifaces
	if listIfaces {
		return listInterfaces(selector)
	}

	// Check consent
	if err := consent.CheckAndPromptConsent(); err != nil {
		return err
	}

	// Detect platform
	detector := platform.NewDetector()
	osInfo := detector.GetOSInfo()

	fmt.Printf("\n%s\n", color.CyanString("Local Network Visibility Sensor"))
	fmt.Printf("OS: %s %s (%s)\n", osInfo.Name, osInfo.Version, osInfo.Arch)

	// Check prerequisites
	if !skipCheck {
		if err := detector.CheckPrerequisites(); err != nil {
			color.Red("\nPrerequisites not met: %v", err)
			fmt.Println()
			color.Yellow("Guidance:")
			fmt.Println(detector.GetGuidance())
			return err
		}
		color.Green("Prerequisites satisfied.")
	}

	// Select interface
	var selectedIface *iface.InterfaceInfo
	var err error

	if ifaceName != "" {
		selectedIface, err = selector.GetInterfaceByName(ifaceName)
		if err != nil {
			return fmt.Errorf("interface %q not found: %w", ifaceName, err)
		}
	} else if autoIface {
		selectedIface, err = selector.AutoSelect()
		if err != nil {
			return fmt.Errorf("auto-select failed: %w", err)
		}
	} else {
		return fmt.Errorf("no interface specified. Use --iface or --auto-iface")
	}

	fmt.Printf("Interface: %s\n", color.GreenString(selectedIface.Name))
	if len(selectedIface.IPs) > 0 {
		fmt.Printf("Local IP: %s\n", selectedIface.IPs[0])
	}
	fmt.Printf("Duration: %d seconds\n", duration)
	fmt.Printf("Active discovery: %v\n", activeMode)
	fmt.Printf("Output: %s\n", outputDir)
	fmt.Println()

	// Get local IP and MAC for active discovery
	localIP := getLocalIP(selectedIface)
	localMAC := getLocalMAC(selectedIface.Name)
	localSubnet := getLocalSubnet(selectedIface)

	// Initialize components
	ouiLookup := oui.NewLookup()
	deviceRegistry := discovery.NewDeviceRegistry()
	passiveDiscovery := discovery.NewPassiveDiscovery(deviceRegistry, ouiLookup)
	trafficAnalyzer := traffic.NewAnalyzer(localIP.String())
	fingerprintEngine := fingerprint.NewEngine(deviceRegistry)

	// Create capture engine
	captureConfig := capture.DefaultConfig(selectedIface.Name)
	captureEngine, err := capture.NewEngine(captureConfig)
	if err != nil {
		return fmt.Errorf("failed to create capture engine: %w", err)
	}

	// Add packet handlers
	captureEngine.AddHandler(func(packet gopacket.Packet) {
		passiveDiscovery.ProcessPacket(packet)
		trafficAnalyzer.ProcessPacket(packet)
		fingerprintEngine.ProcessPacket(packet)
	})

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nStopping capture...")
		cancel()
	}()

	// Run active discovery first if enabled
	if activeMode {
		fmt.Println(color.YellowString("Running active discovery..."))
		activeDisc := discovery.NewActiveDiscovery(
			deviceRegistry,
			ouiLookup,
			selectedIface.Name,
			localIP,
			localMAC,
			localSubnet,
		)

		activeCtx, activeCancel := context.WithTimeout(ctx, 10*time.Second)
		if err := activeDisc.Run(activeCtx); err != nil {
			color.Yellow("Active discovery warning: %v", err)
		}
		activeCancel()
		fmt.Printf("Active discovery found %d devices\n", deviceRegistry.Count())
	}

	// Start passive capture
	startTime := time.Now()
	fmt.Printf(color.YellowString("Capturing for %d seconds... (Ctrl+C to stop early)\n"), duration)

	if err := captureEngine.Start(ctx, time.Duration(duration)*time.Second); err != nil {
		return fmt.Errorf("capture failed: %w", err)
	}

	// Apply fingerprints
	fingerprintEngine.ApplyFingerprints()

	// Build summary
	summary := output.NewSummary(
		osInfo.Name,
		platform.GetHostname(),
		selectedIface.Name,
		localIP.String(),
	)

	summary.SetCaptureInfo(startTime, duration, captureEngine.PacketCount())
	summary.SetDevices(deviceRegistry.ToInfoSlice())
	summary.SetTraffic(trafficAnalyzer.GetResults())

	// Print summary
	fmt.Println(summary.PrettyPrint())

	// Write summary file
	generator := output.NewGenerator(outputDir)
	filepath, err := generator.Generate(summary)
	if err != nil {
		return fmt.Errorf("failed to write summary: %w", err)
	}

	color.Green("\nSummary written to: %s", filepath)
	return nil
}

func listInterfaces(selector *iface.Selector) error {
	ifaces, err := selector.ListInterfaces()
	if err != nil {
		return err
	}

	fmt.Print(iface.FormatInterfaceList(ifaces))
	return nil
}

func getLocalIP(ifaceInfo *iface.InterfaceInfo) net.IP {
	for _, ipStr := range ifaceInfo.IPs {
		ip := net.ParseIP(ipStr)
		if ip != nil && iface.IsRFC1918(ip) {
			return ip
		}
	}
	// Fallback to first IPv4
	for _, ipStr := range ifaceInfo.IPs {
		ip := net.ParseIP(ipStr)
		if ip != nil && ip.To4() != nil {
			return ip
		}
	}
	return nil
}

func getLocalMAC(ifaceName string) net.HardwareAddr {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	for _, i := range ifaces {
		if i.Name == ifaceName {
			return i.HardwareAddr
		}
	}
	return nil
}

func getLocalSubnet(ifaceInfo *iface.InterfaceInfo) *net.IPNet {
	for _, ipStr := range ifaceInfo.IPs {
		ip := net.ParseIP(ipStr)
		if ip == nil || !iface.IsRFC1918(ip) {
			continue
		}

		// Assume /24 for simplicity (common home/office network)
		// A more robust implementation would get the actual mask
		mask := net.CIDRMask(24, 32)
		return &net.IPNet{
			IP:   ip.Mask(mask),
			Mask: mask,
		}
	}
	return nil
}
