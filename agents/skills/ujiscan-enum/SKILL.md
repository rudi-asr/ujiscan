---
name: ujiscan-enum
description: "Enumeration phase — service discovery, port scan, teknologi web. Use after recon to map live services. Triggers: enum, portscan, service, fingerprint, http probe."
version: 1.0.0
phase: ["enumeration"]
category: ["enumeration"]
tools: ["nmap", "masscan", "httpx", "whatweb", "sslscan", "gobuster", "ffuf"]
tags: ["enum", "port", "service", "web", "fingerprint"]
---

# ujiscan Enumeration

Fase kedua **ujiscan Full Scan AI**. Memetakan service yang benar-benar hidup
di atas permukaan yang ditemukan fase recon. Referensi: OWASP WSTG-INFO-07
(service footprint) hingga WSTG-CONF (configuration).

## Tujuan Fase

1. Temukan port terbuka & service yang berjalan (dengan versi).
2. Identifikasi layanan web: status, title, teknologi, framework.
3. Audit SSL/TLS pada service HTTPS.
4. Temukan endpoint/directory tersembunyi (content discovery).

## Urutan Kerja

### 1. Port Scan Terarah — `nmap`

```bash
# Cepat: port umum (fase recon selesai = target hidup sudah pasti)
nmap -sV -T4 -p 22,80,443,3306,5432,6379,8080,8443,8000,5000,3000 <target>

# Lengkap (hanya jika target kecil / sudah disepakati)
nmap -sV -sC -T4 <target>
```

Pola analisis: port 80/443 → web; 22 → SSH (versi?); 3306/5432 → database
(IP publik db = temuan); 8080/8443 → alternate web; 6379 → Redis (RCE risk).

### 2. Probing Web — `httpx`

```bash
httpx -u http://<target> -status-code -title -tech-detect
```

Catat: redirect chain, status, title, teknologi. Title mengandung kata
"login", "admin", "phpMyAdmin", "Directory Listing" = prioritas tinggi.

### 3. Fingerprint — `whatweb`

```bash
whatweb <target>
```

Deteksi: CMS (WordPress/Joomla/Drupal), framework (Laravel/Django/Express),
server (nginx/apache/IIS). Hasil ini menentukan template vulnscan.

### 4. SSL/TLS — `sslscan`

```bash
sslscan <target>:443
```

Catat: protokol lemah (SSLv3, TLSv1.0), cipher lemah (RC4, DES, 3DES),
sertifikat (expired? self-signed? CN cocok?).

### 5. Content Discovery — `gobuster` / `ffuf`

```bash
gobuster dir -u http://<target> -w /usr/share/wordlists/dirb/common.txt \
  -x php,html,txt,json,bak,old
```

Catat: `backup`, `config`, `.git`, `.env`, `admin`, `api`, `swagger`,
`graphql`, `uploads` — semua kandidat temuan.

## Daftar Periksa Teknologi (untuk keputusan vulnscan)

| Ditemukan | Indikasi | Fase selanjutnya fokus |
|-----------|----------|------------------------|
| WordPress | plugin/theme vuln | nuclei wp-* templates |
| Laravel (debug) | APP_DEBUG leak | info disclosure check |
| phpMyAdmin | auth exposure | weak creds / version vuln |
| Swagger/API docs | endpoint list | API testing |
| GraphQL | introspection | schema exposure |
| Admin panel | login page | auth testing |
| .git exposed | source leak | git dump analyze |
| S3 bucket | misconfig | s3 enum |

## Output Fase

```
{
  "phase": "enumeration",
  "summary": "Port 22/80/443 terbuka; WordPress 6.4; TLS1.2 OK; /admin & /.env ditemukan",
  "artifacts": ["ports.json", "tech.json", "dirs.txt"],
  "recommendation": "vulnscan — fokus WordPress + .env exposure + api subdomain",
  "confidence": 0.0-1.0
}
```

## Aturan Khusus Fase

- nmap full `-sC` hanya jika port scan cepat menunjukkan target kecil.
- Rate limit content discovery (jangan banjiri target dengan 100 req/detik).
- Jika WAF terdeteksi (dari recon), pertimbangkan response normalization.