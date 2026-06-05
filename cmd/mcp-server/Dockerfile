# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /mcp-server ./cmd/mcp-server

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates && \
    adduser -D -u 1000 appuser

WORKDIR /app

COPY --from=builder /mcp-server .

USER appuser

EXPOSE 8082

ENV PORT=8082

CMD ["./mcp-server"]