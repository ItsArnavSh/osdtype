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
`FRONTEND_URL`, `DEV_LOGIN`, `COOKIE_SECURE`, `COOKIE_SAMESITE`, `DB_HOST`,
`DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_DBNAME`, `DATABASE_URL`, `DB_SSLMODE`,
`REDIS_ADDR`, `REDIS_URL`, `REDIS_TLS`, `CODEGEN_URL`, `BUG_WEBHOOK_URL`,
`METRICS_TOKEN`, `GITHUB_KEY`, `GITHUB_AUTH` and `GITHUB_REDIRECT_URL`.

## Render

The repository includes a `render.yaml` Blueprint for a lightweight managed
deployment:

- **API** — native Go web service, always on. WebSockets, the matchmaker, the
  scheduler and active game runtimes live in this process, so keep it at one
  instance unless the hub is moved to Redis pub/sub.
- **Web** — static SvelteKit build served from Render's CDN. The SPA fallback
  is already configured.
- **Postgres** — Supabase's pooler. Use port `5432` (session mode) with
  `sslmode=require`; transaction mode on `6543` is not a good fit for GORM.
- **Redis** — the managed Redis URL. Use `rediss://` when the provider requires
  TLS; `REDIS_TLS=true` also works with a plain `redis://` URL.

Create the Blueprint from the Render Dashboard and fill the `sync: false`
values. The API must be on a paid always-on plan; Render's free web services
spin down after 15 minutes, which breaks WebSockets and scheduled contests.
The static site can stay free.

Set these values:

```text
DATABASE_URL=postgresql://postgres.<project-ref>:<password>@aws-0-<region>.pooler.supabase.com:5432/postgres?sslmode=require
REDIS_URL=rediss://default:<password>@<host>:<port>
JWTKEY=<generated by Render>
FRONTEND_URL=https://<web-service>.onrender.com
GITHUB_KEY=<GitHub OAuth app client id>
GITHUB_AUTH=<GitHub OAuth app client secret>
GITHUB_REDIRECT_URL=https://<api-service>.onrender.com/auth/github/callback
VITE_API_URL=https://<api-service>.onrender.com
BUG_WEBHOOK_URL=<optional Discord or Slack webhook>
METRICS_TOKEN=<optional bearer token for /metrics>
```

Percent-encode reserved characters in the Supabase password before putting it
in `DATABASE_URL` (for example, `@` becomes `%40`). `render.yaml` sets
`DB_SSLMODE=require`; the server also appends that mode when the URL omits
`sslmode`.

`render.yaml` sets `DEV_LOGIN=false`, `COOKIE_SECURE=true` and
`COOKIE_SAMESITE=none`. The last one matters on Render's default
`onrender.com` subdomains because they are separate sites; use `lax` if the web
and API are served from subdomains of one custom domain. The code generator is
disabled by default (`CODEGEN_URL=disabled`) so the Go fallback snippets are
used; point `CODEGEN_URL` at a deployed codegen service to restore the full
generator.

Do not commit database passwords, Redis passwords, OAuth secrets or webhook
URLs. Rotate any secret that has been pasted into chat or logs.

## Bug reports

`/bugs` is a public report form. Signed-in reports are attached to the account
and listed under "your reports"; guests can leave an optional contact email.
Reports are stored in Postgres, rate-limited through Redis, and optionally
posted to `BUG_WEBHOOK_URL`. The API exposes `POST /bugs` and
`GET /bugs/mine`. Reports are plain text and should never contain passwords,
cookies or API keys.

## Observability

The API exposes three health/metrics endpoints:

- `GET /healthz` — liveness. Returns `200` while the process is running.
- `GET /readyz` — readiness. Checks Postgres and Redis, returning `503` if
  either dependency is unavailable.
- `GET /metrics` — Prometheus text exposition with request counts, status
  classes, latency totals, goroutines and heap usage. Set `METRICS_TOKEN` to
  require `Authorization: Bearer <token>`; leave it empty to keep the endpoint
  public.

Every response includes an `X-Request-ID` header, and request-scoped errors
include the same value as `request_id` in the JSON body. `LOG_FORMAT=json`
makes the API emit structured access logs with that request id, which Render
collects automatically. The bug form copies a failed request id into the
report when the API returns one.

For Render:

1. Enable deploy-failure and unhealthy-service notifications in the Render
   Dashboard.
2. Point an external uptime monitor (UptimeRobot, Better Stack, or a Render
   cron) at `https://<api-service>.onrender.com/readyz` every minute.
3. Optionally scrape `/metrics` from Grafana Cloud or another Prometheus
   instance. Use a long-lived `METRICS_TOKEN` and send it as a bearer token.
4. Watch Render's built-in CPU, memory and request metrics for the API service.
   Keep the service at one instance; the hub, matchmaker and scheduler are
   in-process.

## Layout

```
app/       SvelteKit client, WASM code generator, game UI
codegen/   resrap-rs HTTP service (and parity wrapper)
server/    Go API, matchmaker, rooms, contests, game runtime
```

## Status

V1 is under active development. Infrastructure and database setup may evolve;
migrations run automatically when the server starts.
