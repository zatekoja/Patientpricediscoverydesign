package main

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/database"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/search"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/application/services"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/postgres"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/typesense"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	pgClient, err := postgres.NewClient(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer pgClient.Close()

	tsClient, err := typesense.NewClient(&cfg.Typesense)
	var searchRepo *search.TypesenseAdapter
	if err == nil {
		searchRepo = search.NewTypesenseAdapter(tsClient)
		searchRepo.InitSchema(context.Background())
	}

	facilityRepo := database.NewFacilityAdapter(pgClient)
	facilityService := services.NewFacilityService(facilityRepo, searchRepo)

	ctx := context.Background()

	// 1. Seed Procedures
	procedures := []entities.Procedure{
		{ID: uuid.New().String(), Name: "MRI Scan", Code: "MRI001", Category: "Imaging", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: uuid.New().String(), Name: "Blood Test", Code: "BLD001", Category: "Lab", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: uuid.New().String(), Name: "Physical Therapy", Code: "PHY001", Category: "Rehabilitation", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	procedureRepo := database.NewProcedureAdapter(pgClient)
	for _, p := range procedures {
		if err := procedureRepo.Create(ctx, &p); err != nil {
			log.Printf("Failed to create procedure %s: %v", p.Name, err)
		}
	}

	// 2. Seed Facilities
	facilities := []entities.Facility{
		{
			ID:   uuid.New().String(),
			Name: "City General Hospital",
			Address: entities.Address{
				Street: "100 Main St", City: "San Francisco", State: "CA", ZipCode: "94105", Country: "USA",
			},
			Location:     entities.Location{Latitude: 37.7749, Longitude: -122.4194},
			FacilityType: "Hospital",
			IsActive:     true,
			Rating:       4.5,
			ReviewCount:  120,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			ID:   uuid.New().String(),
			Name: "Sunrise Wellness Center",
			Address: entities.Address{
				Street: "200 Oak Ave", City: "San Francisco", State: "CA", ZipCode: "94107", Country: "USA",
			},
			Location:     entities.Location{Latitude: 37.7833, Longitude: -122.4167},
			FacilityType: "Clinic",
			IsActive:     true,
			Rating:       4.8,
			ReviewCount:  85,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	}

	for _, f := range facilities {
		if err := facilityService.Create(ctx, &f); err != nil {
			log.Printf("Failed to create facility %s: %v", f.Name, err)
		}
	}

	log.Println("Seeding completed successfully")
}
