# packetsniffer

Simple and useful sniffer that captures traffic based on given preferences.

## Another Sniffer?

The goal of this project was to extend my knowledge around networking and the significance of packet analysis on the defensive side of things. 

## Features

* Interactive CLI

* Packet capture 

* PCAP file analysis 

## Dependencies

WINPCAP - must download driver to prevent any DLL issues

## Plaform support

Windows 8/8.1/10

## Short term goals

* Create a threat detection service 
	- based on ip addresses, domains, ports
	- checks whether workstation visisted a possible suspicious or malicious website 
	- internal host to host communication 
	- utlize free threat intel feeds and open source API's
	- scheduling mechanism to update feeds
	
* Create a UI for pcap file analysis
	- can be used by incident response and SOC teams to find any suspicious network activities on a workstation.

## Further down the road goals... adding more intelligence 

* Configuration to capture traffic based on a set of rules like:
	- CPU resource spikes and deviates from baseline  

* Firewall integrations to block suspicious communcation paths 

