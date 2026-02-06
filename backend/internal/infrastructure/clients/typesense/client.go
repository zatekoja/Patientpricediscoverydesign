package typesense

import (
	"context"
	"fmt"
	"time"

	"github.com/typesense/typesense-go/v2/typesense"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/config"
)

// Client represents a Typesense client
type Client struct {
	client *typesense.Client
}

// NewClient creates a new Typesense client
func NewClient(cfg *config.TypesenseConfig) (*Client, error) {
	client := typesense.NewClient(
		typesense.WithServer(cfg.URL),
		typesense.WithAPIKey(cfg.APIKey),
		typesense.WithConnectionTimeout(5*time.Second),
	)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use Health endpoint or Retrieve Collection to test connection
	if _, err := client.Health(ctx, 2*time.Second); err != nil {
		return nil, fmt.Errorf("failed to connect to Typesense: %w", err)
	}

	return &Client{client: client}, nil
}

// Client returns the underlying Typesense client
func (c *Client) Client() *typesense.Client {
	return c.client
}
