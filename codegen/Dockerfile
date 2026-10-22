FROM rust:1.92-slim-bookworm AS builder

WORKDIR /src
COPY codegen/Cargo.toml codegen/Cargo.lock ./
COPY codegen/src ./src
RUN cargo build --release

FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --system --create-home codegen

COPY --from=builder /src/target/release/codegen /usr/local/bin/codegen

USER codegen
ENV CODEGEN_BIND=0.0.0.0:8081
EXPOSE 8081

HEALTHCHECK --interval=10s --timeout=2s --start-period=5s --retries=5 \
    CMD curl -fsS http://127.0.0.1:8081/health >/dev/null || exit 1

ENTRYPOINT ["codegen"]
