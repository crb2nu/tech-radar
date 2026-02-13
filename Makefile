.PHONY: build test lint clean dev

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

IMAGE_REGISTRY ?= registry.flexinfer.ai
IMAGE_NAME ?= $(IMAGE_REGISTRY)/services/tech-radar
IMAGE_TAG ?= $(VERSION)

build: frontend
	go build $(LDFLAGS) -o bin/radar ./cmd/radar

docker-build:
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) .
	docker tag $(IMAGE_NAME):$(IMAGE_TAG) $(IMAGE_NAME):latest

docker-push:
	docker push $(IMAGE_NAME):$(IMAGE_TAG)
	docker push $(IMAGE_NAME):latest

frontend:
	cd web && pnpm install && pnpm build

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
