.PHONY: build test run clean

build:
	go build -o bin/server ./cmd/server

test:
	go test ./... -cover

run:
	go run ./cmd/server

clean:
	rm -rf bin/
