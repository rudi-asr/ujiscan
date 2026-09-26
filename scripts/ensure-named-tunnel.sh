#!/bin/bash
# Cloudflare Named Tunnel Daemon for ujiscan
# Ensures tunnel ujiscan-prod is always running
# URL: https://ujiscan.rudilab.my.id (PERMANENT)

TUNNEL_NAME="ujiscan-prod"
LOG_FILE="/tmp/ujiscan-tunnel-named.log"
HEALTH_CHECK_URL="http://localhost:8081/api/status"

echo "[$(date)] Starting Cloudflare Named Tunnel daemon for $TUNNEL_NAME..."

while true; do
    # Check if tunnel process is running
    if ! pgrep -f "cloudflared tunnel run $TUNNEL_NAME" > /dev/null; then
        echo "[$(date)] Tunnel not running. Starting..."
        cloudflared tunnel run $TUNNEL_NAME > "$LOG_FILE" 2>&1 &
        sleep 10
    fi
    
    # Health check backend
    if ! curl -sf "$HEALTH_CHECK_URL" > /dev/null 2>&1; then
        echo "[$(date)] Backend down at $HEALTH_CHECK_URL"
    fi
    
    # Check every 30 seconds
    sleep 30
done
