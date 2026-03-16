# ─── Build stage ─────────────────────────────────────────────────────────────
# Build context: picoclaw repo root
# go.mod 中有 replace github.com/sipeed/picoclaw => ../../，需要 repo 根目录可见
FROM golang:1.25-alpine AS builder

# 国内 Go 模块代理
ENV GOPROXY=https://goproxy.cn,direct
ENV GONOSUMCHECK=*

# 复制整个 repo，保证 replace 指令的相对路径 ../../ 在容器内成立
WORKDIR /build
COPY . .

WORKDIR /build/cmd/picoclaw-voice
RUN go mod download
# 纯 Go 静态构建，无 CGO 依赖
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /app/picoclaw-voice .

# ─── Runtime stage ────────────────────────────────────────────────────────────
FROM scratch

COPY --from=golang:1.25-alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/picoclaw-voice /picoclaw-voice

EXPOSE 8765
CMD ["/picoclaw-voice"]
