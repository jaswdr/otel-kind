package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

type Handler struct {
	db *sql.DB
}

func NewHandler(db *sql.DB) *Handler {
	return &Handler{
		db: db,
	}
}

func (h *Handler) HandleRequest(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ctx := r.Context()

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

func doWork(ctx context.Context, duration time.Duration) {
	time.Sleep(duration)
}
