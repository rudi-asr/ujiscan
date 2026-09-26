---
name: "Regular Scan"
description: "Comprehensive security scan - full replication of professional pentest scanning"
author: "ujiscan"
version: "2.0"
entry_phase: "recon"
---

## recon

### dig-lookup
Tool: dig
Args: [ANY]
Description: Perform comprehensive DNS lookup (all record types)
Condition: ""

### subfinder-subdomain-discovery
Tool: subfinder
Args: [-all, -recursive, -silent]
Description: Discover subdomains with all sources
Condition: ""

### nmap-host-discovery
Tool: nmap
Args: [-sn, -PE, -PP, -PM]
Description: Discover live hosts using multiple ping techniques
Condition: ""

## enum

### httpx-http-probing
Tool: httpx
Args: [-follow-redirects, -status-code, -title, -tech-detect, -server]
Description: Comprehensive HTTP service probing with tech detection
Condition: ""

### nmap-tcp-full-scan
Tool: nmap
Args: [-p-, -sV, -sC, -T4, --version-intensity, "5"]
Description: Complete TCP port scan (all 65535 ports) with service detection and default scripts
Condition: ""

### nmap-udp-common-scan
Tool: nmap
Args: [-sU, -p, "53,67,68,69,123,137,138,161,162,500,514,520,631,1434,1900,4500,5353", -sV]
Description: UDP port scan of common services
Condition: ""

### whatweb-fingerprinting
Tool: whatweb
Args: [-a, "3", --log-json=-]
Description: Aggressive web technology fingerprinting (level 3)
Condition: ""

### sslscan-comprehensive
Tool: sslscan
Args: [--show-certificate, --show-client-cas, --show-ciphers, --show-sigs, --no-colour]
Description: Comprehensive SSL/TLS audit - certificates, ciphers, vulnerabilities
Condition: ""

## vulnscan

### nuclei-cve-scan
Tool: nuclei
Args: [-t, "cves/", -t, "vulnerabilities/", -t, "exposures/", -severity, "critical,high,medium", -silent, -jsonl]
Description: Comprehensive vulnerability scan - all CVE and vulnerability templates
Condition: ""

### gobuster-directory-scan
Tool: gobuster
Args: [-w, "/tmp/common.txt", -k, -q, -e, -r]
Description: Web directory and file brute-force discovery
Condition: ""

### nikto-web-vuln-scan
Tool: nikto
Args: [-Tuning, "x", -Display, "V", -Format, "txt"]
Description: Comprehensive web server vulnerability scan (all checks)
Condition: ""
