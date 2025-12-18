# Multi-stage build for Ethereum Validator Watcher (Go)
FROM golang:1.21-alpine AS builder

# Copy MITM certificate if it exists (for local development behind corporate proxy)
# This allows the Dockerfile to work both locally (with cert) and in CI/open-source (without cert)
COPY cert.pem* /tmp/

# Install MITM certificate if present (for local development)
# If cert.pem doesn't exist, this step is skipped gracefully
RUN if [ -f /tmp/cert.pem ]; then \
        mkdir -p /usr/local/share/ca-certificates && \
        cp /tmp/cert.pem /usr/local/share/ca-certificates/company-mitm.crt && \
        cat /usr/local/share/ca-certificates/company-mitm.crt >> /etc/ssl/certs/ca-certificates.crt && \
        update-ca-certificates && \
        echo "MITM certificate installed for local development"; \
    else \
        echo "No MITM certificate found - skipping (normal for open-source builds)"; \
    fi

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
# Use direct mode if behind proxy (will work with or without cert)
ENV GOPROXY=direct
ENV SSL_CERT_FILE=/etc/ssl/certs/ca-certificates.crt
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags="-w -s" \
    -o eth-validator-watcher ./cmd/watcher

# Final stage
FROM alpine:latest

# Copy MITM certificate if it exists (for local development)
COPY cert.pem* /tmp/

# Install MITM certificate BEFORE apk add (needed for SSL verification behind proxy)
# Alpine has a basic ca-certificates.crt even before installing the package
RUN if [ -f /tmp/cert.pem ]; then \
        mkdir -p /etc/ssl/certs && \
        cat /tmp/cert.pem >> /etc/ssl/certs/ca-certificates.crt && \
        echo "MITM certificate added to ca-certificates.crt for local development"; \
    else \
        echo "No MITM certificate found - skipping (normal for open-source builds)"; \
    fi

# Now install runtime dependencies (apk can verify SSL with the cert we just added)
RUN apk add --no-cache ca-certificates tzdata wget

# Properly register the MITM certificate if present (after ca-certificates package is installed)
RUN if [ -f /tmp/cert.pem ]; then \
        mkdir -p /usr/local/share/ca-certificates && \
        cp /tmp/cert.pem /usr/local/share/ca-certificates/company-mitm.crt && \
        update-ca-certificates && \
        echo "MITM certificate properly registered"; \
    fi

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
