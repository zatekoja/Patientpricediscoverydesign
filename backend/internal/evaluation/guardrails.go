package evaluation

type GuardrailConfig struct {
	MinIntentConfidence float64
	MaxExpansionTerms   int
	MaxConceptsPerQuery int
}

type Guardrails struct {
	config GuardrailConfig
}

func NewGuardrails(config GuardrailConfig) *Guardrails {
	if config.MaxExpansionTerms <= 0 {
		config.MaxExpansionTerms = 20
	}
	if config.MaxConceptsPerQuery <= 0 {
		config.MaxConceptsPerQuery = 10
	}
	return &Guardrails{config: config}
}

func (g *Guardrails) ShouldProcess(confidence float64) bool {
	return confidence >= g.config.MinIntentConfidence
}

func (g *Guardrails) LimitExpansion(terms []string) []string {
	if len(terms) > g.config.MaxExpansionTerms {
		return terms[:g.config.MaxExpansionTerms]
	}
	return terms
}
