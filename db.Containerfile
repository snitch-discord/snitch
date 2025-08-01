FROM golang:1.24 AS builder

WORKDIR /app

# Install build dependencies for CGO and libsql
RUN apt-get update && apt-get install -y \
    gcc \
    libc6-dev \
    libsqlite3-dev \
    && rm -rf /var/lib/apt/lists/*

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the database service with CGO enabled
ENV CGO_ENABLED=1
RUN go build -o db-service ./cmd/db

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