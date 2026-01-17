.PHONY: build run test clean docker-up docker-down deps

# Default target
all: deps build

# Download dependencies
deps:
	go mod tidy

# Build the application
build:
	go build -o bin/associate ./cmd/associate

# Run locally (stdio mode)
run: build
	./bin/associate

# Run locally (HTTP mode)
run-http: build
	./bin/associate -http -port 8080

# Run tests
test:
	go test -v ./...

# Run integration tests (requires Neo4j)
test-integration:
	go test -tags=integration -v ./internal/neo4j

# Test persistence (run before and after docker restart)
test-persistence:
	go test -tags=integration -v ./internal/neo4j -run TestPersistence

# Clean build artifacts
clean:
	rm -rf bin/

# Docker commands
docker-up:
	docker-compose up --build -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

# Development: just start Neo4j
neo4j-up:
	docker-compose up -d neo4j

neo4j-down:
	docker-compose down neo4j
