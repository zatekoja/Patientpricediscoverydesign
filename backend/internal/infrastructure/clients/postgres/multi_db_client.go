package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"
	"time"

	_ "github.com/lib/pq"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/config"
)

// ConnectionType represents the type of database connection
type ConnectionType int

const (
	// Primary connection for writes
	Primary ConnectionType = iota
	// Replica connection for reads
	Replica
)

// MultiDBClient manages multiple database connections (primary + replicas)
type MultiDBClient struct {
	primary      *sql.DB
	readReplicas []*sql.DB
	rrIndex      uint32 // Round-robin index for read replica selection
}

// MultiDBConfig holds configuration for multiple database connections
type MultiDBConfig struct {
	// Primary database configuration (for writes)
	PrimaryDSN string

	// Read replica DSNs
	ReplicaDSNs []string

	// Connection pool settings
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// NewMultiDBClient creates a new multi-database client with primary and read replicas
func NewMultiDBClient(cfg MultiDBConfig) (*MultiDBClient, error) {
	client := &MultiDBClient{
		readReplicas: make([]*sql.DB, 0, len(cfg.ReplicaDSNs)),
	}

	// Connect to primary database
	primaryDB, err := connectDB(cfg.PrimaryDSN, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to primary database: %w", err)
	}
	client.primary = primaryDB

	// Connect to read replicas
	for i, replicaDSN := range cfg.ReplicaDSNs {
		replicaDB, err := connectDB(replicaDSN, cfg)
		if err != nil {
			// Log warning but don't fail - can operate without replicas
			fmt.Printf("Warning: failed to connect to read replica %d: %v\n", i, err)
			continue
		}
		client.readReplicas = append(client.readReplicas, replicaDB)
	}

	if len(client.readReplicas) == 0 {
		fmt.Println("Warning: No read replicas available, all reads will go to primary")
	} else {
		fmt.Printf("Connected to primary and %d read replicas\n", len(client.readReplicas))
	}

	return client, nil
}

// NewMultiDBClientFromConfig creates a multi-DB client from application config
func NewMultiDBClientFromConfig(cfg *config.DatabaseConfig, replicaConfigs []config.DatabaseConfig) (*MultiDBClient, error) {
	// Build DSN for primary
	primaryDSN := cfg.DatabaseDSN()

	// Build DSNs for replicas
	replicaDSNs := make([]string, len(replicaConfigs))
	for i, replicaCfg := range replicaConfigs {
		replicaDSNs[i] = replicaCfg.DatabaseDSN()
	}

	return NewMultiDBClient(MultiDBConfig{
		PrimaryDSN:      primaryDSN,
		ReplicaDSNs:     replicaDSNs,
		MaxOpenConns:    100,
		MaxIdleConns:    25,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
	})
}

// connectDB establishes a database connection with pool settings
func connectDB(dsn string, poolConfig MultiDBConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	// Configure connection pool
	db.SetMaxOpenConns(poolConfig.MaxOpenConns)
	db.SetMaxIdleConns(poolConfig.MaxIdleConns)
	db.SetConnMaxLifetime(poolConfig.ConnMaxLifetime)
	db.SetConnMaxIdleTime(poolConfig.ConnMaxIdleTime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

// Primary returns the primary database connection (for writes)
func (c *MultiDBClient) Primary() *sql.DB {
	return c.primary
}

// Read returns a read replica connection using round-robin, or primary if no replicas
func (c *MultiDBClient) Read() *sql.DB {
	if len(c.readReplicas) == 0 {
		return c.primary
	}

	// Round-robin selection
	idx := atomic.AddUint32(&c.rrIndex, 1)
	return c.readReplicas[idx%uint32(len(c.readReplicas))]
}

// ReadReplicas returns all read replica connections
func (c *MultiDBClient) ReadReplicas() []*sql.DB {
	return c.readReplicas
}

// GetConnection returns the appropriate connection based on operation type
func (c *MultiDBClient) GetConnection(connType ConnectionType) *sql.DB {
	switch connType {
	case Primary:
		return c.Primary()
	case Replica:
		return c.Read()
	default:
		return c.Primary()
	}
}

// Close closes all database connections
func (c *MultiDBClient) Close() error {
	var errs []error

	// Close primary
	if err := c.primary.Close(); err != nil {
		errs = append(errs, fmt.Errorf("failed to close primary: %w", err))
	}

	// Close read replicas
	for i, replica := range c.readReplicas {
		if err := replica.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close replica %d: %w", i, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}

	return nil
}

// HealthCheck checks the health of all database connections
func (c *MultiDBClient) HealthCheck(ctx context.Context) error {
	// Check primary
	if err := c.primary.PingContext(ctx); err != nil {
		return fmt.Errorf("primary database unhealthy: %w", err)
	}

	// Check replicas (non-blocking)
	unhealthyReplicas := 0
	for i, replica := range c.readReplicas {
		if err := replica.PingContext(ctx); err != nil {
			fmt.Printf("Warning: read replica %d unhealthy: %v\n", i, err)
			unhealthyReplicas++
		}
	}

	// If all replicas are down but primary is up, return warning
	if unhealthyReplicas == len(c.readReplicas) && len(c.readReplicas) > 0 {
		return fmt.Errorf("all read replicas unhealthy")
	}

	return nil
}

// ConnectionStats holds connection pool statistics
type ConnectionStats struct {
	Primary  sql.DBStats
	Replicas []sql.DBStats
}

// Stats returns connection pool statistics
func (c *MultiDBClient) Stats() ConnectionStats {
	stats := ConnectionStats{
		Primary:  c.primary.Stats(),
		Replicas: make([]sql.DBStats, len(c.readReplicas)),
	}

	for i, replica := range c.readReplicas {
		stats.Replicas[i] = replica.Stats()
	}

	return stats
}
