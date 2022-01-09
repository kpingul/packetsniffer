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
	"github.com/asdine/storm/v3"
	"github.com/jasonlvhit/gocron"

)

//configurations for capture
var (
	timestampLayout = "01-02-2006"
	hardCodedVMNIC string = "{C602633B-AFB8-4C40-B09A-658A8BC3FA45}"
    	snapshot_len int32  = 1024
    	snapshot_lenPCAPFile uint32  = 1024
    	promiscuous  bool   = true
    	err          error
    	handle       *pcap.Handle
    	ethLayer layers.Ethernet
    	ipLayer  layers.IPv4
    	tcpLayer layers.TCP
    	udpLayer layers.UDP
    	dnsLayer layers.DNS
)

type IPRecord struct {
  	ID string `storm:"id"`// primary key
  	Protocol string 
  	SrcIP string 
  	DstIP string 
}
type DNSRecord struct {
	ID string `storm:"id"`// primary key
  	Domain string 
  	SrcIP string 
  	DstIP string 

}

func main() {	
	snifferDB, errDB := storm.Open("sniffer.db")
	if errDB != nil {
		log.Fatal(errDB) 
	}

	defer snifferDB.Close()

	//Initial CLI App Setup
	app := &cli.App{
		Name:        "Packet Sniffer",
		Version:     "0.1.0",
		Description: "PCAP file analysis & sniffer based on protocol and port number",
		Authors: []*cli.Author{
			{Name: "KP",},
		},
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "filepath", Value: "", Usage: "Choose a pcap file to analyze", Required: false,},
			&cli.StringFlag{Name: "protocol", Value: "", Usage: "TCP/UDP", Required: false,},
			&cli.StringFlag{Name: "port", Value: "", Usage: "Choose port between 1-65535", Required: false,},
			&cli.StringFlag{Name: "time", Value: "30", Usage: "Determine how long you want it to run in seconds (min is 30 seconds)", Required: false,},
		},
		Action: func(c *cli.Context) error {

			//flag to check if everything checks out
			valChecks := true

		    	//input validation checks
		    	if (c.String("filepath") == "" ) {
		    		if c.String("protocol") == "" {
		    			fmt.Println("Invalid protocol")
		    			valChecks = false 
		    		}
		    		if c.String("port") == "" {
		    			fmt.Println("Invalid port")
		    			valChecks = false 
		    		}
		    		if c.Int64("time") < 30 {
		    			fmt.Println("Invalid port")
		    			valChecks = false 
		    		}
			    	if strings.ToLower(c.String("protocol")) != "tcp" && strings.ToLower(c.String("protocol")) != "udp" {
			     		fmt.Println("Invalid protocol")
			     		valChecks = false
			     	}

			     	if c.Int64("port") <= 0 || c.Int64("port") > 65535 {
			     		fmt.Println("Invalid port")
			     		valChecks = false
			     	} 
			} else {
				valChecks = false
			     	fmt.Println("ANALYZE FILE..")

			}


		     	// runif input checks out 
		     	if valChecks {

		     		//setup scheduler
		     		gocron.Every(c.Uint64("time")).Second().Do(stopSniffer)

		     		//start scheduler
		     		gocron.Start()

		     		//run sniffer
		     		runSniffer(snifferDB, c.String("protocol"), c.Int64("port"))
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

func runSniffer(snifferDB *storm.DB, protocol string, port int64) {
	fmt.Println("running sniffer...")
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
	handle, err = pcap.OpenLive(hardCodedVMNIC, snapshot_len, promiscuous, pcap.BlockForever)

	if err != nil {
		log.Fatal(err) 
	}

	defer handle.Close()

	// Create filter by combining protocol and port
	var filter string = strings.ToLower(protocol) + " and port " + strconv.FormatInt(port, 10)
	fmt.Println(filter)

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
	            	&udpLayer,
	        	&dnsLayer,
	        )

	        foundLayerTypes := []gopacket.LayerType{}
	        err := parser.DecodeLayers(packet.Data(), &foundLayerTypes)
	        if err != nil {
	        	//fmt.Println("Trouble decoding layers: ", err)
	        }

	        for _, layerType := range foundLayerTypes {

	        	//extract ipv4 data
	        	if layerType == layers.LayerTypeIPv4 {
	                	
	                	fmt.Println(ipLayer.DstIP.String())

	                	//create ip record
	                	record := CreateIPRecord(ipLayer)

	                	//store in db
				errSave := snifferDB.Save(&record)
				if errSave != nil {
					log.Fatal(errSave)
				}

	            	}
	            	
	            	//extract dns data
	            	if layerType == layers.LayerTypeDNS {
	            		dnsResponseCode := int(dnsLayer.ResponseCode)
				dnsANCount := int(dnsLayer.ANCount)

				//check if there is a dns response 
				if (dnsANCount == 0 && dnsResponseCode > 0) || (dnsANCount > 0) {

					for _, dnsQuestion := range dnsLayer.Questions {

						//domain name --
						fmt.Println("DOMAIN NAME - " + string(dnsQuestion.Name))
						
						//record type
						fmt.Println("RECORD TYPE - " + dnsQuestion.Type.String())

						//extract answers
						if dnsANCount > 0 {

							for _, dnsAnswer := range dnsLayer.Answers {
								if dnsAnswer.IP.String() != "<nil>" {
									fmt.Println("DNS Answer: ", dnsAnswer.IP.String())
								}
							}

						}

					}
				}
	            	}
	        }	
	
	}

}


/* Scheduling */
func stopSniffer() {
	fmt.Println("Stopping sniffer..")
	handle.Close()
}


/* Utility */
func CreateIPRecord (ipLayer layers.IPv4) IPRecord{
	return IPRecord{
		ID: ipLayer.SrcIP.String(),
		Protocol: ipLayer.Protocol.String(),
		SrcIP: ipLayer.SrcIP.String(),
		DstIP: ipLayer.DstIP.String(),
	}
}
func CreateDNSRecord (dnsLayer layers.DNS) DNSRecord{
	return DNSRecord{
		ID: ipLayer.SrcIP.String(),
		Domain: ipLayer.Protocol.String(),
		SrcIP: ipLayer.SrcIP.String(),
		DstIP: ipLayer.DstIP.String(),
	}
}