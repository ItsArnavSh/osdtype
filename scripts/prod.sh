#!/usr/bin/env sh
# Production build: adapter-node frontend + compiled Go API.
# Dev login is explicitly disabled in this path.
set -eu

ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
WEB_PORT="${WEB_PORT:-3000}"
API_PORT="${API_PORT:-8080}"

if [ ! -f "$ROOT/.prod-secret" ]; then
	# Keep sessions valid across restarts instead of regenerating on each boot.
	umask 077
	if command -v openssl >/dev/null 2>&1; then
		openssl rand -hex 32 >"$ROOT/.prod-secret"
	else
		node -e "console.log(require('crypto').randomBytes(32).toString('hex'))" >"$ROOT/.prod-secret"
	fi
fi
chmod 600 "$ROOT/.prod-secret" 2>/dev/null || true
JWTKEY="${JWTKEY:-$(cat "$ROOT/.prod-secret")}"

echo "==> building frontend"
# PUBLIC_API_URL is what the browser will call. Local prod defaults to the
# API port above; remote deployments should set the public API origin.
(cd "$ROOT/app" && npm install && VITE_API_URL="${PUBLIC_API_URL:-http://localhost:${API_PORT}}" npm run build)

echo "==> building api"
(cd "$ROOT/server" && go build -trimpath -o bin/osdtyp .)

echo "==> api      http://localhost:${API_PORT}"
echo "==> web      http://localhost:${WEB_PORT}"
echo "==> dev login disabled"

(
	cd "$ROOT/app"
	PORT="$WEB_PORT" \
	ORIGIN="http://localhost:${WEB_PORT}" \
	NODE_ENV="production" \
	node build
) &
APP_PID=$!

(
	cd "$ROOT/server"
	PORT=":${API_PORT}" \
	DEV_LOGIN="false" \
	FRONTEND_URL="http://localhost:${WEB_PORT}" \
	COOKIE_SECURE="false" \
	JWTKEY="$JWTKEY" \
	./bin/osdtyp
) &
SERVER_PID=$!

cleanup() {
	kill "$SERVER_PID" 2>/dev/null || true
	kill "$APP_PID" 2>/dev/null || true
}
trap cleanup INT TERM EXIT

wait
