package services

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
)

// FacilityService handles business logic for facilities
type FacilityService struct {
	repo                 repositories.FacilityRepository
	searchRepo           repositories.FacilitySearchRepository
	procedureRepo        repositories.FacilityProcedureRepository
	procedureCatalogRepo repositories.ProcedureRepository
	insuranceRepo        repositories.InsuranceRepository
	eventBus             providers.EventBus
	termExpander         *TermExpansionService
	queryUnderstanding   *QueryUnderstandingService
	searchRanking        *SearchRankingService
	featureFlags         *FeatureFlags
}

const maxSearchTags = 12

// NewFacilityService creates a new facility service
func NewFacilityService(
	repo repositories.FacilityRepository,
	searchRepo repositories.FacilitySearchRepository,
	procedureRepo repositories.FacilityProcedureRepository,
	procedureCatalogRepo repositories.ProcedureRepository,
	insuranceRepo repositories.InsuranceRepository,
) *FacilityService {
	return &FacilityService{
		repo:                 repo,
		searchRepo:           searchRepo,
		procedureRepo:        procedureRepo,
		procedureCatalogRepo: procedureCatalogRepo,
		insuranceRepo:        insuranceRepo,
		eventBus:             nil, // Injected separately to avoid breaking existing code
	}
}

// SetEventBus sets the event bus for publishing real-time updates
func (s *FacilityService) SetEventBus(eventBus providers.EventBus) {
	s.eventBus = eventBus
}

// SetTermExpander sets the term expansion service
func (s *FacilityService) SetTermExpander(expander *TermExpansionService) {
	s.termExpander = expander
}

// SetQueryUnderstanding sets the query understanding service
func (s *FacilityService) SetQueryUnderstanding(svc *QueryUnderstandingService) {
	s.queryUnderstanding = svc
}

// SetSearchRanking sets the search ranking service
func (s *FacilityService) SetSearchRanking(svc *SearchRankingService) {
	s.searchRanking = svc
}

// SetFeatureFlags sets the feature flags service
func (s *FacilityService) SetFeatureFlags(ff *FeatureFlags) {
	s.featureFlags = ff
}

// ExpandQuery expands a search query using the configured term expander
func (s *FacilityService) ExpandQuery(query string) []string {
	if s.termExpander == nil {
		if query == "" {
			return []string{}
		}
		return []string{query}
	}
	return s.termExpander.Expand(query)
}

// Create creates a new facility and indexes it
func (s *FacilityService) Create(ctx context.Context, facility *entities.Facility) error {
	// 1. Save to database
	if err := s.repo.Create(ctx, facility); err != nil {
		return err
	}

	// 2. Index in search engine
	if s.searchRepo != nil {
		s.enrichFacilityForSearch(ctx, facility)
		if err := s.searchRepo.Index(ctx, facility); err != nil {
			// Log error but don't fail the request (eventual consistency)
			log.Printf("Warning: Failed to index facility %s: %v", facility.ID, err)
		}
	}

	return nil
}

// GetByID retrieves a facility by ID
func (s *FacilityService) GetByID(ctx context.Context, id string) (*entities.Facility, error) {
	return s.repo.GetByID(ctx, id)
}

// Update updates a facility and updates index
func (s *FacilityService) Update(ctx context.Context, facility *entities.Facility) error {
	// Get the existing facility to detect changes
	existing, err := s.repo.GetByID(ctx, facility.ID)
	if err != nil {
		return err
	}

	// 1. Update in database
	if err := s.repo.Update(ctx, facility); err != nil {
		return err
	}

	// 2. Update index
	if s.searchRepo != nil {
		s.enrichFacilityForSearch(ctx, facility)
		if err := s.searchRepo.Index(ctx, facility); err != nil {
			log.Printf("Warning: Failed to update facility index %s: %v", facility.ID, err)
		}
	}

	// 3. Publish real-time update events if relevant fields changed
	if s.eventBus != nil {
		s.publishUpdateEvents(ctx, existing, facility)
	}

	return nil
}

// UpdateServiceAvailability updates availability for a specific facility procedure and publishes an event.
func (s *FacilityService) UpdateServiceAvailability(ctx context.Context, facilityID, procedureID string, isAvailable bool) (*entities.FacilityProcedure, error) {
	if s.procedureRepo == nil {
		return nil, fmt.Errorf("facility procedure repository not configured")
	}

	fp, err := s.procedureRepo.GetByFacilityAndProcedure(ctx, facilityID, procedureID)
	if err != nil {
		return nil, err
	}

	if fp.IsAvailable == isAvailable {
		return fp, nil
	}

	fp.IsAvailable = isAvailable
	if err := s.procedureRepo.Update(ctx, fp); err != nil {
		return nil, err
	}

	if s.eventBus != nil {
		location := entities.Location{}
		if facility, err := s.repo.GetByID(ctx, facilityID); err == nil && facility != nil {
			location = facility.Location
		}

		changedFields := map[string]interface{}{
			"procedure_id":       procedureID,
			"is_available":       isAvailable,
			"price":              fp.Price,
			"currency":           fp.Currency,
			"estimated_duration": fp.EstimatedDuration,
		}

		if s.procedureCatalogRepo != nil {
			if procedure, err := s.procedureCatalogRepo.GetByID(ctx, procedureID); err == nil && procedure != nil {
				if procedure.Name != "" {
					changedFields["procedure_name"] = procedure.Name
				}
				if procedure.Code != "" {
					changedFields["procedure_code"] = procedure.Code
				}
				if procedure.Category != "" {
					changedFields["procedure_category"] = procedure.Category
				}
				if procedure.Description != "" {
					changedFields["procedure_description"] = procedure.Description
				}
			}
		}

		event := entities.NewFacilityEvent(
			facilityID,
			entities.FacilityEventTypeServiceAvailabilityUpdate,
			location,
			changedFields,
		)

		facilityChannel := providers.GetFacilityChannel(facilityID)
		if err := s.eventBus.Publish(ctx, facilityChannel, event); err != nil {
			log.Printf("Warning: Failed to publish service availability event to %s: %v", facilityChannel, err)
		}

		if err := s.eventBus.Publish(ctx, providers.EventChannelFacilityUpdates, event); err != nil {
			log.Printf("Warning: Failed to publish service availability event to global channel: %v", err)
		}

		log.Printf("Published %s event for facility %s", entities.FacilityEventTypeServiceAvailabilityUpdate, facilityID)
	}

	return fp, nil
}

// publishUpdateEvents publishes events for facility updates
func (s *FacilityService) publishUpdateEvents(ctx context.Context, old, new *entities.Facility) {
	changedFields := make(map[string]interface{})
	var eventType entities.FacilityEventType

	// Check for capacity status changes
	if !strPtrEqual(old.CapacityStatus, new.CapacityStatus) {
		changedFields["capacity_status"] = new.CapacityStatus
		eventType = entities.FacilityEventTypeCapacityUpdate
	}

	// Check for wait time changes
	if !intPtrEqual(old.AvgWaitMinutes, new.AvgWaitMinutes) {
		changedFields["avg_wait_minutes"] = new.AvgWaitMinutes
		eventType = entities.FacilityEventTypeWaitTimeUpdate
	}

	// Check for urgent care availability changes
	if !boolPtrEqual(old.UrgentCareAvailable, new.UrgentCareAvailable) {
		changedFields["urgent_care_available"] = new.UrgentCareAvailable
		eventType = entities.FacilityEventTypeUrgentCareUpdate
	}

	// If no relevant changes, don't publish
	if len(changedFields) == 0 {
		return
	}

	// Create and publish event
	event := entities.NewFacilityEvent(
		new.ID,
		eventType,
		new.Location,
		changedFields,
	)

	// Publish to facility-specific channel
	facilityChannel := providers.GetFacilityChannel(new.ID)
	if err := s.eventBus.Publish(ctx, facilityChannel, event); err != nil {
		log.Printf("Warning: Failed to publish event to %s: %v", facilityChannel, err)
	}

	// Publish to global updates channel for regional subscribers
	if err := s.eventBus.Publish(ctx, providers.EventChannelFacilityUpdates, event); err != nil {
		log.Printf("Warning: Failed to publish event to global channel: %v", err)
	}

	log.Printf("Published %s event for facility %s", eventType, new.ID)
}

// Helper functions to compare pointer values
func strPtrEqual(a, b *string) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func intPtrEqual(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func boolPtrEqual(a, b *bool) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

// Delete deletes a facility and removes from index
func (s *FacilityService) Delete(ctx context.Context, id string) error {
	// 1. Delete from database
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	// 2. Delete from index
	if s.searchRepo != nil {
		if err := s.searchRepo.Delete(ctx, id); err != nil {
			log.Printf("Warning: Failed to delete facility from index %s: %v", id, err)
		}
	}

	return nil
}

// List retrieves facilities
func (s *FacilityService) List(ctx context.Context, filter repositories.FacilityFilter) ([]*entities.Facility, error) {
	return s.repo.List(ctx, filter)
}

// Search searches facilities using search engine if available, falling back to database
func (s *FacilityService) Search(ctx context.Context, params repositories.SearchParams) ([]*entities.Facility, error) {
	if s.searchRepo != nil {
		facilities, err := s.searchRepo.Search(ctx, params)
		if err == nil {
			return facilities, nil
		}
		log.Printf("Warning: Typesense search failed, falling back to database: %v", err)
	}
	return s.repo.Search(ctx, params)
}

func (s *FacilityService) searchWithCount(ctx context.Context, params repositories.SearchParams) ([]*entities.Facility, int, *QueryInterpretation, error) {
	var interpretation *QueryInterpretation
	useContextual := s.featureFlags == nil || s.featureFlags.ContextualSearchEnabled()

	if useContextual && s.queryUnderstanding != nil && params.Query != "" {
		interpretation = s.queryUnderstanding.Interpret(params.Query)
		if len(params.ExpandedTerms) == 0 {
			// Limit expansion terms to avoid overly restrictive AND behavior in Typesense
			expanded := interpretation.SearchTerms
			if len(expanded) > 5 {
				expanded = expanded[:5]
			}
			params.ExpandedTerms = expanded
		}
		if params.DetectedIntent == "" {
			params.DetectedIntent = string(interpretation.DetectedIntent)
		}
		if interpretation.MappedConcepts != nil {
			params.ConceptTerms = interpretation.MappedConcepts.AllTerms()
			if len(params.Specialties) == 0 {
				params.Specialties = interpretation.MappedConcepts.Specialties
			}
			// Don't hard-filter by facility types from interpretation as it is too restrictive
		}
	}

	var facilities []*entities.Facility
	var totalCount int
	var err error
	var usedSearchRepo bool

	if s.searchRepo != nil {
		usedSearchRepo = true
		if adapterWithCount, ok := s.searchRepo.(interface {
			SearchWithCount(ctx context.Context, params repositories.SearchParams) ([]*entities.Facility, int, error)
		}); ok {
			facilities, totalCount, err = adapterWithCount.SearchWithCount(ctx, params)
		} else {
			facilities, err = s.searchRepo.Search(ctx, params)
			if err == nil {
				totalCount = len(facilities)
			}
		}
	}

	if !usedSearchRepo || err != nil {
		if usedSearchRepo {
			log.Printf("Warning: Typesense search failed, falling back to database: %v", err)
		}

		if adapterWithCount, ok := s.repo.(interface {
			SearchWithCount(ctx context.Context, params repositories.SearchParams) ([]*entities.Facility, int, error)
		}); ok {
			facilities, totalCount, err = adapterWithCount.SearchWithCount(ctx, params)
		} else {
			facilities, err = s.repo.Search(ctx, params)
			if err == nil {
				totalCount = len(facilities)
			}
		}
	}

	if err != nil {
		return nil, 0, interpretation, err
	}

	if useContextual && s.searchRanking != nil && len(facilities) > 0 {
		ranked := s.searchRanking.Rank(facilities, interpretation, params.Latitude, params.Longitude)
		facilities = make([]*entities.Facility, len(ranked))
		for i, r := range ranked {
			facilities[i] = r.Facility
		}
	}

	return facilities, totalCount, interpretation, nil
}

// SearchResults returns enriched facility search results for the UI.
func (s *FacilityService) SearchResults(ctx context.Context, params repositories.SearchParams) ([]entities.FacilitySearchResult, error) {
	results, _, _, err := s.SearchResultsWithCount(ctx, params)
	return results, err
}

// SearchResultsWithCount returns enriched facility search results and total count for pagination.
func (s *FacilityService) SearchResultsWithCount(ctx context.Context, params repositories.SearchParams) ([]entities.FacilitySearchResult, int, *QueryInterpretation, error) {
	facilities, totalCount, interpretation, err := s.searchWithCount(ctx, params)
	if err != nil {
		return nil, 0, nil, err
	}

	results := make([]entities.FacilitySearchResult, 0, len(facilities))
	for _, facility := range facilities {
		if facility == nil {
			continue
		}

		// If search results are partial (e.g., Typesense), load full record from DB.
		if s.searchRepo != nil {
			tags := facility.Tags
			full, err := s.repo.GetByID(ctx, facility.ID)
			if err == nil && full != nil {
				if len(full.Tags) == 0 && len(tags) > 0 {
					full.Tags = tags
				}
				facility = full
			}
		}

		result := entities.FacilitySearchResult{
			ID:                facility.ID,
			Name:              facility.Name,
			FacilityType:      facility.FacilityType,
			Address:           facility.Address,
			Location:          facility.Location,
			PhoneNumber:       facility.PhoneNumber,
			WhatsAppNumber:    facility.WhatsAppNumber,
			Email:             facility.Email,
			Website:           facility.Website,
			Rating:            facility.Rating,
			ReviewCount:       facility.ReviewCount,
			DistanceKm:        haversineKm(params.Latitude, params.Longitude, facility.Location.Latitude, facility.Location.Longitude),
			Services:          []string{},
			Tags:              facility.Tags,
			AcceptedInsurance: []string{},
			UpdatedAt:         facility.UpdatedAt,
		}

		if facility.CapacityStatus != nil {
			result.CapacityStatus = *facility.CapacityStatus
		}
		if facility.WardStatuses != nil {
			result.WardStatuses = facility.WardStatuses
		}
		if facility.AvgWaitMinutes != nil {
			result.AvgWaitMinutes = facility.AvgWaitMinutes
		}
		if facility.UrgentCareAvailable != nil {
			result.UrgentCareAvailable = facility.UrgentCareAvailable
		}

		if s.procedureRepo != nil {
			if facilityProcedures, err := s.procedureRepo.ListByFacility(ctx, facility.ID); err == nil {
				price := priceRangeFromProcedures(facilityProcedures)
				if price != nil {
					result.Price = price
				}

				if s.procedureCatalogRepo != nil {
					result.ServicePrices, result.MatchedServices = servicePricesFromProcedures(ctx, facilityProcedures, s.procedureCatalogRepo, 8, params.Query, interpretation)
					result.Services = serviceNamesFromPrices(result.ServicePrices)
				}
			}
		}

		if s.insuranceRepo != nil {
			if providers, err := s.insuranceRepo.GetFacilityInsurance(ctx, facility.ID); err == nil {
				for _, provider := range providers {
					if provider == nil {
						continue
					}
					result.AcceptedInsurance = append(result.AcceptedInsurance, provider.Name)
				}
			}
		}

		results = append(results, result)
	}

	return results, totalCount, interpretation, nil
}

// Suggest returns facility suggestions for a query.
func (s *FacilityService) Suggest(ctx context.Context, query string, lat, lon float64, limit int) ([]*entities.Facility, error) {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" {
		return []*entities.Facility{}, nil
	}

	if limit <= 0 {
		limit = 5
	}

	if s.searchRepo != nil {
		if suggestRepo, ok := s.searchRepo.(interface {
			Suggest(ctx context.Context, query string, lat, lon float64, limit int) ([]*entities.Facility, error)
		}); ok {
			return suggestRepo.Suggest(ctx, trimmed, lat, lon, limit)
		}
	}

	params := repositories.SearchParams{
		Latitude:  lat,
		Longitude: lon,
		RadiusKm:  50,
		Limit:     100,
		Offset:    0,
	}

	facilities, err := s.repo.Search(ctx, params)
	if err != nil {
		return nil, err
	}

	queryLower := strings.ToLower(trimmed)
	suggestions := make([]*entities.Facility, 0, limit)
	for _, facility := range facilities {
		if facility == nil {
			continue
		}

		if matchesSuggestion(facility, queryLower) {
			suggestions = append(suggestions, facility)
		}

		if len(suggestions) >= limit {
			break
		}
	}

	return suggestions, nil
}

func (s *FacilityService) enrichFacilityForSearch(ctx context.Context, facility *entities.Facility) {
	if facility == nil {
		return
	}

	var providers []*entities.InsuranceProvider
	if s.insuranceRepo != nil {
		loaded, err := s.insuranceRepo.GetFacilityInsurance(ctx, facility.ID)
		if err == nil {
			providers = loaded
			names := make([]string, 0, len(loaded))
			for _, provider := range loaded {
				if provider == nil || provider.Name == "" {
					continue
				}
				names = append(names, provider.Name)
			}
			if len(names) > 0 {
				facility.AcceptedInsurance = names
			}
		}
	}

	facility.Tags = buildSearchTags(ctx, facility.ID, providers, s.procedureRepo, s.procedureCatalogRepo)
}

func buildSearchTags(
	ctx context.Context,
	facilityID string,
	providers []*entities.InsuranceProvider,
	procedureRepo repositories.FacilityProcedureRepository,
	procedureCatalogRepo repositories.ProcedureRepository,
) []string {
	builder := newSearchTagBuilder(maxSearchTags)

	for _, provider := range providers {
		if provider == nil {
			continue
		}
		builder.add(provider.Name, provider.Code)
		if builder.full() {
			return builder.tags()
		}
	}

	if procedureRepo == nil || procedureCatalogRepo == nil {
		return builder.tags()
	}

	procedures, err := procedureRepo.ListByFacility(ctx, facilityID)
	if err != nil {
		return builder.tags()
	}

	for _, item := range procedures {
		if item == nil {
			continue
		}
		procedure, err := procedureCatalogRepo.GetByID(ctx, item.ProcedureID)
		if err != nil || procedure == nil {
			continue
		}
		builder.add(procedure.Name, procedure.Code, procedure.Category)
		if builder.full() {
			break
		}
	}

	return builder.tags()
}

type searchTagBuilder struct {
	seen  map[string]struct{}
	list  []string
	limit int
}

func newSearchTagBuilder(limit int) *searchTagBuilder {
	if limit <= 0 {
		limit = maxSearchTags
	}
	return &searchTagBuilder{seen: make(map[string]struct{}), limit: limit}
}

func (b *searchTagBuilder) add(values ...string) {
	for _, value := range values {
		if b.full() {
			return
		}
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized == "" {
			continue
		}
		if _, exists := b.seen[normalized]; exists {
			continue
		}
		b.seen[normalized] = struct{}{}
		b.list = append(b.list, normalized)
	}
}

func (b *searchTagBuilder) full() bool {
	return b.limit > 0 && len(b.list) >= b.limit
}

func (b *searchTagBuilder) tags() []string {
	return b.list
}

func matchesSuggestion(facility *entities.Facility, queryLower string) bool {
	name := strings.ToLower(facility.Name)
	if strings.Contains(name, queryLower) {
		return true
	}

	if facility.FacilityType != "" && strings.Contains(strings.ToLower(facility.FacilityType), queryLower) {
		return true
	}

	if facility.Address.City != "" && strings.Contains(strings.ToLower(facility.Address.City), queryLower) {
		return true
	}

	if facility.Address.State != "" && strings.Contains(strings.ToLower(facility.Address.State), queryLower) {
		return true
	}

	return false
}

func priceRangeFromProcedures(items []*entities.FacilityProcedure) *entities.FacilityPriceRange {
	if len(items) == 0 {
		return nil
	}

	minPrice := math.MaxFloat64
	maxPrice := 0.0
	currency := ""

	for _, item := range items {
		if item == nil {
			continue
		}
		if item.Price < minPrice {
			minPrice = item.Price
		}
		if item.Price > maxPrice {
			maxPrice = item.Price
		}
		if currency == "" && item.Currency != "" {
			currency = item.Currency
		}
	}

	if minPrice == math.MaxFloat64 {
		return nil
	}

	return &entities.FacilityPriceRange{
		Min:      minPrice,
		Max:      maxPrice,
		Currency: currency,
	}
}

func servicePricesFromProcedures(
	ctx context.Context,
	items []*entities.FacilityProcedure,
	procedureRepo repositories.ProcedureRepository,
	limit int,
	query string,
	interp *QueryInterpretation,
) ([]entities.ServicePrice, []entities.ServicePrice) {
	if limit <= 0 || len(items) == 0 {
		return []entities.ServicePrice{}, []entities.ServicePrice{}
	}

	filtered := make([]*entities.FacilityProcedure, 0, len(items))
	for _, item := range items {
		if item == nil || !item.IsAvailable {
			continue
		}
		filtered = append(filtered, item)
	}

	if len(filtered) == 0 {
		return []entities.ServicePrice{}, []entities.ServicePrice{}
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Price < filtered[j].Price
	})

	preferredQuery := strings.ToLower(strings.TrimSpace(query))
	services := make([]entities.ServicePrice, 0, limit)
	matchedServices := make([]entities.ServicePrice, 0, limit)
	seen := make(map[string]struct{})

	// 1. First pass: identify conceptual matches
	if interp != nil && len(interp.SearchTerms) > 0 {
		for _, item := range filtered {
			if len(matchedServices) >= limit {
				break
			}
			procedure, err := procedureRepo.GetByID(ctx, item.ProcedureID)
			if err != nil || procedure == nil || procedure.Name == "" {
				continue
			}
			if _, exists := seen[procedure.ID]; exists {
				continue
			}

			if matchesConceptualProcedure(procedure, interp) {
				seen[procedure.ID] = struct{}{}
				sp := mapToServicePrice(item, procedure)
				matchedServices = append(matchedServices, sp)
				services = append(services, sp)
			}
		}
	}

	// 2. Second pass: identify exact lexical matches
	if preferredQuery != "" && len(services) < limit {
		for _, item := range filtered {
			if len(services) >= limit {
				break
			}
			procedure, err := procedureRepo.GetByID(ctx, item.ProcedureID)
			if err != nil || procedure == nil || procedure.Name == "" {
				continue
			}
			if _, exists := seen[procedure.ID]; exists {
				continue
			}

			if matchesProcedure(preferredQuery, procedure) {
				seen[procedure.ID] = struct{}{}
				sp := mapToServicePrice(item, procedure)
				services = append(services, sp)
				// If it didn't match conceptually but matched lexical, maybe it's still relevant
				if len(matchedServices) < 3 {
					matchedServices = append(matchedServices, sp)
				}
			}
		}
	}

	// 3. Third pass: fill remaining slots with cheapest available services
	for _, item := range filtered {
		if len(services) >= limit {
			break
		}
		procedure, err := procedureRepo.GetByID(ctx, item.ProcedureID)
		if err != nil || procedure == nil || procedure.Name == "" {
			continue
		}
		if _, exists := seen[procedure.ID]; exists {
			continue
		}

		seen[procedure.ID] = struct{}{}
		services = append(services, mapToServicePrice(item, procedure))
	}

	return services, matchedServices
}

func matchesConceptualProcedure(procedure *entities.Procedure, interp *QueryInterpretation) bool {
	if procedure == nil || interp == nil {
		return false
	}

	name := strings.ToLower(procedure.Name)
	displayName := strings.ToLower(procedure.DisplayName)
	cat := strings.ToLower(procedure.Category)

	// Match against mapped specialties/conditions (Higher precision)
	if interp.MappedConcepts != nil {
		for _, spec := range interp.MappedConcepts.Specialties {
			if strings.Contains(cat, spec) || strings.Contains(name, spec) {
				return true
			}
		}
		for _, cond := range interp.MappedConcepts.Conditions {
			if strings.Contains(strings.ToLower(procedure.Description), cond) || strings.Contains(name, cond) {
				return true
			}
		}
	}

	// Match against expanded terms with word boundary logic for short terms
	for _, term := range interp.SearchTerms {
		if len(term) <= 5 {
			// Exact word match for short terms to avoid "ache" matching "tracheostomy"
			if containsWord(name, term) || containsWord(displayName, term) || containsWord(cat, term) {
				return true
			}
			continue
		}
		if strings.Contains(name, term) || strings.Contains(displayName, term) || strings.Contains(cat, term) {
			return true
		}
	}

	return false
}

func containsWord(s, word string) bool {
	if s == word {
		return true
	}
	// Simple word boundary check
	return strings.Contains(s, " "+word+" ") || strings.HasPrefix(s, word+" ") || strings.HasSuffix(s, " "+word)
}

func mapToServicePrice(item *entities.FacilityProcedure, procedure *entities.Procedure) entities.ServicePrice {
	displayName := procedure.DisplayName
	if displayName == "" {
		displayName = procedure.Name
	}
	return entities.ServicePrice{
		ProcedureID:       item.ProcedureID,
		Name:              procedure.Name,
		DisplayName:       displayName,
		Price:             item.Price,
		Currency:          item.Currency,
		Description:       procedure.Description,
		Category:          procedure.Category,
		Code:              procedure.Code,
		EstimatedDuration: item.EstimatedDuration,
		NormalizedTags:    procedure.NormalizedTags,
		IsAvailable:       item.IsAvailable,
	}
}

func matchesProcedure(query string, procedure *entities.Procedure) bool {
	if procedure == nil {
		return false
	}
	name := strings.ToLower(strings.TrimSpace(procedure.Name))
	if name != "" && strings.Contains(name, query) {
		return true
	}
	displayName := strings.ToLower(strings.TrimSpace(procedure.DisplayName))
	if displayName != "" && strings.Contains(displayName, query) {
		return true
	}
	code := strings.ToLower(strings.TrimSpace(procedure.Code))
	return code != "" && strings.Contains(code, query)
}

func serviceNamesFromPrices(items []entities.ServicePrice) []string {
	if len(items) == 0 {
		return []string{}
	}
	names := make([]string, 0, len(items))
	for _, item := range items {
		if item.Name == "" {
			continue
		}
		names = append(names, item.Name)
	}
	return names
}

func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0
	lat1Rad := toRadians(lat1)
	lat2Rad := toRadians(lat2)
	deltaLat := toRadians(lat2 - lat1)
	deltaLon := toRadians(lon2 - lon1)

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

func toRadians(degrees float64) float64 {
	return degrees * (math.Pi / 180)
}
