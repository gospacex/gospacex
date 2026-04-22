.PHONY: build test run clean

build:
	go build -o bin/agent ./cmd/agent

test:
	go test ./... -cover

run:
	go run ./cmd/agent

clean:
	rm -rf bin/
