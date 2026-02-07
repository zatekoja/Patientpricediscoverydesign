package main

import (
	"context"
	"log"
	"os"
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
	procedureRepo := database.NewProcedureAdapter(pgClient)
	insuranceRepo := database.NewInsuranceAdapter(pgClient)
	facilityProcedureRepo := database.NewFacilityProcedureAdapter(pgClient)
	facilityService := services.NewFacilityService(facilityRepo, searchRepo, facilityProcedureRepo, procedureRepo, insuranceRepo)

	ctx := context.Background()

	if os.Getenv("RESET_DB") == "true" {
		log.Println("RESET_DB=true detected, truncating tables before seeding")
		_, err := pgClient.DB().ExecContext(ctx, `
			TRUNCATE TABLE
				facility_insurance,
				facility_procedures,
				availability_slots,
				appointments,
				reviews,
				facilities,
				procedures,
				insurance_providers,
				users
			RESTART IDENTITY CASCADE
		`)
		if err != nil {
			log.Fatalf("Failed to reset tables: %v", err)
		}
	}

	// 1. Seed Procedures
	procedures := []entities.Procedure{
		{ID: uuid.New().String(), Name: "MRI Scan (Brain)", Code: "MRI-BRAIN", Category: "Imaging", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: uuid.New().String(), Name: "CT Scan (Chest)", Code: "CT-CHEST", Category: "Imaging", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: uuid.New().String(), Name: "Full Blood Count", Code: "FBC-001", Category: "Lab", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: uuid.New().String(), Name: "Dental Cleaning", Code: "DENT-001", Category: "Dental", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: uuid.New().String(), Name: "Physical Therapy Session", Code: "PT-001", Category: "Rehabilitation", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	for _, p := range procedures {
		if err := procedureRepo.Create(ctx, &p); err != nil {
			log.Printf("Failed to create procedure %s: %v", p.Name, err)
		}
	}

	// 2. Seed Insurance Providers
	insuranceProviders := []entities.InsuranceProvider{
		{ID: uuid.New().String(), Name: "Reliance HMO", Code: "RELIANCE", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: uuid.New().String(), Name: "Hygeia HMO", Code: "HYGEIA", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: uuid.New().String(), Name: "AXA Mansard Health", Code: "AXAMANSARD", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: uuid.New().String(), Name: "NHIS", Code: "NHIS", IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	for _, i := range insuranceProviders {
		if err := insuranceRepo.Create(ctx, &i); err != nil {
			log.Printf("Failed to create insurance provider %s: %v", i.Name, err)
		}
	}

	// 3. Seed Facilities (General Hospitals in Nigeria)
	facilities := []entities.Facility{
		{
			ID:   uuid.New().String(),
			Name: "General Hospital Lagos",
			Address: entities.Address{
				Street: "1-3 Broad Street, Odan", City: "Lagos Island", State: "Lagos", ZipCode: "101221", Country: "Nigeria",
			},
			Location:     entities.Location{Latitude: 6.4531, Longitude: 3.3958},
			FacilityType: "General Hospital",
			IsActive:     true,
			Rating:       4.2,
			ReviewCount:  450,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			ID:   uuid.New().String(),
			Name: "Lagos State University Teaching Hospital (LASUTH)",
			Address: entities.Address{
				Street: "1-5 Oba Akinjobi Way", City: "Ikeja", State: "Lagos", ZipCode: "100271", Country: "Nigeria",
			},
			Location:     entities.Location{Latitude: 6.5967, Longitude: 3.3421},
			FacilityType: "Teaching Hospital",
			IsActive:     true,
			Rating:       4.5,
			ReviewCount:  1200,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			ID:   uuid.New().String(),
			Name: "Garki Hospital",
			Address: entities.Address{
				Street: "Area 8, Tafawa Balewa Way", City: "Garki", State: "Abuja", ZipCode: "900241", Country: "Nigeria",
			},
			Location:     entities.Location{Latitude: 9.0433, Longitude: 7.4833},
			FacilityType: "General Hospital",
			IsActive:     true,
			Rating:       4.4,
			ReviewCount:  320,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			ID:   uuid.New().String(),
			Name: "National Hospital Abuja",
			Address: entities.Address{
				Street: "Plot 272, Samuel Ademulegun St", City: "Central Business District", State: "Abuja", ZipCode: "900211", Country: "Nigeria",
			},
			Location:     entities.Location{Latitude: 9.0333, Longitude: 7.4667},
			FacilityType: "Specialist Hospital",
			IsActive:     true,
			Rating:       4.6,
			ReviewCount:  850,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			ID:   uuid.New().String(),
			Name: "General Hospital Ikorodu",
			Address: entities.Address{
				Street: "T.O.S. Benson Road", City: "Ikorodu", State: "Lagos", ZipCode: "104101", Country: "Nigeria",
			},
			Location:     entities.Location{Latitude: 6.5965, Longitude: 3.5075},
			FacilityType: "General Hospital",
			IsActive:     true,
			Rating:       4.0,
			ReviewCount:  280,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	}

	capacityStatuses := []string{"Available", "Limited", "Busy"}
	avgWaits := []int{15, 30, 45}
	urgentCareFlags := []bool{true, false, false}

	fpRepo := database.NewFacilityProcedureAdapter(pgClient)
	db := pgClient.DB()

	for i := range facilities {
		facilities[i].CapacityStatus = &capacityStatuses[i%len(capacityStatuses)]
		facilities[i].AvgWaitMinutes = &avgWaits[i%len(avgWaits)]
		facilities[i].UrgentCareAvailable = &urgentCareFlags[i%len(urgentCareFlags)]

		f := facilities[i]
		if err := facilityService.Create(ctx, &f); err != nil {
			log.Printf("Failed to create facility %s: %v", f.Name, err)
			continue
		}

		// Add some procedures to each facility with random pricing
		for i, p := range procedures {
			// Each facility only offers 3 random procedures
			if i%2 == 0 {
				price := float64(20000 + (i * 7500))
				fp := &entities.FacilityProcedure{
					ID:                uuid.New().String(),
					FacilityID:        f.ID,
					ProcedureID:       p.ID,
					Price:             price,
					Currency:          "NGN",
					EstimatedDuration: 45,
					IsAvailable:       true,
					CreatedAt:         time.Now(),
					UpdatedAt:         time.Now(),
				}
				if err := fpRepo.Create(ctx, fp); err != nil {
					log.Printf("Failed to link procedure %s to facility %s: %v", p.Name, f.Name, err)
				}
			}
		}

		// Add accepted insurance providers (every other provider for variety)
		for i, provider := range insuranceProviders {
			if i%2 != 0 {
				continue
			}
			_, err := db.ExecContext(
				ctx,
				`INSERT INTO facility_insurance (id, facility_id, insurance_provider_id, is_accepted, created_at, updated_at)
				 VALUES ($1, $2, $3, true, NOW(), NOW())
				 ON CONFLICT (facility_id, insurance_provider_id) DO NOTHING`,
				uuid.New().String(),
				f.ID,
				provider.ID,
			)
			if err != nil {
				log.Printf("Failed to link insurance provider %s to facility %s: %v", provider.Name, f.Name, err)
			}
		}
	}

	log.Println("Seeding completed successfully with Nigerian General Hospitals")
}
