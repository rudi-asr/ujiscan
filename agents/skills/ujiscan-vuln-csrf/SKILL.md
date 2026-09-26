---
name: ujiscan-vuln-csrf
description: "Cross-Site Request Forgery — state-changing requests without anti-CSRF. Use for POST/forms, state changes. Triggers: csrf, cross site request forgery, x-csrf-token."
version: 1.0.0
phase: ["vulnscan"]
category: ["vulnscan"]
tools: ["curl"]
tags: ["vuln", "detection", "verification"]
---

# CSRF Detection# CSRF Detection

## Konsep
Request state-changing (transfer, ubah password, upload) dapat dipicu dari situs lain karena tidak ada token CSRF / double-submit cookie / SameSite.

## Langkah Pengujian

### 1. Identifikasi request state-changing
POST form penting: ubah email, password, transfer, upload, delete.

### 2. Cek mekanisme pelindung
```bash
# Apakah ada token CSRF di form/header?
curl -s http://target/account | grep -iE "csrf|token|_token"
curl -s -X POST http://target/change-password \
  -H "Content-Type: application/json" -d '{"new":"x"}'
# Jika 200 tanpa token → berpotensi CSRF
```

### 3. Cek cookie SameSite
```
# Response cookie: SameSite=None? Lax? Strict?
```
Jika SameSite=None + tanpa token → CSRF valid.

## Klasifikasi
- State-change tanpa token + SameSite tidak Strict: MEDIUM
- + aksi sensitif (transfer/ubah password): HIGH

## Verifikasi
1. Request berhasil TANPA token dan TANPA CSRF header
2. Bukti: respons sukses + tidak ada mekanisme token
3. JANGAN eksekusi aksi merusak (cukup bukti teknis)

