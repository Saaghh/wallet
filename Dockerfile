FROM golang:1.21.5 as builder

COPY . /src
WORKDIR /src

RUN make build

FROM debian:stable-slim

COPY --from=builder /src/bin/apiserver /app/bin/apiserver

WORKDIR /app

ENTRYPOINT ["./bin/apiserver"]
