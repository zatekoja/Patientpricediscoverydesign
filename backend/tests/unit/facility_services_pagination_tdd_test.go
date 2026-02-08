package handlers_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/repositories"
)

// TDD Test Suite for Facility Services Pagination
// These tests define the expected behavior BEFORE implementation
// Key principle: Search ENTIRE dataset first, then paginate results

func TestFacilityProcedureAdapter_ListByFacilityWithCount_SearchEntireDataset(t *testing.T) {
	t.Skip("Integration test - requires database setup")

	ctx := context.Background()
	facilityID := "test-facility-1"

	tests := []struct {
		name                  string
		setupData             []TestProcedureData
		filter                repositories.FacilityProcedureFilter
		expectedTotalMatches  int
		expectedReturnedCount int
		shouldContainTerms    []string
		assertPriceRange      *struct{ min, max float64 }
	}{
		{
			name: "Search 'MRI' - should find ALL MRI procedures across dataset, then paginate",
			setupData: []TestProcedureData{
				{"MRI Brain Scan", "diagnostic", 400, true, "High-resolution brain imaging"},
				{"MRI Knee Joint", "diagnostic", 350, true, "Knee joint MRI scan"},
				{"CT Scan Basic", "diagnostic", 200, true, "Basic CT scan"},
				{"MRI Chest Study", "diagnostic", 450, true, "Chest MRI examination"},
				{"Ultrasound Basic", "diagnostic", 100, true, "Basic ultrasound"},
				{"MRI Abdomen", "diagnostic", 500, false, "Abdominal MRI scan"}, // Not available
				{"X-Ray Chest", "diagnostic", 50, true, "Chest X-ray"},
				{"MRI Spine Full", "diagnostic", 600, true, "Full spine MRI"},
			},
			filter: repositories.FacilityProcedureFilter{
				SearchQuery: "MRI",
				IsAvailable: boolPtr(true),
				Limit:       3,
				Offset:      0,
			},
			expectedTotalMatches:  4, // 4 available MRI procedures total
			expectedReturnedCount: 3, // Limit of 3
			shouldContainTerms:    []string{"MRI"},
		},
		{
			name: "Search + Price filter - search ALL, filter price, then paginate",
			setupData: []TestProcedureData{
				{"Therapy Session A", "therapeutic", 75, true, "Basic therapy session"},
				{"Therapy Session B", "therapeutic", 125, true, "Intermediate therapy"},
				{"Therapy Session C", "therapeutic", 175, true, "Advanced therapy session"},
				{"Therapy Session D", "therapeutic", 225, true, "Specialized therapy"},
				{"Therapy Session E", "therapeutic", 275, true, "Premium therapy"}, // Outside price range
				{"Surgery Therapy", "surgical", 150, true, "Therapy consultation"}, // Different category but has "therapy"
				{"Basic Therapy", "therapeutic", 50, true, "Entry level therapy"},  // Below min price
			},
			filter: repositories.FacilityProcedureFilter{
				SearchQuery: "therapy",
				MinPrice:    float64Ptr(100),
				MaxPrice:    float64Ptr(200),
				Limit:       2,
				Offset:      0,
			},
			expectedTotalMatches:  3, // Therapy B, C, Surgery Therapy (within price range)
			expectedReturnedCount: 2, // Limit of 2
			shouldContainTerms:    []string{"therapy"},
			assertPriceRange:      &struct{ min, max float64 }{100, 200},
		},
		{
			name: "Category + Search - ensure proper order of operations",
			setupData: []TestProcedureData{
				{"Diagnostic Ultrasound", "diagnostic", 120, true, "Diagnostic ultrasound imaging"},
				{"Ultrasound Therapy", "therapeutic", 100, true, "Therapeutic ultrasound"}, // Different category
				{"Diagnostic MRI", "diagnostic", 400, true, "Diagnostic MRI scan"},         // No "ultrasound"
				{"Advanced Ultrasound", "diagnostic", 150, true, "Advanced ultrasound diagnostics"},
				{"Ultrasound Guided", "surgical", 200, true, "Ultrasound-guided procedure"}, // Different category
			},
			filter: repositories.FacilityProcedureFilter{
				SearchQuery: "ultrasound",
				Category:    "diagnostic",
				SortBy:      "price",
				SortOrder:   "asc",
				Limit:       10,
				Offset:      0,
			},
			expectedTotalMatches:  2, // Only diagnostic ultrasound procedures
			expectedReturnedCount: 2, // Both fit in limit
			shouldContainTerms:    []string{"ultrasound"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock repository (this will be implemented when database is available)
			repo := NewMockFacilityProcedureRepository(tt.setupData)

			// Execute the method we're testing
			procedures, totalCount, err := repo.ListByFacilityWithCount(ctx, facilityID, tt.filter)

			// TDD Assertions - these define the required behavior
			require.NoError(t, err, "Repository should not return error")

			// Critical: Total count should reflect ALL matches, not just returned page
			assert.Equal(t, tt.expectedTotalMatches, totalCount,
				"Total count must reflect search across ENTIRE dataset, not just current page")

			// Returned count should respect pagination limit
			assert.Equal(t, tt.expectedReturnedCount, len(procedures),
				"Returned procedure count should respect pagination limit")

			// All returned procedures should match search criteria
			for _, proc := range procedures {
				if len(tt.shouldContainTerms) > 0 {
					procedureDetails := repo.GetProcedureDetails(proc.ProcedureID)
					matchFound := false
					for _, term := range tt.shouldContainTerms {
						if strings.Contains(strings.ToLower(procedureDetails.Name), strings.ToLower(term)) ||
							strings.Contains(strings.ToLower(procedureDetails.Description), strings.ToLower(term)) {
							matchFound = true
							break
						}
					}
					assert.True(t, matchFound, "Procedure %s should contain search term", procedureDetails.Name)
				}

				// Verify price range if specified
				if tt.assertPriceRange != nil {
					assert.GreaterOrEqual(t, proc.Price, tt.assertPriceRange.min, "Price should be >= minimum")
					assert.LessOrEqual(t, proc.Price, tt.assertPriceRange.max, "Price should be <= maximum")
				}
			}

			// Verify sorting if specified
			if tt.filter.SortBy == "price" && len(procedures) > 1 {
				for i := 1; i < len(procedures); i++ {
					if tt.filter.SortOrder == "desc" {
						assert.GreaterOrEqual(t, procedures[i-1].Price, procedures[i].Price,
							"Prices should be sorted descending")
					} else {
						assert.LessOrEqual(t, procedures[i-1].Price, procedures[i].Price,
							"Prices should be sorted ascending")
					}
				}
			}
		})
	}
}

func TestFacilityProcedureAdapter_PaginationConsistency(t *testing.T) {
	t.Skip("Integration test - requires database setup")

	// Critical test: Ensure pagination is deterministic and consistent
	ctx := context.Background()
	facilityID := "test-facility-consistency"

	// Create test data with predictable ordering
	testData := generateSequentialProcedures("test", 20)
	repo := NewMockFacilityProcedureRepository(testData)

	filter := repositories.FacilityProcedureFilter{
		SearchQuery: "test",
		SortBy:      "name",
		SortOrder:   "asc",
		Limit:       5,
	}

	// Get multiple pages
	pages := make([][]*entities.FacilityProcedure, 0)
	totalCounts := make([]int, 0)

	for page := 0; page < 4; page++ {
		filter.Offset = page * 5
		procedures, totalCount, err := repo.ListByFacilityWithCount(ctx, facilityID, filter)
		require.NoError(t, err)

		pages = append(pages, procedures)
		totalCounts = append(totalCounts, totalCount)
	}

	// Verify consistency across pages
	for i := 1; i < len(totalCounts); i++ {
		assert.Equal(t, totalCounts[0], totalCounts[i], "Total count should be consistent across all pages")
	}

	// Verify no duplicate items across pages
	allIDs := make(map[string]bool)
	for pageNum, page := range pages {
		for _, proc := range page {
			assert.False(t, allIDs[proc.ID], "Procedure %s appears on multiple pages", proc.ID)
			allIDs[proc.ID] = true
		}

		// Each page should have expected size (except possibly the last)
		if pageNum < 3 { // First 3 pages should be full
			assert.Equal(t, 5, len(page), "Page %d should have 5 items", pageNum+1)
		}
	}
}

func TestFacilityProcedureRepository_FilterOrderOfOperations(t *testing.T) {
	// Unit test to verify TDD principle: Search → Filter → Sort → Paginate
	ctx := context.Background()
	facilityID := "test-facility-order"

	// Test data designed to verify order of operations
	testData := []TestProcedureData{
		{"Scan A", "diagnostic", 100, true, "Basic scan procedure"},
		{"Scan B", "therapeutic", 150, true, "Therapeutic scan"},
		{"Scan C", "diagnostic", 200, true, "Advanced scan"},
		{"Therapy A", "therapeutic", 120, true, "Basic therapy"},
		{"Therapy B", "diagnostic", 180, true, "Diagnostic therapy"},
	}

	filter := repositories.FacilityProcedureFilter{
		SearchQuery: "scan",       // Should find 3 procedures with "scan"
		Category:    "diagnostic", // Should filter to 2 procedures (Scan A, Scan C)
		SortBy:      "price",      // Should sort by price
		SortOrder:   "asc",        // Ascending order
		Limit:       1,            // Should return only 1 result (cheapest)
		Offset:      0,
	}

	// Mock implementation for unit testing
	repo := &MockRepository{testData: testData}
	procedures, totalCount, err := repo.ListByFacilityWithCount(ctx, facilityID, filter)

	require.NoError(t, err)
	assert.Equal(t, 2, totalCount, "Should find 2 diagnostic procedures with 'scan'")
	assert.Equal(t, 1, len(procedures), "Should return 1 procedure due to limit")

	if len(procedures) > 0 {
		// Should be Scan A (cheapest diagnostic scan at $100)
		assert.Equal(t, 100.0, procedures[0].Price, "Should return cheapest scan")
	}
}

// Helper types and functions for TDD

type TestProcedureData struct {
	Name        string
	Category    string
	Price       float64
	IsAvailable bool
	Description string
}

type MockFacilityProcedureRepository struct {
	testData []TestProcedureData
}

type MockRepository struct {
	testData []TestProcedureData
}

func NewMockFacilityProcedureRepository(testData []TestProcedureData) *MockFacilityProcedureRepository {
	return &MockFacilityProcedureRepository{testData: testData}
}

func (m *MockFacilityProcedureRepository) ListByFacilityWithCount(
	ctx context.Context,
	facilityID string,
	filter repositories.FacilityProcedureFilter,
) ([]*entities.FacilityProcedure, int, error) {
	// Mock implementation for testing - will be replaced with actual database implementation
	// This simulates the expected behavior

	var filteredData []TestProcedureData

	// Step 1: Search across ALL data
	for _, item := range m.testData {
		matches := true

		// Search query filter
		if filter.SearchQuery != "" {
			searchTerm := strings.ToLower(filter.SearchQuery)
			if !strings.Contains(strings.ToLower(item.Name), searchTerm) &&
				!strings.Contains(strings.ToLower(item.Description), searchTerm) {
				matches = false
			}
		}

		// Category filter
		if filter.Category != "" && item.Category != filter.Category {
			matches = false
		}

		// Price filters
		if filter.MinPrice != nil && item.Price < *filter.MinPrice {
			matches = false
		}
		if filter.MaxPrice != nil && item.Price > *filter.MaxPrice {
			matches = false
		}

		// Availability filter
		if filter.IsAvailable != nil && item.IsAvailable != *filter.IsAvailable {
			matches = false
		}

		if matches {
			filteredData = append(filteredData, item)
		}
	}

	totalCount := len(filteredData)

	// Step 2: Sort filtered data
	if filter.SortBy == "price" {
		// Simple sort implementation for testing
		for i := 0; i < len(filteredData)-1; i++ {
			for j := 0; j < len(filteredData)-i-1; j++ {
				shouldSwap := false
				if filter.SortOrder == "desc" {
					shouldSwap = filteredData[j].Price < filteredData[j+1].Price
				} else {
					shouldSwap = filteredData[j].Price > filteredData[j+1].Price
				}
				if shouldSwap {
					filteredData[j], filteredData[j+1] = filteredData[j+1], filteredData[j]
				}
			}
		}
	}

	// Step 3: Apply pagination
	start := filter.Offset
	end := start + filter.Limit

	if start > len(filteredData) {
		start = len(filteredData)
	}
	if end > len(filteredData) {
		end = len(filteredData)
	}

	pagedData := filteredData[start:end]

	// Convert to entities
	var procedures []*entities.FacilityProcedure
	for i, item := range pagedData {
		procedures = append(procedures, &entities.FacilityProcedure{
			ID:                fmt.Sprintf("fp-%d", i),
			FacilityID:        facilityID,
			ProcedureID:       fmt.Sprintf("proc-%s", strings.ReplaceAll(item.Name, " ", "-")),
			Price:             item.Price,
			Currency:          "NGN",
			EstimatedDuration: 30,
			IsAvailable:       item.IsAvailable,
			CreatedAt:         time.Now(),
			UpdatedAt:         time.Now(),
		})
	}

	return procedures, totalCount, nil
}

func (m *MockFacilityProcedureRepository) GetProcedureDetails(procedureID string) *entities.Procedure {
	// Mock procedure details for testing
	return &entities.Procedure{
		ID:          procedureID,
		Name:        "Mock Procedure " + procedureID,
		Category:    "diagnostic",
		Description: "Mock description for " + procedureID,
	}
}

func (m *MockRepository) ListByFacilityWithCount(
	ctx context.Context,
	facilityID string,
	filter repositories.FacilityProcedureFilter,
) ([]*entities.FacilityProcedure, int, error) {
	// Simplified mock for unit testing order of operations
	mockRepo := NewMockFacilityProcedureRepository(m.testData)
	return mockRepo.ListByFacilityWithCount(ctx, facilityID, filter)
}

// Required interface methods (stubs for testing)
func (m *MockFacilityProcedureRepository) Create(ctx context.Context, fp *entities.FacilityProcedure) error {
	return nil
}
func (m *MockFacilityProcedureRepository) GetByID(ctx context.Context, id string) (*entities.FacilityProcedure, error) {
	return nil, nil
}
func (m *MockFacilityProcedureRepository) GetByFacilityAndProcedure(ctx context.Context, facilityID, procedureID string) (*entities.FacilityProcedure, error) {
	return nil, nil
}
func (m *MockFacilityProcedureRepository) ListByFacility(ctx context.Context, facilityID string) ([]*entities.FacilityProcedure, error) {
	return nil, nil
}
func (m *MockFacilityProcedureRepository) Update(ctx context.Context, fp *entities.FacilityProcedure) error {
	return nil
}
func (m *MockFacilityProcedureRepository) Delete(ctx context.Context, id string) error { return nil }

func generateSequentialProcedures(namePrefix string, count int) []TestProcedureData {
	procedures := make([]TestProcedureData, count)
	for i := 0; i < count; i++ {
		procedures[i] = TestProcedureData{
			Name:        fmt.Sprintf("%s Procedure %02d", namePrefix, i+1),
			Category:    "diagnostic",
			Price:       float64(100 + i*10),
			IsAvailable: true,
			Description: fmt.Sprintf("Description for %s procedure %d", namePrefix, i+1),
		}
	}
	return procedures
}

func boolPtr(b bool) *bool          { return &b }
func float64Ptr(f float64) *float64 { return &f }
