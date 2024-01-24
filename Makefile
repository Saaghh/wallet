build:
	go build -o ./bin/apiserver ./cmd/apiserver

tidy:
	go mod tidy

fmt:
	gofumpt -w .
	gci write . --skip-generated -s standard -s default

lint: tidy fmt build
	golangci-lint run

serve: up
	docker build -t wallet_apiserver .
	docker run -p 8080:8080  --name wallet_apiserver -d wallet_apiserver

up: 
	docker-compose up -d

update: clearContainer serve

clearContainer:
	docker stop wallet_apiserver && docker rm wallet_apiserver

.PHONY: build tidy fmt lint serve up update clearContainer

.DEFAULT_GOAL := lint