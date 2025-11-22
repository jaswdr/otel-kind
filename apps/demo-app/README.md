# Demo App - OpenTelemetry Instrumented HTTP Server

A simple Go HTTP server demonstrating full OpenTelemetry instrumentation for traces, metrics, and logs.

## Overview

This application serves as a reference implementation for instrumenting Go applications with OpenTelemetry. It generates:

- **Distributed Traces**: Spans for HTTP handlers and internal operations
- **Metrics**: Request counters and latency histograms
- **Logs**: Structured logging with trace context

## Features

- HTTP server with request handling
- Automatic trace context propagation
- Custom metrics collection
- Structured JSON logging
- OTLP export (HTTP protocol)
- Simulated work with random latency

## Endpoints

- `GET /` - Main handler (generates telemetry)
- `GET /health` - Health check endpoint

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `tempo.lgtm.svc.cluster.local:4318` | OTLP endpoint |

## OpenTelemetry Instrumentation

### Traces

The application creates distributed traces with:

- **Root Span**: `handleRequest` - Represents the entire HTTP request
- **Child Span**: `doWork` - Represents internal business logic

Span attributes include:
- `http.method` - HTTP method (GET, POST, etc.)
- `http.url` - Request path
- `work.duration_ms` - Simulated work duration

### Metrics

Exported metrics:

1. **http_requests_total** (Counter)
   - Description: Total number of HTTP requests
   - Labels: `method`, `path`, `service_name`

2. **http_request_duration_seconds** (Histogram)
   - Description: HTTP request latency distribution
   - Labels: `method`, `path`, `service_name`
   - Buckets: [0, 0.005, 0.01, 0.025, 0.05, 0.075, 0.1, 0.25, 0.5, 0.75, 1, 2.5, 5, 7.5, 10]

### Logs

Structured logs include:
- Timestamp
- Log level
- Message
- Trace ID (for correlation)
- Span ID (for correlation)

Example log output:
```
2025/11/20 22:34:00 Received request: GET / from 10.244.2.14:54058
2025/11/20 22:34:00 Doing work for 252ms
2025/11/20 22:34:00 Request completed in 252.44ms
```

## Dependencies

```go
require (
    go.opentelemetry.io/otel v1.24.0
    go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.24.0
    go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.24.0
    go.opentelemetry.io/otel/metric v1.24.0
    go.opentelemetry.io/otel/sdk v1.24.0
    go.opentelemetry.io/otel/sdk/metric v1.24.0
)
```

## Building

### Local Build

```bash
go build -o demo-app main.go
```

### Docker Build

```bash
docker build -t demo-app:latest .
```

The Dockerfile uses a multi-stage build:
1. Builder stage: Compiles the Go binary
2. Runtime stage: Minimal Alpine image with the binary

## Running Locally

```bash
# Set OTLP endpoint (optional)
export OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4318

# Run the application
./demo-app
```

Or with Docker:

```bash
docker run -p 8080:8080 \
  -e OTEL_EXPORTER_OTLP_ENDPOINT=host.docker.internal:4318 \
  demo-app:latest
```

## Testing

### Send Test Request

```bash
curl http://localhost:8080/
```

Expected response:
```
Hello from demo-app! Request processed in 123.45ms
```

### Generate Load

```bash
for i in {1..100}; do
  curl http://localhost:8080/
  sleep 0.5
done
```

## Code Structure

```go
main.go
├── main()                    // Entry point, initializes OTEL
│   ├── initTracer()         // Configures trace exporter
│   ├── initMeter()          // Configures metrics exporter
│   └── http.ListenAndServe() // Starts HTTP server
│
├── handleRequest()           // HTTP handler
│   ├── Creates trace span
│   ├── Logs request info
│   ├── Calls doWork()
│   └── Records metrics
│
└── doWork()                  // Simulates business logic
    ├── Creates child span
    ├── Simulates work (sleep)
    └── Logs work details
```

## Key Concepts Demonstrated

### 1. Trace Propagation

```go
ctx, span := tracer.Start(r.Context(), "handleRequest")
defer span.End()

// Pass context to child function
doWork(ctx, duration)
```

### 2. Metrics Recording

```go
requestCounter.Add(ctx, 1,
    metric.WithAttributes(
        attribute.String("method", r.Method),
        attribute.String("path", r.URL.Path),
    ),
)
```

### 3. Structured Logging

```go
log.Printf("Request completed in %.2fms", latency*1000)
```

### 4. Resource Attributes

```go
resource.New(ctx,
    resource.WithAttributes(
        semconv.ServiceNameKey.String("demo-app"),
        semconv.ServiceVersionKey.String("1.0.0"),
    ),
)
```

## Observability Best Practices

This application demonstrates:

✅ **Context Propagation**: Trace context flows through all operations
✅ **Semantic Conventions**: Uses OpenTelemetry semantic conventions
✅ **Resource Attributes**: Identifies service name and version
✅ **Graceful Shutdown**: Flushes telemetry on shutdown
✅ **Error Handling**: Proper error handling for OTEL initialization
✅ **Configurable Endpoints**: Environment-based configuration

## Extending the Application

### Adding New Spans

```go
ctx, span := tracer.Start(ctx, "myOperation")
defer span.End()

span.SetAttributes(
    attribute.String("my.attribute", "value"),
)

// Do work...
```

### Adding New Metrics

```go
myCounter, err := meter.Int64Counter(
    "my_metric_total",
    metric.WithDescription("Description of my metric"),
)

myCounter.Add(ctx, 1,
    metric.WithAttributes(
        attribute.String("label", "value"),
    ),
)
```

### Adding Trace Attributes

```go
span.SetAttributes(
    attribute.String("user.id", userID),
    attribute.Int("response.status", statusCode),
)
```

## Troubleshooting

### Telemetry Not Appearing

1. Check OTLP endpoint is reachable:
   ```bash
   curl http://<endpoint>/v1/traces
   ```

2. Check application logs for errors

3. Verify OTEL Collector is receiving data

### High Memory Usage

The OTEL SDK batches telemetry. Adjust batch settings:

```go
sdktrace.WithBatcher(exporter,
    sdktrace.WithBatchTimeout(5*time.Second),
    sdktrace.WithMaxExportBatchSize(512),
)
```

## Resources

- [OpenTelemetry Go Documentation](https://opentelemetry.io/docs/languages/go/)
- [OTLP Specification](https://opentelemetry.io/docs/specs/otlp/)
- [OpenTelemetry Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/)

## License

This demo application is provided for educational purposes.
