# Security Policy — ujiscan

## Prinsip Keamanan 3-Layer

### 1. **Lokal (Host)**
- ✅ File `.env` tidak pernah di-commit (ada di `.gitignore`)
- ✅ Cloudflared credentials (`~/.cloudflared/*.json`) tidak di-commit
- ✅ Database & logs (`data/`, `logs/`) tidak di-commit
- ✅ Tunnel credentials hanya di host (tidak masuk container)

### 2. **Container (Docker)**
- ✅ **Non-root user**: Container run as UID 1000:1000 (bukan root)
- ✅ **Read-only filesystem**: Root FS read-only, hanya `/app/data` & `/app/logs` writable
- ✅ **No new privileges**: Prevent privilege escalation (`security_opt`)
- ✅ **Resource limits**: Max 2GB RAM, 2 CPU cores (prevent DoS)
- ✅ **Network isolation**: Port 8081 hanya bind ke `127.0.0.1` (tidak expose ke internet)
- ✅ **Log rotation**: Max 3 files × 10MB (prevent disk fill)
- ✅ **Secrets via env**: API keys dari `.env` file (tidak hardcode di image)

### 3. **GitHub (Repository)**
- ✅ **`.gitignore` comprehensive**: Semua secrets, credentials, data runtime
- ✅ **No hardcoded secrets**: API keys tidak pernah di code
- ✅ **`.env.example`** template (tanpa key asli)
- ✅ **Dependabot**: GitHub security alerts enabled
- ✅ **Branch protection**: Main branch protect (jika public repo)

---

## Secrets Management

### File yang TIDAK BOLEH di-commit:
```
.env                        # API keys
.env.local                  # Local overrides
data/                       # Database & scans
logs/                       # Application logs
scans.json                  # Scan persistence
*.key, *.pem, *.crt         # Certificates
config.local.yaml           # Local config
secrets/                    # Secret directory
.cloudflared/*.json         # Tunnel credentials
```

### File yang AMAN di-commit:
```
.env.example                # Template (no real keys)
docker-compose.yml          # Tidak ada secrets
playbooks/*.md              # Playbook publik
web/                        # Frontend (no secrets)
```

---

## Docker Security Checklist

- [x] Container run as non-root (UID 1000:1000)
- [x] Read-only root filesystem
- [x] No new privileges (`security_opt`)
- [x] Resource limits (CPU + RAM)
- [x] Port bind ke localhost only (`127.0.0.1:8081`)
- [x] Network isolated (no ICC)
- [x] Log rotation configured
- [x] Health check enabled
- [x] Auto-restart (`unless-stopped`)
- [x] Secrets via `.env` file (not in image)

---

## Deployment Security

### ✅ Production Checklist:
1. **Tunnel**: Cloudflare Named Tunnel (HTTPS only)
2. **Authentication**: JWT tokens (tidak ada anonymous access)
3. **Rate limiting**: 20-30 req/min per IP
4. **CORS**: Whitelist domains only
5. **Security headers**: CSP, HSTS, X-Frame-Options
6. **Database**: File permissions 600 (`chmod 600 data/ujiscan.db`)
7. **Logs**: Rotate & archive (tidak simpan selamanya)
8. **Backups**: Database backup ke storage terpisah

### ⚠️ JANGAN:
- ❌ Commit `.env` file
- ❌ Expose port 8081 ke internet (hanya via tunnel)
- ❌ Run container as root
- ❌ Hardcode API keys di code
- ❌ Disable security headers
- ❌ Share tunnel URL publicly (internal use only)

---

## Incident Response

**Jika API key bocor**:
1. Rotasi key di provider (DeepSeek/OpenAI/Claude)
2. Update `.env` file dengan key baru
3. Restart container: `docker-compose restart`
4. Audit GitHub commits: `git log --all -S "sk-" --source`
5. Jika ada di history: `git filter-branch` atau BFG Repo Cleaner

**Jika database bocor**:
1. Reset semua user passwords
2. Invalidasi semua JWT tokens (restart backend)
3. Audit access logs
4. Restore dari backup terakhir yang aman

---

## Security Updates

- **Docker base image**: Update reguler (`docker pull debian:bookworm-slim`)
- **Go dependencies**: `go get -u` & rebuild
- **Security tools**: Update nmap, nuclei, nikto via `docker build`
- **Monitoring**: Cek GitHub Dependabot alerts

---

**Last updated**: 2026-09-27  
**Contact**: Security issues → private disclosure (bukan public GitHub issue)
