# ============================================================
# Stage 1: Build the picoclaw binary
# ============================================================
FROM golang:1.26.0-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

WORKDIR /src

# Cache dependencies for faster subsequent builds
COPY go.mod go.sum ./
RUN go mod download

# Copy your local source code (where you'll add the Thought Signature fix)
COPY . .

# Compile the binary
RUN make build

# ============================================================
# Stage 2: Minimal runtime image
# ============================================================
FROM alpine:3.23

# Install runtime essentials
RUN apk add --no-cache ca-certificates tzdata curl

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q --spider http://localhost:18790/health || exit 1

# Copy the compiled binary from the builder stage
COPY --from=builder /src/build/picoclaw /usr/local/bin/picoclaw

# Create necessary directories and initialize
RUN /usr/local/bin/picoclaw onboard

# Set the binary as the entrypoint
ENTRYPOINT ["picoclaw"]
CMD ["gateway"]