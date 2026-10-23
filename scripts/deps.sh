#!/usr/bin/env sh
# Best-effort local dependencies for `make dev` / `make prod`.
# Starts PostgreSQL and Redis through Docker Compose when Docker is available.
# Native services on 5432/6379 are left alone.
set -u

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)

if ! command -v docker >/dev/null 2>&1; then
	echo "==> docker not found; using existing PostgreSQL and Redis"
	exit 0
fi

if ! docker compose version >/dev/null 2>&1; then
	echo "==> docker compose not available; using existing PostgreSQL and Redis"
	exit 0
fi

echo "==> ensuring postgres + redis are up"
if docker compose -f "$ROOT/docker-compose.yml" up -d --wait postgres redis >/dev/null 2>&1; then
	exit 0
fi

# Older Compose versions do not support --wait. Starting in the background is
# still useful; the server will retry its own database connection.
if docker compose -f "$ROOT/docker-compose.yml" up -d postgres redis >/dev/null 2>&1; then
	echo "==> postgres + redis containers started"
	exit 0
fi

echo "==> could not start postgres/redis with docker; assuming they are already running"
exit 0
