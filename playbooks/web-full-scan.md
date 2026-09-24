---
name: "Web Server Full Scan"
description: "Complete reconnaissance and enumeration for web servers"
author: "ujiscan"
version: "1.0"
entry_phase: "recon"
---

## recon

### nmap-common-web-ports
Tool: nmap
Args: [-p 80,443,8080,8443,3000,5000 -sV -oJ - {target}]
Description: Scan common web service ports
Condition: ""

### nmap-all-ports
Tool: nmap
Args: [-p- -T4 -oJ - {target}]
Description: Full port scan on all ports
Condition: ""

## enum

### nuclei-web-scan
Tool: nuclei
Args: [-u http://{target} -t http -json]
Description: Scan for web vulnerabilities using nuclei
Condition: ""

### nuclei-ssl-scan
Tool: nuclei
Args: [-u https://{target} -t ssl -json]
Description: Scan SSL/TLS vulnerabilities
Condition: if_port_open_443
