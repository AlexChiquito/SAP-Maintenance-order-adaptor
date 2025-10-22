# Build stage
FROM golang:1.21-alpine3.18 AS builder

# Set working directory
WORKDIR /app

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Generate Swagger documentation
RUN go install github.com/swaggo/swag/cmd/swag@latest
RUN swag init -g cmd/server/main.go

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/server

# Final stage
FROM alpine:3.18

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN adduser -D -s /bin/sh appuser

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/main .

# Copy configuration files
COPY --from=builder /app/config.yaml .
COPY --from=builder /app/env.example .

# Change ownership to non-root user
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./main"]
