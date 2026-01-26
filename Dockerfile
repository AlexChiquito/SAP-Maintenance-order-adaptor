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

RUN apk --no-cache add ca-certificates wget

WORKDIR /app

COPY --from=builder /build/adaptor .
COPY --from=builder /build/config.yaml .

RUN mkdir -p /app/logs

EXPOSE 8080

CMD ["./adaptor"]

# Simulator stage
FROM alpine:latest AS simulator

RUN apk --no-cache add ca-certificates wget

WORKDIR /app

COPY --from=builder /build/simulator .

EXPOSE 8081

CMD ["./simulator"]
