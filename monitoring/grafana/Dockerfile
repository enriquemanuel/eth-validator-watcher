# Grafana with company MITM certificate
FROM grafana/grafana:latest

USER root

# Copy and install company MITM certificate
COPY ../../cert.pem /usr/local/share/ca-certificates/company-mitm.crt

# Update CA certificates (Grafana uses Alpine Linux)
RUN apk add --no-cache ca-certificates && \
    update-ca-certificates

USER grafana
