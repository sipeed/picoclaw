# ============================================================
# Stage 1: Build the picoclaw binary
# ============================================================
FROM golang:1.26.0-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN make build

# ============================================================
# Stage 2: Node.js-based runtime with full MCP support
# ============================================================
FROM node:24-alpine3.23

# Install runtime dependencies
RUN apk add --no-cache \
  ca-certificates \
  curl \
  git \
  python3 \
  py3-pip

# Install uv and symlink to system path
RUN curl -LsSf https://astral.sh/uv/install.sh | sh && \
  ln -s /root/.local/bin/uv /usr/local/bin/uv && \
  ln -s /root/.local/bin/uvx /usr/local/bin/uvx && \
  uv --version

# Copy binary
COPY --from=builder /src/build/picoclaw /usr/local/bin/picoclaw

# Create picoclaw home directory
RUN /usr/local/bin/picoclaw onboard

ENTRYPOINT ["picoclaw"]
CMD ["gateway"]
