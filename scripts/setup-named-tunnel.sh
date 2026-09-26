#!/bin/bash
# Setup Cloudflare Named Tunnel untuk ujiscan.rudilab.my.id
# Run: bash setup-named-tunnel.sh

set -e

TUNNEL_NAME="ujiscan-prod"
HOSTNAME="ujiscan.rudilab.my.id"
LOCAL_SERVICE="http://localhost:8081"

echo "=== Cloudflare Named Tunnel Setup ==="
echo ""
echo "Tunnel name: $TUNNEL_NAME"
echo "Hostname: $HOSTNAME"
echo "Local service: $LOCAL_SERVICE"
echo ""

# Step 1: Login Cloudflare (interactive)
echo "Step 1: Login to Cloudflare..."
echo "Browser akan terbuka untuk login. Pilih domain rudilab.my.id"
cloudflared tunnel login

if [ $? -ne 0 ]; then
    echo "❌ Login failed. Pastikan akun Cloudflare punya akses ke rudilab.my.id"
    exit 1
fi

echo "✅ Login successful"
echo ""

# Step 2: Create tunnel
echo "Step 2: Creating tunnel '$TUNNEL_NAME'..."
cloudflared tunnel create $TUNNEL_NAME

if [ $? -ne 0 ]; then
    echo "⚠️  Tunnel mungkin sudah ada. Listing tunnels..."
    cloudflared tunnel list
fi

echo ""

# Step 3: Get tunnel ID
TUNNEL_ID=$(cloudflared tunnel list | grep $TUNNEL_NAME | awk '{print $1}')
echo "Tunnel ID: $TUNNEL_ID"

if [ -z "$TUNNEL_ID" ]; then
    echo "❌ Tunnel ID not found"
    exit 1
fi

# Step 4: Create config
echo ""
echo "Step 3: Creating config..."
mkdir -p ~/.cloudflared

cat > ~/.cloudflared/config.yml <<EOF
tunnel: $TUNNEL_ID
credentials-file: /Users/rudi/.cloudflared/${TUNNEL_ID}.json

ingress:
  - hostname: $HOSTNAME
    service: $LOCAL_SERVICE
  - service: http_status:404
EOF

echo "✅ Config created at ~/.cloudflared/config.yml"
cat ~/.cloudflared/config.yml
echo ""

# Step 5: Route DNS
echo "Step 4: Routing DNS $HOSTNAME to tunnel..."
cloudflared tunnel route dns $TUNNEL_NAME $HOSTNAME

echo ""
echo "✅ Setup complete!"
echo ""
echo "To start tunnel:"
echo "  cloudflared tunnel run $TUNNEL_NAME"
echo ""
echo "Or run as daemon:"
echo "  nohup cloudflared tunnel run $TUNNEL_NAME > /tmp/ujiscan-tunnel-named.log 2>&1 &"
echo ""
echo "Access: https://$HOSTNAME"
