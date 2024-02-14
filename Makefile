build:
	go build -o ./bin/apiserver ./cmd/apiserver
	go build -o ./bin/xrserver ./cmd/xrserver

tidy:
	go mod tidy

fmt:
	gofumpt -w .
	gci write . --skip-generated -s standard -s default

lint: tidy fmt build
	golangci-lint run

xr:
	go run ./cmd/xrserver -d

serve: up
	go run ./cmd/apiserver

up:
	docker compose up -d

test: build up
	go test -v ./tests

.PHONY: build tidy fmt lint serve up test xr

.DEFAULT_GOAL := lint