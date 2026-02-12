# Build stage
FROM golang:1.25.7 AS builder

WORKDIR /app

# Copy everything
COPY . .

# Force proxy
ENV GOPROXY=https://proxy.golang.org,direct

# Fetch deps
RUN go mod tidy

# Build binary from cmd/picoclaw
RUN go build -tags netgo -ldflags "-s -w" -o app ./cmd/picoclaw

# Final stage
FROM debian:bookworm-slim
WORKDIR /app
COPY --from=builder /app/app .
CMD ["./app"]
