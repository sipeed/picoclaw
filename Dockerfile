# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache make git

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
# We manually set LDFLAGS to ensure they are correct in the container
RUN VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev") && \
    BUILD_TIME=$(date +%FT%T%z) && \
    go build -ldflags "-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME}" -o picoclaw ./cmd/picoclaw

# Runtime stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/picoclaw /usr/local/bin/picoclaw

# Copy builtin skills
COPY --from=builder /app/skills /app/skills

# Set environment variables
ENV PICOCLAW_HOME=/app/.picoclaw

# Create workspace directory
RUN mkdir -p /app/.picoclaw/workspace

# Set the entrypoint
ENTRYPOINT ["picoclaw"]
CMD ["gateway"]
