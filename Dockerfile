# ============================================================
# Stage 1: Build the picoclaw binary
# ============================================================
FROM golang:1.26.0-alpine3.23@sha256:d4c4845f5d60c6a974c6000ce58ae079328d03ab7f721a0734277e69905473e5 AS builder

RUN apk add --no-cache git make ca-certificates \
    && addgroup -S picoclaw \
    && adduser --uid 1000 --shell /bin/false -S picoclaw -G picoclaw \
    && grep picoclaw /etc/passwd > /etc/passwd_picoclaw \
    && grep picoclaw /etc/group > /etc/group_picoclaw

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
FROM alpine:3.23@sha256:25109184c71bdad752c8312a8623239686a9a2071e8825f20acb8f2198c3f659

RUN apk add --no-cache tzdata

COPY --from=builder /etc/passwd_picoclaw /etc/passwd
COPY --from=builder /etc/group_picoclaw /etc/group
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

RUN mkdir -p /home/picoclaw && chown picoclaw:picoclaw /home/picoclaw

# Health check
# BusyBox (Alpine default) already provides wget, no extra package needed
# Consider replacing with application-level health command in the future.
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q --spider http://localhost:18790/health || exit 1

# Copy binary
COPY --from=builder --chown=picoclaw:picoclaw /src/build/picoclaw /usr/local/bin/picoclaw

# Switch to non-root user
USER picoclaw
ENV HOME=/home/picoclaw

# Run onboard to create initial directories and config
RUN /usr/local/bin/picoclaw onboard

ENTRYPOINT ["picoclaw"]

CMD ["gateway"]
