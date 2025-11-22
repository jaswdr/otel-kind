# Quick Start Guide

## Complete LGTM Stack with PostgreSQL and Auto-Instrumentation

This guide will get you up and running with the complete observability stack in minutes.

## Prerequisites

- Docker
- Kind (Kubernetes in Docker)
- kubectl
- Helm 3.x
- make

## One-Command Deploy

```bash
make quick-start
```

This single command will:
1. Create a Kind cluster
2. Deploy the LGTM stack (Loki, Grafana, Tempo, Mimir)
3. Deploy PostgreSQL database
4. Build and deploy the demo app
5. Start a continuous load generator
6. Port-forward Grafana to localhost:3000

## What Gets Deployed

### LGTM Namespace
- **Grafana** - Visualization and querying
- **Tempo** - Distributed tracing
- **Loki** - Log aggregation
- **Mimir** - Metrics storage
- **OTEL Collector** - Telemetry routing

### Default Namespace
- **PostgreSQL** - Database with sample data
- **Demo App** - Go application with full auto-instrumentation
- **Load Generator** - Continuous traffic generator

## Namespace Organization

- **lgtm namespace**: All observability infrastructure
- **default namespace**: All applications (demo-app, postgres, load-generator)

This separation ensures clean organization and demonstrates cross-namespace observability.

## Key Features

### ✅ Auto-Instrumentation

The demo app uses OpenTelemetry auto-instrumentation:

1. **HTTP Requests** - Automatically traced via `otelhttp`
2. **Database Queries** - PostgreSQL queries automatically traced via `otelsql`
3. **Context Propagation** - Trace context flows through all layers

### ✅ PostgreSQL Integration

- **Database**: Pre-populated with products and orders tables
- **Queries Traced**: All SELECT and INSERT operations visible in Tempo
- **Grafana Data Source**: PostgreSQL configured as data source

### ✅ Continuous Load Generation

Load generator continuously sends traffic (1-5 second intervals) to generate:
- Distributed traces with database spans
- Request metrics
- Application logs

## Access Points

### Grafana
```bash
# Already running if you used quick-start, otherwise:
make port-forward

# Access at: http://localhost:3000
# Username: admin
# Password: admin
```

### Data Sources

Grafana comes pre-configured with:
1. **Loki** - For logs
2. **Mimir** - For metrics (default)
3. **Tempo** - For traces (with correlation to logs & metrics)
4. **PostgreSQL** - For database queries

## Verify Everything Works

### 1. Check Pod Status
```bash
make status
```

Expected output:
- All pods in `lgtm` namespace: Running
- All pods in `default` namespace: Running

### 2. View Logs

```bash
# Demo app logs
make logs-demo

# Load generator logs
make logs-load-generator

# PostgreSQL logs
make logs-postgres

# OTEL Collector logs
make logs-otel
```

### 3. Explore in Grafana

#### View Traces (Tempo)
1. Go to **Explore**
2. Select **Tempo** data source
3. Search for service: `demo-app`
4. Click any trace to see:
   - HTTP request span
   - Database query spans (SELECT products, INSERT orders)
   - Work simulation spans

#### View Logs (Loki)
1. Go to **Explore**
2. Select **Loki** data source
3. Query: `{service_name="demo-app"}`
4. See structured logs with trace IDs

#### View Metrics (Mimir)
1. Go to **Explore**
2. Select **Mimir** data source
3. Queries:
   - `http_requests_total` - Request count
   - `rate(http_requests_total[5m])` - Request rate
   - `histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))` - P95 latency

#### Query PostgreSQL
1. Go to **Explore**
2. Select **PostgreSQL** data source
3. Query: `SELECT * FROM products;`
4. See the products table data

## Database Schema

### Products Table
```sql
CREATE TABLE products (
  id SERIAL PRIMARY KEY,
  name VARCHAR(100),
  description TEXT,
  price DECIMAL(10, 2),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Orders Table
```sql
CREATE TABLE orders (
  id SERIAL PRIMARY KEY,
  product_id INTEGER REFERENCES products(id),
  quantity INTEGER,
  total_price DECIMAL(10, 2),
  status VARCHAR(20),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Trace Flow with Database

When a request hits the demo app:

```
HTTP Request
  ↓
[handleRequest span]
  ├─ [queryProducts span]
  │   └─ [SQL: SELECT * FROM products] ← Auto-instrumented
  │
  ├─ [doWork span]
  │   └─ Simulated work
  │
  └─ [createOrder span]
      ├─ [SQL: SELECT price FROM products WHERE id=?] ← Auto-instrumented
      └─ [SQL: INSERT INTO orders ...] ← Auto-instrumented
```

All SQL operations are automatically instrumented and visible in the trace!

## Common Commands

```bash
# Create cluster
make create-cluster

# Deploy everything
make deploy

# Deploy only LGTM stack
make deploy-infra

# Deploy only applications
make deploy-apps

# Rebuild demo app after changes
make rebuild-demo

# Restart components
make restart-demo
make restart-load-generator
make restart-otel

# Clean up
make clean                    # Remove deployments
make delete-cluster           # Delete entire cluster

# Get help
make help
```

## Troubleshooting

### Demo app not connecting to PostgreSQL

```bash
# Check PostgreSQL is running
kubectl get pods -n default -l app=postgres

# Check demo app logs
make logs-demo

# Restart demo app
make restart-demo
```

### No traces appearing

```bash
# Check OTEL Collector logs
make logs-otel

# Check if load generator is running
make logs-load-generator

# Restart OTEL Collector
make restart-otel
```

### Grafana data source errors

1. Check if all pods are running: `make status`
2. Wait a few minutes for services to be fully ready
3. Restart Grafana: `kubectl rollout restart deployment/grafana -n lgtm`

## Architecture Highlights

### Auto-Instrumentation

The demo app achieves auto-instrumentation through:

1. **HTTP Layer**: `otelhttp.NewHandler()` wraps HTTP handlers
2. **Database Layer**: `otelsql.Register()` wraps database driver
3. **Trace Propagation**: Context flows through all layers

No manual span creation needed for:
- HTTP requests
- Database queries
- Connection pool operations

### Continuous Load

The load generator is a Deployment (not a Job), ensuring:
- Continuous traffic 24/7
- Automatic restart if it fails
- Consistent data generation for dashboards

### Namespace Separation

- **lgtm namespace**: Observability infrastructure
  - Isolated from application concerns
  - Can be shared across multiple app namespaces

- **default namespace**: Applications
  - Demo app
  - PostgreSQL database
  - Load generator
  - Mimics real-world deployment patterns

## Next Steps

1. **Create Dashboards**: Build Grafana dashboards for your metrics
2. **Add More Apps**: Deploy additional applications with OTEL instrumentation
3. **Explore Correlations**: Click from traces → logs → metrics in Grafana
4. **Query Database**: Use PostgreSQL data source to explore order data

## Resources

- Main README: [README.md](README.md)
- Architecture Guide: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- Deployment Details: [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)
- Demo App README: [apps/demo-app/README.md](apps/demo-app/README.md)

---

**Ready to explore?** Run `make quick-start` and open http://localhost:3000!
