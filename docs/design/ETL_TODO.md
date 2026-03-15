# ETL Visibility TODO List

This document tracks the tasks required to implement the "Ultimate Visibility" ETL framework.

## 1. Extract (Ingestion & Telemetry Collection)

- [ ] **Structured Logging:** Ensure `zerolog` is used consistently across the codebase for structured JSON logging. Add context to logs where missing (session IDs, tool inputs/outputs).
- [ ] **Basic Metrics Implementation:** Introduce a metrics package (e.g., using `expvar` or a Prometheus client) to expose basic application metrics.
- [x] **Goroutine Tracking:** Implement a metric to track the number of active Goroutines.
- [x] **Memory Tracking:** Implement a metric to track heap allocation and GC pauses.
- [ ] **AgentLoop Telemetry:** Add specific instrumentation to the `AgentLoop` (iteration duration, tool execution duration, failure counts).
- [ ] **LLM Provider Telemetry:** Track API call latency, token usage, and failover reasons for LLM providers.
- [ ] **API Gateway Telemetry:** Track request rates (RPS), latency percentiles, and error rates for HTTP and WebSocket endpoints.
- [ ] **Tracing Instrumentation:** Introduce trace IDs at entry points (HTTP, WebSocket) and propagate them via context to track end-to-end execution flow.

## 2. Transform (Stream Processing & Enrichment)

- [ ] **Log Normalization:** Standardize error classifications (e.g., Model Failure, Infrastructure Failure, Logic Failure) to ensure consistent log querying.
- [ ] **Aggregation Strategy:** Design the pipeline for aggregating high-volume events before they reach the data warehouse (e.g., Vector.dev configuration).

## 3. Load (Storage & Analytics)

- [ ] **Time-Series Database:** Set up or integrate with a time-series database (e.g., Prometheus) for metrics storage.
- [ ] **Log Warehouse:** Set up or integrate with an OLAP database (e.g., ClickHouse, Elasticsearch) for log and trace storage.
- [ ] **Dashboards:** Create initial Grafana dashboards visualizing the Four Golden Signals (Latency, Traffic, Errors, Saturation).
- [ ] **Alerting:** Configure alerts based on metric thresholds (e.g., GC pauses > 20ms, Goroutine counts continuously rising).
