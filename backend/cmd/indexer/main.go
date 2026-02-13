package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/database"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/search"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/postgres"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/typesense"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/config"
)

func main() {
	var reset bool
	var intervalFlag string
	flag.BoolVar(&reset, "reset", false, "delete existing Typesense collection before reindexing")
	flag.StringVar(&intervalFlag, "interval", "", "repeat interval for reindexing (e.g. 6h, 30m)")
	flag.Parse()

	intervalValue := strings.TrimSpace(intervalFlag)
	if intervalValue == "" {
		intervalValue = strings.TrimSpace(os.Getenv("REINDEX_INTERVAL"))
	}

	var interval time.Duration
	var err error
	if intervalValue != "" {
		interval, err = time.ParseDuration(intervalValue)
		if err != nil {
			log.Fatalf("Invalid interval %q: %v", intervalValue, err)
		}
		if interval <= 0 {
			log.Fatalf("Interval must be greater than zero")
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	for {
		if err := indexOnce(ctx, reset); err != nil {
			log.Printf("Reindex failed: %v", err)
		}

		if interval <= 0 {
			break
		}

		reset = false
		log.Printf("Reindex complete. Next run in %s.", interval)

		select {
		case <-ctx.Done():
			log.Println("Reindexer shutting down")
			return
		case <-time.After(interval):
		}
	}
}

func indexOnce(ctx context.Context, reset bool) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	pgClient, err := postgres.NewClient(&cfg.Database)
	if err != nil {
		return err
	}
	defer pgClient.Close()

	facilityRepo := database.NewFacilityAdapter(pgClient)
	facilityProcedureRepo := database.NewFacilityProcedureAdapter(pgClient)
	procedureCatalogRepo := database.NewProcedureAdapter(pgClient)
	insuranceRepo := database.NewInsuranceAdapter(pgClient)
	enrichmentRepo := database.NewProcedureEnrichmentAdapter(pgClient)

	tsClient, err := typesense.NewClient(&cfg.Typesense)
	if err != nil {
		return err
	}

	if reset || os.Getenv("RESET_TYPESENSE") == "true" {
		log.Println("RESET_TYPESENSE=true detected, deleting facilities collection")
		_, err := tsClient.Client().Collection(typesense.FacilitiesCollection).Delete(ctx)
		if err != nil {
			log.Printf("Warning: failed to delete collection: %v", err)
		}
	}

	if err := tsClient.InitSchema(ctx); err != nil {
		return err
	}

	facilities, err := facilityRepo.List(ctx, repositories.FacilityFilter{Limit: 1000})
	if err != nil {
		return err
	}

	procedureByID := map[string]*entities.Procedure{}
	procedures, err := procedureCatalogRepo.List(ctx, repositories.ProcedureFilter{})
	if err != nil {
		log.Printf("Warning: failed to list procedures: %v", err)
	} else {
		for _, procedure := range procedures {
			if procedure == nil {
				continue
			}
			procedureByID[procedure.ID] = procedure
		}
	}

	log.Printf("Indexing %d facilities...", len(facilities))

	for _, f := range facilities {
		if f == nil {
			continue
		}

		tagsBuilder := newTagBuilder(maxFacilityTags)
		tagsBuilder.add(
			f.Name,
			f.FacilityType,
			f.Address.City,
			f.Address.State,
			f.Address.Country,
		)

		var minPrice *float64
		facilityProcedures, err := facilityProcedureRepo.ListByFacility(ctx, f.ID)
		if err != nil {
			log.Printf("Warning: failed to load procedures for %s: %v", f.ID, err)
		} else {
			for _, fp := range facilityProcedures {
				if fp == nil {
					continue
				}
				if fp.Price > 0 {
					if minPrice == nil || fp.Price < *minPrice {
						price := fp.Price
						minPrice = &price
					}
				}

				if procedure, ok := procedureByID[fp.ProcedureID]; ok && procedure != nil {
					tagsBuilder.add(procedure.Name, procedure.Code, procedure.Category)
				}
			}
		}

		insuranceProviders, err := insuranceRepo.GetFacilityInsurance(ctx, f.ID)
		if err != nil {
			log.Printf("Warning: failed to load insurance for %s: %v", f.ID, err)
		}

		insuranceNames := []string{}
		for _, provider := range insuranceProviders {
			if provider == nil {
				continue
			}
			insuranceNames = append(insuranceNames, provider.Name)
			tagsBuilder.add(provider.Name, provider.Code)
		}

		// Collect procedure names for dedicated search field
		procedureNames := []string{}
		// We iterate facilityProcedures again or can do it in previous loop if refactored
		// Since we already iterated above to find minPrice, let's refactor or just iterate again or collect in first loop
		// Let's refactor the loop above to collect names.
		// Actually, let's keep it simple and iterate again or use a map
		uniqueProcNames := map[string]struct{}{}
		for _, fp := range facilityProcedures {
			if fp == nil {
				continue
			}
			if procedure, ok := procedureByID[fp.ProcedureID]; ok && procedure != nil {
				uniqueProcNames[procedure.Name] = struct{}{}
			}
		}
		for name := range uniqueProcNames {
			procedureNames = append(procedureNames, name)
		}

		uniqueProcIDs := make(map[string]struct{})
		var enrichments []*entities.ProcedureEnrichment
		for _, fp := range facilityProcedures {
			if fp == nil {
				continue
			}
			if _, seen := uniqueProcIDs[fp.ProcedureID]; seen {
				continue
			}
			uniqueProcIDs[fp.ProcedureID] = struct{}{}

			enrich, err := enrichmentRepo.GetByProcedureID(ctx, fp.ProcedureID)
			if err == nil && enrich != nil {
				enrichments = append(enrichments, enrich)
			}
		}

		conceptFields := search.BuildConceptFields(enrichments)

		// Manual expansion for common specialties to bridge lay terms and medical jargon
		for _, s := range conceptFields.Specialties {
			if s == "pediatrics" || s == "paediatrics" {
				conceptFields.Concepts = append(conceptFields.Concepts, "baby", "child", "infant", "newborn", "paediatric")
			}
			if s == "obstetrics" || s == "gynecology" || s == "obstetrics_gynaecology" {
				conceptFields.Concepts = append(conceptFields.Concepts, "pregnancy", "maternity", "antenatal", "prenatal", "baby delivery")
			}
			if s == "dental" || s == "dentistry" {
				conceptFields.Concepts = append(conceptFields.Concepts, "tooth", "teeth", "dentist", "mouth", "gum")
			}
		}

		doc := map[string]interface{}{
			"id":            f.ID,
			"name":          f.Name,
			"facility_type": f.FacilityType,
			"location":      []float64{f.Location.Latitude, f.Location.Longitude},
			"rating":        f.Rating,
			"review_count":  f.ReviewCount,
			"is_active":     f.IsActive,
			"created_at":    f.CreatedAt.Unix(),
		}

		if minPrice != nil {
			doc["price"] = *minPrice
		}

		if len(insuranceNames) > 0 {
			doc["insurance"] = insuranceNames
		}

		if len(procedureNames) > 0 {
			doc["procedures"] = procedureNames
		}

		if len(conceptFields.Concepts) > 0 {
			doc["concepts"] = conceptFields.Concepts
		}
		if len(conceptFields.Conditions) > 0 {
			doc["conditions"] = conceptFields.Conditions
		}
		if len(conceptFields.Symptoms) > 0 {
			doc["symptoms"] = conceptFields.Symptoms
		}
		if len(conceptFields.Specialties) > 0 {
			doc["specialties"] = conceptFields.Specialties
		}

		if tags := tagsBuilder.tags(); len(tags) > 0 {
			doc["tags"] = tags
		}

		if err := tsClient.IndexFacility(ctx, doc); err != nil {
			log.Printf("Failed to index facility %s: %v", f.ID, err)
		} else {
			log.Printf("Indexed %s", f.Name)
		}
	}

	log.Println("Indexing complete.")
	return nil
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

const maxFacilityTags = 100
