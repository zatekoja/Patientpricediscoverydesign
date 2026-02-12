package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// UMLSClient handles UMLS REST API calls for unmapped abbreviation expansion
type UMLSClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// UMLSResponse represents UMLS API response for concept lookup
type UMLSResponse struct {
	Result struct {
		Classtype  string `json:"classType"`
		Name       string `json:"name"`
		UI         string `json:"ui"`
		Definition []struct {
			Value string `json:"value"`
		} `json:"definitions"`
	} `json:"result"`
}

// LLMEnricher handles LLM calls for abbreviation expansion fallback
type LLMEnricher struct {
	apiKey      string
	apiEndpoint string
	model       string
	httpClient  *http.Client
}

// LLMRequest represents a request to the LLM API
type LLMRequest struct {
	Model       string       `json:"model"`
	Messages    []LLMMessage `json:"messages"`
	Temperature float64      `json:"temperature"`
	MaxTokens   int          `json:"max_tokens"`
}

// LLMMessage represents a message in LLM conversation
type LLMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// LLMResponse represents response from LLM API
type LLMResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// NewUMLSClient creates a new UMLS client
func NewUMLSClient(apiKey string) *UMLSClient {
	return &UMLSClient{
		apiKey:  apiKey,
		baseURL: "https://uts-ws.nlm.nih.gov/rest/",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ExpandAbbreviationUMLS attempts to expand an unmapped abbreviation via UMLS API
func (u *UMLSClient) ExpandAbbreviationUMLS(ctx context.Context, abbreviation string) (string, error) {
	if u.apiKey == "" {
		return "", fmt.Errorf("UMLS API key not configured")
	}

	// Search for the abbreviation in UMLS
	searchURL := fmt.Sprintf("%ssearch/current?string=%s&searchType=exact&apiKey=%s&pageSize=1",
		u.baseURL, abbreviation, u.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create UMLS request: %w", err)
	}

	resp, err := u.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("UMLS API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("UMLS API error: status %d", resp.StatusCode)
	}

	var result UMLSResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse UMLS response: %w", err)
	}

	if result.Result.Name != "" {
		return result.Result.Name, nil
	}

	return "", fmt.Errorf("no UMLS match found for %s", abbreviation)
}

// NewLLMEnricher creates a new LLM enricher for fallback expansion
func NewLLMEnricher(apiKey, apiEndpoint, model string) *LLMEnricher {
	return &LLMEnricher{
		apiKey:      apiKey,
		apiEndpoint: apiEndpoint,
		model:       model,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// ExpandAbbreviationWithLLM attempts to expand unmapped abbreviation using LLM
func (le *LLMEnricher) ExpandAbbreviationWithLLM(ctx context.Context, abbreviation string, context_ string) (string, error) {
	if le.apiKey == "" {
		return "", fmt.Errorf("LLM API key not configured")
	}

	prompt := fmt.Sprintf(
		"What is the medical/healthcare meaning of the abbreviation '%s'? %s\n\nRespond with only the expanded form, no explanation.",
		abbreviation, context_,
	)

	req := LLMRequest{
		Model: le.model,
		Messages: []LLMMessage{
			{
				Role:    "system",
				Content: "You are a medical terminology expert. Expand healthcare abbreviations to their full forms.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: 0.3,
		MaxTokens:   50,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", le.apiEndpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create LLM request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", le.apiKey))

	resp, err := le.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("LLM API call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("LLM API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result LLMResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse LLM response: %w", err)
	}

	if len(result.Choices) > 0 && result.Choices[0].Message.Content != "" {
		return result.Choices[0].Message.Content, nil
	}

	return "", fmt.Errorf("no response from LLM for %s", abbreviation)
}

// ServiceNormalizerWithLLM extends ServiceNameNormalizer with LLM fallback
type ServiceNormalizerWithLLM struct {
	normalizer  *ServiceNameNormalizer
	umlsClient  *UMLSClient
	llmEnricher *LLMEnricher
}

// NewServiceNormalizerWithLLM creates a normalizer with LLM and UMLS support
func NewServiceNormalizerWithLLM(configPath string, umlsAPIKey string, llmAPIKey string, llmEndpoint string, llmModel string) (*ServiceNormalizerWithLLM, error) {
	normalizer, err := NewServiceNameNormalizer(configPath)
	if err != nil {
		return nil, err
	}

	var umlsClient *UMLSClient
	if umlsAPIKey != "" {
		umlsClient = NewUMLSClient(umlsAPIKey)
	}

	var llmEnricher *LLMEnricher
	if llmAPIKey != "" && llmEndpoint != "" {
		llmEnricher = NewLLMEnricher(llmAPIKey, llmEndpoint, llmModel)
	}

	return &ServiceNormalizerWithLLM{
		normalizer:  normalizer,
		umlsClient:  umlsClient,
		llmEnricher: llmEnricher,
	}, nil
}

// NormalizeWithEnrichment performs normalization with LLM fallback for unmapped terms
func (snl *ServiceNormalizerWithLLM) NormalizeWithEnrichment(ctx context.Context, originalName string) *NormalizedServiceName {
	// First, perform standard normalization
	result := snl.normalizer.Normalize(originalName)

	// Look for unmapped abbreviations that might benefit from LLM expansion
	if snl.shouldEnrichWithLLM(result) {
		enrichedResult := snl.enrichUnmappedTerms(ctx, result)
		return enrichedResult
	}

	return result
}

// shouldEnrichWithLLM determines if normalization should be enhanced with LLM
func (snl *ServiceNormalizerWithLLM) shouldEnrichWithLLM(result *NormalizedServiceName) bool {
	// If display name is too short or has many parenthetical qualifiers, may benefit from enrichment
	words := len(strings.Fields(result.DisplayName))
	return words > 0 && words <= 3 && snl.llmEnricher != nil
}

// enrichUnmappedTerms uses LLM to expand terms not in the static dictionary
func (snl *ServiceNormalizerWithLLM) enrichUnmappedTerms(ctx context.Context, result *NormalizedServiceName) *NormalizedServiceName {
	// Try UMLS first if configured
	if snl.umlsClient != nil {
		// Check for short terms that might be abbreviations
		for _, term := range strings.Fields(result.DisplayName) {
			if len(term) <= 5 && !snl.isCommonWord(term) {
				if expanded, err := snl.umlsClient.ExpandAbbreviationUMLS(ctx, term); err == nil && expanded != "" {
					// Found in UMLS, use it
					return result // Already handled in normalization
				}
			}
		}
	}

	// Fall back to LLM for edge cases
	if snl.llmEnricher != nil {
		if expanded, err := snl.llmEnricher.ExpandAbbreviationWithLLM(ctx, result.OriginalName, ""); err == nil && expanded != "" {
			// LLM successfully expanded, re-normalize with new understanding
			expandedResult := snl.normalizer.Normalize(expanded)
			return expandedResult
		}
	}

	return result
}

// isCommonWord checks if a word is likely a common English word, not an abbreviation
func (snl *ServiceNormalizerWithLLM) isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"and": true, "the": true, "for": true, "with": true, "without": true,
		"per": true, "day": true, "hour": true, "session": true, "visit": true,
		"or": true, "of": true, "at": true, "in": true, "on": true,
	}
	return commonWords[strings.ToLower(word)]
}
