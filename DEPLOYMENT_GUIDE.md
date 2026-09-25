# ujiscan Deployment Guide - Phase 10

## Quick Start (3 minutes)

### Option 1: Docker (Recommended)

```bash
# Build image
docker build -t ujiscan:latest .

# Run container
docker run -d \
  --name ujiscan \
  -p 8081:8081 \
  -v ujiscan_data:/app/data \
  ujiscan:latest

# Test
curl http://localhost:8081/api/status

# Open browser
open http://localhost:8081/html/login.html
# Login: admin@ujiscan.local / admin123
```

### Option 2: Docker Compose (Simplest)

```bash
# Start
docker-compose up -d

# Test
curl http://localhost:8081/api/status

# Stop
docker-compose down
```

### Option 3: Native (Go)

```bash
# Build
go build -o ujiscan .

# Run
./ujiscan

# Open browser
open http://localhost:8081/html/login.html
```

---

## Production Deployment

### AWS EC2

1. **Launch EC2 Instance:**
   - Image: Ubuntu 24.04 LTS
   - Type: t3.medium (2 CPU, 4GB RAM)
   - Security group: Allow 443 (HTTPS), 8081 (internal)

2. **Install Docker:**
   ```bash
   curl -fsSL https://get.docker.com | sh
   sudo usermod -aG docker $USER
   ```

3. **Deploy:**
   ```bash
   git clone https://github.com/rudi-asr/ujiscan.git
   cd ujiscan
   docker-compose -f docker-compose.prod.yml up -d
   ```

4. **Setup HTTPS (LetsEncrypt):**
   ```bash
   docker run --rm -it -v /etc/letsencrypt:/etc/letsencrypt \
     -p 80:80 certbot/certbot certonly --standalone \
     -d ujiscan.example.com
   ```

5. **Configure Nginx:**
   ```nginx
   server {
     listen 443 ssl http2;
     server_name ujiscan.example.com;
     
     ssl_certificate /etc/letsencrypt/live/ujiscan.example.com/fullchain.pem;
     ssl_certificate_key /etc/letsencrypt/live/ujiscan.example.com/privkey.pem;
     
     location / {
       proxy_pass http://localhost:8081;
       proxy_set_header Host $host;
       proxy_set_header X-Real-IP $remote_addr;
     }
   }
   ```

### Google Cloud Run

1. **Enable Cloud Run API:**
   ```bash
   gcloud services enable run.googleapis.com
   ```

2. **Build & Push:**
   ```bash
   gcloud builds submit --tag gcr.io/PROJECT_ID/ujiscan
   ```

3. **Deploy:**
   ```bash
   gcloud run deploy ujiscan \
     --image gcr.io/PROJECT_ID/ujiscan \
     --platform managed \
     --region us-central1 \
     --allow-unauthenticated \
     --set-env-vars DATABASE_PATH=/tmp/ujiscan.db
   ```

### Azure Container Instances

1. **Build & Push to ACR:**
   ```bash
   az acr build --registry myregistry --image ujiscan:latest .
   ```

2. **Deploy:**
   ```bash
   az container create \
     --resource-group mygroup \
     --name ujiscan \
     --image myregistry.azurecr.io/ujiscan:latest \
     --ports 8081 \
     --environment-variables LOG_LEVEL=info
   ```

---

## Docker Image Details

**Image Size:** ~175MB (40.7MB compressed)

**Base:** Debian bookworm-slim

**Included:**
- Go 1.23 (multi-stage builder)
- SQLite 3
- Frontend assets
- Health check

**Environment Variables:**
- `LOG_LEVEL` - Log verbosity (default: info)
- `DATABASE_PATH` - SQLite database path (default: /app/data/ujiscan.db)
- `CORS_ALLOWED_ORIGINS` - CORS whitelist (comma-separated)

**Ports:**
- 8081 - HTTP API

**Health Check:**
- Enabled (30s interval, 10s start period)
- Checks `/api/status` endpoint

---

## Database Persistence

### Volume Mount (Recommended)

```bash
# Named volume
docker volume create ujiscan_data
docker run -v ujiscan_data:/app/data ...

# Host directory
docker run -v ~/ujiscan_data:/app/data ...
```

### Backup & Restore

```bash
# Backup
docker cp ujiscan-container:/app/data/ujiscan.db ./ujiscan_backup.db

# Restore
docker cp ./ujiscan_backup.db ujiscan-container:/app/data/ujiscan.db
docker restart ujiscan-container
```

---

## Monitoring & Logging

### Docker Logs

```bash
# View logs
docker logs ujiscan-container

# Follow logs
docker logs -f ujiscan-container

# Last 100 lines
docker logs --tail 100 ujiscan-container
```

### Health Status

```bash
# Check health
docker inspect --format='{{.State.Health.Status}}' ujiscan-container

# Full health info
docker inspect --format='{{json .State.Health}}' ujiscan-container | jq
```

### Metrics

```bash
# Container stats
docker stats ujiscan-container

# Resource usage
docker inspect --format='{{json .HostConfig.Memory}}' ujiscan-container
```

---

## Scaling (Multi-Container)

### Load Balancer Setup

```yaml
# docker-compose.yml
version: '3.8'

services:
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
    depends_on:
      - ujiscan1
      - ujiscan2

  ujiscan1:
    build: .
    ports:
      - "8081:8081"
    volumes:
      - shared_data:/app/data

  ujiscan2:
    build: .
    ports:
      - "8082:8081"
    volumes:
      - shared_data:/app/data

volumes:
  shared_data:
```

```nginx
# nginx.conf
upstream ujiscan {
  server ujiscan1:8081;
  server ujiscan2:8081;
}

server {
  listen 80;
  location / {
    proxy_pass http://ujiscan;
  }
}
```

---

## Security Checklist

- [ ] HTTPS/TLS configured (LetsEncrypt)
- [ ] Database password changed (SQLite - not applicable, but DB path secured)
- [ ] CORS origins whitelisted
- [ ] Firewall rules: only 443 public, 8081 internal only
- [ ] Regular backups enabled (daily)
- [ ] Monitor logs for errors
- [ ] Docker image signed (optional)
- [ ] Container registry private (ACR/GCR)
- [ ] Resource limits set (CPU, memory)

---

## Troubleshooting

### Container crashes on startup

```bash
# Check logs
docker logs ujiscan-container

# Common issues:
# - Database path permission denied → docker run -u 0
# - SQLite library missing → update Dockerfile base image
# - Port already in use → docker port ujiscan-container
```

### Slow login

```bash
# Check JWT token generation
curl -v -X POST http://localhost:8081/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@ujiscan.local","password":"admin123"}'

# Check CPU/memory
docker stats ujiscan-container
```

### Database file not persisting

```bash
# Verify volume mount
docker inspect ujiscan-container | grep Mounts

# Check permissions
docker exec ujiscan-container ls -la /app/data

# Recreate with explicit volume
docker volume rm ujiscan_data
docker volume create ujiscan_data
docker run -v ujiscan_data:/app/data ...
```

---

## Performance Tuning

### Increase Resource Limits

```bash
docker run \
  --memory 2g \
  --cpus 1.5 \
  --memory-reservation 1g \
  ujiscan:latest
```

### Database Optimization

```bash
# For SQLite, create indexes on frequently-queried columns
sqlite3 /app/data/ujiscan.db

sqlite> CREATE INDEX idx_findings_severity ON findings(severity);
sqlite> CREATE INDEX idx_audit_user_id ON audit_logs(user_id);
sqlite> VACUUM;
```

### Network

```bash
# Use host network (Linux only)
docker run --network host ujiscan:latest

# Limit connections (via reverse proxy)
# nginx: limit_req_zone $binary_remote_addr zone=api:10m rate=100r/s;
```

---

## Rollback Strategy

### Version Tags

```bash
# Build with version tag
docker build -t ujiscan:v1.0.0 .
docker tag ujiscan:v1.0.0 ujiscan:latest

# Push to registry
docker push gcr.io/project/ujiscan:v1.0.0
docker push gcr.io/project/ujiscan:latest

# Rollback to previous version
docker run -d -p 8081:8081 gcr.io/project/ujiscan:v0.9.9
```

### Database Rollback

```bash
# Keep daily backups
0 2 * * * docker cp ujiscan:/app/data/ujiscan.db ~/backups/ujiscan_$(date +%Y%m%d).db

# Restore from backup
docker cp ~/backups/ujiscan_20260924.db ujiscan:/app/data/ujiscan.db
docker restart ujiscan
```

---

## Next Steps

1. **Phase 11 (v1.0 Release):** Tag v1.0.0, update README, announce launch
2. **Post-Launch:** Monitor production, collect feedback
3. **Phase B (AI Agents):** Implement autonomous agent features (Jan-Apr 2027)

---

## Support & Issues

For issues:
1. Check logs: `docker logs ujiscan-container`
2. Review manual testing guide: `MANUAL_TESTING_GUIDE.md`
3. Check E2E test report: `PHASE_9_E2E_REPORT.md`
4. Create GitHub issue: https://github.com/rudi-asr/ujiscan/issues

---

**Ready for production! 🚀**
