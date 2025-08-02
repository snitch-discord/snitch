#!/bin/bash

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

if [ ! -f "$ENV_FILE" ]; then
  cat > "$ENV_FILE" << EOF
SNITCH_DISCORD_TOKEN=REPLACE_ME
BACKUP_BUCKET_ENDPOINT=https://your-account.r2.cloudflarestorage.com
BACKUP_BUCKET_NAME=snitch-backups
BACKUP_BUCKET_ACCESS_KEY=REPLACE_ME
BACKUP_BUCKET_SECRET_KEY=REPLACE_ME
BACKUP_BUCKET_REGION=auto
EOF
  echo "Created .env file. Please configure:"
  echo "  - SNITCH_DISCORD_TOKEN with your Discord bot token"
  echo "  - BACKUP_BUCKET_* variables with your S3-compatible storage credentials"
fi

docker compose up --build --watch
