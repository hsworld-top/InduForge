.PHONY: test build run dev fmt tidy

test:
	go test ./...

build:
	go build ./...

run:
	go run ./cmd/dev_core

dev:
	air

fmt:
	gofmt -w ./cmd ./internal

tidy:
	go mod tidy
