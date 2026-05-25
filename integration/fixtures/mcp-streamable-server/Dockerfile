FROM golang:1.25-bookworm AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /out/mcp-streamable-server ./integration/fixtures/mcp-streamable-server

FROM alpine:3.22

RUN adduser -D -u 10001 appuser

COPY --from=build /out/mcp-streamable-server /usr/local/bin/mcp-streamable-server

USER appuser
EXPOSE 8080

HEALTHCHECK --interval=5s --timeout=3s --retries=12 CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1

ENTRYPOINT ["/usr/local/bin/mcp-streamable-server"]
