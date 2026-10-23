# Build the static SvelteKit site and serve it from the same Caddy container
# that terminates TLS and proxies the API. Build context is the repo root.
# syntax=docker/dockerfile:1

FROM node:22-alpine AS build

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

FROM caddy:2-alpine

COPY deploy/aws/Caddyfile /etc/caddy/Caddyfile
COPY --from=build /app/build /srv

EXPOSE 80 443 443/udp
