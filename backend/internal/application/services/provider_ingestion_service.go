package services

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"log"
	"strings"
	"time"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/providerapi"
	apperrors "github.com/zatekoja/Patientpricediscoverydesign/backend/pkg/errors"
)

type ProviderIngestionSummary struct {
	RecordsProcessed          int `json:"records_processed"`
	FacilitiesCreated         int `json:"facilities_created"`
	FacilitiesUpdated         int `json:"facilities_updated"`
	ProceduresCreated         int `json:"procedures_created"`
	ProceduresUpdated         int `json:"procedures_updated"`
	FacilityProceduresCreated int `json:"facility_procedures_created"`
	FacilityProceduresUpdated int `json:"facility_procedures_updated"`
}

// ProviderIngestionService hydrates provider data into core backend storage.
type ProviderIngestionService struct {
	client                providerapi.Client
	facilityRepo          repositories.FacilityRepository
	facilityService       *FacilityService
	procedureRepo         repositories.ProcedureRepository
	facilityProcedureRepo repositories.FacilityProcedureRepository
	geolocationProvider   providers.GeolocationProvider
	geocodeCache          map[string]*providers.GeocodedAddress
	cacheProvider         providers.CacheProvider
	pageSize              int
}

func NewProviderIngestionService(
	client providerapi.Client,
	facilityRepo repositories.FacilityRepository,
	facilityService *FacilityService,
	procedureRepo repositories.ProcedureRepository,
	facilityProcedureRepo repositories.FacilityProcedureRepository,
	geolocationProvider providers.GeolocationProvider,
	cacheProvider providers.CacheProvider,
	pageSize int,
) *ProviderIngestionService {
	if pageSize <= 0 {
		pageSize = 500
	}
	return &ProviderIngestionService{
		client:                client,
		facilityRepo:          facilityRepo,
		facilityService:       facilityService,
		procedureRepo:         procedureRepo,
		facilityProcedureRepo: facilityProcedureRepo,
		geolocationProvider:   geolocationProvider,
		geocodeCache:          map[string]*providers.GeocodedAddress{},
		cacheProvider:         cacheProvider,
		pageSize:              pageSize,
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

	coords, err := s.geolocationProvider.Geocode(ctx, query)
	if err != nil || coords == nil {
		return false
	}

	region := inferRegionFromTags(tags)
	geo := &providers.GeocodedAddress{
		FormattedAddress: query,
		City:             firstNonEmpty(facility.Address.City, region),
		State:            facility.Address.State,
		Country:          firstNonEmpty(facility.Address.Country, "Nigeria"),
		Coordinates: providers.Coordinates{
			Latitude:  coords.Latitude,
			Longitude: coords.Longitude,
		},
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
	if facility.Address.City == "" && geo.City != "" {
		facility.Address.City = geo.City
	}
	if facility.Address.State == "" && geo.State != "" {
		facility.Address.State = geo.State
	}
	if facility.Address.Country == "" && geo.Country != "" {
		facility.Address.Country = geo.Country
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

	region := inferRegionFromTags(tags)
	if region != "" {
		parts = append(parts, region)
	}

	if !containsToken(parts, "Nigeria") {
		parts = append(parts, "Nigeria")
	}

	return strings.TrimSpace(strings.Join(parts, ", "))
}

func inferRegionFromTags(tags []string) string {
	for _, tag := range tags {
		switch strings.ToLower(tag) {
		case "lagos", "lagos_state":
			return "Lagos"
		case "abuja", "fct", "federal_capital_territory":
			return "Abuja"
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
	procedure := &entities.Procedure{
		ID:          buildProcedureID(code),
		Name:        record.ProcedureDescription,
		Code:        code,
		Category:    category,
		Description: description,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.procedureRepo.Create(ctx, procedure); err != nil {
		return nil, false, err
	}

	return procedure, true, nil
}

func (s *ProviderIngestionService) ensureFacilityProcedure(ctx context.Context, facilityID, procedureID string, record providerapi.PriceRecord) (bool, error) {
	existing, err := s.facilityProcedureRepo.GetByFacilityAndProcedure(ctx, facilityID, procedureID)
	if err == nil && existing != nil {
		existing.Price = record.Price
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
		case strings.Contains(value, "imaging") || strings.Contains(value, "scan") || strings.Contains(value, "x-ray"):
			return "imaging"
		case strings.Contains(value, "lab") || strings.Contains(value, "laboratory") || strings.Contains(value, "test"):
			return "laboratory"
		case strings.Contains(value, "surgery") || strings.Contains(value, "operation"):
			return "surgical"
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

	return changed
}

func hashString(value string) string {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(value))
	return fmt.Sprintf("%x", hasher.Sum32())
}
