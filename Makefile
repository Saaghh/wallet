build:
	go build -o ./bin/apiserver ./cmd/apiserver

tidy:
	go mod tidy

fmt:
	gofumpt -w .
	gci write . --skip-generated -s standard -s default

lint: tidy fmt build
	golangci-lint run

serve:
	go run ./cmd/apiserver/main.go

.PHONY: build tidy fmt lint serve

.DEFAULT_GOAL := build