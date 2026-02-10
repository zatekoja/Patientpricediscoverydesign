package search

import (
	"context"
	"fmt"
	"strings"

	"github.com/typesense/typesense-go/v2/typesense/api"
	"github.com/typesense/typesense-go/v2/typesense/api/pointer"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	tsclient "github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/typesense"
)

const (
	collectionName  = "facilities"
	maxFacilityTags = 100
)

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
		// Collection exists. For dev/prototype, we might need to delete/recreate to update schema
		// or use Update() if supported. For now, assuming the 'reset' flag in reindexer handles this.
		return nil
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
			{Name: "price", Type: "float", Facet: pointer.True(), Optional: pointer.True()},
			{Name: "rating", Type: "float"},
			{Name: "review_count", Type: "int32"},
			{Name: "created_at", Type: "int64"},
			{Name: "insurance", Type: "string[]", Facet: pointer.True(), Optional: pointer.True()},
			{Name: "tags", Type: "string[]", Optional: pointer.True()},
			{Name: "procedures", Type: "string[]", Optional: pointer.True()},
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
	// WARNING: This partial index update might wipe 'procedures' if they were set by the reindexer.
	// To fix this properly, we should fetch existing document or fetch procedures here.
	// For now, minimizing risk by just ensuring schema compliance.
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

	if len(facility.AcceptedInsurance) > 0 {
		document["insurance"] = facility.AcceptedInsurance
	}

	if tags := buildFacilityTags(facility); len(tags) > 0 {
		document["tags"] = tags
	}

	// Use Update instead of Upsert to avoid clearing fields we don't have here (like procedures)?
	// But Update fails if doc doesn't exist. Upsert is safer for "Ensure".
	// Ideally we'd pass procedures to Index.
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
	facilities, _, err := a.SearchWithCount(ctx, params)
	return facilities, err
}

// SearchWithCount searches facilities and returns the total match count.
func (a *TypesenseAdapter) SearchWithCount(ctx context.Context, params repositories.SearchParams) ([]*entities.Facility, int, error) {
	query := "*"
	if params.Query != "" {
		query = params.Query
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}

	filter := "is_active:=true"
	// Only apply location filter if coordinates are provided (non-zero)
	if params.Latitude != 0 || params.Longitude != 0 {
		filter = fmt.Sprintf("%s && location:(%f, %f, %f km)", filter, params.Latitude, params.Longitude, params.RadiusKm)
	}

	if params.InsuranceProvider != "" {
		filter = fmt.Sprintf("%s && insurance:=[%s]", filter, escapeFilterValue(params.InsuranceProvider))
	}
	if params.MinPrice != nil {
		filter = fmt.Sprintf("%s && price:>=%f", filter, *params.MinPrice)
	}
	if params.MaxPrice != nil {
		filter = fmt.Sprintf("%s && price:<=%f", filter, *params.MaxPrice)
	}

	searchParams := &api.SearchCollectionParams{
		Q:        pointer.String(query),
		QueryBy:  pointer.String("name,facility_type,tags,insurance,procedures"),
		FilterBy: pointer.String(filter),
		Page:     pointer.Int(params.Offset/limit + 1),
		PerPage:  pointer.Int(limit),
		NumTypos: pointer.String("2"),
	}
	if params.Query != "" {
		searchParams.MinLen1typo = pointer.Int(4)
		searchParams.MinLen2typo = pointer.Int(7)
	}

	result, err := a.client.Client().Collection(collectionName).Documents().Search(ctx, searchParams)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search facilities: %w", err)
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
		if tagValues, ok := doc["tags"].([]interface{}); ok {
			tags := make([]string, 0, len(tagValues))
			for _, tag := range tagValues {
				if tagStr, ok := tag.(string); ok {
					tags = append(tags, tagStr)
				}
			}
			facility.Tags = tags
		}

		if val, ok := doc["rating"].(float64); ok {
			facility.Rating = val
		}
		if val, ok := doc["review_count"].(float64); ok {
			facility.ReviewCount = int(val)
		}

		facilities = append(facilities, facility)
	}

	totalCount := len(facilities)
	if result.Found != nil {
		totalCount = *result.Found
	}

	return facilities, totalCount, nil
}

// Suggest provides lightweight autocomplete suggestions using Typesense.
func (a *TypesenseAdapter) Suggest(ctx context.Context, query string, lat, lon float64, limit int) ([]*entities.Facility, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return []*entities.Facility{}, nil
	}

	if limit <= 0 {
		limit = 5
	}

	filter := "is_active:=true"
	// Only apply location filter if coordinates are provided (non-zero)
	if lat != 0 || lon != 0 {
		filter = fmt.Sprintf("%s && location:(%f, %f, %f km)", filter, lat, lon, 500.0)
	}

	searchParams := &api.SearchCollectionParams{
		Q:        pointer.String(trimmed),
		QueryBy:  pointer.String("name,facility_type,tags,insurance,procedures"),
		FilterBy: pointer.String(filter),
		Page:     pointer.Int(1),
		PerPage:  pointer.Int(limit),
		NumTypos: pointer.String("2"),
	}
	if trimmed != "" {
		searchParams.MinLen1typo = pointer.Int(4)
		searchParams.MinLen2typo = pointer.Int(7)
	}

	result, err := a.client.Client().Collection(collectionName).Documents().Search(ctx, searchParams)
	if err != nil {
		return nil, fmt.Errorf("failed to suggest facilities: %w", err)
	}

	facilities := []*entities.Facility{}
	for _, hit := range *result.Hits {
		doc := *hit.Document

		locInterface, ok := doc["location"].([]interface{})
		var latVal, lonVal float64
		if ok && len(locInterface) == 2 {
			latVal = locInterface[0].(float64)
			lonVal = locInterface[1].(float64)
		}

		facility := &entities.Facility{
			ID:           doc["id"].(string),
			Name:         doc["name"].(string),
			FacilityType: doc["facility_type"].(string),
			IsActive:     doc["is_active"].(bool),
			Location: entities.Location{
				Latitude:  latVal,
				Longitude: lonVal,
			},
		}
		if tagValues, ok := doc["tags"].([]interface{}); ok {
			tags := make([]string, 0, len(tagValues))
			for _, tag := range tagValues {
				if tagStr, ok := tag.(string); ok {
					tags = append(tags, tagStr)
				}
			}
			facility.Tags = tags
		}

		facilities = append(facilities, facility)
	}

	return facilities, nil
}

func buildFacilityTags(facility *entities.Facility) []string {
	if facility == nil {
		return nil
	}

	builder := newTagBuilder(maxFacilityTags)
	builder.add(facility.Tags...)
	builder.add(
		facility.Name,
		facility.FacilityType,
		facility.Address.City,
		facility.Address.State,
		facility.Address.Country,
	)

	for _, provider := range facility.AcceptedInsurance {
		builder.add(provider)
	}

	return builder.tags()
}

type tagBuilder struct {
	seen  map[string]struct{}
	list  []string
	limit int
}

func newTagBuilder(limit int) *tagBuilder {
	if limit <= 0 {
		limit = maxFacilityTags
	}
	return &tagBuilder{seen: make(map[string]struct{}), limit: limit}
}

func (b *tagBuilder) add(values ...string) {
	for _, value := range values {
		if b.limit > 0 && len(b.list) >= b.limit {
			return
		}
		normalized := normalizeTag(value)
		if normalized == "" {
			continue
		}
		if _, exists := b.seen[normalized]; exists {
			continue
		}
		b.seen[normalized] = struct{}{}
		b.list = append(b.list, normalized)
	}
}

func (b *tagBuilder) tags() []string {
	return b.list
}

func normalizeTag(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func escapeFilterValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "\"\""
	}
	escaped := strings.ReplaceAll(trimmed, "\"", "\\\"")
	return fmt.Sprintf("\"%s\"", escaped)
}
