#!/bin/sh
set -eu

cd "$(dirname "$0")/.."
docker compose up -d --wait --wait-timeout 60
go test -tags=integration ./test/integration -count=1 -v
