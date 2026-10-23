# OSDTYP — shortcuts for running the stack.
#
#   make dev    Go API on :8080 + Vite dev server on :5173 (dev login enabled)
#   make prod   compiled API on :8080 + adapter-node server on :3000 (no dev login)
#   make up     whole app in Docker: PostgreSQL, Redis, codegen, API and web
#   make down   stop the Docker stack
#
# `dev` and `prod` start PostgreSQL and Redis through Docker Compose when
# Docker is available. Override ports with API_PORT=... / WEB_PORT=...

.PHONY: dev prod deps up down build clean

dev: deps
	@sh scripts/dev.sh

prod: deps
	@sh scripts/prod.sh

deps:
	@sh scripts/deps.sh

up:
	docker compose up --build

down:
	docker compose down

# Build both halves without starting them.
build:
	cd app && npm install && npm run build
	cd server && go build -trimpath -o bin/osdtyp .

clean:
	rm -rf app/build server/bin
