package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/XSAM/otelsql"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	tracer         trace.Tracer
	meter          metric.Meter
	requestCounter metric.Int64Counter
	requestLatency metric.Float64Histogram
	db             *sql.DB
)

type Product struct {
	ID          int
	Name        string
	Description string
	Price       float64
}

func initTracer(ctx context.Context) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "otel-collector.lgtm.svc.cluster.local:4318")),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String("demo-app"),
			semconv.ServiceVersionKey.String("2.0.0"),
		),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return tp, nil
}

func initMeter(ctx context.Context) (*sdkmetric.MeterProvider, error) {
	exporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "otel-collector.lgtm.svc.cluster.local:4318")),
		otlpmetrichttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String("demo-app"),
			semconv.ServiceVersionKey.String("2.0.0"),
		),
	)
	if err != nil {
		return nil, err
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(10*time.Second))),
		sdkmetric.WithResource(res),
	)

	otel.SetMeterProvider(mp)

	return mp, nil
}

func initDatabase(ctx context.Context) (*sql.DB, error) {
	// Register the pgx driver with OpenTelemetry instrumentation
	driverName, err := otelsql.Register("pgx",
		otelsql.WithAttributes(semconv.DBSystemPostgreSQL),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register otel driver: %w", err)
	}

	// Build connection string
	dbHost := getEnv("DB_HOST", "postgres.default.svc.cluster.local")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "demouser")
	dbPass := getEnv("DB_PASSWORD", "demopass")
	dbName := getEnv("DB_NAME", "demoapp")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPass, dbName)

	// Open database connection with instrumented driver
	database, err := sql.Open(driverName, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	database.SetMaxOpenConns(25)
	database.SetMaxIdleConns(5)
	database.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection
	if err := database.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Record database stats for monitoring
	err = otelsql.RegisterDBStatsMetrics(database, otelsql.WithAttributes(
		semconv.DBSystemPostgreSQL,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to record stats: %w", err)
	}

	log.Println("Database connection established")
	return database, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	// Get tracer span from auto-instrumentation
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("http.method", r.Method),
		attribute.String("http.url", r.URL.Path),
	)

	log.Printf("Received request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

	// Query database with automatic tracing
	products, err := queryProducts(ctx)
	if err != nil {
		log.Printf("Error querying products: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Simulate some work
	workDuration := time.Duration(rand.Intn(500)) * time.Millisecond
	doWork(ctx, workDuration)

	// Create a sample order (demonstrates INSERT query tracing)
	if len(products) > 0 {
		productID := products[rand.Intn(len(products))].ID
		if err := createOrder(ctx, productID, rand.Intn(5)+1); err != nil {
			log.Printf("Error creating order: %v", err)
		}
	}

	// Record metrics
	requestCounter.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("method", r.Method),
			attribute.String("path", r.URL.Path),
		),
	)

	latency := time.Since(start).Seconds()
	requestLatency.Record(ctx, latency,
		metric.WithAttributes(
			attribute.String("method", r.Method),
			attribute.String("path", r.URL.Path),
		),
	)

	log.Printf("Request completed in %.2fms", latency*1000)

	// Respond with product list
	fmt.Fprintf(w, "Hello from demo-app!\n\nFound %d products:\n", len(products))
	for _, p := range products {
		fmt.Fprintf(w, "- %s: $%.2f\n", p.Name, p.Price)
	}
	fmt.Fprintf(w, "\nRequest processed in %.2fms\n", latency*1000)
}

func queryProducts(ctx context.Context) ([]Product, error) {
	ctx, span := tracer.Start(ctx, "queryProducts")
	defer span.End()

	query := "SELECT id, name, description, price FROM products ORDER BY created_at DESC LIMIT 10"

	// The SQL query will be automatically traced by otelsql
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		span.RecordError(err)
		return nil, err
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price); err != nil {
			span.RecordError(err)
			return nil, err
		}
		products = append(products, p)
	}

	span.SetAttributes(attribute.Int("products.count", len(products)))
	log.Printf("Queried %d products from database", len(products))

	return products, rows.Err()
}

func createOrder(ctx context.Context, productID, quantity int) error {
	ctx, span := tracer.Start(ctx, "createOrder")
	defer span.End()

	span.SetAttributes(
		attribute.Int("order.product_id", productID),
		attribute.Int("order.quantity", quantity),
	)

	// Get product price
	var price float64
	err := db.QueryRowContext(ctx, "SELECT price FROM products WHERE id = $1", productID).Scan(&price)
	if err != nil {
		span.RecordError(err)
		return err
	}

	totalPrice := price * float64(quantity)

	// Insert order (automatically traced)
	_, err = db.ExecContext(ctx,
		"INSERT INTO orders (product_id, quantity, total_price, status) VALUES ($1, $2, $3, $4)",
		productID, quantity, totalPrice, "completed",
	)
	if err != nil {
		span.RecordError(err)
		return err
	}

	log.Printf("Created order: product_id=%d, quantity=%d, total=$%.2f", productID, quantity, totalPrice)
	return nil
}

func doWork(ctx context.Context, duration time.Duration) {
	_, span := tracer.Start(ctx, "doWork")
	defer span.End()

	span.SetAttributes(attribute.Int("work.duration_ms", int(duration.Milliseconds())))

	time.Sleep(duration)
	log.Printf("Doing work for %v", duration)
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	// Check database connection
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, "Database unhealthy: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK\n")
}

func main() {
	ctx := context.Background()

	// Initialize tracer
	tp, err := initTracer(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize tracer: %v", err)
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()

	tracer = otel.Tracer("demo-app")

	// Initialize meter
	mp, err := initMeter(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize meter: %v", err)
	}
	defer func() {
		if err := mp.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down meter provider: %v", err)
		}
	}()

	meter = otel.Meter("demo-app")

	// Create metrics
	requestCounter, err = meter.Int64Counter(
		"http_requests_total",
		metric.WithDescription("Total number of HTTP requests"),
	)
	if err != nil {
		log.Fatalf("Failed to create counter: %v", err)
	}

	requestLatency, err = meter.Float64Histogram(
		"http_request_duration_seconds",
		metric.WithDescription("HTTP request latency in seconds"),
	)
	if err != nil {
		log.Fatalf("Failed to create histogram: %v", err)
	}

	// Initialize database with auto-instrumentation
	db, err = initDatabase(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Set up HTTP handlers with automatic instrumentation
	http.Handle("/", otelhttp.NewHandler(http.HandlerFunc(handleRequest), "handleRequest"))
	http.HandleFunc("/health", healthCheck)

	port := getEnv("PORT", "8080")
	log.Printf("Starting demo-app on port %s", port)
	log.Printf("OTEL endpoint: %s", getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "otel-collector.lgtm.svc.cluster.local:4318"))
	log.Printf("Database: %s", getEnv("DB_HOST", "postgres.default.svc.cluster.local"))

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
