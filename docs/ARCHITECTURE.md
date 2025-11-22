# LGTM Stack Architecture

## Overview

This repository implements a complete observability stack based on the LGTM (Loki, Grafana, Tempo, Mimir) components, deployed on Kubernetes using Helm and custom manifests.

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Kubernetes Cluster                       │
│                                                                   │
│  ┌─────────────┐                                                │
│  │  Demo App   │ HTTP Server (Port 8080)                        │
│  │  (Go)       │ - Request handling                             │
│  └──────┬──────┘ - Business logic                               │
│         │                                                         │
│         │ OTLP (HTTP 4318)                                       │
│         │                                                         │
│  ┌──────▼──────────────────────────────────────────┐            │
│  │        OpenTelemetry Collector                  │            │
│  │  ┌──────────────────────────────────────────┐   │            │
│  │  │ Receivers:                               │   │            │
│  │  │  - OTLP gRPC (4317)                     │   │            │
│  │  │  - OTLP HTTP (4318)                     │   │            │
│  │  └──────────────────────────────────────────┘   │            │
│  │  ┌──────────────────────────────────────────┐   │            │
│  │  │ Processors:                              │   │            │
│  │  │  - Batch (batching telemetry)           │   │            │
│  │  │  - Resource (add/modify attributes)     │   │            │
│  │  └──────────────────────────────────────────┘   │            │
│  │  ┌──────────────────────────────────────────┐   │            │
│  │  │ Exporters:                               │   │            │
│  │  │  - OTLP HTTP → Tempo (traces)           │   │            │
│  │  │  - Prometheus RemoteWrite → Mimir       │   │            │
│  │  │  - Loki → Loki (logs)                   │   │            │
│  │  └──────────────────────────────────────────┘   │            │
│  └──┬──────────┬────────────┬────────────────────┘            │
│     │          │            │                                    │
│     │          │            │                                    │
│  ┌──▼──────┐ ┌─▼───────┐ ┌─▼──────┐                           │
│  │  Tempo  │ │  Loki   │ │ Mimir  │                           │
│  │         │ │         │ │        │                           │
│  │ Traces  │ │  Logs   │ │Metrics │                           │
│  └──┬──────┘ └─┬───────┘ └─┬──────┘                           │
│     │          │            │                                    │
│     └──────────┴────────────┴───────────┐                      │
│                                          │                      │
│                                    ┌─────▼──────┐               │
│                                    │  Grafana   │               │
│                                    │            │               │
│                                    │ Query & UI │               │
│                                    └────────────┘               │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

## Components

### 1. Demo Application

**Technology**: Go with OpenTelemetry SDK

**Responsibilities**:
- HTTP server handling requests
- Generate distributed traces using OTEL SDK
- Emit metrics (counters, histograms)
- Produce structured logs
- Send telemetry via OTLP

**Key Features**:
- Auto-instrumentation for HTTP handlers
- Custom spans for business logic
- Metrics: request count, latency distribution
- Logs: structured JSON with trace context

**Endpoints**:
- `GET /` - Main handler (generates traces, logs, metrics)
- `GET /health` - Health check

### 2. OpenTelemetry Collector

**Type**: Collector Contrib (includes all receivers/exporters)

**Pipeline Configuration**:

#### Traces Pipeline
```yaml
receivers: [otlp]
processors: [batch, resource]
exporters: [otlphttp/tempo, logging]
```

#### Metrics Pipeline
```yaml
receivers: [otlp]
processors: [batch, resource]
exporters: [prometheusremotewrite, logging]
```

#### Logs Pipeline
```yaml
receivers: [otlp]
processors: [batch, resource]
exporters: [loki, logging]
```

**Why OTEL Collector?**:
- Decouples applications from backend implementation
- Centralized configuration for routing
- Batching and retry logic
- Resource attribute enrichment
- Protocol translation (OTLP → Loki/Prometheus/Tempo formats)

### 3. Tempo (Distributed Tracing)

**Deployment**: Single instance with local storage

**Configuration**:
- Backend: Local filesystem (persistent volume)
- OTLP receivers enabled (gRPC:4317, HTTP:4318)
- Trace retention: Based on storage capacity

**Features**:
- Ingests OTLP traces
- Stores trace data efficiently
- Provides TraceQL query language
- Integrated with Grafana for visualization

**Storage**:
- Persistent Volume: 10Gi
- Format: Native Tempo block format

### 4. Loki (Log Aggregation)

**Deployment**: Single binary mode

**Configuration**:
- Storage: Filesystem (TSDB + filesystem chunks)
- Schema: v13 (latest)
- Index: TSDB with 24h period
- No authentication required (internal use)

**Components**:
- **Loki**: Single binary (all-in-one)
- **Gateway**: Nginx for load balancing and routing
- **Caches**: Chunks cache, results cache

**Features**:
- LogQL query language
- Label-based indexing (like Prometheus)
- Efficient storage with compression
- Integration with Grafana for log exploration

### 5. Mimir (Long-term Metrics Storage)

**Deployment**: Distributed mode

**Components**:
- **Distributor**: Receives metrics, validates, replicates
- **Ingester**: Writes to storage, serves recent queries
- **Querier**: Handles PromQL queries
- **Query Frontend**: Query optimization and caching
- **Compactor**: Background compaction
- **Store Gateway**: Queries long-term storage
- **Ruler**: Alerting and recording rules
- **MinIO**: Object storage backend

**Configuration**:
- Replication factor: 3 (across zones)
- Object storage: MinIO (S3-compatible)
- Query sharding: Enabled
- Prometheus remote write endpoint

**Features**:
- PromQL compatible
- Horizontal scalability
- Multi-tenancy support (disabled for simplicity)
- Cardinality limits and rate limiting

### 6. Grafana (Visualization)

**Deployment**: Single instance

**Pre-configured Data Sources**:

1. **Tempo**:
   - Type: Tempo
   - URL: http://tempo.lgtm.svc.cluster.local:3100
   - Trace to logs correlation (→ Loki)
   - Trace to metrics correlation (→ Mimir)
   - Service graph support

2. **Loki**:
   - Type: Loki
   - URL: http://loki-gateway.lgtm.svc.cluster.local
   - Max lines: 1000
   - Derived fields for trace correlation

3. **Mimir**:
   - Type: Prometheus
   - URL: http://mimir-gateway.lgtm.svc:80/prometheus
   - Default data source
   - Scrape interval: 30s

**Features**:
- Unified query interface (Explore)
- Dashboard creation
- Alerting (configured separately)
- User authentication (admin/admin)
- Data source correlation

## Data Flow

### Trace Flow

```
Demo App
  → OTLP Trace (HTTP 4318)
    → OTEL Collector (receives)
      → Batch Processor (batches spans)
        → Resource Processor (adds attributes)
          → OTLP HTTP Exporter
            → Tempo (stores trace blocks)
              → Grafana (queries for visualization)
```

### Metrics Flow

```
Demo App
  → OTLP Metrics (HTTP 4318)
    → OTEL Collector (receives)
      → Batch Processor (batches metrics)
        → Resource Processor (adds attributes)
          → Prometheus Remote Write Exporter
            → Mimir Distributor (validates & replicates)
              → Mimir Ingesters (writes to storage)
                → MinIO (long-term storage)
                  → Grafana (queries via PromQL)
```

### Logs Flow

```
Demo App
  → OTLP Logs (HTTP 4318)
    → OTEL Collector (receives)
      → Batch Processor (batches logs)
        → Resource Processor (adds attributes)
          → Loki Exporter
            → Loki Gateway (routes)
              → Loki Single Binary (indexes & stores)
                → Filesystem (chunks + index)
                  → Grafana (queries via LogQL)
```

## Correlation

### Trace ↔ Logs

- Traces include `trace_id` and `span_id`
- Logs include `trace_id` from context
- Grafana can jump from trace span to related logs
- Configuration in Tempo data source: `tracesToLogsV2`

### Trace ↔ Metrics

- Service-level metrics correlated by `service_name`
- Exemplars link metrics to traces (if configured)
- Grafana shows metrics for services in traces
- Configuration in Tempo data source: `tracesToMetrics`

### Logs ↔ Traces

- LogQL queries can extract trace IDs
- Derived fields in Loki data source
- Click trace ID in log line → Opens trace in Tempo

## Storage

### Tempo
- **Type**: Persistent Volume (local-path)
- **Size**: 10Gi
- **Format**: Native Tempo blocks
- **Retention**: Space-based (oldest blocks deleted first)

### Loki
- **Type**: Persistent Volume (local-path)
- **Size**: 5Gi
- **Components**:
  - Index: TSDB (efficient label indexing)
  - Chunks: Compressed log data
- **Retention**: Configurable (default: unlimited)

### Mimir
- **Type**: MinIO (S3-compatible object storage)
- **Size**: 10Gi
- **Components**:
  - Blocks: Prometheus TSDB blocks
  - Metadata: Block metadata
- **Retention**: Configurable per tenant

## Networking

### Internal Service Communication

| Service | Port | Protocol | Purpose |
|---------|------|----------|---------|
| demo-app | 8080 | HTTP | Application endpoint |
| otel-collector | 4317 | gRPC | OTLP receiver |
| otel-collector | 4318 | HTTP | OTLP receiver |
| tempo | 3100 | HTTP | Query API |
| loki-gateway | 80 | HTTP | Push/Query API |
| mimir-gateway | 80 | HTTP | Remote Write/Query |
| grafana | 80 | HTTP | Web UI |

### External Access

- **Grafana**: NodePort 30000 or port-forward to 3000
- **All others**: Internal only (no external exposure)

## Scalability Considerations

### Current Setup (Development/Testing)
- Single replicas for most components
- Local storage (not production-grade)
- No high availability

### Production Recommendations

1. **Tempo**:
   - Use object storage (S3, GCS, Azure Blob)
   - Scale queriers and distributors independently
   - Enable caching (memcached/redis)

2. **Loki**:
   - Use Simple Scalable or Microservices mode
   - Object storage for chunks
   - Scale read/write paths independently
   - Add querier sharding

3. **Mimir**:
   - Already distributed (good starting point)
   - Increase replication factor (production: 3)
   - Use production-grade object storage
   - Scale ingesters and queriers based on load

4. **OTEL Collector**:
   - Deploy as DaemonSet for logs collection
   - Add load balancer for central collectors
   - Use sampling for high-volume traces

## Security Considerations

### Current Setup
- No TLS/encryption
- Basic authentication for Grafana only
- No network policies
- No pod security policies

### Production Recommendations
- Enable TLS for all services
- Implement mTLS between components
- Use service mesh (Istio, Linkerd)
- Network policies to restrict traffic
- Pod security standards
- Secret management (Vault, Sealed Secrets)
- RBAC for Kubernetes resources

## Observability of Observability

The stack includes self-monitoring capabilities:

1. **Loki Canary**: Generates test logs to verify Loki health
2. **Logging Exporter**: OTEL Collector logs to stdout for debugging
3. **Grafana Health Checks**: Built-in health endpoints
4. **Mimir Metrics**: Self-exported Prometheus metrics

## Troubleshooting

### Common Issues

1. **No traces in Tempo**:
   - Check OTEL Collector logs
   - Verify OTLP endpoint configuration
   - Confirm Tempo receivers are enabled

2. **Missing logs in Loki**:
   - Check Loki gateway status
   - Verify label configuration
   - Check OTEL Collector Loki exporter config

3. **Metrics not in Mimir**:
   - Verify distributor logs
   - Check ingester status
   - Confirm MinIO is accessible

4. **Grafana data source errors**:
   - Test data source connection
   - Verify service URLs
   - Check network policies

### Debug Commands

```bash
# Check all pods
kubectl get pods -n lgtm

# OTEL Collector logs
kubectl logs -n lgtm -l app=otel-collector

# Demo app logs
kubectl logs -n lgtm -l app=demo-app

# Tempo logs
kubectl logs -n lgtm tempo-0

# Loki logs
kubectl logs -n lgtm loki-0

# Mimir distributor logs
kubectl logs -n lgtm -l app.kubernetes.io/component=distributor
```

## References

- [Grafana Tempo Documentation](https://grafana.com/docs/tempo/latest/)
- [Grafana Loki Documentation](https://grafana.com/docs/loki/latest/)
- [Grafana Mimir Documentation](https://grafana.com/docs/mimir/latest/)
- [OpenTelemetry Collector Documentation](https://opentelemetry.io/docs/collector/)
- [OpenTelemetry Go SDK](https://opentelemetry.io/docs/languages/go/)
