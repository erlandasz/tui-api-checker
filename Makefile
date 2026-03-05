.PHONY: build test lint run

build:
	go build -o bin/postmaniux ./cmd/postmaniux

test:
	go test ./...

lint:
	go vet ./...

run:
	go run ./cmd/postmaniux
