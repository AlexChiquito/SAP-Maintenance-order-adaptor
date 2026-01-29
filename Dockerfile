# Multi-stage Dockerfile for SAP Adaptor and Simulator
# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binaries
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o adaptor ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o simulator ./cmd/simulator

# Adaptor stage
FROM alpine:latest AS adaptor

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

RUN apk --no-cache add ca-certificates wget

WORKDIR /app

COPY --from=builder /build/adaptor .
COPY --from=builder /build/config.yaml .

# Create logs directory with proper permissions
RUN mkdir -p /app/logs && \
    chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

EXPOSE 8080

CMD ["./adaptor"]

# Simulator stage
FROM alpine:latest AS simulator

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

RUN apk --no-cache add ca-certificates wget

WORKDIR /app

COPY --from=builder /build/simulator .

# Change ownership to non-root user
RUN chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

EXPOSE 8081

CMD ["./simulator"]
