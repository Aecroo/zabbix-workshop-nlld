# Stage 1: Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /build

# Copy go.mod (and go.sum if it exists) for dependency management
COPY go.mod ./
# go.sum may not exist if using only standard library
RUN test -f go.sum && cp go.sum . || true
RUN go mod download 2>/dev/null || true

# Copy the rest of the source code
COPY . .

# Build the binary with static linking (no CGO) for minimal runtime dependencies
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o api ./cmd/api

# Stage 2: Runtime stage
FROM alpine:3.19

# Install wget for health check
RUN apk --no-cache add wget ca-certificates

# Create non-root user for security
RUN adduser -D appuser

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /build/api .

# Change ownership to non-root user
RUN chown -R appuser:appuser /app

USER appuser

EXPOSE 8080

# Health check endpoint
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:8080/health || exit 1

CMD ["./api"]
