# A Go-only image: it holds the chuckd binary and nothing else. It runs the
# standalone Dogwood disseminator (`chuckd sys dogwood`) and the Dogwood and
# echo benchmarks, but not the coordinator: `chuckd sys coordinator` needs
# scpd, the Rust SCP daemon from the thruput-scp-server crate, which this
# image does not build or package. For a coordinator, use the release archives
# (chuckd-coordinator-linux-*.tar.gz), which ship both binaries.

# Build stage
FROM golang:1.26-alpine AS builder

# Install git and other dependencies
RUN apk add --no-cache git ca-certificates tzdata curl jq && \
    update-ca-certificates

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /chuckd ./chuckd

# Final stage
FROM alpine:3.21

# Add necessary certificates and timezone data
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Install AWS CLI and other utilities
RUN apk add --no-cache aws-cli wget curl jq
RUN adduser -D -g '' chuckd

# Set working directory
WORKDIR /app

# Copy the binary from builder
COPY --from=builder /chuckd /app/chuckd

# Make scripts executable
RUN mkdir -p /app/config /app/data && \
    chown -R chuckd:chuckd /app

# Switch to non-root user
USER chuckd

# Expose app port
EXPOSE 9111

# Expose metrics port
EXPOSE 2222
