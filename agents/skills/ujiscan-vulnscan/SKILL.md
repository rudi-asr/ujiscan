---
name: ujiscan-vulnscan
description: "Vulnerability scan phase — template-based detection + specific vuln checks based on enum results. Use after enumeration. Triggers: vulnscan, nuclei, vulnerability, CVE check, ssl audit."
version: 1.0.0
phase: ["vulnscan"]
category: ["vulnscan"]
tools: ["nuclei", "nikto", "sslscan", "dalfox", "sqlmap", "testssl.sh"]
tags: ["vuln", "cve", "template", "xss", "sqli"]
---

# ujiscan Vulnerability Scan

Fase ketiga **ujiscan Full Scan AI**. Deteksi kerentanan berdasarkan teknologi
yang ditemukan fase enum. Referensi: OWASP WSTG (v4.2) bagian Testing,
nuclei template ecosystem, CVE databases.

## Prinsip Non-Destruktif

- Hanya template/request yang TIDAK merusak atau mengubah data target.
- sqlmap hanya `--batch --technique=BEUSTQ` pada parameter yang jelas
  (tanpa `--dump` kecuali user menyetujui).
- Tidak ada exploitation aktif / reverse shell / upload webshell.

## Urutan Kerja

### 1. Template Scan Luas — `nuclei`

```bash
# Scan default (semua template yang relevan berdasarkan tech)
nuclei -u http://<target> -silent

# Fokus tech tertentu hasil enum (contoh: wordpress)
nuclei -u http://<target> -tags wordpress -silent

# Eksposur config penting
nuclei -u http://<target> -tags exposure,config -silent
```

Pola analisis: bedakan `template matched` vs `false positive`.
Template yang butuh konteks (mis. "possible XSS") = kandidat verifikasi.

### 2. Web Server Check — `nikto`

```bash
nikto -h http://<target>
```

Catat: server misconfig, file default, header tidak aman (X-Frame-Options
hilang, HSTS absen), CORS longgar.

### 3. SSL/TLS Deep — `sslscan` / `testssl.sh`

```bash
sslscan <target>:443
# atau deep
testssl.sh <target>:443
```

### 4. Parameter Fuzzing — `dalfox` (XSS)

```bash
dalfox url http://<target>/<endpoint>?param=value --silence
```

Hanya jalankan pada parameter yang jelas memantulkan input.

### 5. SQLi Detection — `sqlmap`

```bash
sqlmap -u "http://<target>/page?id=1" --batch --technique=BEUSTQ --level=1
```

Hanya mode deteksi aman. Temuan SQLi = lapor dengan PoC minimal.

## Klasifikasi Findings

| Severity | Contoh |
|----------|--------|
| CRITICAL | RCE (template terkonfirmasi), SQLi terkonfirmasi, exposed .env dengan secret |
| HIGH | Auth bypass, path traversal, known CVE dengan PoC publik di service penting |
| MEDIUM | XSS reflektif, SSL/TLS lemah, missing security headers, directory listing |
| LOW | Info disclosure minor, version exposed, cookie tanpa HttpOnly |
| INFO | Tech fingerprint, port terbuka service minor |

## Verifikasi Wajib (sebelum masuk laporan)

Untuk setiap finding CRITICAL/HIGH:
1. Re-run tool sekali (konfirmasi reproducibility).
2. Jika template-based: cek manual via `curl` yang tidak berbahaya.
3. Catat bukti output tool persis (jangan di-summarize berlebihan).

## Output Fase

```
{
  "phase": "vulnscan",
  "findings_count": 5,
  "critical": 1, "high": 2, "medium": 1, "low": 1,
  "summary": "SQLi pada /product?id= (terkonfirmasi), XSS reflektif pada /search, .env exposed",
  "recommendation": "verifikasi & laporan — prioritas SQLi + .env exposure",
  "confidence": 0.0-1.0
}
```

## Aturan Khusus Fase

- JANGAN auto-exploit. Deteksi & verifikasi non-destruktif saja (sesuai
  AGENTS.md §00 safety).
- Finding tanpa bukti output = tidak pernah masuk laporan.
- Severity rendah dengan evidence lemah → downgrade ke INFO.