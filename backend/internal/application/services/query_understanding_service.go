package services

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/evaluation"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// QueryInterpretation holds the result of interpreting a user search query.
type QueryInterpretation struct {
	OriginalQuery    string                   `json:"original_query"`
	NormalizedQuery  string                   `json:"normalized_query"`
	CorrectedQuery   string                   `json:"corrected_query,omitempty"`
	DetectedIntent   evaluation.Intent        `json:"detected_intent"`
	IntentConfidence float64                  `json:"intent_confidence"`
	MappedConcepts   *entities.SearchConcepts `json:"mapped_concepts,omitempty"`
	ExpandedTerms    []string                 `json:"expanded_terms,omitempty"`
	SearchTerms      []string                 `json:"search_terms"`
	UnmatchedTerms   []string                 `json:"unmatched_terms,omitempty"`
}

// ConceptEntry represents a single entry in the concept dictionary.
type ConceptEntry struct {
	CanonicalForm string   `json:"canonical_form"`
	Category      string   `json:"category"` // condition, symptom, procedure, facility
	RelatedTerms  []string `json:"related_terms"`
	Specialties   []string `json:"specialties"`
	FacilityTypes []string `json:"facility_types"`
}

// QueryUnderstandingService interprets user search queries by normalizing,
// spell-correcting, detecting intent, and mapping to medical concepts.
type QueryUnderstandingService struct {
	conceptDict    map[string]*ConceptEntry // term → concept
	spellingDict   map[string]string        // misspelling → correct
	multiWordIndex map[string][]string      // first word → full multi-word keys
	cache          providers.CacheProvider
}

var nonAlphaNumDash = regexp.MustCompile(`[^\p{L}\p{N}\s\-'/]`)

var (
	missingTermCounterOnce sync.Once
	missingTermCounter     metric.Int64Counter
)

var ignoredUnmatchedTerms = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "at": {}, "for": {}, "in": {}, "me": {},
	"near": {}, "of": {}, "on": {}, "or": {}, "the": {}, "to": {}, "with": {},
	"without": {},
}

// NewQueryUnderstandingService creates a new service from config files.
func NewQueryUnderstandingService(conceptDictPath, spellingPath string) (*QueryUnderstandingService, error) {
	svc := &QueryUnderstandingService{
		conceptDict:    make(map[string]*ConceptEntry),
		spellingDict:   make(map[string]string),
		multiWordIndex: make(map[string][]string),
	}

	if err := svc.loadConceptDict(conceptDictPath); err != nil {
		return nil, err
	}
	if err := svc.loadSpellingDict(spellingPath); err != nil {
		return nil, err
	}

	return svc, nil
}

func (s *QueryUnderstandingService) loadConceptDict(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var raw map[string]*ConceptEntry
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for key, entry := range raw {
		k := strings.ToLower(strings.TrimSpace(key))
		s.conceptDict[k] = entry

		// Build multi-word index for phrase matching
		words := strings.Fields(k)
		if len(words) > 1 {
			first := words[0]
			s.multiWordIndex[first] = append(s.multiWordIndex[first], k)
		}
	}
	return nil
}

func (s *QueryUnderstandingService) loadSpellingDict(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var raw map[string]string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for k, v := range raw {
		s.spellingDict[strings.ToLower(strings.TrimSpace(k))] = strings.ToLower(strings.TrimSpace(v))
	}
	return nil
}

// SetCache sets the cache provider for interpretation results.
func (s *QueryUnderstandingService) SetCache(cache providers.CacheProvider) {
	s.cache = cache
}

// Interpret processes a raw search query through the full understanding pipeline.
func (s *QueryUnderstandingService) Interpret(query string) *QueryInterpretation {
	q := strings.TrimSpace(strings.ToLower(query))
	if q == "" {
		return &QueryInterpretation{OriginalQuery: query}
	}

	if s.cache != nil {
		cacheKey := "query_interp:" + q
		if data, err := s.cache.Get(context.Background(), cacheKey); err == nil {
			var cached QueryInterpretation
			if json.Unmarshal(data, &cached) == nil {
				return &cached
			}
		}
	}

	result := &QueryInterpretation{
		OriginalQuery: query,
	}

	// Step 1: Normalize
	normalized := s.normalize(query)
	result.NormalizedQuery = normalized
	if normalized == "" {
		return result
	}

	// Step 2: Spell correct
	corrected, wasChanged := s.spellCorrect(normalized)
	if wasChanged {
		result.CorrectedQuery = corrected
	}
	effectiveQuery := corrected

	// Step 3: Map to concepts (try multi-word first, then individual words)
	concepts, matchedEntries, unmatchedTerms := s.mapToConcepts(effectiveQuery)
	result.MappedConcepts = concepts
	result.UnmatchedTerms = unmatchedTerms

	// Step 4: Detect intent from matched entries
	result.DetectedIntent, result.IntentConfidence = s.detectIntent(effectiveQuery, matchedEntries)

	// Step 5: Build search terms (original + related + concept terms)
	result.SearchTerms = s.buildSearchTerms(effectiveQuery, matchedEntries)
	result.ExpandedTerms = result.SearchTerms
	s.recordMissingTermMetrics(result.UnmatchedTerms)

	if s.cache != nil {
		cacheKey := "query_interp:" + q
		if data, err := json.Marshal(result); err == nil {
			_ = s.cache.Set(context.Background(), cacheKey, data, 86400) // 24 hours
		}
	}

	return result
}

func (s *QueryUnderstandingService) normalize(query string) string {
	q := strings.ToLower(strings.TrimSpace(query))
	q = nonAlphaNumDash.ReplaceAllString(q, "")
	// Collapse multiple spaces
	words := strings.Fields(q)
	return strings.Join(words, " ")
}

func (s *QueryUnderstandingService) spellCorrect(normalized string) (string, bool) {
	words := strings.Fields(normalized)
	changed := false
	corrected := make([]string, len(words))

	for i, w := range words {
		if correction, ok := s.spellingDict[w]; ok {
			corrected[i] = correction
			changed = true
		} else {
			corrected[i] = w
		}
	}

	result := strings.Join(corrected, " ")
	return result, changed
}

func (s *QueryUnderstandingService) mapToConcepts(query string) (*entities.SearchConcepts, []*ConceptEntry, []string) {
	words := strings.Fields(query)
	if len(words) == 0 {
		return nil, nil, nil
	}

	var matchedEntries []*ConceptEntry
	matched := make(map[int]bool) // track which word positions are matched

	// Try matching the full query first
	if entry, ok := s.conceptDict[query]; ok {
		matchedEntries = append(matchedEntries, entry)
		for i := range words {
			matched[i] = true
		}
	}

	// Try multi-word phrases (longest match first)
	if len(matchedEntries) == 0 {
		for i := 0; i < len(words); i++ {
			if matched[i] {
				continue
			}
			// Check multi-word candidates starting with words[i]
			if candidates, ok := s.multiWordIndex[words[i]]; ok {
				bestLen := 0
				var bestEntry *ConceptEntry
				var bestPhrase string
				for _, phrase := range candidates {
					phraseWords := strings.Fields(phrase)
					if len(phraseWords)+i > len(words) {
						continue
					}
					// Check if words match
					candidate := strings.Join(words[i:i+len(phraseWords)], " ")
					if candidate == phrase {
						if len(phraseWords) > bestLen {
							bestLen = len(phraseWords)
							bestEntry = s.conceptDict[phrase]
							bestPhrase = phrase
						}
					}
				}
				if bestEntry != nil {
					matchedEntries = append(matchedEntries, bestEntry)
					phraseWords := strings.Fields(bestPhrase)
					for j := i; j < i+len(phraseWords); j++ {
						matched[j] = true
					}
					i += bestLen - 1 // skip matched words
				}
			}
		}
	}

	// Try individual words
	for i, w := range words {
		if matched[i] {
			continue
		}
		if entry, ok := s.conceptDict[w]; ok {
			matchedEntries = append(matchedEntries, entry)
			matched[i] = true
		}
	}

	if len(matchedEntries) == 0 {
		return nil, nil, collectUnmatchedTerms(words, matched)
	}

	// Merge all matched concepts
	concepts := &entities.SearchConcepts{}
	for _, entry := range matchedEntries {
		concepts.Specialties = appendUnique(concepts.Specialties, entry.Specialties...)
		concepts.FacilityTypes = appendUnique(concepts.FacilityTypes, entry.FacilityTypes...)

		switch entry.Category {
		case "condition":
			concepts.Conditions = appendUnique(concepts.Conditions, entry.CanonicalForm)
		case "symptom":
			concepts.Symptoms = appendUnique(concepts.Symptoms, entry.CanonicalForm)
		case "procedure":
			concepts.LayTerms = appendUnique(concepts.LayTerms, entry.CanonicalForm)
		case "facility":
			// facility types already added above
		}
	}

	return concepts, matchedEntries, collectUnmatchedTerms(words, matched)
}

func (s *QueryUnderstandingService) detectIntent(query string, entries []*ConceptEntry) (evaluation.Intent, float64) {
	if len(entries) == 0 {
		// Try to guess from facility keywords
		facilityKeywords := []string{"hospital", "clinic", "pharmacy", "lab", "center", "centre"}
		q := strings.ToLower(query)
		for _, kw := range facilityKeywords {
			if strings.Contains(q, kw) {
				return evaluation.IntentFacility, 0.6
			}
		}
		if strings.Contains(q, "near me") || strings.Contains(q, "nearby") || strings.Contains(q, "close to") {
			return evaluation.IntentFacility, 0.5
		}
		return evaluation.IntentProcedure, 0.3 // default low confidence
	}

	// Count intents from matched entries
	counts := make(map[string]int)
	for _, e := range entries {
		counts[e.Category]++
	}

	// Check for facility intent first (contains "near me" etc.)
	if strings.Contains(query, "near me") || strings.Contains(query, "nearby") {
		return evaluation.IntentFacility, 0.9
	}

	// Pick the most common intent
	best := ""
	bestCount := 0
	for cat, count := range counts {
		if count > bestCount {
			best = cat
			bestCount = count
		}
	}

	confidence := float64(bestCount) / float64(len(entries))
	if confidence > 1 {
		confidence = 1
	}

	switch best {
	case "condition":
		return evaluation.IntentCondition, confidence
	case "symptom":
		return evaluation.IntentSymptom, confidence
	case "procedure":
		return evaluation.IntentProcedure, confidence
	case "facility":
		return evaluation.IntentFacility, confidence
	}

	return evaluation.IntentProcedure, 0.3
}

func (s *QueryUnderstandingService) buildSearchTerms(query string, entries []*ConceptEntry) []string {
	seen := make(map[string]struct{})
	var terms []string

	add := func(t string) {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			return
		}
		if _, ok := seen[t]; !ok {
			seen[t] = struct{}{}
			terms = append(terms, t)
		}
	}

	// Add original query words
	for _, w := range strings.Fields(query) {
		add(w)
	}

	// Add terms from matched concepts
	for _, entry := range entries {
		add(entry.CanonicalForm)
		for _, rt := range entry.RelatedTerms {
			add(rt)
		}
		for _, sp := range entry.Specialties {
			add(sp)
		}
	}

	return terms
}

func appendUnique(slice []string, items ...string) []string {
	seen := make(map[string]struct{})
	for _, s := range slice {
		seen[s] = struct{}{}
	}
	for _, item := range items {
		if _, ok := seen[item]; !ok {
			seen[item] = struct{}{}
			slice = append(slice, item)
		}
	}
	return slice
}

func collectUnmatchedTerms(words []string, matched map[int]bool) []string {
	seen := make(map[string]struct{})
	terms := make([]string, 0, len(words))
	for i, word := range words {
		if matched[i] {
			continue
		}
		normalized := strings.ToLower(strings.TrimSpace(word))
		if !shouldTrackUnmatchedTerm(normalized) {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		terms = append(terms, normalized)
	}
	return terms
}

func shouldTrackUnmatchedTerm(term string) bool {
	if term == "" || len(term) < 2 || len(term) > 32 {
		return false
	}
	if _, ignore := ignoredUnmatchedTerms[term]; ignore {
		return false
	}
	for _, r := range term {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || r == '-' || r == '/' || r == '\'' {
			continue
		}
		return false
	}
	return true
}

func initMissingTermCounter() {
	meter := otel.Meter("github.com/zatekoja/Patientpricediscoverydesign/backend/query_understanding")
	counter, err := meter.Int64Counter(
		"search.term_not_found.count",
		metric.WithDescription("Count of query terms not found in the concept index"),
	)
	if err == nil {
		missingTermCounter = counter
	}
}

func (s *QueryUnderstandingService) recordMissingTermMetrics(terms []string) {
	if len(terms) == 0 {
		return
	}
	missingTermCounterOnce.Do(initMissingTermCounter)
	if missingTermCounter == nil {
		return
	}
	for _, term := range terms {
		missingTermCounter.Add(
			context.Background(),
			1,
			metric.WithAttributes(attribute.String("search.term", term)),
		)
	}
}
