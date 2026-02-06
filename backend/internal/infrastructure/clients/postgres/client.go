package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/config"
)

// Client represents a PostgreSQL database client
type Client struct {
	db *sql.DB
}

// NewClient creates a new PostgreSQL client
func NewClient(cfg *config.DatabaseConfig) (*Client, error) {
	db, err := sql.Open("postgres", cfg.DatabaseDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close() // Close the database connection on ping failure
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Client{db: db}, nil
}

// DB returns the underlying database connection
func (c *Client) DB() *sql.DB {
	return c.db
}

// Close closes the database connection
func (c *Client) Close() error {
	return c.db.Close()
}

// BeginTx starts a new transaction
func (c *Client) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return c.db.BeginTx(ctx, nil)
}

// Ping verifies the connection to the database
func (c *Client) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}
