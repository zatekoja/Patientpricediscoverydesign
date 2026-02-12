package openai

import (
	"encoding/json"
	"fmt"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

const searchConceptSystemPrompt = `You are a clinical content assistant for a Nigerian healthcare price comparison platform. Return ONLY valid JSON with this schema:
{
  "description": string (1-2 short sentences, simple language),
  "prep_steps": string[] (2-4 items),
  "risks": string[] (2-4 items),
  "recovery": string[] (2-4 items),
  "search_concepts": {
    "conditions": string[] (1-5 conditions this procedure treats or diagnoses),
    "symptoms": string[] (1-8 symptoms a patient might search with),
    "lay_terms": string[] (1-5 layperson-friendly names for this procedure),
    "synonyms": string[] (1-5 alternative clinical or medical names),
    "specialties": string[] (1-3 medical specialties, use snake_case),
    "facility_types": string[] (1-3 from: hospital, clinic, diagnostic_lab, imaging_center, pharmacy, urgent_care, specialty_clinic),
    "intent_tags": string[] (1-3 from: diagnostic, therapeutic, surgical, preventive, emergency)
  }
}
All search_concepts terms must be lowercase. Focus on terms a Nigerian patient might use when searching. Keep language simple and non-alarmist. Do not include medical advice or diagnosis.`

// enrichmentPayloadWithConcepts extends the base payload with search concepts.
type enrichmentPayloadWithConcepts struct {
	enrichmentPayload
	SearchConcepts *entities.SearchConcepts `json:"search_concepts,omitempty"`
}

func buildSearchConceptUserPrompt(name, category, code, description string) string {
	return fmt.Sprintf(
		"Procedure name: %s\nCategory: %s\nCode: %s\nExisting description: %s\n",
		name, category, code, description,
	)
}

func parseEnrichmentPayloadWithConcepts(data []byte) (*enrichmentPayloadWithConcepts, error) {
	var payload enrichmentPayloadWithConcepts
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("failed to parse enrichment payload: %w", err)
	}
	return &payload, nil
}
