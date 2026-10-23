# OSDTYP — shortcuts for running the stack.
#
#   make dev    Go API on :8080 + Vite dev server on :5173 (dev login enabled)
#   make prod   compiled API on :8080 + adapter-node server on :3000 (no dev login)
#
# PostgreSQL and Redis are expected to already be running. On Windows the
# targets call scripts/dev.ps1 and scripts/prod.ps1; elsewhere they use the
# POSIX shell scripts. Override ports with API_PORT=... / WEB_PORT=...

ifeq ($(OS),Windows_NT)
RUN_DEV  := powershell -NoProfile -ExecutionPolicy Bypass -File scripts/dev.ps1
RUN_PROD := powershell -NoProfile -ExecutionPolicy Bypass -File scripts/prod.ps1
else
RUN_DEV  := sh scripts/dev.sh
RUN_PROD := sh scripts/prod.sh
endif

.PHONY: dev prod build clean

dev:
	@$(RUN_DEV)

prod:
	@$(RUN_PROD)

# Build both halves without starting them.
build:
	cd app && npm install && npm run build
	cd server && go build -trimpath -o bin/osdtyp .

clean:
	rm -rf app/build server/bin
