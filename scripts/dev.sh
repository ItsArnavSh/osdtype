#!/usr/bin/env sh
# Development stack: Go API + Vite dev server.
# Prerequisites: Go 1.26+, Node 22+, PostgreSQL on 5432, Redis on 6379.
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
SERVER_PID=""
APP_PID=""

cleanup() {
	[ -n "$SERVER_PID" ] && kill "$SERVER_PID" 2>/dev/null || true
	[ -n "$APP_PID" ] && kill "$APP_PID" 2>/dev/null || true
}
trap cleanup INT TERM EXIT

if [ ! -d "$ROOT/app/node_modules" ]; then
	echo "==> installing frontend dependencies"
	(cd "$ROOT/app" && npm install)
fi

echo "==> api      http://localhost:8080  (dev login enabled)"
echo "==> web      http://localhost:5173"
echo "==> press ctrl+c to stop both"

(
	cd "$ROOT/server"
	PORT=":8080" \
	DEV_LOGIN="true" \
	ADMIN_USERNAME="${ADMIN_USERNAME:-ItsArnavSh}" \
	FRONTEND_URL="http://localhost:5173" \
	COOKIE_SECURE="false" \
	go run .
) &
SERVER_PID=$!

(
	cd "$ROOT/app"
	npm run dev -- --host localhost
) &
APP_PID=$!

wait
