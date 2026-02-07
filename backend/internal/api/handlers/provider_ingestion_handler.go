package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	redislib "github.com/redis/go-redis/v9"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/application/services"
)

// ProviderIngestionHandler triggers provider -> core data sync.
type ProviderIngestionHandler struct {
	service         *services.ProviderIngestionService
	redisClient     *redislib.Client
	idempotencyTTL  time.Duration
}

func NewProviderIngestionHandler(
	service *services.ProviderIngestionService,
	redisClient *redislib.Client,
	idempotencyTTL time.Duration,
) *ProviderIngestionHandler {
	if idempotencyTTL <= 0 {
		idempotencyTTL = 24 * time.Hour
	}
	return &ProviderIngestionHandler{
		service:        service,
		redisClient:    redisClient,
		idempotencyTTL: idempotencyTTL,
	}
}

func (h *ProviderIngestionHandler) TriggerIngestion(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		respondWithError(w, http.StatusServiceUnavailable, "provider ingestion service not configured")
		return
	}

	if duplicate, key := h.isDuplicate(r.Context(), r); duplicate {
		respondWithJSON(w, http.StatusOK, map[string]string{
			"status":            "duplicate",
			"idempotency_key":   key,
		})
		return
	}

	providerID := strings.TrimSpace(r.URL.Query().Get("providerId"))
	summary, err := h.service.SyncCurrentData(r.Context(), providerID)
	if err != nil {
		respondWithError(w, http.StatusBadGateway, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, summary)
}

type idempotencyPayload struct {
	EventID string `json:"eventId"`
}

func (h *ProviderIngestionHandler) isDuplicate(ctx context.Context, r *http.Request) (bool, string) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		key = strings.TrimSpace(r.Header.Get("X-Idempotency-Key"))
	}
	if key == "" {
		if eventID, err := extractEventID(r); err == nil {
			key = eventID
		}
	}
	if key == "" || h.redisClient == nil {
		return false, ""
	}

	redisKey := "provider_ingest_idem:" + key
	ok, err := h.redisClient.SetNX(ctx, redisKey, time.Now().UTC().Format(time.RFC3339Nano), h.idempotencyTTL).Result()
	if err != nil {
		log.Printf("idempotency check failed: %v", err)
		return false, key
	}
	if !ok {
		return true, key
	}
	return false, key
}

func extractEventID(r *http.Request) (string, error) {
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") || r.Body == nil {
		return "", nil
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}
	r.Body.Close()
	r.Body = io.NopCloser(bytes.NewBuffer(body))
	if len(bytes.TrimSpace(body)) == 0 {
		return "", nil
	}
	var payload idempotencyPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	return strings.TrimSpace(payload.EventID), nil
}
