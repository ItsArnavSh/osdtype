FROM golang:1.26 AS builder

WORKDIR /src
ENV CGO_ENABLED=0 GOOS=linux
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
RUN go build -trimpath -ldflags="-s -w" -o /out/osdtyp .

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app \
    && adduser -S -G app app

WORKDIR /app
COPY --from=builder /out/osdtyp /usr/local/bin/osdtyp
COPY server/boot ./boot

USER app
EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=3s --start-period=10s --retries=5 \
    CMD wget -qO- http://127.0.0.1:8080/ping >/dev/null || exit 1

ENTRYPOINT ["osdtyp"]
