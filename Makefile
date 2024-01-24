build:
	go build -o ./bin/apiserver.exe ./cmd/apiserver

tidy:
	go mod tidy

fmt:
	gofumpt -w .
	gci write . --skip-generated -s standard -s default

lint: tidy fmt build
	golangci-lint run

serve: up
	docker build -t wallet .
	docker run -p 8080:8080 wallet

up: 
	docker-compose up -d
	
.PHONY: build tidy fmt lint serve up

.DEFAULT_GOAL := build