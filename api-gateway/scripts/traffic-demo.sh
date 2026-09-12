#!/bin/sh
set -eu

cd "$(dirname "$0")/.."
docker compose up -d --wait --wait-timeout 60
# Restart the fixtures so changes to the mounted backend script take effect.
docker compose restart --no-deps users orders
docker compose up -d --wait --wait-timeout 60
go run ./cmd/traffic "$@"
