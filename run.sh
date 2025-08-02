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
  printf "SNITCH_DISCORD_TOKEN=REPLACE_ME\n" > "$ENV_FILE"
  echo "Created .env file. Please replace SNITCH_DISCORD_TOKEN with your actual Discord bot token."
fi

docker compose up --build --watch
