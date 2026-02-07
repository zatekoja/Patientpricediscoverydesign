package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/api/handlers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

type stubFeedbackService struct {
	created []*entities.Feedback
}

func (s *stubFeedbackService) Create(ctx context.Context, feedback *entities.Feedback) error {
	if feedback.ID == "" {
		feedback.ID = "test-id"
	}
	s.created = append(s.created, feedback)
	return nil
}

func TestFeedbackHandler_SubmitFeedback_Success(t *testing.T) {
	service := &stubFeedbackService{}
	handler := handlers.NewFeedbackHandler(service, nil)

	body := `{"rating":5,"message":"Great flow","email":"test@example.com","page":"/"}`
	req := httptest.NewRequest("POST", "/api/feedback", strings.NewReader(body))
	req.RemoteAddr = "10.0.0.1:1234"
	w := httptest.NewRecorder()

	handler.SubmitFeedback(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Len(t, service.created, 1)

	var response map[string]string
	err := json.NewDecoder(w.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "received", response["status"])
	assert.NotEmpty(t, response["id"])
}

func TestFeedbackHandler_SubmitFeedback_RateLimit(t *testing.T) {
	service := &stubFeedbackService{}
	handler := handlers.NewFeedbackHandler(service, nil)

	for i := 0; i < 5; i++ {
		body := `{"rating":4,"message":"ok-` + strconv.Itoa(i) + `"}`
		req := httptest.NewRequest("POST", "/api/feedback", strings.NewReader(body))
		req.RemoteAddr = "10.0.0.2:1234"
		w := httptest.NewRecorder()
		handler.SubmitFeedback(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)
	}

	body := `{"rating":4,"message":"ok-dup"}`
	req := httptest.NewRequest("POST", "/api/feedback", strings.NewReader(body))
	req.RemoteAddr = "10.0.0.2:1234"
	w := httptest.NewRecorder()
	handler.SubmitFeedback(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestFeedbackHandler_SubmitFeedback_Duplicate(t *testing.T) {
	service := &stubFeedbackService{}
	handler := handlers.NewFeedbackHandler(service, nil)

	body := `{"rating":5,"message":"Great flow","email":"test@example.com","page":"/"}`
	req := httptest.NewRequest("POST", "/api/feedback", strings.NewReader(body))
	req.RemoteAddr = "10.0.0.9:1234"
	w := httptest.NewRecorder()

	handler.SubmitFeedback(w, req)
	assert.Equal(t, http.StatusCreated, w.Code)

	req2 := httptest.NewRequest("POST", "/api/feedback", strings.NewReader(body))
	req2.RemoteAddr = "10.0.0.9:1234"
	w2 := httptest.NewRecorder()

	handler.SubmitFeedback(w2, req2)
	assert.Equal(t, http.StatusAccepted, w2.Code)
	assert.Len(t, service.created, 1)
}
