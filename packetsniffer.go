package main	

import(
	"fmt"
	"log"
	"time"
	"os"
	"strconv"
	"strings"
	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/urfave/cli/v2"
)

//configurations for capture
var (
	hardCodedVMNIC string = "{C602633B-AFB8-4C40-B09A-658A8BC3FA45}"
    snapshot_len int32  = 1024
    promiscuous  bool   = false
    err          error
    timeout      time.Duration = 30 * time.Second
    handle       *pcap.Handle
)

func main() {

	//Initial CLI App Setup
	app := &cli.App{
		Name:        "Packet Sniffer",
		Version:     "0.1.0",
		Description: "Sniffs traffic based on protocol and port number",
		Authors: []*cli.Author{
			{Name: "KP",},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "protocol", Value: "TCP", Usage: "TCP/UDP", Required: true,},
			&cli.StringFlag{Name: "port", Value: "80", Usage: "Choose port between 1-65535", Required: true,},
		},
		Action: func(c *cli.Context) error {

			//flag to check if everything checks out
			valChecks := true
	
	     	//input validation checks
	     	if strings.ToLower(c.String("protocol")) == "tcp" && strings.ToLower(c.String("protocol")) == "udp" {
	     		fmt.Println("Invalid protocol")
	     		valChecks = false
	     	}

	     	if c.Int64("port") <= 0 || c.Int64("port") > 65535 {
	     		fmt.Println("Invalid port")
	     		valChecks = false
	     	} 


	     	//runif input checks out 
	     	if valChecks {
	     		runSniffer(c.String("protocol"), c.Int64("port"))
	     	} else {
	     		fmt.Println("stop program..")
	     		return nil
	     	}

	     	return nil
	    },
	}

	//Run CLI
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}

func runSniffer(protocol string, port int64) {

    // Open device
    handle, err = pcap.OpenLive(hardCodedVMNIC, snapshot_len, promiscuous, timeout)
  
    if err != nil {
    	log.Fatal(err) 
    }
    
    defer handle.Close()

    // Create filter by combining protocol and port
    var filter string = strings.ToLower(protocol) + " and port " + strconv.FormatInt(port, 10)
    
    // Set filter
    err = handle.SetBPFFilter(filter)
    if err != nil {
        log.Fatal(err)
    }

    // Output current capturing config
    fmt.Println("Sniffing traffic on port " + strconv.FormatInt(port, 10) + " via " + protocol)

    // Use the handle as a packet source to process all packets
    packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
    for packet := range packetSource.Packets() {
        // Process packet here
        fmt.Println(packet)
    }

}