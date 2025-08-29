# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o migr8 .

# Final stage
FROM alpine:3.18

# Install runtime dependencies for database clients
RUN apk add --no-cache \
    postgresql-client \
    mysql-client \
    sqlite \
    ca-certificates \
    tzdata

# Create non-root user
RUN adduser -D -s /bin/sh migr8user

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/migr8 .

# Copy configuration template
COPY --from=builder /app/examples/migr8.yaml.example /app/.migr8.yaml.example

# Create directories for migrations, backups, and seeds
RUN mkdir -p /app/migrations /app/backups /app/seeds && \
    chown -R migr8user:migr8user /app

# Switch to non-root user
USER migr8user

# Expose volume for data persistence
VOLUME ["/app/migrations", "/app/backups", "/app/seeds"]

# Set default configuration path
ENV MIGR8_CONFIG=/app/.migr8.yaml

# Default command
ENTRYPOINT ["./migr8"]
CMD ["--help"]