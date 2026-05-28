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
# Stage 2: Runtime with Node.js for Playwright browser automation
# ============================================================
FROM node:20-slim

# Install Chromium system dependencies + utilities
# Using Playwright's recommended deps for Debian/Ubuntu
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates tzdata wget curl \
    # Chromium dependencies
    libasound2 libatk-bridge2.0-0 libatk1.0-0 libcairo2 libcups2 \
    libdbus-1-3 libdrm2 libexpat1 libgbm1 libglib2.0-0 libnspr4 \
    libnss3 libpango-1.0-0 libx11-6 libxcb1 libxcomposite1 \
    libxdamage1 libxext6 libxfixes3 libxkbcommon0 libxrandr2 \
    libatspi2.0-0 libwayland-client0 \
    # Fonts
    fonts-liberation fonts-noto-color-emoji \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# Install Playwright and Chromium browser
RUN npx -y playwright@1.58.2 install chromium

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q --spider http://localhost:18790/health || exit 1

# Copy binary and first-run entrypoint (same as release image).
COPY --from=builder /src/build/picoclaw /usr/local/bin/picoclaw
COPY docker/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Copy entrypoint scripts
COPY docker/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
COPY docker/entrypoint-job.sh /home/picoclaw/entrypoint-job.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh /home/picoclaw/entrypoint-job.sh

# Create non-root user and group
RUN userdel -r node 2>/dev/null || true && \
    groupadd --gid 1000 picoclaw && \
    useradd --create-home --uid 1000 --gid 1000 picoclaw && \
    # Copy Playwright browsers to picoclaw user
    mkdir -p /home/picoclaw/.cache && \
    cp -r /root/.cache/ms-playwright /home/picoclaw/.cache/ms-playwright 2>/dev/null || true && \
    chown -R picoclaw:picoclaw /home/picoclaw

# Switch to non-root user
USER picoclaw

# Run onboard to create initial directories and config
RUN /usr/local/bin/picoclaw onboard

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["gateway"]
