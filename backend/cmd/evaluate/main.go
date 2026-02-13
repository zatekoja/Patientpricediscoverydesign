package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/database"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/search"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/application/services"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/evaluation"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/postgres"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/typesense"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/config"
)

// serviceWrapper adapts FacilityService to evaluation.SearchResultProvider
type serviceWrapper struct {
	svc *services.FacilityService
}

func (w *serviceWrapper) Search(ctx context.Context, params repositories.SearchParams) ([]entities.FacilitySearchResult, int, error) {
	results, count, _, err := w.svc.SearchResultsWithCount(ctx, params)
	return results, count, err
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	pgClient, err := postgres.NewClient(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pgClient.Close()

	tsClient, err := typesense.NewClient(&cfg.Typesense)
	if err != nil {
		log.Fatalf("Failed to connect to Typesense: %v", err)
	}

	// Setup Repos
	facilityRepo := database.NewFacilityAdapter(pgClient)
	searchRepo := search.NewTypesenseAdapter(tsClient)
	facilityProcedureRepo := database.NewFacilityProcedureAdapter(pgClient)
	procedureCatalogRepo := database.NewProcedureAdapter(pgClient)
	insuranceRepo := database.NewInsuranceAdapter(pgClient)

	// Setup Services
	facilityService := services.NewFacilityService(
		facilityRepo,
		searchRepo,
		facilityProcedureRepo,
		procedureCatalogRepo,
		insuranceRepo,
	)

	// Initialize Query Understanding and Search Ranking
	conceptDictPath := "config/concept_dictionary.json"
	spellingPath := "config/spelling_corrections.json"
	if _, err := os.Stat("backend/" + conceptDictPath); err == nil {
		conceptDictPath = "backend/" + conceptDictPath
		spellingPath = "backend/" + spellingPath
	}

	quService, err := services.NewQueryUnderstandingService(conceptDictPath, spellingPath)
	if err == nil {
		facilityService.SetQueryUnderstanding(quService)
	}

	rankingService := services.NewSearchRankingService()
	facilityService.SetSearchRanking(rankingService)

	// Load Golden Queries
	goldenPath := "config/golden_queries.json"
	if _, err := os.Stat("backend/" + goldenPath); err == nil {
		goldenPath = "backend/" + goldenPath
	}

	queries, err := evaluation.LoadGoldenQueries(goldenPath)
	if err != nil {
		log.Fatalf("Failed to load golden queries: %v", err)
	}

	runner := evaluation.NewRunner(&serviceWrapper{svc: facilityService})
	summary, err := runner.Run(context.Background(), queries)
	if err != nil {
		log.Fatalf("Evaluation failed: %v", err)
	}

	// Output results as JSON
	out, _ := json.MarshalIndent(summary, "", "  ")
	fmt.Println(string(out))
}
