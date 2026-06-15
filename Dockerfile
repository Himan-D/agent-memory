FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /agent-memory ./cmd/server

FROM alpine:3.24

RUN apk --no-cache add ca-certificates && \
    adduser -D -u 1000 appuser

WORKDIR /app

COPY --from=builder /agent-memory .

USER appuser

CMD ["./agent-memory"]
