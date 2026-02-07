package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
)

const (
	feedbackRateLimit   = 5
	feedbackRateWindow  = time.Hour
	feedbackDedupWindow = 24 * time.Hour
)

// FeedbackService defines the feedback operations used by the handler.
type FeedbackService interface {
	Create(ctx context.Context, feedback *entities.Feedback) error
}

// FeedbackHandler handles feedback submissions.
type FeedbackHandler struct {
	service FeedbackService
	cache   providers.CacheProvider
	local   *localRateLimiter
	deduper *localDeduper
}

// NewFeedbackHandler creates a new feedback handler.
func NewFeedbackHandler(service FeedbackService, cache providers.CacheProvider) *FeedbackHandler {
	return &FeedbackHandler{
		service: service,
		cache:   cache,
		local:   newLocalRateLimiter(),
		deduper: newLocalDeduper(),
	}
}

type feedbackRequest struct {
	Rating  int    `json:"rating"`
	Message string `json:"message"`
	Email   string `json:"email"`
	Page    string `json:"page"`
}

// SubmitFeedback handles POST /api/feedback
func (h *FeedbackHandler) SubmitFeedback(w http.ResponseWriter, r *http.Request) {
	var payload feedbackRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "invalid request payload")
		return
	}

	if payload.Rating < 1 || payload.Rating > 5 {
		respondWithError(w, http.StatusBadRequest, "rating must be between 1 and 5")
		return
	}

	payload.Message = strings.TrimSpace(payload.Message)
	payload.Email = strings.TrimSpace(payload.Email)
	payload.Page = strings.TrimSpace(payload.Page)

	if len(payload.Message) > 1000 {
		respondWithError(w, http.StatusBadRequest, "message is too long")
		return
	}
	if len(payload.Email) > 200 {
		respondWithError(w, http.StatusBadRequest, "email is too long")
		return
	}
	if len(payload.Page) > 300 {
		respondWithError(w, http.StatusBadRequest, "page is too long")
		return
	}

	key := "feedback:rate:" + clientIP(r)
	allowed, retryAfter := h.allowRequest(r.Context(), key)
	if !allowed {
		w.Header().Set("Retry-After", strconv.Itoa(int(retryAfter.Seconds())))
		respondWithError(w, http.StatusTooManyRequests, "rate limit exceeded")
		return
	}

	dupKey := "feedback:dup:" + feedbackFingerprint(payload, clientIP(r))
	if h.isDuplicate(r.Context(), dupKey) {
		respondWithJSON(w, http.StatusAccepted, map[string]string{
			"status": "duplicate_ignored",
		})
		return
	}

	feedback := &entities.Feedback{
		Rating:    payload.Rating,
		Message:   payload.Message,
		Email:     payload.Email,
		Page:      payload.Page,
		UserAgent: r.UserAgent(),
	}

	if err := h.service.Create(r.Context(), feedback); err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to submit feedback")
		return
	}

	respondWithJSON(w, http.StatusCreated, map[string]string{
		"status": "received",
		"id":     feedback.ID,
	})
}

func (h *FeedbackHandler) allowRequest(ctx context.Context, key string) (bool, time.Duration) {
	if h.cache == nil {
		return h.local.allow(key, feedbackRateLimit, feedbackRateWindow)
	}

	state := rateLimitState{}
	if data, err := h.cache.Get(ctx, key); err == nil {
		_ = json.Unmarshal(data, &state)
	}

	if state.Count >= feedbackRateLimit {
		return false, feedbackRateWindow
	}

	state.Count++
	data, _ := json.Marshal(state)
	_ = h.cache.Set(ctx, key, data, int(feedbackRateWindow.Seconds()))
	return true, feedbackRateWindow
}

type rateLimitState struct {
	Count int `json:"count"`
}

func (h *FeedbackHandler) isDuplicate(ctx context.Context, key string) bool {
	if h.cache == nil {
		return h.deduper.seen(key, feedbackDedupWindow)
	}

	exists, err := h.cache.Exists(ctx, key)
	if err == nil && exists {
		return true
	}

	_ = h.cache.Set(ctx, key, []byte("1"), int(feedbackDedupWindow.Seconds()))
	return false
}

type localRateLimiter struct {
	mu     sync.Mutex
	states map[string]*localRateState
}

type localRateState struct {
	count   int
	resetAt time.Time
}

func newLocalRateLimiter() *localRateLimiter {
	return &localRateLimiter{
		states: make(map[string]*localRateState),
	}
}

func (l *localRateLimiter) allow(key string, limit int, window time.Duration) (bool, time.Duration) {
	now := time.Now()

	l.mu.Lock()
	defer l.mu.Unlock()

	state, ok := l.states[key]
	if !ok || now.After(state.resetAt) {
		state = &localRateState{count: 0, resetAt: now.Add(window)}
		l.states[key] = state
	}

	if state.count >= limit {
		retryAfter := time.Until(state.resetAt)
		if retryAfter < 0 {
			retryAfter = window
		}
		return false, retryAfter
	}

	state.count++
	return true, window
}

type localDeduper struct {
	mu      sync.Mutex
	entries map[string]time.Time
}

func newLocalDeduper() *localDeduper {
	return &localDeduper{
		entries: make(map[string]time.Time),
	}
}

func (d *localDeduper) seen(key string, window time.Duration) bool {
	now := time.Now()

	d.mu.Lock()
	defer d.mu.Unlock()

	if expiresAt, ok := d.entries[key]; ok && now.Before(expiresAt) {
		return true
	}

	d.entries[key] = now.Add(window)
	return false
}

func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func feedbackFingerprint(payload feedbackRequest, ip string) string {
	normalized := []string{
		strconv.Itoa(payload.Rating),
		normalizeFeedback(payload.Message),
		strings.ToLower(strings.TrimSpace(payload.Email)),
		strings.ToLower(strings.TrimSpace(payload.Page)),
		ip,
	}

	hash := sha256.Sum256([]byte(strings.Join(normalized, "|")))
	return hex.EncodeToString(hash[:])
}

func normalizeFeedback(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return ""
	}
	return strings.Join(strings.Fields(trimmed), " ")
}
