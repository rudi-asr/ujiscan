---
name: ujiscan-vuln-header
description: "Missing security headers & dangerous headers (HSTS, CSP, X-Frame, nosniff, CORS). Use for any web target. Triggers: security headers, hsts, csp, x-frame-options, cors misconfiguration."
version: 1.0.0
phase: ["vulnscan"]
category: ["vulnscan"]
tools: ["curl", "nuclei"]
tags: ["vuln", "detection", "verification"]
---

# HEADER Detection# Security Headers & CORS Check

## Langkah Pengujian

### 1. Cek headers dasar
```bash
curl -sI http://target/
curl -sI https://target/ -H "Origin: https://evil.com"
```

### 2. Checklist header wajib
- Strict-Transport-Security (HSTS) — semua HTTPS
- Content-Security-Policy — XSS mitigasi
- X-Frame-Options / frame-ancestors — clickjacking
- X-Content-Type-Options: nosniff
- Referrer-Policy
- Permissions-Policy

### 3. CORS misconfiguration
```bash
# Response harus TIDAK punya Access-Control-Allow-Origin: evil.com
```
Jika ACAO = origin attacker + ACAC: true → bug.

## Klasifikasi
- Missing HSTS + CSP: MEDIUM
- CORS reflect arbitrary origin + credentials: HIGH
- X-Frame-Options hilang di login page: MEDIUM

## Verifikasi
1. Daftar header yang ADA vs HILANG
2. CORS: satu respons perlihatkan reflect origin
3. JANGAN buat proof-of-concept clickjacking aktif

