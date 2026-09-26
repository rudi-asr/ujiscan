---
name: ujiscan-vuln-ssrf
description: "Server-Side Request Forgery — target fetches attacker URLs. Use for webhooks, URL import, image proxy. Triggers: ssrf, server side request, url fetch, webhook."
version: 1.0.0
phase: ["vulnscan"]
category: ["vulnscan"]
tools: ["curl", "interactsh", "nuclei"]
tags: ["vuln", "detection", "verification"]
---

# SSRF Detection# SSRF Detection

## Konsep
Server membuat request ke URL yang dikontrol attacker (import, fetch, webhook, preview, PDF generator).

## Langkah Pengujian

### 1. Temukan fitur URL-fetch
Webhook, RSS importer, image proxy, PDF render, "preview link".

### 2. Probe dasar
```bash
# Request ke kontrol sendiri (interactsh OOB)
curl -s -X POST http://target/fetch -d '{"url":"http://YOUR-COLLABORATOR}"}'

# Lakukan ke service lokal (time/error oracle)
curl -s -X POST http://target/fetch -d '{"url":"http://127.0.0.1:22}"}'
curl -s -X POST http://target/fetch -d '{"url":"http://169.254.169.254/latest/meta-data/}"}'
```

### 3. OOB (Out-of-Band) verification
Gunakan DNS/HTTP callback (interactsh, burp collaborator) — server harus memicu
request ke domain kita untuk bukti kuat.

## Klasifikasi
- SSRF ke cloud metadata (169.254.169.254): CRITICAL
- SSRF internal port scan (error oracle): HIGH
- SSRF terbatas (hanya HTTP GET, whitelist): MEDIUM

## Verifikasi
1. Bukti OOB callback (request masuk ke server kita)
2. Atau perbedaan respons internal vs eksternal
3. JANGAN eksploitasi metadata lebih dalam tanpa izin

