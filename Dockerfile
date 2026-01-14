# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install git for go mod download
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /associate ./cmd/associate

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates

COPY --from=builder /associate /associate

# Default environment for Neo4j connection
ENV NEO4J_URI=bolt://neo4j:7687
ENV NEO4J_USERNAME=neo4j
ENV NEO4J_PASSWORD=password
ENV NEO4J_DATABASE=neo4j

EXPOSE 8080

# Default to stdio mode for MCP client connections
# Use docker-compose or -http flag for HTTP server mode
ENTRYPOINT ["/associate"]
