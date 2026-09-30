# OSDType developer tasks
# Run `just` to list all recipes.

session := "development"
server_dir := "server"
app_dir := "app"

# Build tags shared by the lint, test and coverage recipes. Test files are gated
# behind `unit` / `integration`, so a plain `go build` / `go vet` never sees
# them; anything that is meant to check the whole tree has to opt in.
build_tags := "unit integration"

# golangci-lint ships its own copy of go/types and refuses to load source files
# that need a newer Go language version than the one it was built against. If
# the local Go is newer than the linter, pin the toolchain for the lint run
# instead of failing with "file requires newer Go version". CI pins Go and
# golangci-lint together for exactly this reason.
# Override with: GOLANGCI_GOTOOLCHAIN=local just lint
lint_go := env_var_or_default("GOLANGCI_GOTOOLCHAIN", "go1.26.4")

# Frontend package runner. bun is preferred (it is what the lockfile and CI
# use) but npm works too, so the recipes stay usable on a Node-only machine.
pkg := if `command -v bun > /dev/null 2>&1 && echo bun || echo npm` == "bun" { "bun run" } else { "npm run" }

# Show all available recipes
default:
    @just --list --unsorted

# ---------- Dev ----------
# Full local dev stack in a tmux session (db + server + codegen + frontend)
dev:
    tmux has-session -t {{session}} 2>/dev/null && tmux kill-session -t {{session}} || true

    tmux new-session -d -s {{session}}

    tmux send-keys -t {{session}} "docker compose up" C-m

    tmux split-window -h -t {{session}}
    tmux send-keys -t {{session}} "cd server && JWTKEY=dev_secret air" C-m

    tmux split-window -v -t {{session}}
    tmux send-keys -t {{session}} "cd codegen && cargo run" C-m

    tmux select-pane -t {{session}}:0.1
    tmux split-window -v
    tmux send-keys -t {{session}} "cd app && {{pkg}} dev" C-m

    tmux select-layout -t {{session}} tiled
    tmux attach-session -t {{session}}

# Start the database in the background
db-up:
    docker compose up -d

# Stop the database
db-down:
    docker compose down

# Destroy the dev database and start it over.
# Worth knowing: the checked-in docker_data volume predates the current models
# and still carries columns the app no longer has, and AutoMigrate only adds
# columns, it never drops them. Use this when the two have drifted.
db-reset:
    docker compose down -v
    docker compose up -d --wait

# ---------- Build ----------
# Compile the Go server
build:
    cd {{server_dir}} && go build ./...

# Compile the Go server binary into server/tmp/main
build-bin:
    cd {{server_dir}} && go build -o tmp/main .

# Build the SvelteKit frontend
build-app:
    cd {{app_dir}} && {{pkg}} build

# ---------- Tests ----------
# Unit tests only. No database or network required.
test-unit:
    cd {{server_dir}} && go test -short -tags=unit -count=1 ./...

# Unit tests with verbose output
test-unit-v:
    cd {{server_dir}} && go test -short -tags=unit -count=1 -v ./...

# Integration tests. Requires a reachable Postgres (see test-db-up).
test-integration:
    cd {{server_dir}} && go test -tags=integration -count=1 ./...

# Integration tests with verbose output
test-integration-v:
    cd {{server_dir}} && go test -tags=integration -count=1 -v ./...

# Unit + integration. Assumes a reachable database, see test-db-up.
test-all:
    cd {{server_dir}} && go test -tags="{{build_tags}}" -count=1 ./...

# The -p flag matters: without a distinct compose project name this file would
# share a project with docker-compose.yml and tear down the dev database.
# Start a throwaway Postgres for integration tests on port 5433.
test-db-up:
    docker compose -p osdtype-test -f docker-compose.test.yml up -d --wait

# Stop the throwaway test database
test-db-down:
    docker compose -p osdtype-test -f docker-compose.test.yml down -v

# Full test cycle: boot a temp database, run everything, tear it down
ci-test: test-db-up
    -just test-all
    just test-db-down

# Run tests with the race detector. Slower, catches data races.
test-race:
    cd {{server_dir}} && go test -race -tags="{{build_tags}}" -count=1 ./...

# Report test coverage across the server packages
cover:
    cd {{server_dir}} && mkdir -p tmp && go test -tags="{{build_tags}}" -count=1 -covermode=atomic -coverprofile=tmp/coverage.out ./...
    cd {{server_dir}} && go tool cover -func=tmp/coverage.out | tail -30

# Open the HTML coverage report in a browser
cover-html:
    cd {{server_dir}} && go tool cover -html=tmp/coverage.out

# ---------- Lint / Format ----------
# Check that a file is gofmt-clean, or list the offenders.
gofmt-check:
    #!/usr/bin/env bash
    set -euo pipefail
    cd "{{server_dir}}"
    unformatted="$(gofmt -l .)"
    if [ -n "$unformatted" ]; then
        echo "these files are not gofmt'd (run 'just fmt'):" >&2
        echo "$unformatted" >&2
        exit 1
    fi
    echo "gofmt: clean"

# Everything CI runs against the Go tree, in one command.
lint: gofmt-check
    cd {{server_dir}} && go vet -tags="{{build_tags}}" ./...
    cd {{server_dir}} && GOTOOLCHAIN={{lint_go}} golangci-lint run --config ../.golangci.yml --build-tags="{{build_tags}}" ./...

# golangci-lint on its own
lint-server:
    cd {{server_dir}} && GOTOOLCHAIN={{lint_go}} golangci-lint run --config ../.golangci.yml --build-tags="{{build_tags}}" ./...

# Lint the frontend
lint-app:
    cd {{app_dir}} && {{pkg}} lint

# Type-check the frontend
check-app:
    cd {{app_dir}} && {{pkg}} check

# Format the Go server
fmt:
    cd {{server_dir}} && go fmt ./...

# Format the frontend
fmt-app:
    cd {{app_dir}} && {{pkg}} format

# ---------- Housekeeping ----------
# Tidy Go module files
tidy:
    cd {{server_dir}} && go mod tidy

# Run the Go server locally
run: build-bin
    cd {{server_dir}} && JWTKEY=dev_secret ./tmp/main

# Remove build and coverage artifacts
clean:
    rm -rf {{server_dir}}/tmp/coverage.out {{server_dir}}/tmp/coverage-*.out {{server_dir}}/tmp/main
