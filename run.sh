#!/usr/bin/env bash

# Check for required dependencies
if ! command -v docker &> /dev/null; then
    echo "Error: docker is not installed or not in PATH"
    exit 1
fi

if ! command -v docker compose &> /dev/null; then
    echo "Error: docker compose is not installed or not in PATH"
    exit 1
fi

BASE_DIR=$(dirname "$0")
ENV_FILE="${BASE_DIR}/.env"
CERTS_SCRIPT="${BASE_DIR}/scripts/generate-certs.sh"

# Create .env file if it doesn't exist
if [ ! -f "$ENV_FILE" ]; then
  printf "SNITCH_DISCORD_TOKEN=REPLACE_ME\n" > "$ENV_FILE"
  echo "Created .env file. Please replace SNITCH_DISCORD_TOKEN with your actual Discord bot token."
fi

# Generate JWT secret if it doesn't exist
if ! grep -q "SNITCH_JWT_SECRET" "$ENV_FILE"; then
  echo "Generating JWT secret..."
  JWT_SECRET=$(head /dev/urandom | tr -dc A-Za-z0-9 | head -c 64)
  printf "\nSNITCH_JWT_SECRET=%s\n" "$JWT_SECRET" >> "$ENV_FILE"
  echo "JWT secret added to $ENV_FILE"
fi

# Generate TLS certificates if they don't exist
if [ ! -f "${BASE_DIR}/certs/ca/ca-cert.pem" ]; then
  echo "TLS certificates not found. Generating certificates..."
  if [ -f "$CERTS_SCRIPT" ]; then
    "$CERTS_SCRIPT"
  else
    echo "Warning: Certificate generation script not found at $CERTS_SCRIPT"
    echo "TLS certificates are required for the services to start properly."
    exit 1
  fi
else
  echo "TLS certificates found. Verifying..."
  "$CERTS_SCRIPT" --verify || {
    echo "Certificate verification failed. Consider regenerating with: $CERTS_SCRIPT --force"
    exit 1
  }
fi

echo "Starting Snitch services with TLS enabled..."
docker compose up --build --watch
