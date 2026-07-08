.PHONY: run build seed lint test test-race fmt

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

seed:
	go run ./cmd/seed

lint:
	gofmt -s -l .
	go vet ./...

test:
	go test ./...

test-race:
	go test ./... -race

fmt:
	gofmt -s -w .
