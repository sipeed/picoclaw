# ============================================================
# Stage 1: Build the picoclaw binary
# ============================================================
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN make build

# ============================================================
# Stage 2: Minimal runtime image
# ============================================================
FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata curl

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -fsS http://localhost:18790/health >/dev/null || exit 1

# Copy binary
COPY --from=builder /src/build/picoclaw /usr/local/bin/picoclaw
COPY docker/entrypoint.sh /entrypoint.sh

# Create non-root user and group
RUN addgroup -g 1000 picoclaw && \
    adduser -D -u 1000 -h /home/picoclaw -G picoclaw picoclaw && \
    mkdir -p /home/picoclaw/.picoclaw && \
    chown -R picoclaw:picoclaw /home/picoclaw && \
    chmod +x /entrypoint.sh

ENV HOME=/home/picoclaw

# Switch to non-root user
USER picoclaw

ENTRYPOINT ["/entrypoint.sh"]
CMD ["gateway"]
