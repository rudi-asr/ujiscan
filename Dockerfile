# Dockerfile for ujiscan - builds with sqlite3 support

FROM golang:1.23 AS builder

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
RUN CGO_ENABLED=1 go build -o ujiscan .

# Stage 2: Runtime
FROM debian:bookworm-slim

WORKDIR /app

# Install runtime dependencies only
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    libsqlite3-0 \
    wget \
    && rm -rf /var/lib/apt/lists/*

# Copy binary from builder
COPY --from=builder /build/ujiscan .

# Copy web assets
COPY web ./web

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
