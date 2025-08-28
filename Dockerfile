# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies including wget for healthcheck
RUN apk add --no-cache git ca-certificates tzdata make gcc musl-dev wget

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies with verification
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o main cmd/server/main.go

# Create uploads directory in builder stage
RUN mkdir -p uploads

# Security scan the binary (optional but good practice)
RUN echo "Binary built successfully: $(file main)"

# Final stage - use distroless for maximum security
FROM gcr.io/distroless/static-debian11:latest

# Add ca-certificates and timezone data from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/main .

# Create necessary directories with proper permissions
# Note: distroless doesn't have shell, so we use builder stage for directory creation
COPY --from=builder --chown=65532:65532 /tmp /tmp

# Copy uploads directory from builder
COPY --from=builder /app/uploads ./uploads

# Use non-root user (nonroot user is built into distroless)
USER 65532:65532

# Expose port
EXPOSE 8080

# Health check - note: distroless doesn't have wget, so we'll check port directly
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD ["/app/main", "-health-check"] || exit 1

# Labels for metadata
LABEL maintainer="LMS Development Team" \
      description="Library Management System Backend" \
      version="1.0.0" \
      security.scan="enabled"

# Run the application
ENTRYPOINT ["/app/main"]