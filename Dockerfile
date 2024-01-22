FROM golang:1.21.5 as builder

COPY . /src
WORKDIR /src

RUN make build

FROM alpine:3.14

COPY --from=builder /src/bin /app/bin

WORKDIR /app

ENTRYPOINT ["./bin/apiserver"]