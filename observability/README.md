# Observability demo stack
# Services: OpenTelemetry Collector, Prometheus, Grafana, Loki, Promtail

## Start observability + gateway
# docker compose --profile observability --profile gateway up -d

## Local Langfuse (self-hosted) is included in the observability profile
# Langfuse UI: http://localhost:3001

## Enable PicoClaw -> local Langfuse OTLP export
# export PICOCLAW_OBSERVABILITY_LANGFUSE_ENABLED=true
# export PICOCLAW_OBSERVABILITY_LANGFUSE_HOST=http://langfuse-web:3000
# export PICOCLAW_OBSERVABILITY_LANGFUSE_PUBLIC_KEY=pk-lf-xxxx
# export PICOCLAW_OBSERVABILITY_LANGFUSE_SECRET_KEY=sk-lf-xxxx
# docker compose --profile observability --profile gateway up -d --force-recreate picoclaw-gateway

## Smoke test (inside Linux/devcontainer shell)
# docker compose --profile observability --profile gateway exec picoclaw-gateway \
#   picoclaw agent -m "list files in workspace"

## Verify status
# docker compose --profile observability --profile gateway exec picoclaw-gateway \
#   picoclaw status

## URLs
# Grafana: http://localhost:3000 (admin/admin)
# Prometheus: http://localhost:9090
# Loki API: http://localhost:3100
# Langfuse: http://localhost:3001
