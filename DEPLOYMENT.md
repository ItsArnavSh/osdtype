# OSDTYP production deployment

> **v0.0.1 · experimental.** The first public build is expected to have bugs.
> Keep the bug form and admin triage queue available, and tell users where to
> report issues.

This is the launch runbook for the current single-VM AWS deployment. The
Render + Supabase instructions further down are retained as legacy reference.
The API is a single always-on Go process; the web app is a static SvelteKit
build on Vercel. Do not scale the API horizontally until the hub/matchmaker
state is moved out of process.

## Current production: one AWS VM + Vercel frontend

For v0.0.1 the whole backend runs on one EC2 instance in `ap-southeast-2`.
Vercel serves the public frontend and CloudFront/WAF is not yet in front of the
API because ACM cannot validate the temporary `sslip.io` hostnames.

| Piece | Where |
| --- | --- |
| Frontend | <https://osd-typ.in/> (Vercel) |
| API | <https://api.15.134.14.153.sslip.io> (Caddy → Go API) |
| AWS web fallback | <https://app.15.134.14.153.sslip.io> (same static build) |
| PostgreSQL 17 | Docker container on the EC2 instance, private Compose network |
| Redis 7 | Docker container on the EC2 instance, AOF persistence, private network |
| TLS / reverse proxy | Caddy in the `osdtype-web` image, automatic Let's Encrypt |
| Images | ECR repos `osdtype-api` and `osdtype-web`, built by GitHub Actions |
| Deploy | GitHub Actions → SSM Run Command → `deploy/aws/deploy.sh` |
| Secrets | SSM SecureString `/osdtype/prod/env`; local `.env.prod` is gitignored |

Instance: `i-0899b48fda9fa75f0`, `t4g.small`, Elastic IP `15.134.14.153`.
The security group allows only inbound 80/443 and no SSH; access is through
SSM. AWS Shield Standard is automatic for the public IP.

### Deploy

- Push to `main`. The workflow in `.github/workflows/deploy.yml` builds both
  ARM64 images, pushes them to ECR, deploys through SSM, and smoke-tests the
  API and AWS web fallback. It is gated by the repository variable
  `DEPLOY_ENABLED=true`.
- Manual deploy: run `./deploy/aws/upload-env.sh` from the repository root,
  then run `cd /opt/osdtype && ./deploy/aws/deploy.sh latest` on the instance
  through SSM.
- Health endpoints: `/ping`, `/healthz`, `/readyz`, and token-protected
  `/metrics`.

### Operations

```bash
cd /opt/osdtype
docker compose -f docker-compose.aws.yml --env-file .env.prod ps
docker compose -f docker-compose.aws.yml --env-file .env.prod logs -f api
docker compose -f docker-compose.aws.yml --env-file .env.prod stats
```

- Postgres data: Docker volume `osdtype_postgres_data`.
- Redis data: Docker volume `osdtype_redis_data` with AOF enabled.
- Postgres/Redis publish no host ports; only the Compose network can reach
  them.
- A systemd timer (`osdtype-backup.timer`) runs `pg_dump` daily at 18:00 UTC,
  keeps 7 days locally in `/var/backups/osdtype`, and uploads gzip dumps to
  `s3://osdtype-backups-309857583292-ap-southeast-2/postgres/` (SSE-S3,
  private, 30-day lifecycle).
- CloudFront + AWS WAF is the intended edge DDoS/WAF layer once a real custom
  domain exists (or if the API moves to a stable `cloudfront.net` hostname).
  At that point restrict the EC2 security group to the CloudFront origin-facing
  prefix list.

## 0. Pre-flight

Before touching a provider:

- [ ] Rotate the Redis password if it has ever been pasted into chat, a ticket
      or a log.
- [ ] Rotate the Supabase database password if it has ever been shared.
- [ ] Create a fresh GitHub OAuth app secret.
- [ ] Generate a new `JWTKEY` with `openssl rand -hex 32`. Rotating it later
      logs every existing session out.
- [ ] Confirm the `main` branch is the deploy branch and CI/tests pass.
- [ ] Decide the production hostnames, for example:
      `osdtype.example.com` (web) and `api.osdtype.example.com` (API).
- [ ] Make sure the Render workspace has a payment method. The API cannot use
      the free plan: free services spin down and kill WebSockets, matchmaking
      and scheduled contests.

Never commit `.env`, `.prod-secret`, database URLs, Redis URLs, OAuth secrets
or webhook URLs. `render.yaml` must contain only non-secret values.

## 1. Supabase Postgres

1. Create a Supabase project in the region closest to the Render API
   (`singapore` is used by `render.yaml`).
2. Open **Project Settings → Database** and copy the **pooler** connection
   details. Use **session mode on port 5432**, not transaction mode on 6543.
   GORM needs session-level behavior for migrations and prepared statements.
3. Use this shape for `DATABASE_URL`:

   ```text
   postgresql://postgres.<project-ref>:<url-encoded-password>@aws-0-<region>.pooler.supabase.com:5432/postgres?sslmode=require
   ```

   Percent-encode reserved password characters (`@` → `%40`, `#` → `%23`,
   `/` → `%2F`, `:` → `%3A`).
4. Keep TLS on. `render.yaml` sets `DB_SSLMODE=require`; the server also adds
   `sslmode=require` when the URL does not include a mode.
5. Enable the highest backup/PITR level the plan supports. Run one restore
   drill into a scratch project before launch.
6. The app runs migrations automatically on boot. The first deploy may take
   longer while tables/indexes are created.
7. Do not enable public table access or share the service-role key. The Go API
   is the only database client.

If you use a custom database role instead of the Supabase `postgres` role,
grant schema/DDL rights for the automatic migrations.

## 2. Managed Redis

1. Use a paid Redis-compatible instance with persistence enabled. The queue,
   room state, activity throttle and health ring buffer live here.
2. Prefer TLS. Use `rediss://` in `REDIS_URL`, or set `REDIS_TLS=true` with a
   plain `redis://` URL.
3. Set a long random password and rotate it after any exposure. Update
   `REDIS_URL` in Render immediately after rotation, then redeploy.
4. Avoid `allkeys-lru` eviction for this workload. `noeviction` is safest; if
   the provider requires an eviction policy, use `volatile-lru` and monitor
   memory. Evicting matchmaking/room keys will drop live state.
5. Enable snapshots/backups. Redis is not the source of truth for users,
   ratings, results or bug reports, but losing it drops active queues, rooms
   and in-flight games.
6. If the provider supports IP allowlists but Render egress IPs are not
   stable on your plan, rely on TLS, a strong password, and provider network
   controls rather than a brittle allowlist.

## 3. GitHub OAuth

1. GitHub → **Settings → Developer settings → OAuth Apps → New OAuth App**.
2. Set:
   - Homepage URL: `https://osdtype.example.com`
   - Authorization callback URL:
     `https://api.osdtype.example.com/auth/github/callback`
3. Copy the Client ID to `GITHUB_KEY` and the generated Client secret to
   `GITHUB_AUTH`.
4. Set `GITHUB_REDIRECT_URL` to the same callback URL.
5. When using the default `onrender.com` hostnames, the values are
   `https://<web-service>.onrender.com` and
   `https://<api-service>.onrender.com/auth/github/callback`. Update them
   again after attaching a custom domain.

The OAuth state cookie is `HttpOnly`, `Secure` in production and uses
`SameSite=Lax`; the callback compares it in constant time and clears it.

## 4. Render Blueprint

1. Push `render.yaml` and the app to `main`.
2. Render Dashboard → **New → Blueprint** → connect
   `ItsArnavSh/osdtype` → select the `main` branch → apply.
3. Render creates two services:
   - `osdtype-api` — Go web service, `0.5c-512mb`, always on, one instance,
     health check `/healthz`.
   - `osdtype-web` — static SvelteKit site, SPA rewrite to `/index.html`.
4. Fill every `sync: false` environment variable on the **API**:

   | Variable | Required | Value / notes |
   | --- | --- | --- |
   | `DATABASE_URL` | yes | Supabase session-pooler URL from step 1 |
   | `REDIS_URL` | yes | `rediss://…` from step 2 |
   | `FRONTEND_URL` | yes | `https://osdtype.example.com` (no trailing slash) |
   | `JWTKEY` | yes | Render-generated 256-bit value; at least 32 chars |
   | `GITHUB_KEY` | yes | GitHub OAuth client id |
   | `GITHUB_AUTH` | yes | GitHub OAuth client secret |
   | `GITHUB_REDIRECT_URL` | yes | `https://api.osdtype.example.com/auth/github/callback` |
   | `METRICS_TOKEN` | recommended | Long random bearer token for `/metrics` |
   | `BUG_WEBHOOK_URL` | optional | Discord/Slack webhook for new reports |
   | `ADMIN_USERNAME` | yes | `ItsArnavSh` |

   `render.yaml` already sets `DB_SSLMODE=require`,
   `DB_MAX_OPEN_CONNS=10`, `DB_MAX_IDLE_CONNS=5`, `COOKIE_SECURE=true`,
   `COOKIE_SAMESITE=none`, `DEV_LOGIN=false`, `GIN_MODE=release`,
   `LOG_FORMAT=json`, `CODEGEN_URL=disabled`, and one API instance.
5. Fill the **web** variable:
   - `VITE_API_URL=https://api.osdtype.example.com` (no trailing slash).
   This value is baked into the static bundle. After changing it you must
   redeploy the web service, not just restart it.
6. Deploy the API first. Wait for `/healthz` and `/readyz` to return `200`.
   Then deploy the web service.
7. If the API deploy fails with `JWTKEY must be set…`, the Blueprint's
   `generateValue` did not apply (for example, an existing service retained an
   old value). Set a fresh 32+ character secret manually and redeploy.
8. Confirm `build/_headers` exists in the static build (it comes from
   `app/static/_headers`). Render also applies the header rules in
   `render.yaml`.

The API refuses to start with the development JWT key when
`DEV_LOGIN=false` or `GIN_MODE=release`, and refuses
`COOKIE_SAMESITE=none` without `COOKIE_SECURE=true`.

### Custom domains

In Render, add `osdtype.example.com` to the web service and
`api.osdtype.example.com` to the API. Render shows the required CNAME targets.
Attach them at the DNS provider (or Cloudflare) before the next step. Once the
custom domains work, optionally disable the `onrender.com` subdomains in
Render if you do not need them.

`onrender.com` is on the Public Suffix List, so the default web and API hosts
are **different sites**. `COOKIE_SAMESITE=none` is required there. With custom
subdomains of one registrable domain, `lax` is usually preferable; change
`COOKIE_SAMESITE` to `lax` and redeploy if that matches your setup. Keep
`COOKIE_SECURE=true` either way.

## 5. Cloudflare / edge protection

Render provides network-level DDoS protection, but put Cloudflare in front for
WAF, bot management, rate limits and DNS control.

1. Add the site to Cloudflare and create the CNAME records Render gave you.
   Orange-cloud both web and API records. Apex domains can use Cloudflare's
   CNAME flattening.
2. **SSL/TLS → Overview**: use **Full (strict)**. Do not use Flexible.
3. **SSL/TLS → Edge Certificates**:
   - Always Use HTTPS: on
   - HSTS: on, include subdomains, preload only after you are certain
   - Minimum TLS version: 1.2
   - Opportunistic Encryption: on
4. **Speed → Optimization**: disable Rocket Loader for this site. It can
   interfere with SvelteKit and WebSocket startup.
5. **Caching → Cache Rules**:
   - Bypass cache for `api.osdtype.example.com/*`, `/admin*`, `/auth/*`,
     `/bugs*`, `/ws*`, `/healthz`, `/readyz` and `/metrics`.
   - Cache immutable `/_app/*` assets for a long time.
   - Never cache authenticated API responses or the SPA HTML behind a stale
     edge copy.
6. **Security → WAF**:
   - Enable Cloudflare Managed Rules and OWASP Core Rules.
   - Enable Bot Fight Mode (or Super Bot Fight Mode on paid plans).
   - Add custom rules to block obviously malicious methods/paths and
     oversized bodies at the edge (the app also caps bodies at 1 MiB).
   - Consider a managed challenge for `/auth/*` and `/bugs` if abuse starts.
7. **Security → Rate limiting** (thresholds are starting points; tune with
   real traffic):
   - `/auth/github/callback` and `/auth/dev-login`: 20 requests/minute/IP.
   - `/login/github`: 20/minute/IP.
   - `/bugs`: 5/hour/IP (the app also enforces this in Redis).
   - `/ws`: 30 connection attempts/minute/IP.
   - Catch-all API: 300 requests/minute/IP.
   - Use `CF-Connecting-IP` as the client identity in Cloudflare rules.
   - Free plans have limited rule slots; prioritise the auth, bug and
     catch-all rules.
8. **WebSockets**: Cloudflare supports WebSockets on all plans. Keep the proxy
   on, allow `Upgrade`, and do not add a cache rule for `/ws`. If you see
   disconnects, check the client heartbeat/reconnect path and Cloudflare
   analytics before changing the API.
9. **Emergency**: if an attack is active, raise Security Level / enable
   **Under Attack Mode**, tighten the rate-limit rules, and block offending
   ASNs or countries. Challenge mode can interrupt WebSocket clients, so
   prefer targeted rate limits for game traffic.
10. Cloudflare does not hide the Render origin by itself. Render's
    `onrender.com` host remains reachable unless disabled after a custom
    domain is attached. Do not rely on obscurity: the API still enforces auth,
    admin 404s, CORS, cookies and rate limits. Render IP allowlisting or a
    Cloudflare Tunnel/proxy architecture is a separate paid/infra decision.

## 6. Application security checklist

- [ ] `DEV_LOGIN=false` and `GIN_MODE=release` in production.
- [ ] `JWTKEY` is a fresh 32+ character secret; never the default.
- [ ] `COOKIE_SECURE=true`; `COOKIE_SAMESITE=none` only for cross-site
      `onrender.com` hosts, otherwise `lax`.
- [ ] `FRONTEND_URL` exactly matches the browser origin (scheme + host, no
      trailing slash). CORS is an allowlist, never `*`.
- [ ] `ADMIN_USERNAME=ItsArnavSh`, and the GitHub account is protected with
      MFA. The primary admin is case-insensitive and immutable.
- [ ] `/admin` returns `404` to a guest, a normal player and a revoked admin.
- [ ] `METRICS_TOKEN` is set; `/metrics` is not public.
- [ ] `BUG_WEBHOOK_URL` points to a private channel and contains no secrets.
- [ ] Supabase and Redis passwords are unique, rotated and stored in a
      password manager.
- [ ] Supabase backups/PITR are enabled and a restore has been tested.
- [ ] Redis persistence is on; eviction policy is understood.
- [ ] Cloudflare WAF/rate limits are live and alerts reach a human.
- [ ] Render deploy-failure and unhealthy-service notifications are enabled.
- [ ] `render.yaml` contains no real secrets.
- [ ] API is one instance; do not enable autoscaling until state is shared.

Application-level protections already in the code:

- Security headers and a deny-all API CSP; static headers in
  `app/static/_headers` and `render.yaml`.
- 1 MiB request body cap, 10s header timeout, 30s read timeout, 32 KiB header
  cap.
- `HttpOnly`, `Secure`, scoped auth cookies; explicit CORS allowlist.
- Cross-site OAuth fallback: the callback also returns the JWT in the URL
  fragment, which the SPA stores and sends as a Bearer token. The fragment is
  stripped before the page is used and is never sent to a server. Prefer a
  same-site web/API domain when possible so the HttpOnly cookie remains the
  primary session transport.
- OAuth state validation with constant-time comparison and a cleared state
  cookie.
- Hidden admin routes, `no-store` on admin responses and audited admin
  actions.
- Rate-limited bug reports; optional bearer token on `/metrics`; request IDs
  on every response and in error bodies.
- Paginated/capped admin queries (no unbounded table scans in the UI).

## 7. Verify the launch

Run from a machine outside Render:

```sh
curl -fsS https://api.osdtype.example.com/healthz
curl -fsS https://api.osdtype.example.com/readyz
curl -fsS -H "Authorization: Bearer $METRICS_TOKEN" \
  https://api.osdtype.example.com/metrics | head
curl -fsSI https://osdtype.example.com | grep -Ei \
  'strict-transport|x-content-type|x-frame|referrer|permissions'
```

Then in a browser:

- [ ] GitHub login completes and the session cookie is `Secure`, `HttpOnly`
      and scoped correctly.
- [ ] Practice works with the API up and offline where supported.
- [ ] Ranked queue finds a match; a second browser/account can play.
- [ ] A room can be created, joined and started.
- [ ] A contest can be scheduled and appears on `/contests`.
- [ ] WebSocket stays connected for several minutes and reconnects after a
      network blip; check the browser console and Render logs.
- [ ] `/bugs` submits a report; it appears in `/admin` with a request id.
- [ ] `/admin` works for `ItsArnavSh`; `/admin` is `404` for a normal user,
      a guest and a revoked admin.
- [ ] Admin pagination works: next/prev, search, filters, total counts.
- [ ] `/admin` logs filter by user, level and query; health pages; activity
      window; events/audits filters.
- [ ] A 404 URL shows the keyboard error page.
- [ ] The hub footer links (OSDC, author, GitHub) work, and no other page
      shows the old about/footer content.
- [ ] Mobile widths (390px) do not overlap the back key, footer or admin
      controls.

## 8. Secrets rotation

If a secret is exposed, rotate it in this order:

1. **Redis password**: rotate at the provider, update `REDIS_URL` in Render,
   redeploy, verify `/readyz` and one game.
2. **Supabase password**: rotate in Supabase, update `DATABASE_URL`, redeploy,
   verify login and a write. Keep the old password only for the rollback
   window.
3. **GitHub OAuth secret**: generate a new secret, update `GITHUB_AUTH` and the
   OAuth app, redeploy, test login.
4. **JWTKEY**: generate a new value, update Render, redeploy. All sessions are
   logged out; tell users to log in again.
5. **METRICS_TOKEN / BUG_WEBHOOK_URL**: rotate independently; update the
   scraper or webhook receiver.
6. Revoke any admin who should no longer have access from `/admin` and verify
   their next request returns `404`.

Rotate the Redis password **now** if it was ever pasted into chat. Changing it
in the provider without updating Render will take the API down; update both in
the same maintenance window.

## 9. Monitoring and alerts

- Uptime monitor → `https://api.osdtype.example.com/readyz` every minute.
  Alert after 2 consecutive failures.
- Second monitor → `https://osdtype.example.com` for the static site.
- Scrape `/metrics` with the bearer token. Alert on:
  - 5xx rate above 2% for 5 minutes
  - p95 latency above 1s for 5 minutes
  - in-flight requests stuck high
  - goroutine or heap growth across hours
- Render alerts: deploy failed, service unhealthy, high CPU/memory.
- Cloudflare alerts: spike in 4xx/5xx, WAF blocks, rate-limit hits.
- `/admin` for bug reports, active users per hour, health history and recent
  logs. Render's log stream is the durable request-log sink; export it to your
  log provider if you need longer retention.
- Supabase alerts for connection saturation, disk and backup failures.
- Redis alerts for memory, evictions and connection errors.

## 10. Incident response

1. **Suspected DDoS**: Cloudflare Security Level → High or Under Attack;
   tighten `/auth`, `/bugs` and catch-all rate limits; block bad ASNs/countries
   if clearly targeted. Keep `/readyz` reachable for monitoring.
2. **Credential leak**: rotate the affected secret immediately using the order
   above; revoke admin access; review `audit_logs`, `system_events`, request
   logs and Cloudflare logs.
3. **Database outage**: `/readyz` goes `503`; Redis-backed health history and
   Render/Cloudflare alerts remain visible. Put the API in maintenance mode if
   writes cannot be trusted; restore from Supabase backup/PITR; redeploy.
4. **Redis outage**: matchmaking/rooms/games fail; preserve Postgres data.
   Restore Redis from snapshot or accept loss of live state and let players
   re-queue.
5. **Bad deploy**: roll back to the last healthy Render deploy. Rotate secrets
   only if the bad build exposed them; check the build diff first.
6. **Abusive admin**: revoke them with the primary account; their next
   `/admin` request is a `404`. Rotate `JWTKEY` if their token may be stolen.
7. After recovery: write a short timeline, rotate anything touched, verify
   backups, and add a WAF/rate-limit or code guard for the failure mode.

## 11. Rollback and repeat deploys

- Render keeps deploy history: use **Rollback** on the service.
- The web static bundle is immutable per deploy; changing `VITE_API_URL`
  requires a new build.
- Database migrations are forward-only in practice. Back up before a schema
  change and test the migration against a copy.
- Keep `render.yaml` in sync with the dashboard. Blueprint changes can add or
  modify services; review the plan before applying.

## 12. Launch order (cheat sheet)

1. Rotate exposed Redis password.
2. Supabase project + backup/PITR + `DATABASE_URL`.
3. Managed Redis + TLS/persistence + `REDIS_URL`.
4. GitHub OAuth app + callback.
5. Push `main`; create Render Blueprint.
6. Set API env vars, deploy API, verify `/healthz` and `/readyz`.
7. Set `VITE_API_URL`, deploy web.
8. Attach custom domains; update `FRONTEND_URL`, `GITHUB_REDIRECT_URL`, OAuth
   callback, and redeploy.
9. Put Cloudflare in front; enable WAF, TLS and rate limits.
10. Log in as `ItsArnavSh`; verify admin 404s and the dashboard.
11. Run the full browser checklist in section 7.
12. Enable monitoring/alerts and announce.
