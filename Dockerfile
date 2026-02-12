FROM golang:1.25-alpine AS builder
RUN apk add --no-cache git make
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION=docker
ARG BUILD_TIME
RUN BUILD_TIME=$(date +%FT%T%z) && \
    GO_VERSION=$(go version | awk '{print $3}') && \
    go build -v \
    -ldflags "-X main.version=${VERSION} -X main.buildTime=${BUILD_TIME} -X main.goVersion=${GO_VERSION}" \
    -o /app/picoclaw \
    ./cmd/picoclaw

# Runtime stage
FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/picoclaw /usr/local/bin/picoclaw
COPY --from=builder /app/skills /app/skills
RUN mkdir -p /root/.picoclaw/workspace
ENV PICOCLAW_HOME=/root/.picoclaw
ENV WORKSPACE_DIR=/root/.picoclaw/workspace
EXPOSE 18790
ENTRYPOINT ["picoclaw"]
CMD ["agent"]