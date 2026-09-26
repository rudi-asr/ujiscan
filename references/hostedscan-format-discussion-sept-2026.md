# Diskusi: Format Laporan ala hostedscan untuk Reguler & Quick Scan

**Tanggal**: 27 September 2026  
**Konteks**: User meminta Regular & Quick scan mengeluarkan data format laporan seperti hostedscan

---

## Kondisi Sekarang

| Scan Type       | Output saat ini                                                                                      |
|-----------------|------------------------------------------------------------------------------------------------------|
| **Regular**     | JSON array `results[]` (tool name, command, stdout raw, exit code) + `findings[]` (parsed findings) |
| **Quick**       | Sama seperti Regular (subset tools, struktur data identik)                                          |
| **Full Scan AI**| JSON + `ai_decisions[]` + `ai_report` markdown (sudah punya format laporan AI-generated)             |

**Perbedaan dengan hostedscan**:
- hostedscan menampilkan laporan HTML/web terstruktur: Executive Summary, Severity Breakdown (Critical/High/Medium/Low), Detailed Findings per tool, Evidence, Remediation
- ujiscan sekarang menampilkan raw tool output + parsed findings — **belum punya struktur laporan profesional**

---

## Filosofi Produk (dari skill)

> "Reguler Scan — fokus tools dari hostedscan.com: 9 tools (dig, subfinder, nmap, httpx, whatweb, sslscan, nuclei, gobuster, nikto), diinstall lokal/Docker."

> "Laporan reguler & quick = format laporan ala hostedscan, **versi web/HTML dulu**, nanti bisa Save/Print PDF dari situ."

---

## Yang Perlu Ditambah

### 1. Backend: Report Generator untuk Regular/Quick

**Struktur laporan hostedscan** (dari observasi produk mereka):

```
Executive Summary
├── Target: example.com
├── Scan Type: Regular Scan (9 tools)
├── Started: 2026-09-27 14:35 WIB
├── Duration: 18m 24s
├── Risk Score: 7.2/10 (High)
└── Severity Breakdown
    ├── Critical: 2 findings
    ├── High: 5 findings
    ├── Medium: 12 findings
    ├── Low: 8 findings
    └── Info: 15 findings

Detailed Findings
├── Finding #1: SQL Injection in login.php
│   ├── Severity: Critical (CVSS 9.8)
│   ├── Tool: sqlmap
│   ├── Evidence: [request/response snapshot]
│   ├── Description: ...
│   ├── Remediation: Use prepared statements
│   └── References: CWE-89, OWASP A03:2021
├── Finding #2: Open Port 22 (SSH)
│   ├── Severity: Medium (CVSS 5.3)
│   ├── Tool: nmap
│   ├── Evidence: nmap output snippet
│   └── Remediation: ...
...

Tool Execution Details (Appendix)
├── dig → DNS records found: A, MX, TXT
├── nmap → 5 open ports (22, 80, 443, 3306, 8080)
├── nuclei → 12 templates matched
...
```

**Yang harus dibuat**:
- Fungsi `GenerateHostedscanReport(scan *models.Scan) *HostedscanReport`
- Struct `HostedscanReport` dengan field:
  - `ExecutiveSummary` (target, duration, risk score, severity counts)
  - `DetailedFindings[]` (per finding dengan severity, tool, evidence, remediation, CWE/OWASP)
  - `ToolExecutionDetails[]` (ringkasan per tool: berapa findings, durasi, status)
- Endpoint baru: `GET /api/scan/{id}/report/hostedscan` → return JSON report
- Endpoint HTML: `GET /api/scan/{id}/report/html` → render HTML dari JSON

**Severity classification**:
- Critical: CVSS 9.0-10.0 (RCE, SQL injection)
- High: CVSS 7.0-8.9 (XSS, sensitive data exposure)
- Medium: CVSS 4.0-6.9 (open ports, weak ciphers)
- Low: CVSS 0.1-3.9 (info disclosure, banners)
- Info: CVSS 0.0 (DNS records, tech stack)

**Risk score calculation**:
```
risk_score = weighted_avg(
  critical * 10 +
  high * 7 +
  medium * 4 +
  low * 2 +
  info * 0
) / total_findings
```

---

### 2. Frontend: Tab "Report" dengan View ala hostedscan

**UI structure** (di tab Report):

```
[Download HTML] [Print PDF] [Export JSON]

╔═══════════════════════════════════════════════╗
║  Executive Summary                             ║
║  ────────────────────────────────────────────  ║
║  🎯 Target: 127.0.0.1                          ║
║  📅 Started: Sep 27, 2026 14:35 WIB            ║
║  ⏱️ Duration: 18m 24s                          ║
║  ⚠️ Risk Score: 7.2 / 10 (High)                ║
║                                                ║
║  Severity Breakdown:                           ║
║  🔴 Critical: 2   🟠 High: 5   🟡 Medium: 12   ║
║  🔵 Low: 8        ⚪ Info: 15                   ║
╚═══════════════════════════════════════════════╝

╔═══════════════════════════════════════════════╗
║  🔴 CRITICAL — SQL Injection in /login         ║
║  ────────────────────────────────────────────  ║
║  Tool: sqlmap | CVSS: 9.8 | CWE-89             ║
║  Evidence:                                     ║
║    GET /login?user=admin' OR '1'='1            ║
║    [200 OK] Vulnerable to boolean-based blind  ║
║  Remediation:                                  ║
║    Use prepared statements with parameterized  ║
║    queries. Update framework to latest version.║
╚═══════════════════════════════════════════════╝

... (repeat per finding, descending by severity)
```

**Tombol aksi**:
- **Download HTML**: download file `report-127.0.0.1-20260927.html` (standalone, bisa dibuka offline)
- **Print PDF**: `window.print()` → browser print dialog → save as PDF
- **Export JSON**: download raw JSON report untuk integrasi/parsing

---

### 3. Template HTML Report (standalone)

**File template**: `web/templates/hostedscan-report.html`

Struktur:
```html
<!DOCTYPE html>
<html>
<head>
  <style>
    /* Dark theme, print-friendly CSS */
    @media print {
      /* Optimize untuk PDF export */
    }
  </style>
</head>
<body>
  <header>
    <h1>Penetration Test Report</h1>
    <p>Target: {{.Target}}</p>
    <p>Generated: {{.GeneratedAt}}</p>
  </header>
  
  <section class="executive-summary">
    <!-- Executive summary content -->
  </section>
  
  <section class="findings">
    {{range .Findings}}
    <div class="finding severity-{{.Severity}}">
      <!-- Finding detail -->
    </div>
    {{end}}
  </section>
  
  <section class="appendix">
    <!-- Tool execution details -->
  </section>
</body>
</html>
```

---

## Implementation Checklist

Backend:
- [ ] `internal/reports/hostedscan.go` — generator untuk format hostedscan
- [ ] Struct `HostedscanReport` dengan semua field
- [ ] Fungsi `GenerateHostedscanReport(scan)` → hitung severity, risk score, format findings
- [ ] Endpoint `GET /api/scan/{id}/report/hostedscan` (JSON)
- [ ] Endpoint `GET /api/scan/{id}/report/html` (HTML standalone)

Frontend:
- [ ] Tab "Report" di dashboard (setelah tab Results)
- [ ] Fetch `/api/scan/{id}/report/hostedscan` dan render
- [ ] UI cards: Executive Summary, Findings list (sortable by severity)
- [ ] Tombol "Download HTML", "Print PDF", "Export JSON"
- [ ] Styling ala hostedscan (dark mode, severity badges, code blocks untuk evidence)

Template:
- [ ] `web/templates/hostedscan-report.html` — template HTML standalone
- [ ] CSS print-friendly (page breaks, color untuk screen/print)

Testing:
- [ ] Run Regular scan → verify report endpoint returns valid JSON
- [ ] Verify severity classification correct (critical/high/medium/low/info)
- [ ] Verify risk score calculation
- [ ] Download HTML → open offline → verify styling & content
- [ ] Print PDF → verify layout & readability

---

## Prioritas (diskusi dengan user)

**Pertanyaan**:
1. Format HTML standalone dulu, atau JSON API dulu?
2. Mau severity otomatis (dari CVSS) atau manual classification per tool output?
3. Remediation advice: generic (dari CWE database) atau custom per finding?
4. UI report: inline di dashboard atau popup modal?

---

## Referensi

- hostedscan.com — contoh laporan profesional (executive summary, detailed findings)
- CVSS v3.1 calculator — severity scoring
- OWASP WSTG — remediation best practices
- CWE Top 25 — weakness classification

**File terkait**:
- `internal/playbook/report.go` — sudah ada `ReportData.ToHTML()` untuk Full Scan AI
- `internal/models/models.go` — struct `Finding` dengan `Severity`, `CWE`, `OWASP`
- `web/index.html` — tab Report untuk AI report (bisa di-extend untuk hostedscan format)

---

**Status**: Diskusi — belum ada eksekusi kode
