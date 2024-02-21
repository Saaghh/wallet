FROM golang:1.22 as builder

COPY ../.. /src
WORKDIR /src

RUN CGO_ENABLED=0 GOOS=linux go build -o bin/xrserver cmd/xrserver/main.go

FROM debian:stable-slim

COPY --from=builder /src/bin/xrserver /app/bin/xrserver

WORKDIR /app

ENTRYPOINT ["./bin/xrserver"]