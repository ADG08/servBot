# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /servbot ./cmd/bot

# Run stage (minimal, non-root)
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata && \
    adduser -D -g "" appuser

WORKDIR /app

COPY --from=builder --chown=appuser:appuser /servbot .

USER appuser

ENTRYPOINT ["./servbot"]
