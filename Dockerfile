# ============================================================
# Stage 1: Build the picoclaw binary
# ============================================================
FROM golang:1.26.0-alpine AS builder

RUN apk add --no-cache git make nodejs npm

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN make ui-build
RUN make build

# ============================================================
# Stage 2: Minimal runtime image
# ============================================================
FROM alpine:3.23

RUN apk add --no-cache ca-certificates tzdata curl

# Copy binary
COPY --from=builder /src/build/picoclaw /usr/local/bin/picoclaw

# Copy built Web UI assets
COPY --from=builder /src/ui/dist /usr/local/share/picoclaw/ui/dist

# Create picoclaw home directory
RUN /usr/local/bin/picoclaw onboard

ENTRYPOINT ["picoclaw"]
CMD ["gateway"]
