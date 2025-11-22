# LGTM Stack on Kubernetes

A complete observability stack deployment featuring **Loki** (logs), **Grafana** (visualization), **Tempo** (traces), and **Mimir** (metrics) on Kubernetes, with a fully instrumented demo application.

## Quick Start

```bash
# Deploy everything
make deploy

# Port-forward to Grafana
make port-forward

# Open browser to http://localhost:3000
# Login: admin/admin

# Generate traffic
make traffic

# Verify deployment
make verify
```

## Repository Structure

```
.
├── apps/                           # Applications
│   └── demo-app/                  # Demo Go application with OTEL instrumentation
│       ├── main.go                # Application code
│       ├── go.mod                 # Go dependencies
│       ├── Dockerfile             # Container image
│       └── README.md              # App-specific documentation
│
├── infrastructure/                 # Infrastructure as Code
│   ├── helm/                      # Helm chart values
│   │   ├── grafana/               # Grafana configuration
│   │   ├── tempo/                 # Tempo configuration
│   │   ├── loki/                  # Loki configuration
│   │   └── mimir/                 # Mimir configuration
│   │
│   └── kubernetes/                # Kubernetes manifests
│       ├── otel-collector/        # OpenTelemetry Collector
│       ├── demo-app/              # Demo app deployment
│       └── traffic-generator/     # Traffic generation job
│
├── scripts/                       # Utility scripts
│   ├── deploy.sh                 # Full deployment script
│   ├── verify.sh                 # Verification script
│   └── cleanup.sh                # Cleanup script
│
├── docs/                          # Documentation
│   ├── DEPLOYMENT.md             # Deployment guide
│   └── ARCHITECTURE.md           # Architecture documentation
│
├── Makefile                       # Common operations
├── .gitignore                    # Git ignore rules
└── README.md                     # This file
```

## What's Included

### Observability Stack

- **Grafana** - Unified visualization and querying interface
- **Tempo** - Distributed tracing backend (OpenTelemetry compatible)
- **Loki** - Log aggregation and querying
- **Mimir** - Long-term metrics storage (Prometheus compatible)
- **OpenTelemetry Collector** - Telemetry data collection and routing

### Demo Application

- Go HTTP server with complete OpenTelemetry instrumentation
- Generates traces, metrics, and logs
- Sends telemetry via OTLP protocol
- Demonstrates real-world observability patterns

## Prerequisites

- Kubernetes cluster (Kind, Minikube, or any K8s cluster)
- `kubectl` configured
- `helm` 3.x installed
- `docker` for building images
- `make` (optional, for convenience commands)

## Deployment

### Option 1: Using Make (Recommended)

```bash
# Deploy everything
make deploy

# Or deploy step by step
make deploy-infra    # Deploy Helm charts only
make deploy-apps     # Deploy applications only
```

### Option 2: Using Script

```bash
./scripts/deploy.sh
```

### Option 3: Manual Deployment

```bash
# 1. Create namespace
kubectl create namespace lgtm

# 2. Add Helm repo
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update

# 3. Deploy Helm charts
helm install tempo grafana/tempo -n lgtm -f infrastructure/helm/tempo/values.yaml
helm install loki grafana/loki -n lgtm -f infrastructure/helm/loki/values.yaml
helm install mimir grafana/mimir-distributed -n lgtm -f infrastructure/helm/mimir/values.yaml
helm install grafana grafana/grafana -n lgtm -f infrastructure/helm/grafana/values.yaml

# 4. Deploy OTEL Collector
kubectl apply -f infrastructure/kubernetes/otel-collector/deployment.yaml

# 5. Build and deploy demo app
docker build -t demo-app:latest apps/demo-app/
kind load docker-image demo-app:latest  # For Kind clusters
kubectl apply -f infrastructure/kubernetes/demo-app/deployment.yaml
```

## Accessing Grafana

### Port Forward

```bash
# Using Make
make port-forward

# Or directly
kubectl port-forward -n lgtm svc/grafana 3000:80
```

Then open http://localhost:3000

**Credentials**:
- Username: `admin`
- Password: `admin`

## Using the Stack

### View Logs (Loki)

1. Go to **Explore** in Grafana
2. Select **Loki** data source
3. Query: `{service_name="demo-app"}`

### View Metrics (Mimir)

1. Go to **Explore** in Grafana
2. Select **Mimir** data source
3. Try these queries:
   - `http_requests_total` - Total requests
   - `rate(http_requests_total[5m])` - Request rate
   - `histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))` - P95 latency

### View Traces (Tempo)

1. Go to **Explore** in Grafana
2. Select **Tempo** data source
3. Use **Search** to find traces by service name: `demo-app`
4. Click any trace to see the full request flow

## Common Operations

### Generate Traffic

```bash
make traffic
```

This creates a Kubernetes job that sends 50 requests to the demo app over ~100 seconds.

### Check Status

```bash
# All pods
make status

# Demo app logs
make logs-demo

# OTEL Collector logs
make logs-otel

# Grafana logs
make logs-grafana
```

### Verify Deployment

```bash
make verify
```

Runs a verification script that checks:
- Grafana accessibility
- Data source configuration
- Pod health
- Traffic flow

### Rebuild Demo App

```bash
# After making changes to apps/demo-app/
make rebuild-demo
```

This rebuilds the image, loads it into the cluster, and restarts the pod.

## Cleanup

```bash
# Remove everything except namespace
make clean

# Remove namespace as well
make clean-namespace

# Or use the interactive script
./scripts/cleanup.sh
```

## Architecture

```
┌──────────┐
│ Demo App │ (Instrumented with OpenTelemetry)
└────┬─────┘
     │ OTLP
     ↓
┌─────────────┐
│ OTEL        │ Routes telemetry to appropriate backends
│ Collector   │
└──┬──┬───┬──┘
   │  │   │
   ↓  ↓   ↓
┌──────┐ ┌─────┐ ┌───────┐
│Tempo │ │Loki │ │ Mimir │
│      │ │     │ │       │
│Traces│ │Logs │ │Metrics│
└──┬───┘ └──┬──┘ └───┬───┘
   │        │        │
   └────────┴────────┘
            │
       ┌────▼────┐
       │ Grafana │ (Visualization & Querying)
       └─────────┘
```

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for detailed architecture documentation.

## Documentation

- [DEPLOYMENT.md](docs/DEPLOYMENT.md) - Detailed deployment guide
- [ARCHITECTURE.md](docs/ARCHITECTURE.md) - Architecture and design decisions
- [Demo App README](apps/demo-app/README.md) - Application-specific documentation

## Key Features

✅ **Complete LGTM Stack** - All four pillars of observability
✅ **Pre-configured Integration** - Data sources automatically configured in Grafana
✅ **Correlation** - Traces, logs, and metrics are correlated
✅ **Production-Ready Patterns** - Based on Grafana best practices
✅ **OpenTelemetry** - Using modern observability standards
✅ **Infrastructure as Code** - All configuration versioned and repeatable
✅ **Easy to Deploy** - Single command deployment
✅ **Well Documented** - Comprehensive documentation

## Troubleshooting

### Pods Not Starting

```bash
# Check pod status
kubectl get pods -n lgtm

# Describe problematic pod
kubectl describe pod <pod-name> -n lgtm

# Check logs
kubectl logs <pod-name> -n lgtm
```

### No Data in Grafana

1. Verify pods are running: `make status`
2. Check OTEL Collector logs: `make logs-otel`
3. Generate more traffic: `make traffic`
4. Wait a few minutes for data to propagate

### Port Forward Not Working

```bash
# Kill existing port forwards
pkill -f "port-forward"

# Restart
make port-forward
```

## Development

### Adding a New Application

1. Create directory under `apps/<app-name>/`
2. Add Dockerfile and source code
3. Create deployment manifest in `infrastructure/kubernetes/<app-name>/`
4. Update Makefile with build/deploy targets

### Modifying Infrastructure

1. Update Helm values in `infrastructure/helm/<component>/values.yaml`
2. Redeploy: `helm upgrade --install <release> ... -f <values-file>`

### Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## Resources

- [Grafana Documentation](https://grafana.com/docs/)
- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Helm Documentation](https://helm.sh/docs/)

## License

This project is provided as-is for educational and demonstration purposes.

## Credits

Built with:
- [Grafana](https://grafana.com/)
- [Tempo](https://grafana.com/oss/tempo/)
- [Loki](https://grafana.com/oss/loki/)
- [Mimir](https://grafana.com/oss/mimir/)
- [OpenTelemetry](https://opentelemetry.io/)
- [Kubernetes](https://kubernetes.io/)

---

**Status**: ✅ Deployed and Operational

For questions or issues, please check the [troubleshooting section](#troubleshooting) or review the [architecture documentation](docs/ARCHITECTURE.md).
