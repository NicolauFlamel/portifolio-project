# Build stage
FROM golang:1.23.0-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o api ./cmd/api

# Runtime stage
FROM alpine:latest

# Install ca-certificates for TLS connections and wget for health checks
RUN apk --no-cache add ca-certificates wget

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /build/api .

# Copy config file (optional, can be overridden by environment variables)
COPY config.yaml ./config.yaml

# Expose the default port (can be overridden by environment variable)
EXPOSE 3000

# Run the application
CMD ["./api"]
