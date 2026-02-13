package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/adapters/database"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/application/services"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/openai"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/postgres"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/config"
)

func main() {
	var workers int
	var maxRetries int
	var procedureID string

	flag.IntVar(&workers, "workers", 3, "Number of concurrent workers")
	flag.IntVar(&maxRetries, "max-retries", 3, "Max retries per procedure")
	flag.StringVar(&procedureID, "procedure", "", "Single procedure ID to backfill")
	flag.Parse()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Setup DB
	pgClient, err := postgres.NewClient(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pgClient.Close()

	// Setup repos
	procRepo := database.NewProcedureAdapter(pgClient)
	enrichRepo := database.NewProcedureEnrichmentAdapter(pgClient)

	// Setup provider
	provider, err := openai.NewClient(&cfg.OpenAI)
	if err != nil {
		log.Fatalf("Failed to create OpenAI client: %v", err)
	}

	// Setup service
	svc := services.NewConceptBackfillService(procRepo, enrichRepo, provider, workers, maxRetries)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	start := time.Now()

	if procedureID != "" {
		log.Printf("Backfilling single procedure: %s", procedureID)
		if err := svc.BackfillSingle(ctx, procedureID); err != nil {
			log.Fatalf("Failed to backfill procedure %s: %v", procedureID, err)
		}
		log.Printf("Successfully backfilled %s", procedureID)
	} else {
		log.Printf("Starting backfill with %d workers...", workers)
		summary, err := svc.BackfillAll(ctx)
		if err != nil {
			log.Printf("Backfill failed: %v", err)
		}

		if summary != nil {
			log.Printf("Backfill complete in %s", time.Since(start))
			log.Printf("Total processed: %d", summary.TotalProcessed)
			log.Printf("Success: %d", summary.SuccessCount)
			log.Printf("Failed: %d", summary.FailureCount)
		}
	}
}
