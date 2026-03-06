.PHONY: build test lint run

build:
	go build -o bin/ratatuile ./cmd/ratatuile

test:
	go test ./...

lint:
	go vet ./...

run:
	go run ./cmd/ratatuile
