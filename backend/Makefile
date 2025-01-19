.PHONY: build test clean run docker-build docker-run lint

# Variables
APP_NAME=smart-insights
DOCKER_IMAGE=smart-insights:latest

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}"

build:
	@echo "Building ${APP_NAME}..."
	go build ${LDFLAGS} -o ${APP_NAME}

test:
	@echo "Running tests..."
	go test -v -race -cover ./...

clean:
	@echo "Cleaning..."
	rm -f ${APP_NAME}
	go clean

run: build
	./${APP_NAME} api

docker-build:
	@echo "Building Docker image..."
	docker build -t ${DOCKER_IMAGE} .

docker-run: docker-build
	@echo "Running Docker container..."
	docker run -p 8080:8080 ${DOCKER_IMAGE}

lint:
	@echo "Running linter..."
	go vet ./...
	if command -v golangci-lint >/dev/null; then \
		golangci-lint run; \
	else \
		echo "golangci-lint is not installed"; \
	fi