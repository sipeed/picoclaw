# ============================================================
# Stage 1: Build the picoclaw binary
# ============================================================
FROM golang:1.26.0-alpine AS builder

RUN apk add --no-cache git make ca-certificates tzdata

WORKDIR /src

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 make build GOFLAGS="-v -trimpath" LDFLAGS='-ldflags "-s -w"'

# Create non-root user entry for scratch
RUN echo "picoclaw:x:10001:10001::/home/picoclaw:/sbin/nologin" > /tmp/passwd && \
  echo "picoclaw:x:10001:" > /tmp/group && \
  mkdir -p /home/picoclaw

# ============================================================
# Stage 2: Minimal runtime image (scratch)
# ============================================================
FROM scratch

# Copy user/group files for non-root execution
COPY --from=builder /tmp/passwd /etc/passwd
COPY --from=builder /tmp/group /etc/group

# Copy SSL certs and timezone data
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy home directory (owned by picoclaw user)
COPY --from=builder --chown=10001:10001 /home/picoclaw /home/picoclaw

# Copy binary
COPY --from=builder /src/build/picoclaw /usr/local/bin/picoclaw

USER 10001

EXPOSE 18790

ENTRYPOINT ["picoclaw"]
CMD ["gateway"]
