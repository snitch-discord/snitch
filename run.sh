#!/bin/bash

BASE_DIR=$(dirname "$0")
ENV_FILE="${BASE_DIR}/.env"

if [ ! -f "$ENV_FILE" ]; then
  printf "SNITCH_DISCORD_TOKEN=REPLACE_ME\n" > "$ENV_FILE"
  echo "Created .env file. Please replace SNITCH_DISCORD_TOKEN with your actual Discord bot token."
fi

docker compose up --build --watch
