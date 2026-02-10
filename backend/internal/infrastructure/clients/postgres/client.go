package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/config"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/retry"
)

// Client represents a PostgreSQL database client
type Client struct {
	db *sql.DB
}

// NewClient creates a new PostgreSQL client with exponential backoff retry
func NewClient(cfg *config.DatabaseConfig) (*Client, error) {
	db, err := sql.Open("postgres", cfg.DatabaseDSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Test the connection with retry
	retryConfig := retry.DefaultConfig()
	err = retry.DoWithLog(
		context.Background(),
		retryConfig,
		"PostgreSQL",
		func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			return db.PingContext(ctx)
		},
		func(attempt int, err error, nextDelay time.Duration) {
			log.Printf("PostgreSQL connection attempt %d failed: %v. Retrying in %v...", attempt, err, nextDelay)
		},
	)

	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to PostgreSQL after retries: %w", err)
	}

	log.Println("Successfully connected to PostgreSQL")
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
