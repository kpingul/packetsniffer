package main	

import(
	"fmt"
	"log"
	"time"
	"os"
	"strconv"
	"strings"
	"crypto/md5"
    	"encoding/hex"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/pcapgo"
	"github.com/urfave/cli/v2"
)

//configurations for capture
var (
	timestampLayout = "01-02-2006"
	hardCodedVMNIC string = "{C602633B-AFB8-4C40-B09A-658A8BC3FA45}"
    	snapshot_len int32  = 1024
    	snapshot_lenPCAPFile uint32  = 1024
    	promiscuous  bool   = false
    	err          error
    	timeout      time.Duration = 30 * time.Second
    	handle       *pcap.Handle
    	ethLayer layers.Ethernet
    	ipLayer  layers.IPv4
    	tcpLayer layers.TCP
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

    	//unix timestamp
    	currentTimestamp := time.Now()

    	//create hash using current timestamp
	hash := md5.Sum([]byte(currentTimestamp.String()))
   	hashValue := hex.EncodeToString(hash[:])
	
	//filename based on timestamp and md5 hash
	fileName := "pcap-" + currentTimestamp.Format(timestampLayout) + "-" + hashValue + ".pcap"
		
	//create new file for pcap 
	pcapFile, _ := os.Create(fileName)
	
	// Open output pcap file and write header
	w := pcapgo.NewWriter(pcapFile)
	w.WriteFileHeader(snapshot_lenPCAPFile, layers.LinkTypeEthernet)
	defer pcapFile.Close()

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

		//write packet to pcap file
		w.WritePacket(packet.Metadata().CaptureInfo, packet.Data())


		//use existing structures to store the packet information 
		//instead of creating new structs for every packet which 
		//takes time and memory. 
		parser := gopacket.NewDecodingLayerParser(
	            layers.LayerTypeEthernet,
	            &ethLayer,
	            &ipLayer,
	            &tcpLayer,
	        )
	        foundLayerTypes := []gopacket.LayerType{}

	        err := parser.DecodeLayers(packet.Data(), &foundLayerTypes)
	        if err != nil {
	            fmt.Println("Trouble decoding layers: ", err)
	        }

	        for _, layerType := range foundLayerTypes {
	            if layerType == layers.LayerTypeIPv4 {
	                fmt.Println("IPv4: ", ipLayer.SrcIP, "->", ipLayer.DstIP)
	            }
	            // if layerType == layers.LayerTypeTCP {
	            //     fmt.Println("TCP Port: ", tcpLayer.SrcPort, "->", tcpLayer.DstPort)
	            //     fmt.Println("TCP SYN:", tcpLayer.SYN, " | ACK:", tcpLayer.ACK)
	            // }
	        }	
	
	}

}