#!/bin/sh
set -eu

cd "$(dirname "$0")/.."
traffic_dir=$(mktemp -d "${TMPDIR:-/tmp}/api-gateway-traffic.XXXXXX")
trap 'rm -f "$traffic_dir/traffic"; rmdir "$traffic_dir"' 0
go build -o "$traffic_dir/traffic" ./cmd/traffic
docker compose up -d --wait --wait-timeout 60
# Restart the fixtures so changes to the mounted backend script take effect.
docker compose restart --no-deps users orders
docker compose up -d --wait --wait-timeout 60
"$traffic_dir/traffic" "$@"
