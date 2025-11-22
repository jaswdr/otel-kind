package main

import (
	"context"
	"log"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

func main() {
	ctx := context.Background()

	// Load configuration
	cfg := LoadConfig()

	// Initialize OpenTelemetry
	tp, err := InitTracer(ctx, cfg.OtelEndpoint)
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer tp.Shutdown(ctx)

	mp, err := InitMeter(ctx, cfg.OtelEndpoint)
	if err != nil {
		log.Fatalf("Failed to initialize meter: %v", err)
	}
	defer mp.Shutdown(ctx)

	// Create metrics
	meter := otel.Meter("demo-app")
	requestCounter, err := meter.Int64Counter("http_requests_total",
		metric.WithDescription("Total number of HTTP requests"))
	if err != nil {
		log.Fatalf("Failed to create counter: %v", err)
	}

	requestLatency, err := meter.Float64Histogram("http_request_duration_seconds",
		metric.WithDescription("HTTP request latency in seconds"))
	if err != nil {
		log.Fatalf("Failed to create histogram: %v", err)
	}

	// Initialize database
	db, err := InitDatabase(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create handler
	handler := NewHandler(db, requestCounter, requestLatency)

	// Setup routes
	http.Handle("/", otelhttp.NewHandler(http.HandlerFunc(handler.HandleRequest), "handleRequest"))
	http.HandleFunc("/health", handler.HealthCheck)

	// Start server
	log.Printf("Starting server on :%s", cfg.Port)
	log.Printf("OTEL endpoint: %s", cfg.OtelEndpoint)
	log.Printf("Database: %s", cfg.DBHost)

	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
