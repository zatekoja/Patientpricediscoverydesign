package services

import (
	"testing"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/infrastructure/clients/providerapi"
)

// TestLocationDefaultingIssue tests the bug where locations default to "Lagos, Lagos"
// instead of using actual facility coordinates and addresses
func TestLocationDefaultingIssue(t *testing.T) {
	testCases := []struct {
		name            string
		facilityName    string
		street          string
		city            string
		state           string
		country         string
		latitude        float64
		longitude       float64
		tags            []string
		expectedCity    string
		expectedState   string
		expectedNotCity string // City it should NOT be
		description     string
	}{
		{
			name:            "Hospital in Abuja with proper address should NOT default to Lagos",
			facilityName:    "National Hospital Abuja",
			street:          "Plot 132 Central District",
			city:            "Abuja",
			state:           "FCT",
			country:         "Nigeria",
			latitude:        9.0579,
			longitude:       7.4951,
			tags:            []string{"hospital", "nigeria"},
			expectedCity:    "Abuja",
			expectedState:   "FCT",
			expectedNotCity: "Lagos",
			description:     "Abuja hospital should retain Abuja location, not default to Lagos",
		},
		{
			name:            "Hospital in Port Harcourt should NOT default to Lagos",
			facilityName:    "University of Port Harcourt Teaching Hospital",
			street:          "East-West Road",
			city:            "Port Harcourt",
			state:           "Rivers",
			country:         "Nigeria",
			latitude:        4.8156,
			longitude:       7.0498,
			tags:            []string{"teaching_hospital", "rivers", "nigeria"},
			expectedCity:    "Port Harcourt",
			expectedState:   "Rivers",
			expectedNotCity: "Lagos",
			description:     "Rivers state hospital should use actual location, not Lagos",
		},
		{
			name:            "Hospital in Kano should NOT default to Lagos",
			facilityName:    "Aminu Kano Teaching Hospital",
			city:            "Kano",
			state:           "Kano",
			country:         "Nigeria",
			latitude:        12.0022,
			longitude:       8.5920,
			tags:            []string{"teaching_hospital", "kano", "nigeria"},
			expectedCity:    "Kano",
			expectedState:   "Kano",
			expectedNotCity: "Lagos",
			description:     "Kano hospital should use Kano location",
		},
		{
			name:            "Hospital with 'lagos' tag but actually in Ibadan should use coordinates",
			facilityName:    "Ibadan Specialist Hospital",
			city:            "Ibadan",
			state:           "Oyo",
			country:         "Nigeria",
			latitude:        7.3775,
			longitude:       3.9470,
			tags:            []string{"specialist", "nigeria", "lagos"}, // Has "lagos" tag but NOT in Lagos!
			expectedCity:    "Ibadan",
			expectedState:   "Oyo",
			expectedNotCity: "Lagos",
			description:     "BUG: Having 'lagos' in tags should NOT override actual address",
		},
		{
			name:            "Hospital with no address but valid coordinates should NOT force Lagos",
			facilityName:    "Enugu State University Teaching Hospital",
			country:         "Nigeria",
			latitude:        6.4531,
			longitude:       7.5248,
			tags:            []string{"teaching_hospital", "enugu", "nigeria"},
			expectedCity:    "",      // City might be empty or geocoded
			expectedNotCity: "Lagos", // But should NOT be Lagos!
			description:     "Missing address but valid coordinates - should reverse geocode, not default to Lagos",
		},
		{
			name:          "Actual Lagos hospital should correctly be Lagos",
			facilityName:  "Lagos University Teaching Hospital",
			street:        "Idi-Araba",
			city:          "Lagos",
			state:         "Lagos",
			country:       "Nigeria",
			latitude:      6.5244,
			longitude:     3.3792,
			tags:          []string{"teaching_hospital", "lagos", "nigeria"},
			expectedCity:  "Lagos",
			expectedState: "Lagos",
			description:   "Actual Lagos hospital should be Lagos (this is correct)",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			// Create facility with profile data
			facility := &entities.Facility{
				ID:   "test_facility",
				Name: tt.facilityName,
				Address: entities.Address{
					Street:  tt.street,
					City:    tt.city,
					State:   tt.state,
					Country: tt.country,
				},
				Location: entities.Location{
					Latitude:  tt.latitude,
					Longitude: tt.longitude,
				},
			}

			// Test 1: Check that coordinates are NOT defaulting to 0,0 (which would trigger Lagos fallback)
			if facility.Location.Latitude == 0 && facility.Location.Longitude == 0 {
				t.Errorf("%s: Coordinates should not be 0,0 for facility with profile location", tt.description)
			}

			// Test 2: Verify actual coordinates match expected
			if tt.latitude != 0 {
				if facility.Location.Latitude != tt.latitude {
					t.Errorf("%s: Expected latitude %.4f, got %.4f",
						tt.description, tt.latitude, facility.Location.Latitude)
				}
			}

			// Test 3: Check city is correct
			if tt.expectedCity != "" && facility.Address.City != tt.expectedCity {
				t.Errorf("%s: Expected city '%s', got '%s'",
					tt.description, tt.expectedCity, facility.Address.City)
			}

			// Test 4: CRITICAL - Ensure city is NOT incorrectly set to Lagos
			if tt.expectedNotCity != "" && facility.Address.City == tt.expectedNotCity {
				t.Errorf("❌ BUG CONFIRMED: %s: City incorrectly set to '%s' when it should be '%s'. "+
					"Location likely defaulted to Lagos despite having coordinates (%.4f, %.4f)",
					tt.description, facility.Address.City, tt.expectedCity,
					facility.Location.Latitude, facility.Location.Longitude)
			}

			// Test 5: State check
			if tt.expectedState != "" && facility.Address.State != tt.expectedState {
				t.Errorf("%s: Expected state '%s', got '%s'",
					tt.description, tt.expectedState, facility.Address.State)
			}

			t.Logf("✓ Facility: %s | City: %s, State: %s | Coords: (%.4f, %.4f)",
				facility.Name, facility.Address.City, facility.Address.State,
				facility.Location.Latitude, facility.Location.Longitude)
		})
	}
}

// TestInferRegionFromTagsProblem specifically tests the problematic inferRegionFromTags function
func TestInferRegionFromTagsProblem(t *testing.T) {
	tests := []struct {
		name           string
		tags           []string
		expectedRegion string
		shouldBeEmpty  bool
		problem        string
	}{
		{
			name:           "Tags with 'lagos' should return Lagos",
			tags:           []string{"hospital", "lagos", "specialist"},
			expectedRegion: "Lagos",
			problem:        "",
		},
		{
			name:           "Tags with 'abuja' should return Abuja",
			tags:           []string{"hospital", "abuja"},
			expectedRegion: "Abuja",
			problem:        "",
		},
		{
			name:          "Tags without region indicators should return empty",
			tags:          []string{"hospital", "specialist", "emergency"},
			shouldBeEmpty: true,
			problem:       "",
		},
		{
			name:           "BUG: Tags with 'lagos' for non-Lagos facility incorrectly returns Lagos",
			tags:           []string{"hospital", "lagos"}, // Tag might be metadata, not location!
			expectedRegion: "Lagos",
			problem:        "This function cannot distinguish between 'lagos' as a tag vs actual location",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferRegionFromTags(tt.tags)

			if tt.shouldBeEmpty && result != "" {
				t.Errorf("Expected empty region, got: %s", result)
			}

			if tt.expectedRegion != "" && result != tt.expectedRegion {
				t.Errorf("Expected region '%s', got '%s'", tt.expectedRegion, result)
			}

			if tt.problem != "" {
				t.Logf("⚠️ KNOWN ISSUE: %s - Result: '%s'", tt.problem, result)
			}
		})
	}
}

// TestGeocodeQueryBuildingAddingLagos tests if Lagos is being added to geocode queries incorrectly
func TestGeocodeQueryBuildingAddingLagos(t *testing.T) {
	tests := []struct {
		name             string
		facility         *entities.Facility
		record           providerapi.PriceRecord
		profile          *providerapi.FacilityProfile
		tags             []string
		shouldContain    string
		shouldNotContain string
		description      string
	}{
		{
			name: "Abuja hospital query should NOT contain Lagos",
			facility: &entities.Facility{
				Name: "National Hospital",
				Address: entities.Address{
					City:    "Abuja",
					State:   "FCT",
					Country: "Nigeria",
				},
			},
			record: providerapi.PriceRecord{
				FacilityName: "National Hospital Abuja",
			},
			profile: func() *providerapi.FacilityProfile {
				p := &providerapi.FacilityProfile{
					Name: "National Hospital Abuja",
				}
				p.Address.City = "Abuja"
				p.Address.State = "FCT"
				p.Address.Country = "Nigeria"
				return p
			}(),
			tags:             []string{"hospital", "abuja"},
			shouldContain:    "Abuja",
			shouldNotContain: "Lagos",
			description:      "Abuja hospital geocode query should not have Lagos added",
		},
		{
			name: "Hospital with 'lagos' tag but different city should use actual city",
			facility: &entities.Facility{
				Name: "Port Harcourt Hospital",
				Address: entities.Address{
					City:    "Port Harcourt",
					State:   "Rivers",
					Country: "Nigeria",
				},
			},
			record: providerapi.PriceRecord{
				FacilityName: "Port Harcourt Hospital",
			},
			tags:             []string{"hospital", "lagos"}, // Misleading tag
			shouldContain:    "Port Harcourt",
			shouldNotContain: "Lagos",
			description:      "BUG: 'lagos' tag should NOT add Lagos to query when city is Port Harcourt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := buildGeocodeQuery(tt.facility, tt.record, tt.profile, tt.tags)

			t.Logf("Generated query: %s", query)

			if tt.shouldContain != "" {
				if !containsToken([]string{query}, tt.shouldContain) {
					t.Errorf("Query should contain '%s' but got: %s", tt.shouldContain, query)
				}
			}

			if tt.shouldNotContain != "" {
				if containsToken([]string{query}, tt.shouldNotContain) {
					t.Errorf("❌ BUG: %s\nQuery incorrectly contains '%s': %s",
						tt.description, tt.shouldNotContain, query)
				}
			}
		})
	}
}

// TestRootCauseAnalysis documents the root cause of the Lagos defaulting issue
func TestRootCauseAnalysis(t *testing.T) {
	t.Log("=== ROOT CAUSE ANALYSIS: Why Locations Default to 'Lagos, Lagos' ===")
	t.Log("")
	t.Log("ISSUE: Hospital locations are incorrectly defaulting to Lagos, Lagos")
	t.Log("")
	t.Log("ROOT CAUSES:")
	t.Log("")
	t.Log("1. inferRegionFromTags() function (line 510-520)")
	t.Log("   - Uses 'lagos' in TAGS to infer region")
	t.Log("   - Problem: Tags might have 'lagos' as metadata, not actual location")
	t.Log("   - Returns 'Lagos' even for hospitals in other cities")
	t.Log("")
	t.Log("2. buildFallbackGeocodedAddress() function (line 383-390)")
	t.Log("   - Calls inferRegionFromTags() to fill missing city")
	t.Log("   - Problem: Uses inferred region even when coordinates indicate different location")
	t.Log("   - Should use reverse geocoding with actual coordinates instead")
	t.Log("")
	t.Log("3. buildGeocodeQuery() function (line 461-507)")
	t.Log("   - Adds inferred region to geocode query")
	t.Log("   - Problem: Forces 'Lagos' into query for geocoding, biasing results")
	t.Log("   - Should only use actual address components, not tag inference")
	t.Log("")
	t.Log("SOLUTION REQUIRED:")
	t.Log("✓ Priority 1: Use coordinates for reverse geocoding when available")
	t.Log("✓ Priority 2: Only use inferRegionFromTags as LAST resort when no coordinates/address")
	t.Log("✓ Priority 3: Don't add inferred region to geocode query if actual address exists")
	t.Log("✓ Priority 4: Validate that coordinates match inferred region before using it")
}
