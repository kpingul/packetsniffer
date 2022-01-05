# packetsniffer

*The goal of packet sniffer:* 

	1.interactive CLI

	2.choose either TCP/UDP traffic

	3.choose a port number ( will extend to multiple  ports later on..)

	4.capture network traffic

	5.store results in pcap file

*short term goals*

	1.Create a threat detection service 
		-based on ip addresses and domains
		-checks whether host visisted a suspicious/malicious website 
		-utlize free threat intel feeds and open source API's
		-scheduling mechanism to update feeds
	
	2.Create a UI for pcap file analysis
		-can be used by incident response and SOC teams to find any suspicious network activies on a workstation.

*further down the road goals... adding more intelligence to the sniffer*

	1.Configuration to capture traffic based on a set of rules like:
		-CPU resource spikes and deviatiates from baseline  

	2.Firewall integrations to block suspicious communcation paths 

## platform support

Windows 8/8.1/10

## dependencies

WINPCAP - must download driver to prevent any DLL issues