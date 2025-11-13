# Multi-stage build for Ethereum Validator Watcher (Go)
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o eth-validator-watcher ./cmd/watcher

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 watcher && \
    adduser -D -u 1000 -G watcher watcher

# Set working directory
WORKDIR /home/watcher

# Copy binary from builder
COPY --from=builder /app/eth-validator-watcher /usr/local/bin/eth-validator-watcher

# Copy example config
COPY config.example.yaml /home/watcher/config.example.yaml

# Change ownership
RUN chown -R watcher:watcher /home/watcher

# Switch to non-root user
USER watcher

# Expose metrics port
EXPOSE 8080

# Set entrypoint
ENTRYPOINT ["eth-validator-watcher"]

# Default command
CMD ["-config", "config.yaml"]
