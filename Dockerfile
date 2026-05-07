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

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S picoclaw \
    && adduser -S -G picoclaw -h /home/picoclaw picoclaw

ENV HOME=/home/picoclaw
ENV PICOCLAW_GATEWAY_HOST=0.0.0.0

WORKDIR /home/picoclaw

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q --spider http://localhost:18790/health || exit 1

# Copy binary and first-run entrypoint.
COPY --from=builder /src/build/picoclaw /usr/local/bin/picoclaw
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh \
    && mkdir -p /home/picoclaw/.picoclaw \
    && chown -R picoclaw:picoclaw /home/picoclaw

USER picoclaw

VOLUME ["/home/picoclaw/.picoclaw"]
EXPOSE 18790

ENTRYPOINT ["/entrypoint.sh"]
