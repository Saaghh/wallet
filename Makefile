.PHONY: build
build: 
	go mod tidy
	gofumpt -w .
	gci write . --skip-generated -s standard -s default
	golangci-lint run
	go build ./cmd/apiserver

.PHONY: lint
lint: 
	go mod tidy
	gofumpt -w .
	gci write . --skip-generated -s standard -s default
	golangci-lint run

.DEFAULT_GOAL := build