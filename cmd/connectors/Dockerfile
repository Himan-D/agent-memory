# Build stage
FROM golang:1.27-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /connectors ./cmd/connectors

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates && \
    adduser -D -u 1000 appuser

WORKDIR /app

COPY --from=builder /connectors .

USER appuser

EXPOSE 8083

ENV PORT=8083

CMD ["./connectors"]