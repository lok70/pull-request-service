.PHONY: build run test lint

build:
	go build ./cmd/server

run:
	DB_DSN=postgres://pruser:prpassword@localhost:5432/prreviewer?sslmode=disable go run ./cmd/server

test:
	go test ./...

lint:
	golangci-lint run ./...
