package main

import (
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
    snapshot_len int32 = 1024 snapshot_lenPCAPFile uint32 = 1024 promiscuous bool = true err error snifferDB * storm.DB pcapDB * storm.DB errDB error handle * pcap.Handle ethLayer layers.Ethernet ipLayer layers.IPv4 tcpLayer layers.TCP udpLayer layers.UDP dnsLayer layers.DNS
)

type DNSRecord struct {
    Domain string
    Type string
}
type Record struct {
    ID int `storm:"id,increment"` // primary key
    Protocol string
    SrcIP string
    DstIP string
    Payload string
    HTTPHeader map[string] string
    HostToHost bool
    DNS DNSRecord
}

func main() {
    snifferDB,
    errDB = storm.Open("sniffer.db")
    if errDB != nil {
        log.Fatal(errDB)
    }

    defer snifferDB.Close()

    //Initial CLI App Setup
    app: = & cli.App {
        Name: "Packet Sniffer",
        Version: "0.1.0",
        Description: "PCAP file analysis & sniffer based on protocol and port number",
        Authors: [] * cli.Author {
            {
                Name: "KP",
            },
        },
        Flags: [] cli.Flag { &
            cli.StringFlag {
                    Name: "web",
                    Value: "no",
                    Usage: "Enable web server for GUI",
                    Required: false,
                }, &
                cli.StringFlag {
                    Name: "filepath",
                    Value: "",
                    Usage: "Choose a pcap file to analyze",
                    Required: false,
                }, &
                cli.StringFlag {
                    Name: "protocol",
                    Value: "",
                    Usage: "TCP/UDP",
                    Required: false,
                }, &
                cli.StringFlag {
                    Name: "port",
                    Value: "",
                    Usage: "Choose port between 1-65535",
                    Required: false,
                }, &
                cli.StringFlag {
                    Name: "time",
                    Value: "30",
                    Usage: "Determine how long you want it to run in seconds (min is 30 seconds)",
                    Required: false,
                },
        },
        Action: func(c * cli.Context) error {

            //flag to check if everything checks out
            valChecks: = true
            webCheck: = false

            // input validation checks
                if (c.String("web") == "yes") {
                    webCheck = true
                } else {

                    if (c.String("filepath") == "") {
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
                        openPCAPFileAndAnalyze(c.String("filepath"))

                    }
                }


                // run if input checks out 
            if webCheck {
                fmt.Println("RUNNING WEB SERVER")
                fileServer: = http.FileServer(http.Dir("./frontend"))
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
        err: = app.Run(os.Args)
    if err != nil {
        log.Fatal(err)
    }

}

func runSniffer(snifferDB * storm.DB, protocol string, port int64) {
    fmt.Println("running sniffer...")
        //unix timestamp
    currentTimestamp: = time.Now()

    //create hash using current timestamp
    hash: = md5.Sum([] byte(currentTimestamp.String()))
    hashValue: = hex.EncodeToString(hash[: ])

    //filename based on timestamp and md5 hash
    fileName: = "pcap-" + currentTimestamp.Format(timestampLayout) + "-" + hashValue + ".pcap"

    //create new file for pcap 
    pcapFile, _: = os.Create(fileName)

    // Open output pcap file and write header
    w: = pcapgo.NewWriter(pcapFile)
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
    packetSource: = gopacket.NewPacketSource(handle, handle.LinkType())
    for packet: = range packetSource.Packets() {

        //write packet to pcap file
        w.WritePacket(packet.Metadata().CaptureInfo, packet.Data())


        //use existing structures to store the packet information 
        //instead of creating new structs for every packet which 
        //takes time and memory. 
        parser: = gopacket.NewDecodingLayerParser(
            layers.LayerTypeEthernet, &
            ethLayer, &
            ipLayer, &
            tcpLayer, &
            udpLayer, &
            dnsLayer,
        )

        foundLayerTypes: = [] gopacket.LayerType {}
        err: = parser.DecodeLayers(packet.Data(), & foundLayerTypes)
        if err != nil {
            //fmt.Println("Trouble decoding layers: ", err)
        }



        for _, layerType: = range foundLayerTypes {

            //extract ipv4 data
            if layerType == layers.LayerTypeIPv4 {

                fmt.Println(ipLayer.DstIP.String())

                //create record
                record: = CreateRecord(ipLayer)

                //store in db
                errSave: = snifferDB.Save( & record)
                if errSave != nil {
                    log.Fatal(errSave)
                }

            }

            //extract tcp data
            if layerType == layers.LayerTypeTCP {
                fmt.Println("TCP Port: ", tcpLayer.SrcPort, "->", tcpLayer.DstPort)
                fmt.Println("TCP SYN:", tcpLayer.SYN, " | ACK:", tcpLayer.ACK)

                //extract SNI
                if len(tcpLayer.Payload) > 0 {
                    // TLS Handshake?
                    if tcpLayer.Payload[0] == 0x16 {
                        // Parse the packet as a TLS handshake
                        tls: = & layers.TLS {}
                        err: = tls.DecodeFromBytes(tcpLayer.Payload, gopacket.NilDecodeFeedback)
                        if err != nil {
                            continue
                        }

                        // Check for ClientHello
                        if len(tls.Handshake) > 0 && tls.Handshake[0].Type == layers.HandshakeTypeClientHello {
                            // Extract SNI
                            for _, ext: = range tls.Handshake[0].Extensions {
                                if ext.Type == layers.TLSExtensionTypeServerName {
                                    sni: = string(ext.Data[2: ])
                                    log.Printf("SNI: %s\n", sni)
                                }
                            }
                        }
                    }
                }

                // Application layer contains things like HTTP
                //FTP, SMTP, etc..
                applicationLayer: = packet.ApplicationLayer()
                if applicationLayer != nil {
                    fmt.Println("Application layer/Payload found.")

                    // Search for a protocols inside the payload
                    switch true {
                        case strings.Contains(string(applicationLayer.Payload()), "HTTP"):
                            //create record
                            record: = Record {
                                Protocol: "HTTP",
                                SrcIP: ipLayer.SrcIP.String(),
                                DstIP: ipLayer.DstIP.String(),
                                Payload: string(applicationLayer.Payload()),
                                HTTPHeader: parseHTTPHeader(string(applicationLayer.Payload())),
                            }

                            //store in db
                            errSave: = snifferDB.Save( & record)
                            if errSave != nil {
                                log.Fatal(errSave)
                            }

                        case strings.Contains(string(applicationLayer.Payload()), "FTP"):
                            //create record
                            record: = Record {
                                Protocol: "FTP",
                                SrcIP: ipLayer.SrcIP.String(),
                                DstIP: ipLayer.DstIP.String(),
                                Payload: string(applicationLayer.Payload()),
                            }

                            //store in db
                            errSave: = snifferDB.Save( & record)
                            if errSave != nil {
                                log.Fatal(errSave)
                            }
                        case strings.Contains(string(applicationLayer.Payload()), "TELNET"):
                            //create record
                            record: = Record {
                                Protocol: "TELNET",
                                SrcIP: ipLayer.SrcIP.String(),
                                DstIP: ipLayer.DstIP.String(),
                                Payload: string(applicationLayer.Payload()),
                            }

                            //store in db
                            errSave: = snifferDB.Save( & record)
                            if errSave != nil {
                                log.Fatal(errSave)
                            }
                        case strings.Contains(string(applicationLayer.Payload()), "SMB"):
                            //create record
                            record: = Record {
                                Protocol: "SMB",
                                SrcIP: ipLayer.SrcIP.String(),
                                DstIP: ipLayer.DstIP.String(),
                                Payload: string(applicationLayer.Payload()),
                            }

                            //store in db
                            errSave: = snifferDB.Save( & record)
                            if errSave != nil {
                                log.Fatal(errSave)
                            }

                        case strings.Contains(string(applicationLayer.Payload()), "SSH"):
                            //create record
                            record: = Record {
                                Protocol: "SSH",
                                SrcIP: ipLayer.SrcIP.String(),
                                DstIP: ipLayer.DstIP.String(),
                                Payload: string(applicationLayer.Payload()),
                            }

                            //store in db
                            errSave: = snifferDB.Save( & record)
                            if errSave != nil {
                                log.Fatal(errSave)
                            }


                        case strings.Contains(string(applicationLayer.Payload()), "SMTP"):
                            //create record
                            record: = Record {
                                Protocol: "SMTP",
                                SrcIP: ipLayer.SrcIP.String(),
                                DstIP: ipLayer.DstIP.String(),
                                Payload: string(applicationLayer.Payload()),
                            }

                            //store in db
                            errSave: = snifferDB.Save( & record)
                            if errSave != nil {
                                log.Fatal(errSave)
                            }


                        default:
                    }

                }
            }

            //extract dns data
            if layerType == layers.LayerTypeDNS {
                dnsResponseCode: = int(dnsLayer.ResponseCode)
                dnsANCount: = int(dnsLayer.ANCount)

                //check if there is a dns response 
                if (dnsANCount == 0 && dnsResponseCode > 0) || (dnsANCount > 0) {

                    for _, dnsQuestion: = range dnsLayer.Questions {

                        //domain name --
                        fmt.Println("DOMAIN NAME - " + string(dnsQuestion.Name))

                        //record type
                        fmt.Println("RECORD TYPE - " + dnsQuestion.Type.String())

                        //extract answers
                        if dnsANCount > 0 {

                            for _, dnsAnswer: = range dnsLayer.Answers {
                                if dnsAnswer.IP.String() != "<nil>" {
                                    fmt.Println("DNS Answer: ", dnsAnswer.IP.String())


                                    //create record
                                    record: = Record {
                                        Protocol: "DNS",
                                        SrcIP: ipLayer.SrcIP.String(),
                                        DstIP: ipLayer.DstIP.String(),
                                        DNS: DNSRecord {
                                            Domain: string(dnsAnswer.Name),
                                            Type: dnsQuestion.Type.String(),
                                        },
                                    }

                                    //store in db
                                    errSave: = snifferDB.Save( & record)
                                    if errSave != nil {
                                        log.Fatal(errSave)
                                    }
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
func CreateRecord(ipLayer layers.IPv4) Record {
    return Record {
        Protocol: ipLayer.Protocol.String(),
        SrcIP: ipLayer.SrcIP.String(),
        DstIP: ipLayer.DstIP.String(),
        HostToHost: checkHostToHostInternalCommunication(ipLayer.SrcIP.String(), ipLayer.DstIP.String()),
    }
}
func parseHTTPHeader(header string) map[string] string {

    //find a way to parse http headers consistently 
    //and create a struct to represent each of those fields
    //Host: test.com
    //Connection: keep alive
    //Accept-Encoding: gzip,deflate etc..
    //Problem we run into is the type of HTTP request for example:
    //GET /HTTP/1.1 or POST /HTTP/1.1 etc..
    parsedHeader: = strings.Split(header, "\n")
    httpMap: = make(map[string] string)

    for i: = 0;i < len(parsedHeader);i++{
        if strings.Contains(parsedHeader[i], "HTTP/") {
            httpMap["Type"] = strings.TrimSpace(parsedHeader[i])
        } else {
            fields: = strings.Split(strings.TrimSpace(parsedHeader[i]), ": ")
            if len(fields) > 1 {
                httpMap[fields[0]] = fields[1]
            }

        }
    }

    return httpMap


}

func openPCAPFileAndAnalyze(fileName string) {

    pcapDB,
    errDB = storm.Open("pcap.db")
    if errDB != nil {
        log.Fatal(errDB)
    }

    defer pcapDB.Close()

    // Open file instead of device
    handle,
    err = pcap.OpenOffline(fileName)
    if err != nil {
        log.Fatal(err)
    }

    defer handle.Close()

    // Loop through packets in file
    packetSource: = gopacket.NewPacketSource(handle, handle.LinkType())
    for packet: = range packetSource.Packets() {

        //use existing structures to store the packet information 
        //instead of creating new structs for every packet which 
        //takes time and memory. 
        parser: = gopacket.NewDecodingLayerParser(
            layers.LayerTypeEthernet, &
            ethLayer, &
            ipLayer, &
            tcpLayer, &
            udpLayer, &
            dnsLayer,
        )

            foundLayerTypes: = [] gopacket.LayerType {}
        err: = parser.DecodeLayers(packet.Data(), & foundLayerTypes)
        if err != nil {
            //fmt.Println("Trouble decoding layers: ", err)
        }


        for _,
        layerType: = range foundLayerTypes {

            //extract ipv4 data
            if layerType == layers.LayerTypeIPv4 {

                // create record
                record: = CreateRecord(ipLayer)

            }

            //extract tcp data
            if layerType == layers.LayerTypeTCP {

                // Application layer contains things like HTTP
                //FTP, SMTP, etc..
                applicationLayer: = packet.ApplicationLayer()
                if applicationLayer != nil {
                    fmt.Println("Application layer/Payload found.")
                        // fmt.Printf("%s\n", applicationLayer.Payload())

                    // Search for a protocols inside the payload
                    switch true {
                        case strings.Contains(string(applicationLayer.Payload()), "HTTP"):

                            //create record
                            record: = Record {
                                Protocol: "HTTP",
                                SrcIP: ipLayer.SrcIP.String(),
                                DstIP: ipLayer.DstIP.String(),
                                Payload: string(applicationLayer.Payload()),
                                HTTPHeader: parseHTTPHeader(string(applicationLayer.Payload())),
                            }

                            //store in db
                            errSave: = pcapDB.Save( & record)
                            if errSave != nil {
                                log.Fatal(errSave)
                            }

                        case strings.Contains(string(applicationLayer.Payload()), "FTP"):
                            //create record
                            record: = Record {
                                Protocol: "FTP",
                                SrcIP: ipLayer.SrcIP.String(),
                                DstIP: ipLayer.DstIP.String(),
                                Payload: string(applicationLayer.Payload()),
                            }

                            //store in db
                            errSave: = pcapDB.Save( & record)
                            if errSave != nil {
                                log.Fatal(errSave)
                            }
                        case strings.Contains(string(applicationLayer.Payload()), "TELNET"):
                            //create record
                            record: = Record {
                                Protocol: "TELNET",
                                SrcIP: ipLayer.SrcIP.String(),
                                DstIP: ipLayer.DstIP.String(),
                                Payload: string(applicationLayer.Payload()),
                            }

                            //store in db
                            errSave: = pcapDB.Save( & record)
                            if errSave != nil {
                                log.Fatal(errSave)
                            }
                        case strings.Contains(string(applicationLayer.Payload()), "SMB"):
                            //create record
                            record: = Record {
                                Protocol: "SMB",
                                SrcIP: ipLayer.SrcIP.String(),
                                DstIP: ipLayer.DstIP.String(),
                                Payload: string(applicationLayer.Payload()),
                            }

                            //store in db
                            errSave: = pcapDB.Save( & record)
                            if errSave != nil {
                                log.Fatal(errSave)
                            }

                        default:
                    }
                }
            }

            //extract dns data
            if layerType == layers.LayerTypeDNS {
                dnsResponseCode: = int(dnsLayer.ResponseCode)
                dnsANCount: = int(dnsLayer.ANCount)

                //check if there is a dns response 
                if (dnsANCount == 0 && dnsResponseCode > 0) || (dnsANCount > 0) {

                    for _, dnsQuestion: = range dnsLayer.Questions {

                        //domain name --
                        fmt.Println("DOMAIN NAME - " + string(dnsQuestion.Name))

                        //record type
                        fmt.Println("RECORD TYPE - " + dnsQuestion.Type.String())

                        //extract answers
                        if dnsANCount > 0 {

                            for _, dnsAnswer: = range dnsLayer.Answers {
                                if dnsAnswer.IP.String() != "<nil>" {
                                    fmt.Println("DNS Answer: ", dnsAnswer.IP.String())

                                    //create record
                                    record: = Record {
                                        Protocol: "DNS",
                                        SrcIP: ipLayer.SrcIP.String(),
                                        DstIP: ipLayer.DstIP.String(),
                                        DNS: DNSRecord {
                                            Domain: string(dnsAnswer.Name),
                                            Type: dnsQuestion.Type.String(),
                                        },
                                    }

                                    //store in db
                                    errSave: = snifferDB.Save( & record)
                                    if errSave != nil {
                                        log.Fatal(errSave)
                                    }
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

func getRecords(w http.ResponseWriter, req * http.Request) {

    records: = getAllEventRecords()

        jsonData,
    err: = json.Marshal(records)
    if err != nil {
        log.Fatal(err)
    }
    w.Header().Set("Content-Type", "application/json")
    w.Write(jsonData)

}


func getAllEventRecords()[] Record {

    var records[] Record

    errFetch: = snifferDB.All( & records)
    if errFetch != nil {
        log.Fatal(errFetch)
        return records
    } else {
        return records
    }

}

func checkHostToHostInternalCommunication(srcIP string, dstIP string) bool {
    if (isPrivateIPv4(srcIP) && isPrivateIPv4(dstIP)) {
        return true
    }

    return false
}

func isPrivateIPv4(ipStr string) bool {
    ip: = net.ParseIP(ipStr)
    if ip == nil {
        return false
    }

    if ip.IsLoopback() {
        return true
    }

    if ip4: = ip.To4();ip4 != nil {
        switch {
            case ip4[0] == 10:
                return true
            case ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31:
                return true
            case ip4[0] == 192 && ip4[1] == 168:
                return true
            default:
                return false
        }
    }

    return false
}