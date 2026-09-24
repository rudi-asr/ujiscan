---
name: "Network Discovery"
description: "Discover live hosts and basic network reconnaissance"
author: "ujiscan"
version: "1.0"
entry_phase: "recon"
---

## recon

### nmap-ping-discovery
Tool: nmap
Args: [-sn -oJ - {target}]
Description: Discover live hosts using ping scan
Condition: ""

### nmap-port-scan
Tool: nmap
Args: [-p 22,80,443,3306,5432,6379,27017,8080 -sV -oJ - {target}]
Description: Scan for common service ports
Condition: ""

## enum

### nuclei-network-scan
Tool: nuclei
Args: [-l {target} -json]
Description: Run nuclei templates on discovered hosts
Condition: ports_open
