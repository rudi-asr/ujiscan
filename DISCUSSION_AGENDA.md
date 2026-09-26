# ujiscan — Agenda Diskusi (disimpan untuk dibahas kembali)

> Status: PENDING — akan ditampilkan/dibahas ulang atas permintaan user.
> Tanggal dibuat: 2026-09-26

---

## Topik Diskusi yang Ditunda

### 1. Keamanan API Key DeepSeek
- Rotasi key: key `sk-25087...` pernah lewat chat + terminal → praktik terbaik = ganti di platform.deepseek.com
- Simpan key di `~/.ujiscan.env` (luar repo), bukan command line `docker run -e`
- Tambah `*.env` ke .gitignore (saat ini belum ada proteksi)
- Production: pakai secret manager (Docker secrets / CI env), bukan file
- Catatan: key TIDAK pernah masuk git/repo (terverifikasi audit)

### 2. Arsitektur Full Scan AI
- Katalog 139 tools (24 lokal, 115 remote) — sudah bekerja, AI pilih tools nyata
- Flow adaptif: recon → enum → vulnscan → report, AI putuskan lanjut/stop per fase
- Yang mungkin kurang: eksekusi tools remote (115 tools katalog belum benar-benar bisa dijalankan, baru rekomendasi)
- Opsi: install on-demand di container vs runner remote (agent di server terpisah)

### 3. Roadmap Selanjutnya
- Ekspansi skill vuln: 10 → ~40 skill (pola OmOP)
- Domain ujiscan.rudilab.my.id → arahkan ke Cloudflare tunnel (saat ini GitHub Pages + tunnel aktif)
- Laporan PDF: saat ini print browser; opsi generate PDF server-side
- Scan history persistence: saat ini in-memory (hilang saat container restart)

### 4. Produksi / Deployment Nyata
- Saat ini: localhost + GitHub Pages + Cloudflare quick tunnel
- Untuk production: domain permanen, HTTPS, container restart policy, volume untuk DB
- Multi-user & RBAC: saat ini admin tunggal (admin@ujiscan.local)
- Audit log & compliance

### 5. Keamanan Frontend
- Token JWT di localStorage (risiko XSS) — opsi: httpOnly cookie, short-lived token
- CORS whitelist
- Rate limiting API publik (tunnel ekspos ke internet)

---

## Keputusan yang Sudah Disepakati (referensi)

- **3-tier scanning**: Reguler (9 tools hostedscan) / Quick (user pilih) / Full AI (agentic)
- **OmOP-style**: kunci ada di file .md rahasia (agents/), BUKAN di LLM; dibuat dari scratch (bukan copas, aman HAKI)
- **Laporan**: ala hostedscan, web/HTML dulu + print/save PDF
- **Full Scan AI**: 3 engine (DeepSeek Flash / OpenAI / Claude), tools TIDAK perlu diinstall lokal
- **Dark mode default** (aturan rudilab)

---

## Status Saat Ini (2026-09-26)

- ✅ Backend: Docker localhost:8081, 9 tools terinstall, Full Scan AI bekerja (DeepSeek real)
- ✅ Frontend: redesign profesional (sidebar, dashboard, report, AI decisions, tool catalog)
- ✅ Katalog tools: 139 (tools-catalog.json)
- ✅ Skills: 4 fase + 10 vuln spesifik (agents/skills/)
- ✅ GitHub Pages live: https://rudi-asr.github.io/ujiscan/
- ⏳ Pending diskusi: kelima topik di atas