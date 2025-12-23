# Build stage
FROM golang:1.24-bookworm AS builder

WORKDIR /build

# Install build dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    git ca-certificates gcc libc6-dev && \
    rm -rf /var/lib/apt/lists/*

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the ingestion service
RUN CGO_ENABLED=1 GOOS=linux go build -o ingestion-service ./cmd/ingestion

# Runtime stage
FROM debian:bookworm-slim

# Use China mirror for apt (comment out if not needed)
RUN sed -i 's/deb.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list.d/debian.sources

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates tzdata && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/ingestion-service .

# Copy configuration files
COPY config/ingestion.defaults.yml ./config/

# Expose HTTP and gRPC ports
EXPOSE 8091 50051

# Run the service
CMD ["./ingestion-service"]