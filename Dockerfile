
FROM golang:latest AS builder

ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64

WORKDIR /app/server

COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/ .
RUN go build -o /app/app .

FROM alpine:latest

WORKDIR /root/
COPY --from=builder /app/app .
COPY server/boot ./boot
COPY server/grammar ./grammar
EXPOSE 8080

CMD ["./app"]
