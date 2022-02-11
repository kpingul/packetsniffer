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
    	"net/http"
  	"encoding/json"
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
	hardCodedVMNIC string = `\\Device\\NPF_{C602633B-AFB8-4C40-B09A-658A8BC3FA45}`
    	snapshot_len int32  = 1024
    	snapshot_lenPCAPFile uint32  = 1024
    	promiscuous  bool   = true
    	err          error
    	snifferDB *storm.DB
    	errDB error
    	handle       *pcap.Handle
    	ethLayer layers.Ethernet
    	ipLayer  layers.IPv4
    	tcpLayer layers.TCP
    	udpLayer layers.UDP
    	dnsLayer layers.DNS
)

type Record struct {
  	ID  int `storm:"id,increment"` // primary key
  	Protocol string 
  	SrcIP string 
  	DstIP string 
  	Payload string
}
type DNSRecord struct {
	ID string `storm:"id,increment"`// primary key
  	Domain string 
  	SrcIP string 
  	DstIP string 

}

func main() {	
	snifferDB, errDB = storm.Open("sniffer.db")
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
			&cli.StringFlag{Name: "web", Value: "no", Usage: "Enable web server for GUI", Required: false,},
			&cli.StringFlag{Name: "filepath", Value: "", Usage: "Choose a pcap file to analyze", Required: false,},
			&cli.StringFlag{Name: "protocol", Value: "", Usage: "TCP/UDP", Required: false,},
			&cli.StringFlag{Name: "port", Value: "", Usage: "Choose port between 1-65535", Required: false,},
			&cli.StringFlag{Name: "time", Value: "30", Usage: "Determine how long you want it to run in seconds (min is 30 seconds)", Required: false,},
		},
		Action: func(c *cli.Context) error {

			//flag to check if everything checks out
			valChecks := true
			webCheck := false 

		    	//input validation checks
		    	if (c.String("web") == "yes" ) {
		    		webCheck = true
		    	} else {

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
			    			fmt.Println("Invalid time")
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
		    	}


		     	// run if input checks out 
	     		if webCheck {
	     			fmt.Println("RUNNING WEB SERVER")
			    	fileServer := http.FileServer(http.Dir("./frontend")) 
			    	http.Handle("/", fileServer) 
				http.HandleFunc("/api/records", getRecords)
				http.ListenAndServe(":8090", nil)
	     		} else {
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

	                	//create record
	                	record := CreateRecord(ipLayer)

	                	//store in db
				errSave := snifferDB.Save(&record)
				if errSave != nil {
					log.Fatal(errSave)
				}

	            	}

	            	//extract tcp data
	            	if layerType == layers.LayerTypeTCP {
		                fmt.Println("TCP Port: ", tcpLayer.SrcPort, "->", tcpLayer.DstPort)
		                fmt.Println("TCP SYN:", tcpLayer.SYN, " | ACK:", tcpLayer.ACK)

			        // Application layer contains things like HTTP
			        //FTP, SMTP, etc..
			    	applicationLayer := packet.ApplicationLayer()
			    	if applicationLayer != nil {
			        	fmt.Println("Application layer/Payload found.")
			        	fmt.Printf("%s\n", applicationLayer.Payload())

			        	// Search for a protocols inside the payload
			        	if strings.Contains(string(applicationLayer.Payload()), "HTTP") {
			            		//create record
	                			record := Record{
							Protocol: "HTTP",
							SrcIP: ipLayer.SrcIP.String(),
							DstIP: ipLayer.DstIP.String(),
							Payload: string(applicationLayer.Payload()),
						}

						//store in db
						errSave := snifferDB.Save(&record)
						if errSave != nil {
							log.Fatal(errSave)
						}
			        	}
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
func CreateRecord (ipLayer layers.IPv4) Record{
	return Record{
		Protocol: ipLayer.Protocol.String(),
		SrcIP: ipLayer.SrcIP.String(),
		DstIP: ipLayer.DstIP.String(),
	}
}
func CreateDNSRecord (dnsLayer layers.DNS) DNSRecord{
	return DNSRecord{
		Domain: ipLayer.Protocol.String(),
		SrcIP: ipLayer.SrcIP.String(),
		DstIP: ipLayer.DstIP.String(),
	}
}

func openPCAPFileAndAnalyze(fileName string) {

	// Open file instead of device
    	handle, err = pcap.OpenOffline(fileName)
    	if err != nil { 
    		log.Fatal(err) 
    	}
    	
    	defer handle.Close()

    	// Loop through packets in file
    	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
    	for packet := range packetSource.Packets() {
        	
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

	                	//create record
	                	record := CreateRecord(ipLayer)
	                	fmt.Println(record)

	            	}

	            	//extract tcp data
	            	if layerType == layers.LayerTypeTCP {
		                fmt.Println("TCP Port: ", tcpLayer.SrcPort, "->", tcpLayer.DstPort)
		                fmt.Println("TCP SYN:", tcpLayer.SYN, " | ACK:", tcpLayer.ACK)

			        // Application layer contains things like HTTP
			        //FTP, SMTP, etc..
			    	applicationLayer := packet.ApplicationLayer()
			    	if applicationLayer != nil {
			        	fmt.Println("Application layer/Payload found.")
			        	fmt.Printf("%s\n", applicationLayer.Payload())

			        	// Search for a protocols inside the payload
			        	if strings.Contains(string(applicationLayer.Payload()), "HTTP") {
			            		fmt.Println("HTTP found!")
			        	}
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


/* DB  Management */

func getRecords(w http.ResponseWriter, req *http.Request) {

	records := getAllEventRecords()

	jsonData, err := json.Marshal(records)
	if err != nil {
	    log.Fatal(err)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
   	
}


func getAllEventRecords () []Record{

	var records []Record

	errFetch := snifferDB.All(&records)
	if errFetch != nil {
		log.Fatal(errFetch)
		return records
	} else {
		return records
	}

}