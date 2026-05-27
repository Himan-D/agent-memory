# Multi-stage build for Hystersis Cloud MCP Server
# Usage: docker build -f deploy/mcp/Dockerfile -t hystersis-mcp-cloud .

# --- Build stage ---
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Cache dependency downloads
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build the cloud MCP server binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /mcp-cloud \
    ./cmd/mcp-cloud

# --- Runtime stage ---
FROM scratch

# Import CA certificates for HTTPS upstream calls
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary
COPY --from=builder /mcp-cloud /mcp-cloud

# Cloud Run uses PORT env var; default to 8090
ENV PORT=8090

EXPOSE 8090

ENTRYPOINT ["/mcp-cloud"]
