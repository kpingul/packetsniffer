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
    promiscuous  bool   = true
    err          error
    timeout      time.Duration = 1 * time.Second
    handle       *pcap.Handle
)


func main() {

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
        go func(device pcap.Interface){

	        // Open device
		    handle, err = pcap.OpenLive(device.Name, snapshot_len, promiscuous, timeout)
		  
		    if err != nil {
		    	log.Fatal(err) 
		    }
		    
		    defer handle.Close()

		    // Use the handle as a packet source to process all packets
		    packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
		    for packet := range packetSource.Packets() {
		        // Process packet here
		        fmt.Println(packet)
		    }
        }(device)

    }

    time.Sleep(60 * time.Second)

}