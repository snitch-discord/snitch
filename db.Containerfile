FROM golang:1.24-bookworm AS builder

WORKDIR /app

# Install build dependencies for CGO and libsql
RUN apt-get update && apt-get install -y \
    gcc \
    libc6-dev \
    libsqlite3-dev \
    && rm -rf /var/lib/apt/lists/*

# Set up build environment for performance
ENV CGO_ENABLED=1
ENV CGO_CFLAGS="-O3"
ENV GOCACHE=/root/.cache/go-build

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Copy source code
COPY . .

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

# Copy the built binary
COPY --from=builder /app/db-service .

# Create data directory
RUN mkdir -p /app/data


# Expose port
EXPOSE 5200

# Run the database service
CMD ["./db-service", "-port=5200", "-db-dir=/app/data"]