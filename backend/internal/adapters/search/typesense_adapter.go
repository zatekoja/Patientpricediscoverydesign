package search

import (
	"context"
	"fmt"

	"github.com/typesense/typesense-go/v2/typesense/api"
	"github.com/typesense/typesense-go/v2/typesense/api/pointer"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	tsclient "github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/typesense"
)

const collectionName = "facilities"

// TypesenseAdapter implements facility search using Typesense

type TypesenseAdapter struct {

       client *tsclient.Client

}



// Ensure TypesenseAdapter implements FacilitySearchRepository

var _ repositories.FacilitySearchRepository = (*TypesenseAdapter)(nil)



// NewTypesenseAdapter creates a new Typesense adapter


func NewTypesenseAdapter(client *tsclient.Client) *TypesenseAdapter {
	return &TypesenseAdapter{client: client}
}

// InitSchema ensures the collection exists
func (a *TypesenseAdapter) InitSchema(ctx context.Context) error {
	// Check if collection exists
	_, err := a.client.Client().Collection(collectionName).Retrieve(ctx)
	if err == nil {
		return nil // Collection exists
	}

	// Create collection
	schema := &api.CollectionSchema{
		Name: collectionName,
		Fields: []api.Field{
			{Name: "id", Type: "string"},
			{Name: "name", Type: "string"},
			{Name: "facility_type", Type: "string", Facet: pointer.True()},
			{Name: "is_active", Type: "bool"},
			{Name: "location", Type: "geopoint"},
			{Name: "rating", Type: "float"},
			{Name: "review_count", Type: "int32"},
			{Name: "created_at", Type: "int64"},
		},
		DefaultSortingField: pointer.String("created_at"),
	}

	_, err = a.client.Client().Collections().Create(ctx, schema)
	if err != nil {
		return fmt.Errorf("failed to create typesense collection: %w", err)
	}

	return nil
}

// Index indexes a facility
func (a *TypesenseAdapter) Index(ctx context.Context, facility *entities.Facility) error {
	document := map[string]interface{}{
		"id":            facility.ID,
		"name":          facility.Name,
		"facility_type": facility.FacilityType,
		"is_active":     facility.IsActive,
		"location":      []float64{facility.Location.Latitude, facility.Location.Longitude},
		"rating":        facility.Rating,
		"review_count":  facility.ReviewCount,
		"created_at":    facility.CreatedAt.Unix(),
	}

	_, err := a.client.Client().Collection(collectionName).Documents().Upsert(ctx, document)
	if err != nil {
		return fmt.Errorf("failed to index facility: %w", err)
	}

	return nil
}

// Delete removes a facility from index
func (a *TypesenseAdapter) Delete(ctx context.Context, id string) error {
	_, err := a.client.Client().Collection(collectionName).Document(id).Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete facility from index: %w", err)
	}
	return nil
}

// Search searches facilities
func (a *TypesenseAdapter) Search(ctx context.Context, params repositories.SearchParams) ([]*entities.Facility, error) {
	searchParams := &api.SearchCollectionParams{
		Q:        pointer.String("*"),
		QueryBy:  pointer.String("name"),
		FilterBy: pointer.String(fmt.Sprintf("is_active:=true && location:(%f, %f, %f km)", params.Latitude, params.Longitude, params.RadiusKm)),
		Page:     pointer.Int(params.Offset/params.Limit + 1),
		PerPage:  pointer.Int(params.Limit),
	}

	result, err := a.client.Client().Collection(collectionName).Documents().Search(ctx, searchParams)
	if err != nil {
		return nil, fmt.Errorf("failed to search facilities: %w", err)
	}

	facilities := []*entities.Facility{}
	for _, hit := range *result.Hits {
		doc := *hit.Document
		
		// Parse location
		locInterface, ok := doc["location"].([]interface{})
		var lat, lon float64
		if ok && len(locInterface) == 2 {
			lat = locInterface[0].(float64)
			lon = locInterface[1].(float64)
		}

		// Reconstruct entity (partial)
		// Note: Typesense returns map[string]interface{}, so we need to cast safely
		// For a real app, we might want to fetch full details from DB using IDs if Typesense doesn't have everything
		facility := &entities.Facility{
			ID:           doc["id"].(string),
			Name:         doc["name"].(string),
			FacilityType: doc["facility_type"].(string),
			IsActive:     doc["is_active"].(bool),
			Location: entities.Location{
				Latitude:  lat,
				Longitude: lon,
			},
		}
		
		if val, ok := doc["rating"].(float64); ok {
			facility.Rating = val
		}
		if val, ok := doc["review_count"].(float64); ok {
			facility.ReviewCount = int(val)
		}

		facilities = append(facilities, facility)
	}

	return facilities, nil
}
