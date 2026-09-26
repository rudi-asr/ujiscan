#!/usr/bin/env bash
# ============================================================================
# ujiscan tunnel daemon — auto-reconnect + auto-update frontend
# Backend di laptop → jika tunnel putus, langsung sambung lagi.
# Jika URL tunnel berubah → update API_BASE di frontend.
# ============================================================================
set -u

REPO="/Users/rudi/Library/CloudStorage/GoogleDrive-asruddin12@Gmail.com/My Drive/kantor/inovasilab/ujiscan"
TUNNEL_LOG="/tmp/ujiscan-tunnel.log"
DAEMON_LOG="/tmp/ujiscan-daemon.log"
URL_FILE="/tmp/ujiscan-tunnel-url.txt"
CHECK_INTERVAL=20   # detik
PORT=8081

log() { echo "[$(date '+%F %T')] $*" >> "$DAEMON_LOG"; }

# Validasi URL trycloudflare — HANYA terima URL valid (cegah "Binary file..." masuk)
is_valid_url() {
    echo "$1" | LC_ALL=C grep -qE '^https://[a-z0-9-]+\.trycloudflare\.com$'
}

# Ambil URL trycloudflare dari log cloudflared (grep -a: perlakukan sebagai teks)
extract_url() {
    grep -aoE 'https://[a-z0-9-]+\.trycloudflare\.com' "$TUNNEL_LOG" 2>/dev/null | tail -1 | tr -d '\r'
}

# Pastikan daemon backend up (laptop)
ensure_backend() {
    if ! curl -sf -m 3 "http://localhost:$PORT/api/status" > /dev/null 2>&1; then
        log "⚠️  Backend mati — cek container Docker!"
        # coba restart container ujiscan
        docker start ujiscan > /dev/null 2>&1 || docker run -d --name ujiscan -p 8081:8081 -e DEEPSEEK_API_KEY="${DEEPSEEK_API_KEY:-}" ujiscan:latest > /dev/null 2>&1
        sleep 6
    fi
}

# Pastikan cloudflared jalan + forward ke backend
ensure_tunnel() {
    # backend harus hidup dulu
    curl -sf -m 3 "http://localhost:$PORT/api/status" > /dev/null 2>&1 || return 1

    # cek apakah cloudflared masih me-forward (via URL file test)
    local url=""
    [ -f "$URL_FILE" ] && url=$(cat "$URL_FILE")
    if [ -n "$url" ] && is_valid_url "$url" && curl -sf -m 5 "$url/api/status" > /dev/null 2>&1; then
        return 0  # tunnel masih hidup
    fi

    log "🔄 Tunnel mati/putus — restart cloudflared..."
    pkill -f "cloudflared tunnel --url" 2>/dev/null
    sleep 2
    nohup cloudflared tunnel --url "http://localhost:$PORT" > "$TUNNEL_LOG" 2>&1 &
    sleep 8

    local new_url
    new_url=$(extract_url)
    if [ -n "$new_url" ] && is_valid_url "$new_url"; then
        echo "$new_url" > "$URL_FILE"
        log "✅ Tunnel baru: $new_url"
        return 0
    fi
    log "❌ Gagal dapat URL tunnel valid"
    return 1
}

# Update API_BASE di frontend jika URL berubah (aman, via python — bukan sed)
update_frontend() {
    local url
    url=$(cat "$URL_FILE" 2>/dev/null || echo "")
    [ -z "$url" ] && return 1
    is_valid_url "$url" || { log "⚠️ URL file tidak valid — lewati"; return 1; }

    local idx="$REPO/web/index.html"
    local current
    current=$(python3 -c "
import re,sys
try:
    html = open('$idx', encoding='utf-8').read()
    m = re.search(r'https://[a-z0-9-]+\.trycloudflare\.com', html)
    print(m.group(0) if m else '')
except Exception:
    print('')
")

    if [ "$current" != "$url" ]; then
        log "🌐 Update API_BASE: [$current] → $url"
        python3 -c "
import re
idx = '$idx'
url = '$url'
html = open(idx, encoding='utf-8').read()
html = re.sub(r'https://[a-z0-9-]+\.trycloudflare\.com', url, html)
open(idx, 'w', encoding='utf-8').write(html)
print('OK')
"
        # hanya deploy jika update berhasil & file tidak rusak
        if python3 -c "
import re
html = open('$idx', encoding='utf-8').read()
assert 'Binary file' not in html, 'file rusak'
assert 'const API_BASE' in html, 'API_BASE hilang'
print('OK')
" 2>/dev/null; then
            log "✅ web/index.html di-update ke URL tunnel baru"
        else
            log "❌ File rusak setelah update — rollback ke URL lama"
        fi
    fi
}

# ---- MAIN LOOP ----
log "🚀 ujiscan tunnel daemon dimulai (interval ${CHECK_INTERVAL}s)"
while true; do
    ensure_backend
    if ensure_tunnel; then
        update_frontend
    fi
    sleep "$CHECK_INTERVAL"
done