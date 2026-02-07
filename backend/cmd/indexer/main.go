package main

import (
	"context"
	"log"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/database"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/postgres"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/typesense"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/config"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Init Postgres
	pgClient, err := postgres.NewClient(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to init postgres: %v", err)
	}
	defer pgClient.Close()

	facilityRepo := database.NewFacilityAdapter(pgClient)

	// Init Typesense
	tsClient := typesense.NewClient(cfg)
	if err := tsClient.InitSchema(context.Background()); err != nil {
		log.Fatalf("Failed to init typesense schema: %v", err)
	}

	// Fetch all facilities
	ctx := context.Background()
	facilities, err := facilityRepo.List(ctx, repositories.FacilityFilter{Limit: 1000})
	if err != nil {
		log.Fatalf("Failed to list facilities: %v", err)
	}

	log.Printf("Indexing %d facilities...", len(facilities))

	for _, f := range facilities {
		// Prepare document for Typesense
		// Note: Typesense location field expects [lat, lon]
		doc := map[string]interface{}{
			"id":            f.ID,
			"name":          f.Name,
			"facility_type": f.FacilityType,
			"location":      []float64{f.Location.Latitude, f.Location.Longitude},
			"rating":        f.Rating,
			"is_active":     f.IsActive,
			// "insurance": f.AcceptedInsurance, // Add when available
		}

		if err := tsClient.IndexFacility(ctx, doc); err != nil {
			log.Printf("Failed to index facility %s: %v", f.ID, err)
		} else {
			log.Printf("Indexed %s", f.Name)
		}
	}

	log.Println("Indexing complete.")
}
