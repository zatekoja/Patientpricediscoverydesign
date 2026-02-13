package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

const defaultBaseURL = "https://api.openai.com/v1"

// Client implements the OpenAI procedure enrichment provider.
type Client struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
	limiter    *tokenBucket
}

// NewClient creates a new OpenAI client.
func NewClient(cfg *config.OpenAIConfig) (*Client, error) {
	if cfg == nil || cfg.APIKey == "" {
		return nil, errors.New("openai api key is required")
	}

	model := cfg.Model
	if model == "" {
		model = "gpt-4o-mini"
	}

	limiter := newTokenBucket(cfg.RateLimitRPM, cfg.RateLimitBurst)

	return &Client{
		apiKey:  cfg.APIKey,
		model:   model,
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
		limiter: limiter,
	}, nil
}

type responseContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type responseOutput struct {
	Content []responseContent `json:"content"`
}

type responseEnvelope struct {
	Output []responseOutput `json:"output"`
}

type enrichmentPayload struct {
	Description string   `json:"description"`
	PrepSteps   []string `json:"prep_steps"`
	Risks       []string `json:"risks"`
	Recovery    []string `json:"recovery"`
}

// EnrichProcedure returns AI-generated content for a procedure.
func (c *Client) EnrichProcedure(ctx context.Context, procedure *entities.Procedure) (*entities.ProcedureEnrichment, error) {
	if procedure == nil {
		return nil, errors.New("procedure is required")
	}

	if c.limiter != nil {
		waitStart := time.Now()
		if err := c.limiter.Wait(ctx); err != nil {
			recordOpenAIMetric(ctx, c.model, 0, 0, err)
			return nil, err
		}
		recordOpenAIRateLimitWait(ctx, c.model, time.Since(waitStart))
	}

	userPrompt := buildSearchConceptUserPrompt(
		procedure.Name,
		procedure.Category,
		procedure.Code,
		procedure.Description,
	)

	payload := map[string]interface{}{
		"model": c.model,
		"input": []map[string]string{
			{"role": "system", "content": searchConceptSystemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature":       0.2,
		"max_output_tokens": 600,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/responses", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		recordOpenAIMetric(ctx, c.model, 0, time.Since(start), err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		recordOpenAIMetric(ctx, c.model, resp.StatusCode, time.Since(start), fmt.Errorf("status %d", resp.StatusCode))
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, fmt.Errorf("%w: openai request failed with status %d", providers.ErrProcedureEnrichmentUnauthorized, resp.StatusCode)
		}
		return nil, fmt.Errorf("openai request failed with status %d", resp.StatusCode)
	}

	var envelope responseEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		recordOpenAIMetric(ctx, c.model, resp.StatusCode, time.Since(start), err)
		return nil, err
	}

	var text string
	for _, out := range envelope.Output {
		for _, content := range out.Content {
			if content.Type == "output_text" && content.Text != "" {
				text = content.Text
				break
			}
		}
		if text != "" {
			break
		}
	}

	if text == "" {
		recordOpenAIMetric(ctx, c.model, resp.StatusCode, time.Since(start), errors.New("missing output text"))
		return nil, errors.New("openai response missing output text")
	}

	// Clean Markdown code blocks if present
	cleaned := text
	if strings.HasPrefix(cleaned, "```json") {
		cleaned = strings.TrimPrefix(cleaned, "```json")
		cleaned = strings.TrimSuffix(cleaned, "```")
	} else if strings.HasPrefix(cleaned, "```") {
		cleaned = strings.TrimPrefix(cleaned, "```")
		cleaned = strings.TrimSuffix(cleaned, "```")
	}
	cleaned = strings.TrimSpace(cleaned)

	parsed, err := parseEnrichmentPayloadWithConcepts([]byte(cleaned))
	if err != nil {
		recordOpenAIMetric(ctx, c.model, resp.StatusCode, time.Since(start), err)
		return nil, fmt.Errorf("failed to parse openai response: %w", err)
	}

	// Validate and sanitize search concepts if present
	if parsed.SearchConcepts != nil {
		_ = parsed.SearchConcepts.Validate()
	}

	recordOpenAIMetric(ctx, c.model, resp.StatusCode, time.Since(start), nil)
	return &entities.ProcedureEnrichment{
		Description:    parsed.Description,
		PrepSteps:      parsed.PrepSteps,
		Risks:          parsed.Risks,
		Recovery:       parsed.Recovery,
		SearchConcepts: parsed.SearchConcepts,
		Provider:       "openai",
		Model:          c.model,
	}, nil
}

func newTokenBucket(rpm int, burst int) *tokenBucket {
	if rpm == 0 {
		rpm = 60
	}
	if rpm < 0 {
		return nil
	}
	if burst <= 0 {
		burst = 5
	}
	if rpm <= 0 {
		return nil
	}
	return newTokenBucketWithRate(rpm, burst)
}

type tokenBucket struct {
	tokens chan struct{}
}

func newTokenBucketWithRate(rpm int, burst int) *tokenBucket {
	bucket := &tokenBucket{
		tokens: make(chan struct{}, burst),
	}

	for i := 0; i < burst; i++ {
		bucket.tokens <- struct{}{}
	}

	interval := time.Minute / time.Duration(rpm)
	if interval <= 0 {
		interval = time.Millisecond
	}

	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			select {
			case bucket.tokens <- struct{}{}:
			default:
			}
		}
	}()

	return bucket
}

func (b *tokenBucket) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-b.tokens:
		return nil
	}
}

type openAIMetrics struct {
	requestCount    metric.Int64Counter
	requestDuration metric.Float64Histogram
	requestErrors   metric.Int64Counter
	rateLimitWait   metric.Float64Histogram
}

var openaiMetricsInit = false
var openaiMetrics openAIMetrics

func ensureOpenAIMetrics() {
	if openaiMetricsInit {
		return
	}
	meter := otel.Meter("github.com/zatekoja/Patientpricediscoverydesign/backend/openai")

	requestCount, err := meter.Int64Counter(
		"ai.openai.request.count",
		metric.WithDescription("Number of OpenAI requests"),
	)
	if err != nil {
		return
	}
	requestDuration, err := meter.Float64Histogram(
		"ai.openai.request.duration",
		metric.WithDescription("OpenAI request duration in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return
	}
	requestErrors, err := meter.Int64Counter(
		"ai.openai.request.errors",
		metric.WithDescription("Number of OpenAI request errors"),
	)
	if err != nil {
		return
	}
	rateLimitWait, err := meter.Float64Histogram(
		"ai.openai.rate_limit.wait",
		metric.WithDescription("Time spent waiting for OpenAI rate limiter in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return
	}

	openaiMetrics = openAIMetrics{
		requestCount:    requestCount,
		requestDuration: requestDuration,
		requestErrors:   requestErrors,
		rateLimitWait:   rateLimitWait,
	}
	openaiMetricsInit = true
}

func recordOpenAIMetric(ctx context.Context, model string, statusCode int, duration time.Duration, err error) {
	ensureOpenAIMetrics()
	if !openaiMetricsInit {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("ai.provider", "openai"),
		attribute.String("ai.model", model),
	}
	if statusCode > 0 {
		attrs = append(attrs, attribute.Int("http.status_code", statusCode))
	}

	openaiMetrics.requestCount.Add(ctx, 1, metric.WithAttributes(attrs...))
	openaiMetrics.requestDuration.Record(ctx, float64(duration.Milliseconds()), metric.WithAttributes(attrs...))
	if err != nil {
		openaiMetrics.requestErrors.Add(ctx, 1, metric.WithAttributes(attrs...))
	}
}

func recordOpenAIRateLimitWait(ctx context.Context, model string, wait time.Duration) {
	ensureOpenAIMetrics()
	if !openaiMetricsInit {
		return
	}
	attrs := []attribute.KeyValue{
		attribute.String("ai.provider", "openai"),
		attribute.String("ai.model", model),
	}
	openaiMetrics.rateLimitWait.Record(ctx, float64(wait.Milliseconds()), metric.WithAttributes(attrs...))
}
