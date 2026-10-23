# OSDType

Speedtyping, but for coders. A server-authoritative typing arena with ranked Elo,
multiplayer rooms, contests, bots and anti-cheat.

![Main Page](assets/Home.png)

## V1 modes

- **Ranked** — Redis-backed matchmaking at 30s, 90s and 300s. Free-for-all
  pairwise Elo with divisions from Bronze to Grandmaster. A bot opponent is
  used when nobody else is queued.
- **Rooms** — public or private 2–16 player lobbies, six-character codes and
  invite links, optional passwords, owner/mod controls, kicks and bans.
- **Contests** — scheduled room events that open a lobby and start
  automatically. Funnels can include qualifiers, practice/warm-up stages,
  HotLaps and live finals. Enrolled players and spectators get a public
  timing tower with the top 10 and a username lookup. Results and final
  standings are persisted.
- **Practice and Zen** — unranked single-player typing with the same generated
  snippets used in every other mode. Practice runs are recorded, can be saved
  to profile history and shared as simulated replay links, and logged-in
  players can race adaptive Lewis/Max/Lance bots calibrated from recent form.
- **Results** — every finished run has a labelled speed graph and simulated
  replay. Ranked and room results combine every player on one exportable graph.

All matches are generated from the same language, seed and token count so the
server and browser produce byte-identical code. The backend owns the clock,
the start offset, scoring, presence, resume state and rating updates.
Anti-cheat flags are persisted and freeze the flagged player's rating change.

## Stack

- **Frontend:** SvelteKit, TypeScript, Tailwind
- **Backend:** Go, Gin, WebSockets
- **State and queueing:** Redis
- **Durable storage:** PostgreSQL
- **Code generation:** Rust (`resrap-rs` 1.0.0), served over HTTP and browser WASM

## Run locally

Prerequisites: Go 1.26+ and Node 22+. PostgreSQL 16 and Redis 7 must be
available on `5432` and `6379`; when Docker is installed, `make dev` starts
both containers for you. Rust is optional; without the codegen service the
server uses its built-in snippet fallback.

```sh
make dev
```

This starts PostgreSQL/Redis if needed, the Go API on
`http://localhost:8080`, and the Vite dev server on `http://localhost:5173`.
Dev login is enabled. Open `http://localhost:5173`.

For a production-style run:

```sh
make prod
```

This builds the SvelteKit app with `adapter-node`, compiles the Go API, and
starts the web server on `http://localhost:3000` with the API on `:8080`.
`DEV_LOGIN` is forced to `false`, so the login page only offers GitHub OAuth.
Ports can be changed with `WEB_PORT` and `API_PORT`. When the API is not at
`http://localhost:8080` for the browser, build with
`PUBLIC_API_URL=https://api.example.com make prod`.

If `make` is unavailable, run the scripts directly: `sh scripts/deps.sh`,
`sh scripts/dev.sh` or `sh scripts/prod.sh`. The manual path is still:

```sh
docker compose up -d postgres redis

# terminal 1: code generator (optional)
cd codegen && cargo run

# terminal 2: API and WebSocket server
cd server && go run .

# terminal 3: frontend
cd app && npm install && npm run dev
```

Open `http://localhost:5173`. Dev login is enabled by default; GitHub OAuth
needs `GITHUB_KEY` and `GITHUB_AUTH` in the server environment.

Guests can open the hub and play Practice or Zen without logging in. Those two
modes also keep working when the Go backend is not running; ranked, rooms,
contests, leaderboards and results are gated behind an account.

Use `localhost` rather than `127.0.0.1` for the frontend: auth cookies are
same-site and the dev server binds that origin.

## Docker

`make up` (or `docker compose up --build`) starts the whole application:
PostgreSQL, Redis, the Rust code generator, the Go API, and the adapter-node
SvelteKit web server. Open `http://localhost:3000`. The production-style stack
has `DEV_LOGIN=false`, so only GitHub OAuth is offered.

Set these variables for a real deployment:

```sh
PUBLIC_API_URL=https://api.example.com \
PUBLIC_ORIGIN=https://example.com \
FRONTEND_URL=https://example.com \
JWTKEY="$(openssl rand -hex 32)" \
GITHUB_KEY=... GITHUB_AUTH=... \
docker compose up --build -d
```

`PUBLIC_API_URL` is baked into the browser bundle at build time; the others
are runtime configuration. For local Docker development with the dev login
button, run `DEV_LOGIN=true docker compose up --build`.

Environment overrides understood by the server include `PORT`, `JWTKEY`,
`FRONTEND_URL`, `DEV_LOGIN`, `COOKIE_SECURE`, `DB_HOST`, `DB_PORT`, `DB_USER`,
`DB_PASSWORD`, `DB_DBNAME`, `REDIS_ADDR`, `CODEGEN_URL`, `GITHUB_KEY`,
`GITHUB_AUTH` and `GITHUB_REDIRECT_URL`.

## Layout

```
app/       SvelteKit client, WASM code generator, game UI
codegen/   resrap-rs HTTP service (and parity wrapper)
server/    Go API, matchmaker, rooms, contests, game runtime
```

## Status

V1 is under active development. Infrastructure and database setup may evolve;
migrations run automatically when the server starts.
