.PHONY: build run test lint clean

build:
	go build -o bin/oriontelescope ./cmd/oriontelescope

run:
	go run ./cmd/oriontelescope

test:
	go test ./...

lint:
	go vet ./...
	staticcheck ./...

clean:
	rm -rf bin/
