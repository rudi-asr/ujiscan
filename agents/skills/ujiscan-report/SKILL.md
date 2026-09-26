---
name: ujiscan-report
description: "Final report generation — executive summary, findings table, risk prioritization. Use after all phases complete. Triggers: report, summary, finalize, export, print pdf."
version: 1.0.0
phase: ["report"]
category: ["report"]
tools: []
tags: ["report", "summary", "risk", "export"]
---

# ujiscan Report Generation

Fase akhir **ujiscan Full Scan AI**. Mengubah temuan terverifikasi menjadi
laporan profesional ala hostedscan yang bisa diprint/diexport PDF dari web.

## Struktur Laporan Wajib

### 1. Executive Summary
- 2-3 kalimat non-teknis, friendly untuk manajemen/klien.
- Fokus: postur risiko keseluruhan + urgensi, bukan nama tool.
- Formula: "Assessment of <target> completed; <N> confirmed findings
  (<X> critical, <Y> high). Primary risk: <top risk>. Immediate action:
  <satu rekomendasi terpenting>."

### 2. Scope & Metodologi
- Target, scan mode (reguler/quick/full-ai), AI engine dipakai.
- Fase yang dijalankan (recon → enum → vulnscan) + tools.

### 3. Findings Table
| Severity | Title | Tool | Deskripsi | Bukti |
|----------|-------|------|-----------|-------|
- Sorting: CRITICAL → HIGH → MEDIUM → LOW → INFO.
- Setiap finding wajib punya evidence (output tool / curl PoC).

### 4. Tool Execution Log
- Tool, status (success/fail), exit code, durasi, output preview.

### 5. Rekomendasi Prioritas
- Top 3 remediasi dengan alasan bisnis singkat.

## Severity Scoring (konsisten dengan vulnscan)

| Severity | Rentang dampak | Contoh |
|----------|----------------|--------|
| CRITICAL | Kompromi total / data breach langsung | RCE, SQLi dump mungkin, exposed .env+cred |
| HIGH | Akses terbatas / sistem penting terdampak | Auth bypass, path traversal |
| MEDIUM | Informasi bocor / kondisi lemah | XSS reflektif, missing headers |
| LOW | Hygiene minor | version banner, cookie flag |
| INFO | Fakta teknis tanpa risiko langsung | tech fingerprint, open service |

## Format Export

1. **HTML (default)** — tampil di tab Report, bisa `window.print()` → "Save as PDF"
2. **JSON** — untuk integrasi/otomasi
3. **Markdown** — untuk dokumentasi

## Aturan Khusus

- Jangan menulis temuan yang tidak terverifikasi (fase vulnscan).
- Jangan menyebut "kemungkinan" tanpa konteks — tulis confidence.
- Jangan bocorkan: API keys, credentials target, data pribadi.
- Executive summary harus bisa dibaca non-teknis dalam 30 detik.