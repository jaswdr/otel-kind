package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type Handler struct {
	db             *sql.DB
	requestCounter metric.Int64Counter
	requestLatency metric.Float64Histogram
}

func NewHandler(db *sql.DB, requestCounter metric.Int64Counter, requestLatency metric.Float64Histogram) *Handler {
	return &Handler{
		db:             db,
		requestCounter: requestCounter,
		requestLatency: requestLatency,
	}
}

func (h *Handler) HandleRequest(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("http.method", r.Method),
		attribute.String("http.url", r.URL.Path),
	)

	log.Printf("Request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

	products, err := QueryProducts(ctx, h.db)
	if err != nil {
		log.Printf("Error querying products: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	doWork(ctx, time.Duration(rand.Intn(500))*time.Millisecond)

	if len(products) > 0 {
		productID := products[rand.Intn(len(products))].ID
		if err := CreateOrder(ctx, h.db, productID, rand.Intn(5)+1); err != nil {
			log.Printf("Error creating order: %v", err)
		}
	}

	h.recordMetrics(ctx, r, start)

	latency := time.Since(start).Seconds()
	log.Printf("Request completed in %.2fms", latency*1000)

	fmt.Fprintf(w, "Hello from demo-app!\n\nFound %d products:\n", len(products))
	for _, p := range products {
		fmt.Fprintf(w, "- %s: $%.2f\n", p.Name, p.Price)
	}
	fmt.Fprintf(w, "\nProcessed in %.2fms\n", latency*1000)
}

func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, "Database unhealthy: %v\n", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK\n")
}

func (h *Handler) recordMetrics(ctx context.Context, r *http.Request, start time.Time) {
	attrs := metric.WithAttributes(
		attribute.String("method", r.Method),
		attribute.String("path", r.URL.Path),
	)

	h.requestCounter.Add(ctx, 1, attrs)
	h.requestLatency.Record(ctx, time.Since(start).Seconds(), attrs)
}

func doWork(ctx context.Context, duration time.Duration) {
	time.Sleep(duration)
}
