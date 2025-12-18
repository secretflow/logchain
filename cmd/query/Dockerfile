# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the query service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o query-service ./cmd/query

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/query-service .

# Copy configuration files
COPY config/query.defaults.yml ./config/
COPY config/clients/ ./config/clients/

# Create log directory
RUN mkdir -p /var/log/query

# Expose HTTP port
EXPOSE 8082

# Run the service
CMD ["./query-service"]
