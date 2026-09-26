---
name: ujiscan-vuln-open-redirect
description: "Open redirect — target redirects to attacker-controlled domain. Use for login/logout/next/return params. Triggers: open redirect, url redirect, next param."
version: 1.0.0
phase: ["vulnscan"]
category: ["vulnscan"]
tools: ["curl", "ffuf"]
tags: ["vuln", "detection", "verification"]
---

# OPEN-REDIRECT Detection# Open Redirect Detection

## Langkah Pengujian

### 1. Temukan param redirect
`?next=`, `?redirect=`, `?return=`, `?url=`, `?dest=`, `?continue=`, `?goto=`

### 2. Probe variasi
```bash
curl -sI "http://target/login?next=https://evil.com"
curl -sI "http://target/logout?return=//evil.com"      # protokol-relative
curl -sI "http://target/?url=https://evil.com%2f%2f"   # encoding
curl -sI "http://target/?redirect=javascript:alert(1)" # scheme non-http

# Bypass whitelist umum
curl -sI "http://target/?next=https://evil.com@target.com"
curl -sI "http://target/?next=https://target.com.evil.com"
curl -sI "http://target/?next=https://evil.com/target.com"
```

### 3. Cek header Location
`-I` → response header `Location: <redirect-url>`.

## Klasifikasi
- Open redirect terkonfirmasi: LOW-MEDIUM
- + digunakan untuk OAuth token theft: HIGH

## Verifikasi
1. Header Location menunjuk domain attacker
2. Re-run lintas method (GET/POST)
3. JANGAN klik link di report — cukup bukti header

