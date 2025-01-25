#!/bin/bash

BASE_DIR=$(dirname "$0")
ENV_FILE="${BASE_DIR}/.env"

if [ ! -f "$ENV_FILE" ]; then
  PRIVATE_KEY=$(openssl genpkey -algorithm ed25519 -outform PEM)
  PUBLIC_KEY=$(openssl pkey -pubout <<< "$PRIVATE_KEY")
  
  B64_PRIVATE_KEY=$(echo "$PRIVATE_KEY" | openssl enc -A -base64)
  B64_PUBLIC_KEY=$(echo "$PUBLIC_KEY" | openssl enc -A -base64)
  
  printf "PRIVATE_KEY=%s\nPUBLIC_KEY=%s" "$B64_PRIVATE_KEY" "$B64_PUBLIC_KEY" > "$ENV_FILE"
  printf "SNITCH_DISCORD_TOKEN=REPLACE_ME" >> "$ENV_FILE"
fi

docker compose up --build --watch
