# packetsniffer

*The goal of packet sniffer:* 

	1.interactive CLI

	2.choose either TCP/UDP traffic

	3.choose a port number ( will extend to multiple  ports later on..)

	4.capture network traffic

	5.store results in pcap file

*closer down the road goals*

	1.Provide a UI to analyze pcap files
		-Can be used by incident response and SOC teams to find any suspicious network activies on a workstation.

*further down the road goals..adding more intelligence to the sniffer*

	1.Configuration to capture traffic based on a set of rules like:
		-CPU resource spikes and deviatiates from baseline  

	2.Firewall integrations to block suspicious communcation paths 

## platform support

Windows 8/8.1/10

## dependencies

WINPCAP - must download driver to prevent any DLL issues