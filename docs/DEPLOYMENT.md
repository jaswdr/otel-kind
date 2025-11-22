# LGTM Stack Deployment Summary

## Deployed Components

### 1. Grafana
- **Status**: Deployed and Running
- **Access**: http://localhost:3000 (via port-forward)
- **Credentials**:
  - Username: `admin`
  - Password: `admin`
- **Port Forward Command**: `kubectl port-forward -n lgtm svc/grafana 3000:80`

### 2. Tempo (Traces)
- **Status**: Deployed and Running
- **Endpoint**: http://tempo.lgtm.svc.cluster.local:3100
- **Data Source**: Configured in Grafana as "Tempo"

### 3. Loki (Logs)
- **Status**: Deployed and Running
- **Endpoint**: http://loki-gateway.lgtm.svc.cluster.local
- **Data Source**: Configured in Grafana as "Loki"

### 4. Mimir (Metrics)
- **Status**: Deployed and Running
- **Endpoint**: http://mimir-gateway.lgtm.svc:80/prometheus
- **Data Source**: Configured in Grafana as "Mimir" (Prometheus type)

### 5. OpenTelemetry Collector
- **Status**: Deployed and Running
- **Purpose**: Receives OTLP data from applications and routes to:
  - Traces → Tempo
  - Metrics → Mimir
  - Logs → Loki
- **Endpoints**:
  - OTLP gRPC: otel-collector.lgtm.svc.cluster.local:4317
  - OTLP HTTP: otel-collector.lgtm.svc.cluster.local:4318

### 6. Demo Go Application
- **Status**: Deployed and Running
- **Description**: HTTP server with OpenTelemetry instrumentation
- **Features**:
  - Generates distributed traces
  - Emits metrics (request count, latency)
  - Produces structured logs
- **Endpoint**: http://demo-app.lgtm.svc.cluster.local

## Configuration Files

All Helm values and Kubernetes manifests are saved in the current directory:

- `tempo-values.yaml` - Tempo configuration
- `loki-simple-values.yaml` - Loki configuration
- `mimir-values.yaml` - Mimir configuration
- `grafana-values.yaml` - Grafana configuration with all data sources
- `otel-collector-config.yaml` - OTEL Collector ConfigMap and Deployment
- `demo-app-deployment.yaml` - Demo application Deployment and Service
- `demo-app/` - Demo application source code
  - `main.go` - Application with OTEL instrumentation
  - `go.mod` - Go dependencies
  - `Dockerfile` - Container image definition

## Verification Steps

### 1. Access Grafana

```bash
# Ensure port-forward is running
kubectl port-forward -n lgtm svc/grafana 3000:80 &

# Open browser to http://localhost:3000
# Login with admin/admin
```

### 2. Verify Data Sources in Grafana

1. Navigate to **Configuration → Data Sources** (or http://localhost:3000/datasources)
2. You should see three data sources:
   - **Loki** (for logs)
   - **Mimir** (for metrics)
   - **Tempo** (for traces)

### 3. Verify Logs in Loki

1. Go to **Explore** in Grafana
2. Select **Loki** as the data source
3. Use the query: `{service_name="demo-app"}`
4. You should see logs from the demo application

### 4. Verify Metrics in Mimir

1. Go to **Explore** in Grafana
2. Select **Mimir** as the data source
3. Use the query: `http_requests_total{service_name="demo-app"}`
4. You should see request count metrics
5. Try: `histogram_quantile(0.95, rate(http_request_duration_seconds_bucket{service_name="demo-app"}[5m]))`
6. You should see P95 latency metrics

### 5. Verify Traces in Tempo

1. Go to **Explore** in Grafana
2. Select **Tempo** as the data source
3. Use **Search** tab
4. Filter by Service: `demo-app`
5. You should see distributed traces showing:
   - `handleRequest` span
   - `doWork` child span
   - Request attributes (method, URL, etc.)

## Generating More Traffic

To generate more test data:

```bash
kubectl apply -f traffic-generator.yaml
```

This will send 50 HTTP requests to the demo application over approximately 100 seconds.

## Pod Status

Check all pods are running:

```bash
kubectl get pods -n lgtm
```

Expected pods:
- grafana-*
- tempo-0
- loki-0, loki-gateway-*
- mimir-* (multiple components)
- otel-collector-*
- demo-app-*

## Architecture

```
[Demo App] --OTLP--> [OTEL Collector] ---> [Tempo] <--query-- [Grafana]
                           |                [Loki]
                           |                [Mimir]
                           |
                           +---> Prometheus RemoteWrite --> [Mimir]
                           +---> Loki Push --> [Loki]
                           +---> OTLP HTTP --> [Tempo]
```

## Troubleshooting

### Check OTEL Collector Logs
```bash
kubectl logs -n lgtm -l app=otel-collector
```

### Check Demo App Logs
```bash
kubectl logs -n lgtm -l app=demo-app
```

### Check Specific Component
```bash
kubectl logs -n lgtm <pod-name>
```

### Port Forward to Specific Services
```bash
# Loki
kubectl port-forward -n lgtm svc/loki-gateway 3100:80

# Mimir
kubectl port-forward -n lgtm svc/mimir-gateway 8080:80

# Tempo
kubectl port-forward -n lgtm svc/tempo 3200:3100
```

## Clean Up

To remove all components:

```bash
helm uninstall grafana -n lgtm
helm uninstall tempo -n lgtm
helm uninstall loki -n lgtm
helm uninstall mimir -n lgtm
kubectl delete namespace lgtm
```
