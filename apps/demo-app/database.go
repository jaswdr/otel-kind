package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/XSAM/otelsql"
	_ "github.com/jackc/pgx/v5/stdlib"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

func InitDatabase(ctx context.Context, cfg *Config) (*sql.DB, error) {
	driverName, err := otelsql.Register("pgx",
		otelsql.WithAttributes(semconv.DBSystemPostgreSQL),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register driver: %w", err)
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)

	db, err := sql.Open(driverName, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	err = otelsql.RegisterDBStatsMetrics(db, otelsql.WithAttributes(
		semconv.DBSystemPostgreSQL,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to register stats: %w", err)
	}

	return db, nil
}

type Product struct {
	ID          int
	Name        string
	Description string
	Price       float64
}

func QueryProducts(ctx context.Context, db *sql.DB) ([]Product, error) {
	rows, err := db.QueryContext(ctx, "SELECT id, name, description, price FROM products ORDER BY created_at DESC LIMIT 10")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []Product
	for rows.Next() {
		var p Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price); err != nil {
			return nil, err
		}
		products = append(products, p)
	}

	return products, rows.Err()
}

func CreateOrder(ctx context.Context, db *sql.DB, productID, quantity int) error {
	var price float64
	err := db.QueryRowContext(ctx, "SELECT price FROM products WHERE id = $1", productID).Scan(&price)
	if err != nil {
		return err
	}

	totalPrice := price * float64(quantity)
	_, err = db.ExecContext(ctx,
		"INSERT INTO orders (product_id, quantity, total_price, status) VALUES ($1, $2, $3, $4)",
		productID, quantity, totalPrice, "completed",
	)

	return err
}
