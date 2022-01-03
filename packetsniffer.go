package main	

import(
	"fmt"
	"log"
	"time"
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
)

//configurations for capture
var (
    snapshot_len int32  = 1024
    promiscuous  bool   = false
    err          error
    timeout      time.Duration = 30 * time.Second
    handle       *pcap.Handle
)


func main() {

    time.Sleep(60 * time.Second)

}

func runSniffer() {

	// Find all devices
    devices, err := pcap.FindAllDevs()
    if err != nil {
        log.Fatal(err)
    }

    // Print device information
    fmt.Println("Devices found:")
    for _, device := range devices {
        
    	//creating a go routine to capture 
    	//traffic on all available NICs 
        //go func(device pcap.Interface){

        	fmt.Println("\nName: ", device.Name)
        	fmt.Println("Devices addresses: ", device.Description)
 			for _, address := range device.Addresses {
	            fmt.Println("- IP address: ", address.IP)
	            fmt.Println("- Subnet mask: ", address.Netmask)
	        }

	        // Open device
		    handle, err = pcap.OpenLive("{C602633B-AFB8-4C40-B09A-658A8BC3FA45}", snapshot_len, promiscuous, timeout)
		  
		    if err != nil {
		    	log.Fatal(err) 
		    }
		    
		    defer handle.Close()

		    // Set filter
		    /*
		    var filter string = "port 53"
		    err = handle.SetBPFFilter(filter)
		    if err != nil {
		        log.Fatal(err)
		    }
		    */

		    // Use the handle as a packet source to process all packets
		    packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
		    for packet := range packetSource.Packets() {
		        // Process packet here
		        fmt.Println(packet)
		    }
        //}(device)

    }

}