package main

import (
	"context"
	"log"
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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

	// Initialize database
	db, err := InitDatabase(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create handler
	handler := NewHandler(db)

	// Setup routes with automatic instrumentation
	http.Handle("/", otelhttp.NewHandler(http.HandlerFunc(handler.HandleRequest), "handleRequest"))
	http.Handle("/health", otelhttp.NewHandler(http.HandlerFunc(handler.HealthCheck), "healthCheck"))

	// Start server
	log.Printf("Starting server on :%s", cfg.Port)
	log.Printf("OTEL endpoint: %s", cfg.OtelEndpoint)
	log.Printf("Database: %s", cfg.DBHost)

	if err := http.ListenAndServe(":"+cfg.Port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
