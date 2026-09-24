# ujiscan Roadmap: Build OmOP-like System Legally & Originally

## Foundation: Legal References ONLY

```
OWASP WSTG v4.2           (CC BY-SA 4.0 - Commercial OK)
MITRE ATT&CK              (Free - public domain)
PayloadsAllTheThings      (MIT License - Commercial OK)
SecLists                  (MIT License - Commercial OK)
PTES Methodology          (Free - public domain)
```

**NOT using:**
- oh-my-open-pentest code (SUL-1.0 - Commercial FORBIDDEN)
- oh-my-openagent code (SUL-1.0 - Commercial FORBIDDEN)

---

## Phase 1: Tool Registry System (1-2 weeks)

### Goal
Replace hardcoded tool definitions with data-driven registry (tools.yaml)

### Current State
```
internal/tools/nmap.go      ← Hardcoded
internal/tools/nuclei.go    ← Hardcoded
```

### Target State
```
tools.yaml                  ← Tool definitions (data)
internal/tools/registry.go  ← Tool loader (code)
internal/tools/executor.go  ← Uses registry, not hardcoded
```

### Benefit
Adding new tool = edit YAML, no code change needed

### Implementation
1. ✅ Create tools.yaml (DONE - based on OWASP WSTG)
2. Create Go Tool Registry struct:
```go
type Tool struct {
    ID           string
    Name         string
    Description  string
    Phase        string
    Execution    ExecutionConfig
    Installation InstallationConfig
    Output       OutputConfig
}
```
3. Create ToolRegistry loader:
```go
func LoadToolRegistry(filePath string) (*ToolRegistry, error)
```
4. Refactor executor to use registry
5. Test: Add new tool to YAML, verify it runs

---

## Phase 2: OWASP WSTG Phased Execution (1-2 weeks)

### Goal
Implement 10 OWASP WSTG phases with intelligent orchestration

### Current State
```
nmap → nuclei (fixed sequence)
```

### Target State
```
Phase 1: Information Gathering
  ├─ whois (parallel)
  └─ dig (parallel)

Phase 2: Configuration Analysis
  └─ nmap

Phase 7: Input Validation
  └─ nuclei

Phase 10: API Testing
  └─ curl

...and adapt execution based on findings
```

### Implementation
1. Define Phase struct in Go
2. Tag tools with phases (DONE in tools.yaml)
3. Create PhaseExecutor that:
   - Runs phase tools in order
   - Collects findings
   - Passes to next phase
4. Implement findings collection between phases

---

## Phase 3: Target Analyzer & Mode Detection (1 week)

### Goal
User enters target → system detects type → auto-selects mode

### Current State
```
User: "pick Regular, Playbook, or Agentic"
```

### Target State
```
User: "scan example.com" or "scan 192.168.1.0/24"
System detects: URL? IP? CIDR? Binary?
System selects: web-app mode? network-recon mode? api mode?
System loads: matching tool chain + execution strategy
```

### Implementation
1. Create TargetAnalyzer:
```go
func AnalyzeTarget(target string) TargetType
// Returns: URL, IP, CIDR, Domain, Binary, etc.
```
2. Create ModeSelector:
```go
func SelectMode(targetType TargetType) EngagementMode
// Returns: web-app, network, api, mobile, etc.
```
3. Load tool chain based on mode (from tools.yaml)

---

## Phase 4: Scope Enforcement (1 week)

### Goal
Validate findings stay within user-defined scope

### Current State
```
No scope validation
```

### Target State
```
User: "scope: example.com + *.example.com only"
Finding: XSS in sub.evil.com
System: REJECT (out of scope)
Finding: RCE in app.example.com
System: ACCEPT (in scope)
```

### Implementation
1. Create ScopeValidator:
```go
type Scope struct {
    Targets []string  // e.g., "example.com", "*.example.com"
}
```
2. ScopeMatcher - check if finding is in scope
3. Filter results before reporting

---

## Phase 5: Rules Engine for Adaptive Tool Selection (2 weeks)

### Goal
Next tools selected based on findings, not predetermined

### Current State
```
Tools run sequentially regardless of findings
```

### Target State
```
Finding: XSS detected → load dalfox, xsstrike
Finding: SQL injection → load sqlmap
Finding: Open port 3389 → load rdp_scanner
```

### Implementation
1. Define Rule format in YAML:
```yaml
rules:
  - id: "found_xss"
    condition:
      tool: "nuclei"
      finding_type: "xss"
    actions:
      - load_tools: ["dalfox", "xsstrike"]
      - phase: "exploitation"
```
2. Create RuleEngine to evaluate conditions
3. Dynamic tool loading based on findings

---

## Phase 6: Finding Verification (1-2 weeks)

### Goal
Confirm each finding before including in report

### Current State
```
Tools output → report as-is (trust tool)
```

### Target State
```
Tools output → Verification phase
├─ Re-run finding (confirm it still exists)
├─ Manual PoC (curl, browser interaction)
├─ CVSS scoring
└─ Confirmed findings → report
```

### Implementation
1. Add Verification phase to executor
2. PoC Runner - re-execute finding to confirm
3. CVSS Scorer - rate severity using CVSS v3.1
4. Evidence collector - store PoC output

---

## Phase 7: Format-Specific Reports (1-2 weeks)

### Goal
Output tailored to engagement type

### Current State
```
JSON API only
```

### Target State
```
Engagement: Bug Bounty
  → HackerOne format (severity, description, impact, reproduction)

Engagement: Red Team
  → Executive Summary (overview) + Detailed findings

Engagement: CTF
  → Flag submission format

Engagement: API
  → REST API endpoints, auth bypass, injection points
```

### Implementation
1. Define ReportFormat interface
2. Implement formatters for each mode:
   - HackerOneFormatter
   - ExecutiveSummaryFormatter
   - CTFFormatter
   - DetailedTechnicalFormatter
3. Populate with finding data + PoC

---

## Phase 8: Playbook Engine Enhancement (1-2 weeks)

### Goal
Playbooks adapt based on findings & mode

### Current State
```
Fixed playbook steps
```

### Target State
```
Playbook: "Web App Security Testing"
  IF (found authentication)
    → Load OWASP-ATHN-01 through 05
  IF (found api endpoints)
    → Load OWASP-APIT-01 through 10
  IF (found sql injection)
    → Escalate to exploitation phase
```

### Implementation
1. Add conditional logic to playbook format
2. Finding-based step selection
3. Dynamic phase injection based on results

---

## Phase 9: Dashboard Enhancements (1 week)

### Goal
Show real-time execution, phases, findings

### Current State
```
Logs + results view
```

### Target State
```
┌─ Current Phase: Input Validation
├─ Running Tool: nuclei
├─ Progress: [=====>    ] 55%
├─ Findings: 3 (1 critical, 2 high)
├─ Next Phase: Verification (auto-select)
└─ Timeline: recon→config→identity→auth→authz→session→input→business→client→api
```

### Implementation
1. Phase progress indicator
2. Finding severity dashboard
3. Real-time tool output streaming
4. Attack chain visualization

---

## Phase 10: Intelligence Layer (2-3 weeks)

### Goal
Integration with MITRE ATT&CK, CVE data, WAF signatures

### Current State
```
Tool output only
```

### Target State
```
Finding: Open port 445 (SMB)
├─ MITRE link: Lateral Movement Techniques
├─ Relevant tools: enum4linux, impacket, netexec
├─ Related CVEs: CVE-2017-10271, CVE-2020-1040, ...
└─ Attack chain: Initial Access → Lateral Movement → Persistence

Finding: WordPress detected
├─ MITRE: Collection via WordPress plugins
├─ Relevant tools: WPScan, msfconsole
└─ Common exploits: plugin vuln, theme vuln, xmlrpc
```

### Implementation
1. Load MITRE ATT&CK matrix locally
2. Map findings to techniques
3. CVE database integration
4. Attack chain recommendations

---

## IMPLEMENTATION ROADMAP (Timeline)

| Week | Phase | Deliverable |
|------|-------|-------------|
| 1 | Tool Registry | tools.yaml + Go loader ✅ |
| 2-3 | Phased Execution | OWASP WSTG phases working |
| 4 | Target Analyzer | URL/IP detection + mode selection |
| 5 | Scope Enforcement | Scope validation in findings |
| 6-7 | Rules Engine | Adaptive tool selection |
| 8-9 | Verification | PoC runner + CVSS scoring |
| 10 | Reports | Format-specific outputs |
| 11-12 | Dashboard | Real-time progress + phases |
| 13-14 | Intelligence | MITRE + CVE integration |

---

## REFERENCES (All Legal & Free)

### Methodology
- OWASP Web Security Testing Guide v4.2
  https://owasp.org/www-project-web-security-testing-guide/v4.2/

- PTES (Penetration Testing Execution Standard)
  https://www.pentest-standard.org/

- MITRE ATT&CK Framework
  https://attack.mitre.org/

### Tools & Payloads (MIT License)
- PayloadsAllTheThings
  https://github.com/swisskyrepo/PayloadsAllTheThings

- SecLists
  https://github.com/danielmiessler/SecLists

### Testing Guides
- OWASP Top 10 Web Application Security Risks
  https://owasp.org/www-project-top-ten/

- OWASP Testing Checklist
  https://owasp.org/www-project-web-security-testing-guide/v4.2/checklists/

---

## Key Principle

**We are building based on ESTABLISHED METHODOLOGIES (OWASP, MITRE, PTES), not copying proprietary code.**

This means:
- ✅ 100% legal for commercial use
- ✅ Based on industry standards
- ✅ Easy to defend IP-wise
- ✅ Can be patented (methodology innovations)
- ✅ Can be sold or funded

---

## Next Action

Ready to start Phase 1 implementation (tool registry loader in Go)?
