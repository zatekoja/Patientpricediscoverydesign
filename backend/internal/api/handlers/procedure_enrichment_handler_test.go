package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/api/handlers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

type MockProcedureEnrichmentService struct {
	mock.Mock
}

func (m *MockProcedureEnrichmentService) GetEnrichment(ctx context.Context, procedureID string, refresh bool) (*entities.ProcedureEnrichment, error) {
	args := m.Called(ctx, procedureID, refresh)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entities.ProcedureEnrichment), args.Error(1)
}

func TestProcedureHandler_GetProcedureEnrichment(t *testing.T) {
	mockService := new(MockProcedureEnrichmentService)
	handler := handlers.NewProcedureHandler(nil, mockService)

	now := time.Date(2026, 2, 7, 12, 0, 0, 0, time.UTC)
	expected := &entities.ProcedureEnrichment{
		ID:          "enrich-1",
		ProcedureID: "proc-1",
		Description: "Short MRI description.",
		PrepSteps:   []string{"Remove metal items", "Arrive early"},
		Risks:       []string{"Noise discomfort", "Claustrophobia"},
		Recovery:    []string{"Resume activities", "Follow clinician guidance"},
		Provider:    "openai",
		Model:       "gpt-4o-mini",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	mockService.On("GetEnrichment", mock.Anything, "proc-1", false).Return(expected, nil)

	req := httptest.NewRequest("GET", "/api/procedures/proc-1/enrichment", nil)
	req.SetPathValue("id", "proc-1")
	w := httptest.NewRecorder()

	handler.GetProcedureEnrichment(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp entities.ProcedureEnrichment
	err := json.NewDecoder(w.Body).Decode(&resp)
	assert.NoError(t, err)
	assert.Equal(t, expected.Description, resp.Description)
	assert.Equal(t, expected.PrepSteps, resp.PrepSteps)
	assert.Equal(t, expected.Risks, resp.Risks)
	assert.Equal(t, expected.Recovery, resp.Recovery)
	assert.Equal(t, expected.Provider, resp.Provider)
	assert.Equal(t, expected.Model, resp.Model)
}
