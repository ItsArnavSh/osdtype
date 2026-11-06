# Build the static SvelteKit site and serve it from the same Caddy container
# that terminates TLS and proxies the API. Build context is the repo root.
# syntax=docker/dockerfile:1

# The static site is platform-independent, so build it on the native build
# platform. Running npm under QEMU for an arm64 image is needlessly slow.
FROM --platform=$BUILDPLATFORM node:22-alpine AS build

WORKDIR /app
ARG VITE_API_URL
ARG VITE_APP_VERSION=v0.0.1
ENV ADAPTER=static
ENV VITE_API_URL=$VITE_API_URL
ENV VITE_APP_VERSION=$VITE_APP_VERSION

COPY app/package.json app/package-lock.json ./
RUN npm ci

COPY app/ ./
RUN npm run build

# Caddy's official image has no rate-limit handler. Cross-compile the same
# Caddy release with mholt/caddy-ratelimit on the native build platform; Go
# cross-compilation is dramatically faster than compiling under QEMU. The
# plugin is pinned to a commit so production builds are reproducible.
FROM --platform=$BUILDPLATFORM caddy:2.11.4-builder AS caddy-build
ARG TARGETARCH
ENV CGO_ENABLED=0 GOOS=linux GOARCH=$TARGETARCH
RUN xcaddy build v2.11.4 \
	--with github.com/mholt/caddy-ratelimit@5625512f24f6f59d6f64fb3aafe5eecff0b286db

FROM caddy:2.11.4-alpine

COPY --from=caddy-build /usr/bin/caddy /usr/bin/caddy
RUN caddy list-modules | grep -qx 'http.handlers.rate_limit'

COPY deploy/aws/Caddyfile /etc/caddy/Caddyfile
COPY --from=build /app/build /srv

EXPOSE 80 443 443/udp
