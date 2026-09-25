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
Args: [+short {target}]
Description: Perform DNS lookup
Condition: ""

### subfinder-subdomain-discovery
Tool: subfinder
Args: [-d {target} -json]
Description: Discover subdomains
Condition: ""

### nmap-ping-discovery
Tool: nmap
Args: [-sn -oJ - {target}]
Description: Discover live hosts using ping scan
Condition: ""

### httpx-http-probing
Tool: httpx
Args: [-json {target}]
Description: Probe HTTP services
Condition: ""

## enum

### nmap-port-scan
Tool: nmap
Args: [-p 22,80,443,3306,5432,6379,27017,8080,8443,8000,5000,3000 -sV -sC -oJ - {target}]
Description: Comprehensive port scan with service detection
Condition: ""

### whatweb-fingerprinting
Tool: whatweb
Args: [--color=never --log-json=- {target}]
Description: Web technology fingerprinting
Condition: ports_open

### sslscan-ssl-audit
Tool: sslscan
Args: [--no-failed {target}]
Description: SSL/TLS certificate and configuration audit
Condition: ports_open

## scan

### nuclei-vuln-scan
Tool: nuclei
Args: [-u {target} -json -severity high,critical]
Description: Scan for vulnerabilities with nuclei
Condition: ports_open

### gobuster-dir-brute
Tool: gobuster
Args: [dir -u http://{target} -w /usr/share/wordlists/dirbuster/directory-list-2.3-small.txt -q]
Description: Brute-force web directories
Condition: ""

### nikto-web-scan
Tool: nikto
Args: [-h {target} -Format JSON]
Description: Nikto web server vulnerability scan
Condition: ports_open

## exploit

### (placeholder for manual exploit phase)
Description: Manual exploitation steps would go here
Condition: critical_vuln_found
