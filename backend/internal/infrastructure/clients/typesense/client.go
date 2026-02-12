package typesense

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/typesense/typesense-go/v2/typesense"
	"github.com/typesense/typesense-go/v2/typesense/api"
	"github.com/typesense/typesense-go/v2/typesense/api/pointer"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/config"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/retry"
)

const (
	FacilitiesCollection = "facilities"
)

// Client represents a Typesense client
type Client struct {
	client *typesense.Client
}

// NewClient creates a new Typesense client with exponential backoff retry
func NewClient(cfg *config.TypesenseConfig) (*Client, error) {
	client := typesense.NewClient(
		typesense.WithServer(cfg.URL),
		typesense.WithAPIKey(cfg.APIKey),
		typesense.WithConnectionTimeout(5*time.Second),
	)

	// Test connection with retry
	retryConfig := retry.DefaultConfig()
	err := retry.DoWithLog(
		context.Background(),
		retryConfig,
		"Typesense",
		func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, err := client.Health(ctx, 2*time.Second)
			return err
		},
		func(attempt int, err error, nextDelay time.Duration) {
			log.Printf("Typesense connection attempt %d failed: %v. Retrying in %v...", attempt, err, nextDelay)
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to connect to Typesense after retries: %w", err)
	}

	log.Println("Successfully connected to Typesense")
	return &Client{client: client}, nil
}

// Client returns the underlying Typesense client
func (c *Client) Client() *typesense.Client {
	return c.client
}

// InitSchema ensures the facilities collection exists
func (c *Client) InitSchema(ctx context.Context) error {
	collections, err := c.client.Collections().Retrieve(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve collections: %w", err)
	}

	for _, col := range collections {
		if col.Name == FacilitiesCollection {
			log.Println("Typesense collection 'facilities' already exists")
			return nil
		}
	}

	schema := &api.CollectionSchema{
		Name: FacilitiesCollection,
		Fields: []api.Field{
			{
				Name: "id",
				Type: "string",
			},
			{
				Name: "name",
				Type: "string",
			},
			{
				Name:  "facility_type",
				Type:  "string",
				Facet: pointer.True(),
			},
			{
				Name: "location",
				Type: "geopoint",
			},
			{
				Name:     "price",
				Type:     "float",
				Facet:    pointer.True(),
				Optional: pointer.True(),
			},
			{
				Name:  "rating",
				Type:  "float",
				Facet: pointer.True(),
			},
			{
				Name: "review_count",
				Type: "int32",
			},
			{
				Name: "created_at",
				Type: "int64",
			},
			{
				Name: "is_active",
				Type: "bool",
			},
			{
				Name:     "insurance",
				Type:     "string[]",
				Facet:    pointer.True(),
				Optional: pointer.True(),
			},
			{
				Name:     "procedures",
				Type:     "string[]",
				Optional: pointer.True(),
			},
			{
				Name:     "tags",
				Type:     "string[]",
				Optional: pointer.True(),
			},
		},
		DefaultSortingField: pointer.String("created_at"),
	}

	_, err = c.client.Collections().Create(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	log.Println("Created Typesense collection 'facilities'")
	return nil
}

// IndexFacility indexes a facility document
func (c *Client) IndexFacility(ctx context.Context, document map[string]interface{}) error {
	_, err := c.client.Collection(FacilitiesCollection).Documents().Upsert(ctx, document)
	return err
}
