package services

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"log"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/providerapi"
	apperrors "github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/errors"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/utils"
)

const enrichWorkerCount = 5

type ProviderIngestionSummary struct {
	RecordsProcessed            int `json:"records_processed"`
	FacilitiesCreated           int `json:"facilities_created"`
	FacilitiesUpdated           int `json:"facilities_updated"`
	ProceduresCreated           int `json:"procedures_created"`
	ProceduresUpdated           int `json:"procedures_updated"`
	FacilityProceduresCreated   int `json:"facility_procedures_created"`
	FacilityProceduresUpdated   int `json:"facility_procedures_updated"`
	ProcedureEnrichmentsCreated int `json:"procedure_enrichments_created"`
	ProcedureEnrichmentsFailed  int `json:"procedure_enrichments_failed"`
}

// ProviderIngestionService hydrates provider data into core backend storage.
type ProviderIngestionService struct {
	client                providerapi.Client
	facilityRepo          repositories.FacilityRepository
	facilityService       *FacilityService
	procedureRepo         repositories.ProcedureRepository
	facilityProcedureRepo repositories.FacilityProcedureRepository
	enrichmentRepo        repositories.ProcedureEnrichmentRepository
	enrichmentProvider    providers.ProcedureEnrichmentProvider
	geolocationProvider   providers.GeolocationProvider
	geocodeCache          map[string]*providers.GeocodedAddress
	cacheProvider         providers.CacheProvider
	pageSize              int
	normalizer            *utils.ServiceNameNormalizer
}

func NewProviderIngestionService(
	client providerapi.Client,
	facilityRepo repositories.FacilityRepository,
	facilityService *FacilityService,
	procedureRepo repositories.ProcedureRepository,
	facilityProcedureRepo repositories.FacilityProcedureRepository,
	enrichmentRepo repositories.ProcedureEnrichmentRepository,
	enrichmentProvider providers.ProcedureEnrichmentProvider,
	geolocationProvider providers.GeolocationProvider,
	cacheProvider providers.CacheProvider,
	pageSize int,
) *ProviderIngestionService {
	if pageSize <= 0 {
		pageSize = 500
	}

	// Initialize service normalizer (graceful degradation if config not found)
	configPath := os.Getenv("MEDICAL_ABBREVIATIONS_CONFIG")
	if configPath == "" {
		configPath = "config/medical_abbreviations.json"
	}
	normalizer, err := utils.NewServiceNameNormalizer(configPath)
	if err != nil {
		log.Printf("WARNING: Failed to initialize service name normalizer: %v", err)
		normalizer = nil // Graceful degradation
	}

	return &ProviderIngestionService{
		client:                client,
		facilityRepo:          facilityRepo,
		facilityService:       facilityService,
		procedureRepo:         procedureRepo,
		facilityProcedureRepo: facilityProcedureRepo,
		enrichmentRepo:        enrichmentRepo,
		enrichmentProvider:    enrichmentProvider,
		geolocationProvider:   geolocationProvider,
		geocodeCache:          map[string]*providers.GeocodedAddress{},
		cacheProvider:         cacheProvider,
		pageSize:              pageSize,
		normalizer:            normalizer,
	}
}

func (s *ProviderIngestionService) SyncCurrentData(ctx context.Context, providerID string) (*ProviderIngestionSummary, error) {
	if s.client == nil {
		return nil, fmt.Errorf("provider api client not configured")
	}

	summary := &ProviderIngestionSummary{}
	offset := 0

	facilityCache := map[string]*entities.Facility{}
	facilityTags := map[string]map[string]struct{}{}
	facilityUpdated := map[string]bool{}
	facilityNeedsUpdate := map[string]bool{}
	facilityProfiles := map[string]*providerapi.FacilityProfile{}

	for {
		resp, err := s.client.GetCurrentData(ctx, providerapi.CurrentDataRequest{
			ProviderID: providerID,
			Limit:      s.pageSize,
			Offset:     offset,
		})
		if err != nil {
			return summary, err
		}

		if len(resp.Data) == 0 {
			break
		}

		for _, record := range resp.Data {
			summary.RecordsProcessed++

			facilityID := strings.TrimSpace(record.FacilityID)
			if facilityID == "" {
				facilityID = buildFacilityID(providerID, record.FacilityName)
			}
			if facilityID == "" {
				continue
			}

			profile, ok := facilityProfiles[facilityID]
			if !ok && s.client != nil {
				if fetched, fetchErr := s.client.GetFacilityProfile(ctx, facilityID); fetchErr == nil {
					profile = fetched
				}
				facilityProfiles[facilityID] = profile
			}

			facility, exists := facilityCache[facilityID]
			if !exists {
				var created bool
				var ensureErr error
				facility, created, ensureErr = s.ensureFacility(ctx, facilityID, record, profile, mergeTags(record.Tags, profile))
				if ensureErr != nil {
					return summary, ensureErr
				}
				facilityCache[facilityID] = facility
				if created {
					summary.FacilitiesCreated++
				}
			}

			if len(record.Tags) > 0 {
				if facilityTags[facilityID] == nil {
					facilityTags[facilityID] = map[string]struct{}{}
				}
				for _, tag := range normalizeTags(record.Tags) {
					if tag == "" {
						continue
					}
					facilityTags[facilityID][tag] = struct{}{}
				}
			}
			if profile != nil && len(profile.Tags) > 0 {
				if facilityTags[facilityID] == nil {
					facilityTags[facilityID] = map[string]struct{}{}
				}
				for _, tag := range normalizeTags(profile.Tags) {
					if tag == "" {
						continue
					}
					facilityTags[facilityID][tag] = struct{}{}
				}
			}

			if profile != nil {
				if applyProfileStatus(facility, profile) {
					facilityNeedsUpdate[facilityID] = true
				}
			}

			procedure, created, err := s.ensureProcedure(ctx, record)
			if err != nil {
				return summary, err
			}
			if created {
				summary.ProceduresCreated++
			}

			updated, err := s.ensureFacilityProcedure(ctx, facility.ID, procedure.ID, record)
			if err != nil {
				return summary, err
			}
			if updated {
				summary.FacilityProceduresUpdated++
			} else {
				summary.FacilityProceduresCreated++
			}
		}

		offset += len(resp.Data)
		if resp.Metadata != nil {
			if resp.Metadata.Total > 0 && offset >= resp.Metadata.Total {
				break
			}
			if resp.Metadata.HasMore != nil && !*resp.Metadata.HasMore && offset > 0 {
				break
			}
		}
		if len(resp.Data) < s.pageSize {
			break
		}
	}

	// Update facility tags for search indexing
	for facilityID, tags := range facilityTags {
		facility := facilityCache[facilityID]
		if facility == nil {
			var err error
			facility, err = s.facilityRepo.GetByID(ctx, facilityID)
			if err != nil {
				continue
			}
		}

		mergedTags := make([]string, 0, len(tags))
		for tag := range tags {
			mergedTags = append(mergedTags, tag)
		}
		facility.Tags = mergedTags
		if facility.FacilityType == "" {
			facility.FacilityType = inferFacilityType(facility.Name, mergedTags)
		}

		if s.facilityService != nil {
			if err := s.facilityService.Update(ctx, facility); err != nil {
				return summary, err
			}
			if !facilityUpdated[facilityID] {
				summary.FacilitiesUpdated++
				facilityUpdated[facilityID] = true
			}
		}
	}

	for facilityID, needsUpdate := range facilityNeedsUpdate {
		if !needsUpdate || facilityUpdated[facilityID] {
			continue
		}
		facility := facilityCache[facilityID]
		if facility == nil {
			var err error
			facility, err = s.facilityRepo.GetByID(ctx, facilityID)
			if err != nil {
				continue
			}
		}
		if s.facilityService != nil {
			if err := s.facilityService.Update(ctx, facility); err != nil {
				return summary, err
			}
			summary.FacilitiesUpdated++
			facilityUpdated[facilityID] = true
		}
	}

	s.invalidateSearchCaches(ctx)

	// Enrich procedures in the background with its own long-lived context
	// so it doesn't get killed by the 2-minute ingestion timeout.
	go func() {
		enrichCtx, enrichCancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer enrichCancel()
		s.enrichProceduresBatch(enrichCtx)
	}()

	return summary, nil
}

func (s *ProviderIngestionService) ensureFacility(ctx context.Context, id string, record providerapi.PriceRecord, profile *providerapi.FacilityProfile, tags []string) (*entities.Facility, bool, error) {
	facility, err := s.facilityRepo.GetByID(ctx, id)
	if err == nil {
		updated := s.ensureFacilityLocation(ctx, facility, record, profile, tags)
		if updated {
			if s.facilityService != nil {
				if updateErr := s.facilityService.Update(ctx, facility); updateErr != nil {
					return facility, false, updateErr
				}
			} else if updateErr := s.facilityRepo.Update(ctx, facility); updateErr != nil {
				return facility, false, updateErr
			}
		}
		return facility, false, nil
	}

	if !isNotFound(err) {
		return nil, false, err
	}

	now := time.Now()
	facilityName := record.FacilityName
	facilityType := inferFacilityType(record.FacilityName, record.Tags)
	if profile != nil {
		if profile.Name != "" {
			facilityName = profile.Name
		}
		if profile.FacilityType != "" {
			facilityType = profile.FacilityType
		}
	}

	facility = &entities.Facility{
		ID:           id,
		Name:         facilityName,
		FacilityType: facilityType,
		Location: entities.Location{
			Latitude:  0,
			Longitude: 0,
		},
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if profile != nil {
		facility.Description = profile.Description
		facility.PhoneNumber = profile.PhoneNumber
		facility.WhatsAppNumber = profile.WhatsAppNumber
		facility.Email = profile.Email
		facility.Website = profile.Website
		facility.Address = entities.Address{
			Street:  profile.Address.Street,
			City:    profile.Address.City,
			State:   profile.Address.State,
			ZipCode: profile.Address.ZipCode,
			Country: profile.Address.Country,
		}
		facility.Location = entities.Location{
			Latitude:  profile.Location.Latitude,
			Longitude: profile.Location.Longitude,
		}
		if len(profile.Tags) > 0 {
			facility.Tags = profile.Tags
		}
		if profile.CapacityStatus != nil {
			facility.CapacityStatus = profile.CapacityStatus
		}
		if profile.AvgWaitMinutes != nil {
			facility.AvgWaitMinutes = profile.AvgWaitMinutes
		}
		if profile.UrgentCareAvailable != nil {
			facility.UrgentCareAvailable = profile.UrgentCareAvailable
		}
	}

	s.ensureFacilityLocation(ctx, facility, record, profile, tags)

	if s.facilityService != nil {
		if err := s.facilityService.Create(ctx, facility); err != nil {
			return nil, false, err
		}
		return facility, true, nil
	}

	if err := s.facilityRepo.Create(ctx, facility); err != nil {
		return nil, false, err
	}

	return facility, true, nil
}

func (s *ProviderIngestionService) ensureFacilityLocation(ctx context.Context, facility *entities.Facility, record providerapi.PriceRecord, profile *providerapi.FacilityProfile, tags []string) bool {
	if facility == nil || s.geolocationProvider == nil {
		return false
	}

	query := buildGeocodeQuery(facility, record, profile, tags)
	if query == "" {
		return false
	}

	if hasLocation(facility.Location) && !shouldOverrideLocation(facility.Location, tags, query) {
		return false
	}

	if cached, ok := s.geocodeCache[query]; ok && cached != nil {
		applyGeocodedAddress(facility, cached)
		return true
	}

	geo, err := s.geolocationProvider.Geocode(ctx, query)
	if err != nil || geo == nil {
		return false
	}

	// Geocode now returns the full address, so we use it directly
	// We verify if the returned location matches our expectations (e.g., inside Nigeria)
	loc := entities.Location{
		Latitude:  geo.Coordinates.Latitude,
		Longitude: geo.Coordinates.Longitude,
	}
	if isLikelyNigeria(tags, query) && isOutsideNigeria(loc) {
		// If the geocoded result puts us outside Nigeria when we expect to be inside,
		// we might want to fallback or discard.
		// For now, we trust the specific geocode result over the generic check.
	}

	applyGeocodedAddress(facility, geo)
	s.geocodeCache[query] = geo
	return true
}

func applyGeocodedAddress(facility *entities.Facility, geo *providers.GeocodedAddress) {
	if facility == nil || geo == nil {
		return
	}
	facility.Location.Latitude = geo.Coordinates.Latitude
	facility.Location.Longitude = geo.Coordinates.Longitude

	// Populate street address
	if facility.Address.Street == "" {
		if geo.Street != "" {
			facility.Address.Street = geo.Street
		} else if geo.FormattedAddress != "" {
			// Use formatted address without country/state suffix as street
			street := geo.FormattedAddress
			// Remove country and state suffixes
			if geo.Country != "" {
				street = strings.TrimSuffix(street, ", "+geo.Country)
				street = strings.TrimSuffix(street, ","+geo.Country)
			}
			if geo.State != "" {
				street = strings.TrimSuffix(street, ", "+geo.State)
				street = strings.TrimSuffix(street, ","+geo.State)
			}
			facility.Address.Street = strings.TrimSpace(street)
		}
	}

	// Populate city
	if facility.Address.City == "" && geo.City != "" {
		facility.Address.City = geo.City
	}

	// Populate state
	if facility.Address.State == "" && geo.State != "" {
		facility.Address.State = geo.State
	}

	// Populate country
	if facility.Address.Country == "" && geo.Country != "" {
		facility.Address.Country = geo.Country
	}

	// Populate zip code
	if facility.Address.ZipCode == "" && geo.ZipCode != "" {
		facility.Address.ZipCode = geo.ZipCode
	}
}

func hasLocation(loc entities.Location) bool {
	return loc.Latitude != 0 || loc.Longitude != 0
}

func shouldOverrideLocation(loc entities.Location, tags []string, query string) bool {
	if isDefaultMockLocation(loc) {
		return true
	}
	if isLikelyNigeria(tags, query) && isOutsideNigeria(loc) {
		return true
	}
	return false
}

func isDefaultMockLocation(loc entities.Location) bool {
	return nearlyEqual(loc.Latitude, 37.7749, 0.0001) && nearlyEqual(loc.Longitude, -122.4194, 0.0001)
}

func isOutsideNigeria(loc entities.Location) bool {
	// Rough bounding box for Nigeria
	if loc.Latitude >= 4.0 && loc.Latitude <= 14.5 && loc.Longitude >= 2.0 && loc.Longitude <= 15.5 {
		return false
	}
	return true
}

func isLikelyNigeria(tags []string, query string) bool {
	if strings.Contains(strings.ToLower(query), "nigeria") {
		return true
	}
	for _, tag := range tags {
		switch strings.ToLower(tag) {
		case "nigeria", "lagos", "lagos_state", "abuja", "fct", "federal_capital_territory":
			return true
		}
	}
	return false
}

func nearlyEqual(a, b, epsilon float64) bool {
	if a > b {
		return a-b < epsilon
	}
	return b-a < epsilon
}

func buildGeocodeQuery(facility *entities.Facility, record providerapi.PriceRecord, profile *providerapi.FacilityProfile, tags []string) string {
	parts := make([]string, 0, 4)

	if profile != nil {
		if profile.Address.Street != "" {
			parts = append(parts, profile.Address.Street)
		}
		if profile.Address.City != "" {
			parts = append(parts, profile.Address.City)
		}
		if profile.Address.State != "" {
			parts = append(parts, profile.Address.State)
		}
		if profile.Address.Country != "" {
			parts = append(parts, profile.Address.Country)
		}
	}

	if len(parts) == 0 {
		if facility.Address.Street != "" {
			parts = append(parts, facility.Address.Street)
		}
		if facility.Address.City != "" {
			parts = append(parts, facility.Address.City)
		}
		if facility.Address.State != "" {
			parts = append(parts, facility.Address.State)
		}
		if facility.Address.Country != "" {
			parts = append(parts, facility.Address.Country)
		}
	}

	if len(parts) == 0 {
		if record.FacilityName != "" {
			parts = append(parts, record.FacilityName)
		} else if facility.Name != "" {
			parts = append(parts, facility.Name)
		}
	}

	// FIX: Only use tag-based region inference as last resort when no address components exist
	// Don't add inferred region if we have actual city/state - it biases geocoding results
	hasAddress := false
	if profile != nil {
		hasAddress = profile.Address.City != "" || profile.Address.State != ""
	} else {
		hasAddress = facility.Address.City != "" || facility.Address.State != ""
	}

	if !hasAddress && len(parts) > 0 {
		// Only infer region if we have no address components but have facility name
		region := inferRegionFromTags(tags)
		if region != "" {
			parts = append(parts, region)
		}
	}

	if !containsToken(parts, "Nigeria") {
		parts = append(parts, "Nigeria")
	}

	return strings.TrimSpace(strings.Join(parts, ", "))
}

func inferRegionFromTags(tags []string) string {
	// FIX: Made more strict - only exact matches, not substring matches
	// This prevents tags like "teaching_hospital_lagos_affiliated" from being treated as location
	for _, tag := range tags {
		tagLower := strings.ToLower(strings.TrimSpace(tag))
		switch tagLower {
		case "lagos", "lagos_state", "lagos state":
			return "Lagos"
		case "abuja", "fct", "federal_capital_territory", "federal capital territory":
			return "Abuja"
		case "port_harcourt", "port harcourt", "rivers", "rivers_state", "rivers state":
			return "Port Harcourt"
		case "kano", "kano_state", "kano state":
			return "Kano"
		case "ibadan", "oyo", "oyo_state", "oyo state":
			return "Ibadan"
		case "enugu", "enugu_state", "enugu state":
			return "Enugu"
		}
	}
	return ""
}

func containsToken(parts []string, token string) bool {
	tokenLower := strings.ToLower(token)
	for _, part := range parts {
		if strings.Contains(strings.ToLower(part), tokenLower) {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func mergeTags(a []string, profile *providerapi.FacilityProfile) []string {
	combined := append([]string{}, a...)
	if profile != nil && len(profile.Tags) > 0 {
		combined = append(combined, profile.Tags...)
	}
	return normalizeTags(combined)
}

func (s *ProviderIngestionService) invalidateSearchCaches(ctx context.Context) {
	if s.cacheProvider == nil {
		return
	}
	patterns := []string{
		"http:cache:*search*",
		"http:cache:*suggest*",
		"http:cache:*facilities*",
	}
	for _, pattern := range patterns {
		if err := s.cacheProvider.DeletePattern(ctx, pattern); err != nil {
			log.Printf("Warning: failed to invalidate cache pattern %s: %v", pattern, err)
		}
	}
}

func (s *ProviderIngestionService) ensureProcedure(ctx context.Context, record providerapi.PriceRecord) (*entities.Procedure, bool, error) {
	code := deriveProcedureCode(record.ProcedureCode, record.ProcedureDescription)
	if code == "" {
		return nil, false, fmt.Errorf("missing procedure code/description")
	}

	existing, err := s.procedureRepo.GetByCode(ctx, code)
	if err == nil {
		// Backfill normalization for existing procedures (older ingestions may have display_name=name).
		if s.normalizer != nil {
			// Prefer normalizing the stored original name to keep deterministic output.
			normalized := s.normalizer.Normalize(existing.Name)
			if normalized != nil {
				if normalized.DisplayName != "" && normalized.DisplayName != existing.DisplayName {
					existing.DisplayName = normalized.DisplayName
				}
				if len(normalized.NormalizedTags) > 0 {
					// Backfill if missing, or if tags changed due to improved normalization rules.
					if len(existing.NormalizedTags) == 0 || strings.Join(existing.NormalizedTags, ",") != strings.Join(normalized.NormalizedTags, ",") {
						existing.NormalizedTags = normalized.NormalizedTags
					}
				}
			}
		}
		if record.ProcedureCategory != "" && existing.Category != record.ProcedureCategory {
			existing.Category = record.ProcedureCategory
		}
		if record.ProcedureDetails != "" && existing.Description != record.ProcedureDetails {
			existing.Description = record.ProcedureDetails
		} else if existing.Description == "" {
			existing.Description = record.ProcedureDescription
		}
		if updateErr := s.procedureRepo.Update(ctx, existing); updateErr != nil {
			return existing, false, updateErr
		}
		return existing, false, nil
	}
	if !isNotFound(err) {
		return nil, false, err
	}

	now := time.Now()
	description := record.ProcedureDescription
	if record.ProcedureDetails != "" {
		description = record.ProcedureDetails
	}
	category := inferProcedureCategory(record.ProcedureDescription, record.Tags)
	if record.ProcedureCategory != "" {
		category = record.ProcedureCategory
	}

	// Apply normalization
	displayName := record.ProcedureDescription
	var normalizedTags []string
	if s.normalizer != nil {
		normalized := s.normalizer.Normalize(record.ProcedureDescription)
		displayName = normalized.DisplayName
		normalizedTags = normalized.NormalizedTags
	}

	procedure := &entities.Procedure{
		ID:             buildProcedureID(code),
		Name:           record.ProcedureDescription,
		DisplayName:    displayName,
		Code:           code,
		Category:       category,
		Description:    description,
		NormalizedTags: normalizedTags,
		IsActive:       true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.procedureRepo.Create(ctx, procedure); err != nil {
		return nil, false, err
	}

	return procedure, true, nil
}

func (s *ProviderIngestionService) ensureFacilityProcedure(ctx context.Context, facilityID, procedureID string, record providerapi.PriceRecord) (bool, error) {
	existing, err := s.facilityProcedureRepo.GetByFacilityAndProcedure(ctx, facilityID, procedureID)
	if err == nil && existing != nil {
		// Price Aggregation Strategy: Average prices from multiple providers
		// When a facility-procedure already exists (from another provider),
		// calculate the average price between the existing price and the new price
		averagePrice := calculateAveragePrice(existing.Price, record.Price)

		existing.Price = averagePrice
		existing.Currency = record.Currency
		existing.IsAvailable = true
		if record.EstimatedDurationMin != nil {
			existing.EstimatedDuration = *record.EstimatedDurationMin
		}
		existing.UpdatedAt = time.Now()
		if updateErr := s.facilityProcedureRepo.Update(ctx, existing); updateErr != nil {
			return false, updateErr
		}
		return true, nil
	}

	if err != nil && !isNotFound(err) {
		return false, err
	}

	now := time.Now()
	fp := &entities.FacilityProcedure{
		ID:          buildFacilityProcedureID(facilityID, procedureID),
		FacilityID:  facilityID,
		ProcedureID: procedureID,
		Price:       record.Price,
		Currency:    record.Currency,
		EstimatedDuration: func() int {
			if record.EstimatedDurationMin != nil {
				return *record.EstimatedDurationMin
			}
			return 0
		}(),
		IsAvailable: true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.facilityProcedureRepo.Create(ctx, fp); err != nil {
		return false, err
	}

	return false, nil
}

func isNotFound(err error) bool {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		return appErr.Type == apperrors.ErrorTypeNotFound
	}
	return false
}

// calculateAveragePrice computes the average of two prices
// Used for price aggregation strategy when multiple providers report prices for same facility-procedure
func calculateAveragePrice(existingPrice, newPrice float64) float64 {
	if existingPrice == 0 && newPrice == 0 {
		return 0
	}
	if existingPrice == 0 {
		return newPrice
	}
	if newPrice == 0 {
		return existingPrice
	}
	return (existingPrice + newPrice) / 2.0
}

func buildFacilityID(providerID, name string) string {
	normalized := normalizeIdentifier(name)
	if normalized == "" {
		return ""
	}
	if providerID == "" {
		providerID = "provider"
	}
	return fmt.Sprintf("%s_%s", providerID, normalized)
}

func buildProcedureID(code string) string {
	normalized := normalizeIdentifier(code)
	if normalized == "" {
		return ""
	}
	return fmt.Sprintf("proc_%s", normalized)
}

func buildFacilityProcedureID(facilityID, procedureID string) string {
	return fmt.Sprintf("fp_%s_%s", normalizeIdentifier(facilityID), normalizeIdentifier(procedureID))
}

func deriveProcedureCode(code, description string) string {
	trimmed := strings.TrimSpace(code)
	if trimmed != "" {
		return normalizeIdentifier(trimmed)
	}
	trimmed = strings.TrimSpace(description)
	if trimmed == "" {
		return ""
	}
	hash := hashString(trimmed)
	return fmt.Sprintf("desc_%s", hash)
}

func inferFacilityType(name string, tags []string) string {
	candidates := []string{strings.ToLower(name)}
	for _, tag := range tags {
		candidates = append(candidates, strings.ToLower(tag))
	}

	for _, value := range candidates {
		switch {
		case strings.Contains(value, "urgent"):
			return "urgent_care"
		case strings.Contains(value, "imaging") || strings.Contains(value, "radiology"):
			return "imaging_center"
		case strings.Contains(value, "lab") || strings.Contains(value, "laboratory") || strings.Contains(value, "diagnostic"):
			return "diagnostic_lab"
		case strings.Contains(value, "specialty"):
			return "specialty_clinic"
		case strings.Contains(value, "surgery"):
			return "outpatient_surgery"
		case strings.Contains(value, "clinic"):
			return "clinic"
		}
	}

	return "hospital"
}

func inferProcedureCategory(description string, tags []string) string {
	candidates := []string{strings.ToLower(description)}
	for _, tag := range tags {
		candidates = append(candidates, strings.ToLower(tag))
	}

	for _, value := range candidates {
		switch {
		case strings.Contains(value, "hiv") || strings.Contains(value, "sti") || strings.Contains(value, "std") || strings.Contains(value, "vdrl") || strings.Contains(value, "hepatitis") || strings.Contains(value, "sexual health") || strings.Contains(value, "gonorrh") || strings.Contains(value, "chlamydia") || strings.Contains(value, "syphilis"):
			return "sti_testing"
		case strings.Contains(value, "imaging") || strings.Contains(value, "scan") || strings.Contains(value, "x-ray") || strings.Contains(value, "radiolog") || strings.Contains(value, "ultrasound") || strings.Contains(value, "mri") || strings.Contains(value, "echo"):
			return "imaging"
		case strings.Contains(value, "lab") || strings.Contains(value, "laboratory") || strings.Contains(value, "diagnostic"):
			return "laboratory"
		case strings.Contains(value, "ophthal") || strings.Contains(value, "cataract") || strings.Contains(value, "glaucoma") || strings.Contains(value, "retina") || strings.Contains(value, "cornea"):
			return "ophthalmology"
		case strings.Contains(value, "dental") || strings.Contains(value, "tooth") || strings.Contains(value, "orthodontic") || strings.Contains(value, "maxillofacial"):
			return "dental"
		case strings.Contains(value, "surgery") || strings.Contains(value, "operation") || strings.Contains(value, "theatre") || strings.Contains(value, "laparotomy") || strings.Contains(value, "laparoscop"):
			return "surgical"
		case value == "ent" || strings.Contains(value, "tonsil") || strings.Contains(value, "adenoid") || strings.Contains(value, "mastoid") || strings.Contains(value, "laryngoscop") || strings.Contains(value, "tracheostomy") || strings.Contains(value, "ear nose"):
			return "ent"
		case strings.Contains(value, "psychiatr") || strings.Contains(value, "psycholog") || strings.Contains(value, "mental") || strings.Contains(value, "electroconvulsive"):
			return "psychiatry"
		case strings.Contains(value, "dermatol") || strings.Contains(value, "skin biopsy") || strings.Contains(value, "hyfrecation"):
			return "dermatology"
		case strings.Contains(value, "physiotherapy") || strings.Contains(value, "physio") || strings.Contains(value, "rehab"):
			return "physiotherapy"
		case strings.Contains(value, "dietary") || strings.Contains(value, "nutrition") || strings.Contains(value, "meal") || strings.Contains(value, "feeding"):
			return "dietary"
		case strings.Contains(value, "endoscop") || strings.Contains(value, "colonoscop") || strings.Contains(value, "polypectomy"):
			return "endoscopy"
		case strings.Contains(value, "urol") || strings.Contains(value, "catheter") || strings.Contains(value, "cystoscop") || strings.Contains(value, "prostat"):
			return "urology"
		case strings.Contains(value, "oncol") || strings.Contains(value, "chemo") || strings.Contains(value, "radiotherapy") || strings.Contains(value, "cancer"):
			return "oncology"
		case strings.Contains(value, "orthopaed") || strings.Contains(value, "orthoped") || strings.Contains(value, "fracture") || strings.Contains(value, "plaster of paris"):
			return "orthopaedics"
		case strings.Contains(value, "stroke") || strings.Contains(value, "cerebrovascular") || strings.Contains(value, "eeg") || strings.Contains(value, "electroencephalogr"):
			return "neurology"
		case strings.Contains(value, "ward") || strings.Contains(value, "admission") || strings.Contains(value, "accommodation") || strings.Contains(value, "bed"):
			return "accommodation"
		case strings.Contains(value, "ambulance") || strings.Contains(value, "emergency") || strings.Contains(value, "casualty") || strings.Contains(value, "triage"):
			return "emergency"
		case strings.Contains(value, "registration") || strings.Contains(value, "card") || strings.Contains(value, "folder"):
			return "registration"
		case strings.Contains(value, "report") || strings.Contains(value, "certificate"):
			return "administrative"
		case strings.Contains(value, "therapy"):
			return "therapeutic"
		case strings.Contains(value, "checkup") || strings.Contains(value, "screen"):
			return "preventive"
		}
	}

	return ""
}

func normalizeIdentifier(value string) string {
	lowered := strings.ToLower(strings.TrimSpace(value))
	if lowered == "" {
		return ""
	}
	var builder strings.Builder
	builder.Grow(len(lowered))
	lastUnderscore := false
	for _, r := range lowered {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			builder.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			builder.WriteRune('_')
			lastUnderscore = true
		}
	}
	result := strings.Trim(builder.String(), "_")
	return result
}

func normalizeTags(tags []string) []string {
	normalized := make([]string, 0, len(tags))
	for _, tag := range tags {
		trimmed := strings.TrimSpace(strings.ToLower(tag))
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func applyProfileStatus(facility *entities.Facility, profile *providerapi.FacilityProfile) bool {
	if facility == nil || profile == nil {
		return false
	}
	changed := false

	if profile.CapacityStatus != nil {
		if facility.CapacityStatus == nil || *facility.CapacityStatus != *profile.CapacityStatus {
			facility.CapacityStatus = profile.CapacityStatus
			changed = true
		}
	}
	if profile.AvgWaitMinutes != nil {
		if facility.AvgWaitMinutes == nil || *facility.AvgWaitMinutes != *profile.AvgWaitMinutes {
			facility.AvgWaitMinutes = profile.AvgWaitMinutes
			changed = true
		}
	}
	if profile.UrgentCareAvailable != nil {
		if facility.UrgentCareAvailable == nil || *facility.UrgentCareAvailable != *profile.UrgentCareAvailable {
			facility.UrgentCareAvailable = profile.UrgentCareAvailable
			changed = true
		}
	}
	if profile.WardStatuses != nil {
		facility.WardStatuses = profile.WardStatuses
		changed = true
	}

	return changed
}

// enrichProceduresBatch enriches all procedures that don't have enrichment data yet.
// This runs after ingestion to populate enrichment data once for all procedures.
// Uses a worker pool for concurrent OpenAI calls.
func (s *ProviderIngestionService) enrichProceduresBatch(ctx context.Context) *struct {
	Created int
	Failed  int
} {
	result := &struct {
		Created int
		Failed  int
	}{}

	if s.enrichmentProvider == nil || s.enrichmentRepo == nil {
		log.Println("procedure enrichment provider or repository not configured, skipping enrichment")
		return result
	}

	// Get all procedures
	procedures, err := s.procedureRepo.List(ctx, repositories.ProcedureFilter{})
	if err != nil {
		log.Printf("failed to list procedures for enrichment: %v", err)
		return result
	}

	if len(procedures) == 0 {
		return result
	}

	// Check which procedures need enrichment
	var proceduresToEnrich []*entities.Procedure
	for _, proc := range procedures {
		existing, err := s.enrichmentRepo.GetByProcedureID(ctx, proc.ID)
		if err == nil && existing != nil {
			continue
		}
		proceduresToEnrich = append(proceduresToEnrich, proc)
	}

	if len(proceduresToEnrich) == 0 {
		log.Println("all procedures already enriched, nothing to do")
		return result
	}

	log.Printf("enriching %d procedures with %d workers...", len(proceduresToEnrich), enrichWorkerCount)

	// Worker pool for concurrent enrichment
	var created, failed int64
	var authFailed int32
	var authWarnOnce sync.Once
	type enrichJob struct {
		proc *entities.Procedure
	}

	jobs := make(chan enrichJob, len(proceduresToEnrich))
	var wg sync.WaitGroup

	for w := 0; w < enrichWorkerCount; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobs {
				if ctx.Err() != nil {
					atomic.AddInt64(&failed, 1)
					continue
				}

				// If credentials are rejected, don't keep hammering the provider.
				if atomic.LoadInt32(&authFailed) == 1 {
					atomic.AddInt64(&failed, 1)
					continue
				}
				proc := job.proc
				enriched, err := s.enrichmentProvider.EnrichProcedure(ctx, proc)
				if err != nil {
					if errors.Is(err, providers.ErrProcedureEnrichmentUnauthorized) {
						atomic.StoreInt32(&authFailed, 1)
						authWarnOnce.Do(func() {
							log.Printf("procedure enrichment disabled for this run: provider credentials rejected (%v)", err)
						})
						atomic.AddInt64(&failed, 1)
						continue
					}
					log.Printf("failed to enrich procedure %s (%s): %v", proc.ID, proc.Name, err)
					atomic.AddInt64(&failed, 1)
					continue
				}

				now := time.Now()
				if enriched.ID == "" {
					enriched.ID = fmt.Sprintf("enrich_%s_%d", proc.ID, now.UnixNano())
				}
				if enriched.ProcedureID == "" {
					enriched.ProcedureID = proc.ID
				}
				if enriched.CreatedAt.IsZero() {
					enriched.CreatedAt = now
				}
				if enriched.UpdatedAt.IsZero() {
					enriched.UpdatedAt = now
				}
				if enriched.Description == "" {
					enriched.Description = proc.Description
				}

				if err := s.enrichmentRepo.Upsert(ctx, enriched); err != nil {
					log.Printf("failed to store enrichment for procedure %s: %v", proc.ID, err)
					atomic.AddInt64(&failed, 1)
					continue
				}

				c := atomic.AddInt64(&created, 1)
				if c%50 == 0 {
					log.Printf("enrichment progress: %d/%d done", c, len(proceduresToEnrich))
				}
			}
		}()
	}

	for _, proc := range proceduresToEnrich {
		jobs <- enrichJob{proc: proc}
	}
	close(jobs)
	wg.Wait()

	result.Created = int(atomic.LoadInt64(&created))
	result.Failed = int(atomic.LoadInt64(&failed))
	log.Printf("batch enriched %d procedures (%d failed)", result.Created, result.Failed)
	return result
}

func hashString(value string) string {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(value))
	return fmt.Sprintf("%x", hasher.Sum32())
}
