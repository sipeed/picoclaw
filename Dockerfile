# ============================================================
# Stage 1: Build the picoclaw binary
# ============================================================
FROM golang:1.25.7-alpine AS builder

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
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

# Copy binary
COPY --from=builder /src/build/picoclaw /usr/local/bin/picoclaw

# Copy builtin skills
COPY --from=builder /src/skills /opt/picoclaw/skills

# Create picoclaw home directory
RUN mkdir -p /root/.picoclaw/workspace/skills && \
    cp -r /opt/picoclaw/skills/* /root/.picoclaw/workspace/skills/ 2>/dev/null || true

ENTRYPOINT ["picoclaw"]
CMD ["gateway"]
