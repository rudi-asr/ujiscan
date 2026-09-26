#!/bin/bash
# Deploy ujiscan dengan docker-compose (SECURE)
# Usage: bash deploy.sh [build|start|stop|restart|logs]

set -e

ACTION="${1:-start}"

echo "=== ujiscan Secure Deployment ==="
echo "Action: $ACTION"
echo ""

# Security checks
check_security() {
    echo "🔒 Security checks..."
    
    # 1. Check .env exists
    if [ ! -f .env ]; then
        echo "⚠️  .env file not found. Creating from template..."
        cp .env.example .env
        echo "⚠️  EDIT .env and add your API keys before starting!"
        exit 1
    fi
    
    # 2. Check .env not in git
    if git check-ignore .env > /dev/null 2>&1; then
        echo "✅ .env is gitignored"
    else
        echo "❌ .env NOT in .gitignore! Adding..."
        echo ".env" >> .gitignore
    fi
    
    # 3. Check directory permissions
    mkdir -p data logs
    chmod 700 data logs
    echo "✅ data/ & logs/ secured (700)"
    
    echo ""
}

case "$ACTION" in
    build)
        echo "🔨 Building image..."
        docker build -t ujiscan:comprehensive .
        ;;
    
    start)
        check_security
        echo "🚀 Starting ujiscan..."
        docker-compose up -d
        echo ""
        echo "✅ ujiscan started"
        echo "   Logs: docker-compose logs -f"
        echo "   Status: docker-compose ps"
        echo "   Access: http://localhost:8081 (local)"
        echo "           https://ujiscan.rudilab.my.id (public)"
        ;;
    
    stop)
        echo "🛑 Stopping ujiscan..."
        docker-compose down
        ;;
    
    restart)
        echo "🔄 Restarting ujiscan..."
        docker-compose restart
        ;;
    
    logs)
        docker-compose logs -f --tail=100
        ;;
    
    *)
        echo "Usage: $0 [build|start|stop|restart|logs]"
        exit 1
        ;;
esac
