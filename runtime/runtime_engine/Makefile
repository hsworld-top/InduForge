GO ?= go

.PHONY: build test vet
build:
	$(GO) build -o bin/runtime-engine ./cmd/runtime-engine
test:
	$(GO) test ./...
vet:
	$(GO) vet ./...
