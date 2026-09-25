---
name: "Regular Scan"
description: "Comprehensive regular security scan with multiple tools"
author: "ujiscan"
version: "1.0"
entry_phase: "recon"
---

## recon

### dig-lookup
Tool: dig
Args: []
Description: Perform DNS lookup
Condition: ""

### subfinder-subdomain-discovery
Tool: subfinder
Args: [-json]
Description: Discover subdomains
Condition: ""

### nmap-ping-discovery
Tool: nmap
Args: [-sn -oJ -]
Description: Discover live hosts using ping scan
Condition: ""

## enum

### httpx-http-probing
Tool: httpx
Args: []
Description: Probe HTTP services
Condition: ""

### nmap-port-scan
Tool: nmap
Args: [-p 22,80,443,3306,5432,6379,27017,8080,8443,8000,5000,3000 -sV -sC -oJ -]
Description: Comprehensive port scan with service detection
Condition: ""

### whatweb-fingerprinting
Tool: whatweb
Args: []
Description: Web technology fingerprinting
Condition: ""

### sslscan-ssl-audit
Tool: sslscan
Args: []
Description: SSL/TLS certificate and configuration audit
Condition: ""

## exploit

### nuclei-vuln-scan
Tool: nuclei
Args: [-json]
Description: Scan for vulnerabilities with nuclei
Condition: ""

### nikto-web-scan
Tool: nikto
Args: []
Description: Web server vulnerability scanning
Condition: ""
