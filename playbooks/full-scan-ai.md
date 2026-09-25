---
name: "Full Scan (AI-Driven)"
description: "AI Agent-driven comprehensive assessment with dynamic tool selection"
author: "ujiscan"
version: "1.0"
entry_phase: "ai_reconnaissance"
mode: "agentic"
---

## ai_reconnaissance

### ai-recon-planning
Tool: ai-agent
Description: AI analyzes target and plans comprehensive reconnaissance
Agent: ReconAgent
Autonomy: full
Actions:
  - Determine target type (domain, IP, CIDR)
  - Identify optimal reconnaissance approach
  - Select from: dig, subfinder, nmap, httpx, whois, shodan queries
Condition: ""

## ai_enumeration

### ai-service-discovery
Tool: ai-agent
Description: AI orchestrates service discovery and fingerprinting
Agent: EnumAgent
Autonomy: full
Actions:
  - Execute nmap with optimal flags
  - Run whatweb for tech stack
  - Probe with httpx for all services
  - Audit SSL/TLS with sslscan
Condition: ""

## ai_vulnerability_assessment

### ai-vuln-selection
Tool: ai-agent
Description: AI selects optimal vulnerability scanners based on services found
Agent: VulnAgent
Autonomy: full
Actions:
  - Run nuclei with appropriate templates
  - Execute gobuster on web servers
  - Run nikto on HTTP services
  - Check DNS for anomalies with dig
Condition: ""

## ai_exploitation

### ai-exploit-planning
Tool: ai-agent
Description: AI plans and executes exploitation based on findings
Agent: ExploitAgent
Autonomy: supervised
Actions:
  - Analyze vulnerabilities found
  - Recommend exploitation chain
  - Execute with human approval
  - Capture evidence and POC
Condition: critical_vuln_found

## ai_reporting

### ai-report-generation
Tool: ai-agent
Description: AI generates comprehensive assessment report
Agent: ReportAgent
Autonomy: full
Actions:
  - Correlate all findings
  - Risk scoring with CVSS
  - Remediation recommendations
  - Executive summary
  - Technical details with evidence
Condition: ""
