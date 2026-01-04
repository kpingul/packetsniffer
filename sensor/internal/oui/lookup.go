package oui

import (
	"bufio"
	"embed"
	"strings"
	"sync"
)

//go:embed data/oui.txt
var ouiData embed.FS

// Lookup provides MAC address vendor lookup
type Lookup struct {
	vendors map[string]string
	mu      sync.RWMutex
}

// Common OUI prefixes for when the embedded database isn't available
var commonVendors = map[string]string{
	"00:00:0C": "Cisco",
	"00:01:42": "Cisco",
	"00:03:93": "Apple",
	"00:05:02": "Apple",
	"00:0A:27": "Apple",
	"00:0A:95": "Apple",
	"00:0D:93": "Apple",
	"00:10:FA": "Apple",
	"00:11:24": "Apple",
	"00:14:51": "Apple",
	"00:16:CB": "Apple",
	"00:17:F2": "Apple",
	"00:19:E3": "Apple",
	"00:1B:63": "Apple",
	"00:1C:B3": "Apple",
	"00:1D:4F": "Apple",
	"00:1E:52": "Apple",
	"00:1E:C2": "Apple",
	"00:1F:5B": "Apple",
	"00:1F:F3": "Apple",
	"00:21:E9": "Apple",
	"00:22:41": "Apple",
	"00:23:12": "Apple",
	"00:23:32": "Apple",
	"00:23:6C": "Apple",
	"00:23:DF": "Apple",
	"00:24:36": "Apple",
	"00:25:00": "Apple",
	"00:25:4B": "Apple",
	"00:25:BC": "Apple",
	"00:26:08": "Apple",
	"00:26:4A": "Apple",
	"00:26:B0": "Apple",
	"00:26:BB": "Apple",
	"00:27:02": "Apple",
	"00:50:56": "VMware",
	"00:0C:29": "VMware",
	"00:50:C2": "VMware",
	"00:1C:14": "VMware",
	"00:05:69": "VMware",
	"08:00:27": "VirtualBox",
	"0A:00:27": "VirtualBox",
	"52:54:00": "QEMU",
	"00:16:3E": "Xen",
	"00:15:5D": "Hyper-V",
	"00:1A:11": "Google",
	"3C:5A:B4": "Google",
	"94:EB:2C": "Google",
	"F4:F5:D8": "Google",
	"00:17:88": "Philips Hue",
	"00:1D:C9": "Sonos",
	"5C:AA:FD": "Sonos",
	"78:28:CA": "Sonos",
	"94:9F:3E": "Sonos",
	"B8:E9:37": "Sonos",
	"00:04:20": "Philips",
	"00:09:B0": "Philips",
	"00:0F:E2": "Philips",
	"00:12:EE": "Philips",
	"00:17:EE": "Philips",
	"00:1E:45": "Philips",
	"00:22:A0": "Philips",
	"18:B4:30": "Nest",
	"64:16:66": "Nest",
	"00:11:22": "Dell",
	"00:12:3F": "Dell",
	"00:14:22": "Dell",
	"00:15:C5": "Dell",
	"00:18:8B": "Dell",
	"00:19:B9": "Dell",
	"00:1A:A0": "Dell",
	"00:1C:23": "Dell",
	"00:1D:09": "Dell",
	"00:1E:4F": "Dell",
	"00:1E:C9": "Dell",
	"00:21:9B": "Dell",
	"00:22:19": "Dell",
	"00:24:E8": "Dell",
	"00:25:64": "Dell",
	"00:26:B9": "Dell",
	"18:03:73": "Dell",
	"28:F1:0E": "Dell",
	"34:17:EB": "Dell",
	"00:0D:56": "Dell",
	"00:06:5B": "Dell",
	"00:08:74": "Dell",
	"F8:DB:88": "Dell",
	"00:21:5A": "HP",
	"00:22:64": "HP",
	"00:23:7D": "HP",
	"00:24:81": "HP",
	"00:25:B3": "HP",
	"00:26:55": "HP",
	"00:30:C1": "HP",
	"00:0B:CD": "HP",
	"00:0D:9D": "HP",
	"00:0E:7F": "HP",
	"00:0F:20": "HP",
	"00:10:83": "HP",
	"00:11:0A": "HP",
	"00:11:85": "HP",
	"00:12:79": "HP",
	"00:13:21": "HP",
	"00:14:38": "HP",
	"00:15:60": "HP",
	"00:16:35": "HP",
	"00:17:08": "HP",
	"00:17:A4": "HP",
	"00:18:71": "HP",
	"00:18:FE": "HP",
	"00:19:BB": "HP",
	"00:1A:4B": "HP",
	"00:1B:78": "HP",
	"00:1C:C4": "HP",
	"00:1E:0B": "HP",
	"00:1F:29": "HP",
	"00:1F:FE": "HP",
	"00:20:74": "HP",
	"00:50:8B": "HP",
	"00:60:B0": "HP",
	"00:80:A0": "HP",
	"00:A0:68": "HP",
	"08:00:09": "HP",
	"10:1F:74": "HP",
	"10:60:4B": "HP",
	"24:BE:05": "HP",
	"28:80:23": "HP",
	"30:E1:71": "HP",
	"00:0C:6E": "ASUSTek",
	"00:11:2F": "ASUSTek",
	"00:13:D4": "ASUSTek",
	"00:15:F2": "ASUSTek",
	"00:17:31": "ASUSTek",
	"00:18:F3": "ASUSTek",
	"00:1A:92": "ASUSTek",
	"00:1B:FC": "ASUSTek",
	"00:1D:60": "ASUSTek",
	"00:1E:8C": "ASUSTek",
	"00:1F:C6": "ASUSTek",
	"00:22:15": "ASUSTek",
	"00:23:54": "ASUSTek",
	"00:24:8C": "ASUSTek",
	"00:26:18": "ASUSTek",
	"04:92:26": "ASUSTek",
	"08:60:6E": "ASUSTek",
	"10:BF:48": "ASUSTek",
	"14:DA:E9": "ASUSTek",
	"1C:87:2C": "ASUSTek",
	"20:CF:30": "ASUSTek",
	"24:4B:FE": "ASUSTek",
	"2C:56:DC": "ASUSTek",
	"30:85:A9": "ASUSTek",
	"38:2C:4A": "ASUSTek",
	"38:D5:47": "ASUSTek",
	"40:16:7E": "ASUSTek",
	"48:5B:39": "ASUSTek",
	"50:46:5D": "ASUSTek",
	"54:04:A6": "ASUSTek",
	"60:45:CB": "ASUSTek",
	"74:D0:2B": "ASUSTek",
	"AC:22:0B": "ASUSTek",
	"AC:9E:17": "ASUSTek",
	"B0:6E:BF": "ASUSTek",
	"BC:EE:7B": "ASUSTek",
	"C8:60:00": "ASUSTek",
	"D8:50:E6": "ASUSTek",
	"E0:3F:49": "ASUSTek",
	"E8:9D:87": "ASUSTek",
	"F4:6D:04": "ASUSTek",
	"F8:32:E4": "ASUSTek",
	"FC:C2:DE": "Samsung",
	"00:00:F0": "Samsung",
	"00:02:78": "Samsung",
	"00:07:AB": "Samsung",
	"00:09:18": "Samsung",
	"00:0D:AE": "Samsung",
	"00:0D:E5": "Samsung",
	"00:0F:73": "Samsung",
	"00:12:47": "Samsung",
	"00:12:FB": "Samsung",
	"00:13:77": "Samsung",
	"00:15:99": "Samsung",
	"00:15:B9": "Samsung",
	"00:16:32": "Samsung",
	"00:16:6B": "Samsung",
	"00:16:6C": "Samsung",
	"00:16:DB": "Samsung",
	"00:17:C9": "Samsung",
	"00:17:D5": "Samsung",
	"00:18:AF": "Samsung",
	"00:1A:8A": "Samsung",
	"00:1B:98": "Samsung",
	"00:1C:43": "Samsung",
	"00:1D:25": "Samsung",
	"00:1D:F6": "Samsung",
	"00:1E:7D": "Samsung",
	"00:1E:E1": "Samsung",
	"00:1E:E2": "Samsung",
	"00:1F:CC": "Samsung",
	"00:1F:CD": "Samsung",
	"00:21:19": "Samsung",
	"00:21:4C": "Samsung",
	"00:21:D1": "Samsung",
	"00:21:D2": "Samsung",
	"00:23:39": "Samsung",
	"00:23:3A": "Samsung",
	"00:23:99": "Samsung",
	"00:23:C2": "Samsung",
	"00:23:D6": "Samsung",
	"00:23:D7": "Samsung",
	"00:24:54": "Samsung",
	"00:24:90": "Samsung",
	"00:24:91": "Samsung",
	"00:24:E9": "Samsung",
	"00:25:38": "Samsung",
	"00:25:66": "Samsung",
	"00:25:67": "Samsung",
	"00:26:37": "Samsung",
	"00:26:5D": "Samsung",
	"00:26:5F": "Samsung",
	"54:88:0E": "Samsung",
	"5C:0A:5B": "Samsung",
	"5C:3C:27": "Samsung",
	"60:6B:BD": "Samsung",
	"64:77:91": "Samsung",
	"6C:2F:2C": "Samsung",
	"78:25:AD": "Samsung",
	"78:52:1A": "Samsung",
	"84:55:A5": "Samsung",
	"88:32:9B": "Samsung",
	"8C:77:12": "Samsung",
	"94:35:0A": "Samsung",
	"94:51:03": "Samsung",
	"9C:02:98": "Samsung",
	"9C:3A:AF": "Samsung",
	"A0:07:98": "Samsung",
	"A0:21:95": "Samsung",
	"A4:07:B6": "Samsung",
	"A8:06:00": "Samsung",
	"AC:5F:3E": "Samsung",
	"B0:47:BF": "Samsung",
	"B0:72:BF": "Samsung",
	"B0:EC:71": "Samsung",
	"B4:3A:28": "Samsung",
	"B4:79:A7": "Samsung",
	"B8:5A:73": "Samsung",
	"B8:C6:8E": "Samsung",
	"BC:14:01": "Samsung",
	"BC:20:A4": "Samsung",
	"BC:44:86": "Samsung",
	"BC:72:B1": "Samsung",
	"BC:79:AD": "Samsung",
	"C0:BD:D1": "Samsung",
	"C4:42:02": "Samsung",
	"C4:57:6E": "Samsung",
	"C4:73:1E": "Samsung",
	"C8:14:79": "Samsung",
	"C8:19:F7": "Samsung",
	"CC:07:AB": "Samsung",
	"D0:22:BE": "Samsung",
	"D0:59:E4": "Samsung",
	"D0:66:7B": "Samsung",
	"D4:87:D8": "Samsung",
	"D4:88:90": "Samsung",
	"D8:57:EF": "Samsung",
	"D8:90:E8": "Samsung",
	"DC:71:44": "Samsung",
	"E0:99:71": "Samsung",
	"E4:12:1D": "Samsung",
	"E4:40:E2": "Samsung",
	"E4:58:B8": "Samsung",
	"E4:7C:F9": "Samsung",
	"E4:92:FB": "Samsung",
	"E8:03:9A": "Samsung",
	"E8:4E:84": "Samsung",
	"EC:1F:72": "Samsung",
	"EC:9B:F3": "Samsung",
	"F0:25:B7": "Samsung",
	"F0:5A:09": "Samsung",
	"F4:09:D8": "Samsung",
	"F4:42:8F": "Samsung",
	"F4:7B:5E": "Samsung",
	"F8:04:2E": "Samsung",
	"F8:D0:AC": "Samsung",
	"FC:A1:3E": "Samsung",
	"A4:5E:60": "Apple",
	"AC:BC:32": "Apple",
	"B8:C7:5D": "Apple",
	"D0:03:4B": "Apple",
	"E0:B5:2D": "Apple",
	"F0:99:BF": "Apple",
	"F4:F1:5A": "Apple",
	"AC:E4:B5": "Apple",
}

// NewLookup creates a new OUI lookup instance
func NewLookup() *Lookup {
	l := &Lookup{
		vendors: make(map[string]string),
	}

	// Try to load embedded OUI database
	if err := l.loadEmbeddedDB(); err != nil {
		// Fall back to common vendors
		for k, v := range commonVendors {
			l.vendors[strings.ToUpper(k)] = v
		}
	}

	return l
}

// loadEmbeddedDB attempts to load the embedded OUI database
func (l *Lookup) loadEmbeddedDB() error {
	file, err := ouiData.Open("data/oui.txt")
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		// Parse OUI format: "XX:XX:XX\tVendor Name" or "XX-XX-XX\tVendor Name"
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		prefix := strings.ToUpper(strings.ReplaceAll(parts[0], "-", ":"))
		l.vendors[prefix] = parts[1]
	}

	return scanner.Err()
}

// GetVendor looks up the vendor for a MAC address
func (l *Lookup) GetVendor(mac string) string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Normalize MAC format
	mac = strings.ToUpper(mac)
	mac = strings.ReplaceAll(mac, "-", ":")

	// Try OUI prefix (first 3 octets)
	if len(mac) >= 8 {
		prefix := mac[:8]
		if vendor, ok := l.vendors[prefix]; ok {
			return vendor
		}
	}

	return "Unknown"
}

// IsVirtualVendor checks if the vendor indicates a virtual machine
func (l *Lookup) IsVirtualVendor(vendor string) bool {
	vendor = strings.ToLower(vendor)
	virtualKeywords := []string{"vmware", "virtualbox", "qemu", "xen", "hyper-v", "virtual"}
	for _, kw := range virtualKeywords {
		if strings.Contains(vendor, kw) {
			return true
		}
	}
	return false
}
