---
name: ujiscan-vuln-ssl
description: "SSL/TLS audit — weak protocols, ciphers, cert issues. Use before/after HTTPS targets. Triggers: ssl, tls, heartbleed, poodle, certificate, weak cipher."
version: 1.0.0
phase: ["vulnscan"]
category: ["vulnscan"]
tools: ["sslscan", "testssl.sh", "nmap"]
tags: ["vuln", "detection", "verification"]
---

# SSL Detection# SSL/TLS Audit

## Langkah Pengujian

### 1. Scan dasar
```bash
sslscan target:443
# atau deep
testssl.sh target:443
```

### 2. Fokus temuan umum
- Protokol lemah: SSLv2, SSLv3 (POODLE), TLSv1.0, TLSv1.1
- Cipher lemah: RC4, DES, 3DES, CBC-mode ECDHE
- Cipher export-grade
- Perfect Forward Secrecy TIDAK tersedia
- Sertifikat: expired, self-signed, CN mismatch, SHA-1 signature
- Heartbleed (OpenSSL 1.0.1-1.0.1f)

### 3. Nmap script
```bash
nmap -sV --script ssl-enum-ciphers -p 443 target
nmap -sV --script ssl-heartbleed -p 443 target
```

## Klasifikasi
- Heartbleed aktif: CRITICAL
- TLSv1.0/1.1 + cipher lemah: MEDIUM
- Sertifikat expired: MEDIUM
- Sertifikat self-signed: LOW-INFO

## Verifikasi
1. Output tool menunjukkan protocol/cipher spesifik
2. Re-run 2 tools untuk cross-check

