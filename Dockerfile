# Dockerfile for ujiscan - builds with sqlite3 support

# Stage 1: Builder
FROM golang:1.25 AS builder

WORKDIR /build

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    sqlite3 \
    libsqlite3-dev \
    git \
    && rm -rf /var/lib/apt/lists/*

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build with CGO for SQLite
RUN CGO_ENABLED=1 go build -o ujiscan . \
    && mkdir -p /build/go-tools

# Build Go-based security tools (subfinder, httpx, nuclei, gobuster) as static binaries
# Pin to versions compatible with Go 1.25 toolchain
RUN go install github.com/projectdiscovery/subfinder/v2/cmd/subfinder@v2.6.6 \
    && go install github.com/projectdiscovery/httpx/cmd/httpx@v1.6.10 \
    && go install github.com/projectdiscovery/nuclei/v3/cmd/nuclei@v3.3.7 \
    && go install github.com/OJ/gobuster/v3@v3.6.0 \
    && ls /go/bin/

# Stage 2: Runtime
FROM debian:bookworm-slim

WORKDIR /app

# Install runtime dependencies + all 9 hostedscan scanning tools
# apt-based: dig (dnsutils), nmap, whatweb, sslscan
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    libsqlite3-0 \
    wget \
    curl \
    dnsutils \
    nmap \
    net-tools \
    iputils-ping \
    whatweb \
    sslscan \
    && rm -rf /var/lib/apt/lists/*

# Copy Go security tools from builder (subfinder, httpx, nuclei, gobuster)
COPY --from=builder /go/bin/ /usr/local/bin/

# Install nikto (not in Debian repos) - official tarball
RUN apt-get update && apt-get install -y --no-install-recommends perl libnet-ssleay-perl \
    && rm -rf /var/lib/apt/lists/* \
    && wget -q https://github.com/sullo/nikto/archive/refs/heads/master.tar.gz -O /tmp/nikto.tar.gz \
    && mkdir -p /opt/nikto \
    && tar -xzf /tmp/nikto.tar.gz -C /opt/nikto --strip-components=1 \
    && chmod +x /opt/nikto/program/nikto.pl \
    && ln -sf /opt/nikto/program/nikto.pl /usr/local/bin/nikto \
    && rm -f /tmp/nikto.tar.gz

# Copy binary from builder
COPY --from=builder /build/ujiscan .

# Copy web assets
COPY web ./web

# Copy tool registry + playbooks (required for registry/playbook/agentic scans)
COPY tools.yaml ./
COPY playbooks ./playbooks

# Copy agent knowledge files (.md rahasia - otak Full Scan AI)
COPY agents ./agents

# Create directory for database
RUN mkdir -p /app/data

# Expose port
EXPOSE 8081

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8081/api/status || exit 1

# Set environment variables
ENV LOG_LEVEL=info
ENV DATABASE_PATH=/app/data/ujiscan.db
ENV CORS_ALLOWED_ORIGINS=http://localhost:8081,http://localhost:3000,https://rudi-asr.github.io

# Run the application
CMD ["./ujiscan"]
