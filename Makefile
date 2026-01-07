.PHONY: build test lint clean dev

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

build:
	go build $(LDFLAGS) -o bin/radar ./cmd/radar

test:
	go test -race -cover ./...

lint:
	golangci-lint run ./...

clean:
	rm -rf bin/

dev: build
	./bin/radar serve

init-example: build
	./bin/radar init
	./bin/radar list

fmt:
	go fmt ./...

tidy:
	go mod tidy
