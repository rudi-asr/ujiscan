---
name: ujiscan-vuln-ssti
description: "Server-Side Template Injection. Use for email templates, preview, render, invoice templates. Triggers: ssti, template injection, jinja, twig, freemarker."
version: 1.0.0
phase: ["vulnscan"]
category: ["vulnscan"]
tools: ["curl", "nuclei"]
tags: ["vuln", "detection", "verification"]
---

# SSTI Detection# SSTI Detection

## Langkah Pengujian (deteksi dulu, exploiter butuh izin)

### 1. Temukan input template
Email template preview, invoice number formatting, error page render, profile "about" text.

### 2. Probe math expression (deteksi universal)
```bash
# Input: 7*7 → 49 berarti template engine mengeksekusi
curl -s -X POST http://target/preview -d '{"name":"{{7*7}}"}'
curl -s -X POST http://target/preview -d '{"name":"${7*7}"}'    # alt syntax
curl -s -X POST http://target/preview -d '{"name":"{{7*'7'}}"}' # string concat
```

### 3. Identifikasi engine dari error/syntax behavior
- `{{7*7}}` → Jinja2 (Python), Twig (PHP)
- `${7*7}` → FreeMarker, Velocity (Java)
- `#{7*7}` → Ruby

### 4. Deep probe HANYA dengan izin eksploitasi
Jika user menyetujui, lanjut ke read-file / RCE — TANPA itu, cukup bukti evaluasi.

## Klasifikasi
- Evaluasi template terkonfirmasi: HIGH
- + RCE terverifikasi (dengan izin): CRITICAL
- Sandbox escape: CRITICAL

## Verifikasi
1. Output berisi hasil evaluasi (49 untuk 7*7)
2. Re-run 2 engine syntax untuk eliminasi false positive

