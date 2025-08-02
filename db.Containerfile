FROM golang:1.24-bookworm AS builder

WORKDIR /app

# Install build dependencies for CGO and libsql
RUN apt-get update && apt-get install -y \
    gcc \
    libc6-dev \
    libsqlite3-dev \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Install sqlc
RUN curl -L https://github.com/sqlc-dev/sqlc/releases/download/v1.29.0/sqlc_1.29.0_linux_amd64.tar.gz | \
    tar -xz -C /usr/local/bin sqlc && \
    chmod +x /usr/local/bin/sqlc

# Set up build environment for performance
ENV CGO_ENABLED=1
ENV CGO_CFLAGS="-O3"
ENV GOCACHE=/root/.cache/go-build

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy source code
COPY . .

# Generate sqlc code
RUN sqlc generate

# Build with performance optimizations and caching
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    go build \
    -ldflags="-s -w" \
    -gcflags="-l=4" \
    -o db-service ./cmd/db

FROM debian:bookworm-slim

# Install ca-certificates for HTTPS requests
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the built binary and schema files
COPY --from=builder /app/db-service .
COPY --from=builder /app/internal/db/schemas/ ./internal/db/schemas/

# Create data directory
RUN mkdir -p /app/data

# Expose port
EXPOSE 5200

# Run the database service
CMD ["./db-service"]