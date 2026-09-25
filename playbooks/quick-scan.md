---
name: "Quick Scan"
description: "Ultra-fast security scan - nmap port scan only"
author: "ujiscan"
version: "1.0"
entry_phase: "recon"
---

## recon

### nmap-quick-port-scan
Tool: nmap
Args: [-p 22,80,443,8080 -sV --open -oJ - {target}]
Description: Quick nmap scan of most common ports
Condition: ""
