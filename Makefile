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
	docker build -t wallet .
	docker run -p 8080:8080 -d wallet

.PHONY: build tidy fmt lint serve

.DEFAULT_GOAL := build