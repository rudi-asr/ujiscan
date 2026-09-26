---
name: ujiscan-recon
description: "Reconnaissance phase — passive & active footprinting. Use when starting a Full Scan AI on a new target. Triggers: recon, footprint, subdomain, dns, osint."
version: 1.0.0
phase: ["recon"]
category: ["recon"]
tools: ["dig", "subfinder", "amass", "theHarvester", "wafw00f", "gitleaks"]
tags: ["recon", "osint", "dns", "subdomain"]
---

# ujiscan Reconnaissance

Bagian dari **ujiscan Full Scan AI**. Fase ini memetakan permukaan serangan
sebelum menjalankan tool apa pun yang lebih dalam. Referensi metodologi:
OWASP WSTG-INFO (Information Gathering) & PTES Intelligence Gathering.

## Tujuan Fase

1. Tentukan identitas target (tipe: domain/IP/URL/service).
2. Kumpulkan passive data SEBELUM menyentuh target (dan hanya data yang legal
   untuk dikumpulkan).
3. Enumerasi subdomain & record DNS.
4. Deteksi teknologi perimeter (WAF, reverse proxy, CDN).

## Urutan Kerja (rekomendasi)

### 1. DNS Dasar — `dig`

```bash
# A record
dig A <target>

# Semua record umum
dig <target> ANY +noall +answer

# Zone transfer check (jarang berhasil, cepat coba)
dig AXFR <target> @<nameserver>
```

Pola analisis: catat A/AAAA → IP publik; MX → penyedia email; NS → DNS host;
TXT → SPF/DKIM (indikasi maturitas keamanan).

### 2. Subdomain Enumeration — `subfinder`

```bash
subfinder -d <target> -silent
```

Catat subdomain yang menarik: `api.`, `admin.`, `dev.`, `staging.`,
`test.`, `internal.`, `grafana.`, `jenkins.`, `gitlab.`.
Subdomain ini adalah kandidat fase enumeration.

### 3. Fingerprint Perimeter

```bash
# Deteksi WAF
wafw00f <target>

# HTTP fingerprint via nmap (banner service)
nmap -sV -p 80,443 <target>
```

Catat: server header, teknologi, WAF vendor. Ini menentukan tool fase
vulnscan (mis. WAF aktif → pertimbangkan waf-bypass template).

## Output Fase (handoff)

```
{
  "phase": "recon",
  "target": "<target>",
  "summary": "N subdomain ditemukan, IP x.x.x.x, WAF <ada/tidak>",
  "artifacts": ["subdomains.txt"],
  "recommendation": "lanjut ke enumeration — fokus <subdomain paling menarik>",
  "confidence": 0.0-1.0
}
```

## Aturan Khusus Fase

- Jangan pernah menjalankan active scan (port sweep penuh) di fase ini —
  itu tugas fase enumeration.
- Data OSINT hanya dari sumber yang memperbolehkan otomatisasi.
- 0 subdomain ditemukan ≠ berhenti: lanjut active footprint (fase enum).