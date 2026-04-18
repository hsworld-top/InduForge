.PHONY: test build run fmt tidy

test:
	go test ./...

build:
	go build ./...

run:
	go run ./cmd

fmt:
	gofmt -w ./cmd ./internal

tidy:
	go mod tidy
