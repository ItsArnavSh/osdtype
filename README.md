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
  automatically. Results and final standings are persisted.
- **Practice and Zen** — unranked single-player typing with the same generated
  snippets used in every other mode.

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

Prerequisites: Go 1.26+, Node 22+, Rust 1.92+, Docker, PostgreSQL 16 and Redis 7.

```sh
docker compose up -d postgres redis

# terminal 1: code generator
cd codegen && cargo run

# terminal 2: API and WebSocket server
cd server && go run .

# terminal 3: frontend
cd app && npm install && npm run dev
```

Open `http://localhost:5173`. Dev login is enabled by default; GitHub OAuth
needs `GITHUB_KEY` and `GITHUB_AUTH` in the server environment.

Use `localhost` rather than `127.0.0.1` for the frontend: auth cookies are
same-site and the dev server binds that origin.

## Docker

`docker compose up --build` starts PostgreSQL, Redis, the Rust code generator
and the Go server. The SvelteKit app still runs separately with
`cd app && npm run dev` (or build it with your preferred adapter).

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
