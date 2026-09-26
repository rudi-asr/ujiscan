---
name: ujiscan-vuln-idor
description: "Insecure Direct Object Reference / BOLA-BOPLA. Use for object endpoints (user, order, file, invoice). Triggers: idor, bfla, object reference, horizontal access, vertical access."
version: 1.0.0
phase: ["vulnscan"]
category: ["vulnscan"]
tools: ["curl", "ffuf"]
tags: ["vuln", "detection", "verification"]
---

# IDOR Detection# IDOR / BOLA / BOPLA Detection

## Konsep
Akses objek dengan mengubah identifier tanpa otorisasi:
- BOLA: horizontal (id user lain)
- BOPLA: vertical (id admin)

## Langkah Pengujian

### 1. Identifikasi endpoint objek
`/api/users/123`, `/invoice/456`, `/file/download?id=789`, `/order/AB12`

### 2. Uji dengan dua akun (jika mungkin)
```bash
# Akun A: ambil ID objek milik akun B dari history/list
curl -s -H "Authorization: Bearer A" http://target/api/users/B_ID

# Bandingkan respons: 200 vs 403
```

### 3. Brute ID sederhana (rate-limited, wajar)
```bash
# UUID sulit ditebak — fokus integer sequence
for id in $(seq 100 110); do
  curl -s -o /dev/null -w "%{http_code} %{size_download}\n" \
    -H "Authorization: Bearer $TOKEN" "http://target/api/users/$id"
done
```

### 4. UUID enumeration via mass-assignment / metadata
Periksa apakah ID objek bocor di daftar publik (list-order dihitung).

## Klasifikasi
- BOLA dengan data pribadi (PII): HIGH
- BOPLA (admin data): CRITICAL
- Mass assignment: HIGH

## Verifikasi
1. Bukti respons 200 dengan data objek pihak lain
2. Tanpa manipulasi header/role — hanya ID
3. JANGAN download/ubah data massal — cukup bukti akses

