package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/config"
)

const defaultBaseURL = "https://api.openai.com/v1"

// Client implements the OpenAI procedure enrichment provider.
type Client struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
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

	return &Client{
		apiKey:  cfg.APIKey,
		model:   model,
		baseURL: defaultBaseURL,
		httpClient: &http.Client{
			Timeout: 20 * time.Second,
		},
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

	systemPrompt := "You are a clinical content assistant. Return ONLY valid JSON. " +
		"Use the schema: {\"description\": string, \"prep_steps\": string[], \"risks\": string[], \"recovery\": string[]}. " +
		"Description must be 1-2 short sentences. Provide 2-4 items in each list. " +
		"Keep the language simple and non-alarmist. Do not include medical advice or diagnosis."

	userPrompt := fmt.Sprintf(
		"Procedure name: %s\nCategory: %s\nCode: %s\nExisting description: %s\n",
		procedure.Name,
		procedure.Category,
		procedure.Code,
		procedure.Description,
	)

	payload := map[string]interface{}{
		"model": c.model,
		"input": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
		"temperature":       0.2,
		"max_output_tokens": 400,
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai request failed with status %d", resp.StatusCode)
	}

	var envelope responseEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
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
		return nil, errors.New("openai response missing output text")
	}

	var parsed enrichmentPayload
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse openai response: %w", err)
	}

	return &entities.ProcedureEnrichment{
		Description: parsed.Description,
		PrepSteps:   parsed.PrepSteps,
		Risks:       parsed.Risks,
		Recovery:    parsed.Recovery,
		Provider:    "openai",
		Model:       c.model,
	}, nil
}
