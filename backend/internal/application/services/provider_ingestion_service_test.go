package services

import (
	"testing"

	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/providers"
)

// TestCalculateAveragePrice tests the price averaging logic
func TestCalculateAveragePrice(t *testing.T) {
	tests := []struct {
		name          string
		existingPrice float64
		newPrice      float64
		expectedPrice float64
		description   string
	}{
		{
			name:          "both prices are zero",
			existingPrice: 0,
			newPrice:      0,
			expectedPrice: 0,
			description:   "When both prices are zero, result should be zero",
		},
		{
			name:          "existing price is zero",
			existingPrice: 0,
			newPrice:      100,
			expectedPrice: 100,
			description:   "When existing price is zero, use new price",
		},
		{
			name:          "new price is zero",
			existingPrice: 100,
			newPrice:      0,
			expectedPrice: 100,
			description:   "When new price is zero, use existing price",
		},
		{
			name:          "same prices",
			existingPrice: 100,
			newPrice:      100,
			expectedPrice: 100,
			description:   "When both prices are the same, average should equal the price",
		},
		{
			name:          "different prices - lower first",
			existingPrice: 100,
			newPrice:      200,
			expectedPrice: 150,
			description:   "Average of 100 and 200 should be 150",
		},
		{
			name:          "different prices - higher first",
			existingPrice: 200,
			newPrice:      100,
			expectedPrice: 150,
			description:   "Average of 200 and 100 should be 150",
		},
		{
			name:          "decimal prices",
			existingPrice: 99.99,
			newPrice:      100.01,
			expectedPrice: 100.00,
			description:   "Average of 99.99 and 100.01 should be 100.00",
		},
		{
			name:          "small difference",
			existingPrice: 1000,
			newPrice:      1001,
			expectedPrice: 1000.5,
			description:   "Average of 1000 and 1001 should be 1000.5",
		},
		{
			name:          "large difference",
			existingPrice: 50,
			newPrice:      5000,
			expectedPrice: 2525,
			description:   "Average of 50 and 5000 should be 2525",
		},
		{
			name:          "three decimal places",
			existingPrice: 123.456,
			newPrice:      654.321,
			expectedPrice: 388.8885,
			description:   "Average of 123.456 and 654.321 should be 388.8885",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateAveragePrice(tt.existingPrice, tt.newPrice)
			if result != tt.expectedPrice {
				t.Errorf("%s: got %.4f, want %.4f", tt.description, result, tt.expectedPrice)
			}
		})
	}
}

// TestCalculateAveragePriceMultipleProviders tests averaging across multiple providers
func TestCalculateAveragePriceMultipleProviders(t *testing.T) {
	tests := []struct {
		name     string
		prices   []float64
		expected float64
	}{
		{
			name:     "two providers",
			prices:   []float64{100, 150},
			expected: 125,
		},
		{
			name:     "three providers",
			prices:   []float64{100, 150, 200},
			expected: 162.5, // (100+150)/2 = 125, then (125+200)/2 = 162.5 (sequential averaging)
		},
		{
			name:     "four providers",
			prices:   []float64{80, 100, 120, 140},
			expected: 122.5, // (80+100)/2=90, (90+120)/2=105, (105+140)/2=122.5 (sequential averaging)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if len(tt.prices) < 2 {
				t.Skip("Need at least 2 prices")
			}
			// Simulate sequential averaging from multiple providers
			result := tt.prices[0]
			for i := 1; i < len(tt.prices); i++ {
				result = calculateAveragePrice(result, tt.prices[i])
			}
			if result != tt.expected {
				t.Errorf("averaging %v: got %.2f, want %.2f", tt.prices, result, tt.expected)
			}
		})
	}
}

// TestCalculateAveragePriceEdgeCases tests edge cases for price averaging
func TestCalculateAveragePriceEdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		existingPrice float64
		newPrice      float64
		shouldNotZero bool
	}{
		{
			name:          "very small prices",
			existingPrice: 0.01,
			newPrice:      0.02,
			shouldNotZero: true,
		},
		{
			name:          "very large prices",
			existingPrice: 999999.99,
			newPrice:      999999.99,
			shouldNotZero: true,
		},
		{
			name:          "negative existing (should not happen but test robustness)",
			existingPrice: -100,
			newPrice:      100,
			shouldNotZero: false, // result would be 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateAveragePrice(tt.existingPrice, tt.newPrice)
			if tt.shouldNotZero && result == 0 {
				t.Errorf("expected non-zero result, got 0")
			}
		})
	}
}

// BenchmarkCalculateAveragePrice benchmarks the averaging function
func BenchmarkCalculateAveragePrice(b *testing.B) {
	for i := 0; i < b.N; i++ {
		calculateAveragePrice(100.0, 200.0)
	}
}

func TestApplyGeocodedAddress(t *testing.T) {
	tests := []struct {
		name            string
		facility        *entities.Facility
		geo             *providers.GeocodedAddress
		expectedStreet  string
		expectedCity    string
		expectedState   string
		expectedCountry string
		expectedZipCode string
		expectedLat     float64
		expectedLon     float64
	}{
		{
			name: "should populate all address fields from geocoded data",
			facility: &entities.Facility{
				ID:   "test-facility",
				Name: "Test Hospital",
				Address: entities.Address{
					Street:  "",
					City:    "",
					State:   "",
					Country: "",
					ZipCode: "",
				},
			},
			geo: &providers.GeocodedAddress{
				FormattedAddress: "Hospital Rd, Badagry 103101, Lagos, Nigeria",
				Street:           "Hospital Rd",
				City:             "Badagry",
				State:            "Lagos",
				ZipCode:          "103101",
				Country:          "Nigeria",
				Coordinates: providers.Coordinates{
					Latitude:  6.413688,
					Longitude: 2.8972871,
				},
			},
			expectedStreet:  "Hospital Rd",
			expectedCity:    "Badagry",
			expectedState:   "Lagos",
			expectedCountry: "Nigeria",
			expectedZipCode: "103101",
			expectedLat:     6.413688,
			expectedLon:     2.8972871,
		},
		{
			name: "should not override existing street address if present",
			facility: &entities.Facility{
				ID:   "test-facility",
				Name: "Test Hospital",
				Address: entities.Address{
					Street:  "123 Main St",
					City:    "",
					State:   "",
					Country: "",
				},
			},
			geo: &providers.GeocodedAddress{
				FormattedAddress: "Hospital Rd, Badagry 103101, Lagos, Nigeria",
				Street:           "Hospital Rd",
				City:             "Badagry",
				State:            "Lagos",
				Country:          "Nigeria",
				Coordinates: providers.Coordinates{
					Latitude:  6.413688,
					Longitude: 2.8972871,
				},
			},
			expectedStreet:  "123 Main St",
			expectedCity:    "Badagry",
			expectedState:   "Lagos",
			expectedCountry: "Nigeria",
			expectedLat:     6.413688,
			expectedLon:     2.8972871,
		},
		{
			name: "should use formatted address as street if no street component",
			facility: &entities.Facility{
				ID:   "test-facility",
				Name: "Test Hospital",
				Address: entities.Address{
					Street: "",
				},
			},
			geo: &providers.GeocodedAddress{
				FormattedAddress: "Ishaga Rd, Idi-Araba, Lagos 102215, Lagos, Nigeria",
				Street:           "", // No specific street component
				City:             "Lagos",
				State:            "Lagos",
				Country:          "Nigeria",
				Coordinates: providers.Coordinates{
					Latitude:  6.5176223,
					Longitude: 3.3537453,
				},
			},
			expectedStreet:  "Ishaga Rd, Idi-Araba, Lagos 102215",
			expectedCity:    "Lagos",
			expectedState:   "Lagos",
			expectedCountry: "Nigeria",
			expectedLat:     6.5176223,
			expectedLon:     3.3537453,
		},
		{
			name: "should update zip code from geocoded data",
			facility: &entities.Facility{
				ID:   "test-facility",
				Name: "Test Hospital",
				Address: entities.Address{
					Street:  "",
					City:    "Lagos",
					ZipCode: "",
				},
			},
			geo: &providers.GeocodedAddress{
				FormattedAddress: "Some Address, Lagos 102215, Nigeria",
				Street:           "Some Address",
				City:             "Lagos",
				State:            "Lagos",
				ZipCode:          "102215",
				Country:          "Nigeria",
				Coordinates: providers.Coordinates{
					Latitude:  6.5,
					Longitude: 3.3,
				},
			},
			expectedStreet:  "Some Address",
			expectedCity:    "Lagos",
			expectedState:   "Lagos",
			expectedCountry: "Nigeria",
			expectedZipCode: "102215",
			expectedLat:     6.5,
			expectedLon:     3.3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applyGeocodedAddress(tt.facility, tt.geo)

			if tt.facility.Address.Street != tt.expectedStreet {
				t.Errorf("Street: expected %q, got %q", tt.expectedStreet, tt.facility.Address.Street)
			}
			if tt.facility.Address.City != tt.expectedCity {
				t.Errorf("City: expected %q, got %q", tt.expectedCity, tt.facility.Address.City)
			}
			if tt.facility.Address.State != tt.expectedState {
				t.Errorf("State: expected %q, got %q", tt.expectedState, tt.facility.Address.State)
			}
			if tt.facility.Address.Country != tt.expectedCountry {
				t.Errorf("Country: expected %q, got %q", tt.expectedCountry, tt.facility.Address.Country)
			}
			if tt.facility.Address.ZipCode != tt.expectedZipCode {
				t.Errorf("ZipCode: expected %q, got %q", tt.expectedZipCode, tt.facility.Address.ZipCode)
			}
			if tt.facility.Location.Latitude != tt.expectedLat {
				t.Errorf("Latitude: expected %f, got %f", tt.expectedLat, tt.facility.Location.Latitude)
			}
			if tt.facility.Location.Longitude != tt.expectedLon {
				t.Errorf("Longitude: expected %f, got %f", tt.expectedLon, tt.facility.Location.Longitude)
			}
		})
	}
}

func TestInferProcedureCategory(t *testing.T) {
	tests := []struct {
		name        string
		description string
		tags        []string
		expected    string
	}{
		{name: "imaging from description", description: "CT Scan of the chest", tags: nil, expected: "imaging"},
		{name: "imaging from ultrasound", description: "Ultrasound examination", tags: nil, expected: "imaging"},
		{name: "imaging from MRI tag", description: "Brain study", tags: []string{"mri"}, expected: "imaging"},
		{name: "laboratory from description", description: "Blood laboratory test", tags: nil, expected: "laboratory"},
		{name: "surgical from operation", description: "Major operation theatre", tags: nil, expected: "surgical"},
		{name: "surgical from laparoscopy", description: "Laparoscopic cholecystectomy", tags: nil, expected: "surgical"},
		{name: "dental from description", description: "Dental extraction", tags: nil, expected: "dental"},
		{name: "dental from orthodontic", description: "Orthodontic appliance", tags: nil, expected: "dental"},
		{name: "ophthalmology from cataract", description: "Cataract surgery", tags: nil, expected: "ophthalmology"},
		{name: "ophthalmology from glaucoma", description: "Glaucoma screening", tags: nil, expected: "ophthalmology"},
		{name: "ent from tonsil", description: "Tonsillectomy", tags: nil, expected: "ent"},
		{name: "ent from tag", description: "Ear procedure", tags: []string{"ent"}, expected: "ent"},
		{name: "psychiatry from description", description: "Psychiatric evaluation", tags: nil, expected: "psychiatry"},
		{name: "psychiatry from electroconvulsive", description: "Electroconvulsive therapy", tags: nil, expected: "psychiatry"},
		{name: "dermatology from description", description: "Dermatology consultation", tags: nil, expected: "dermatology"},
		{name: "dermatology from skin biopsy", description: "Skin biopsy procedure", tags: nil, expected: "dermatology"},
		{name: "physiotherapy from description", description: "Physiotherapy session", tags: nil, expected: "physiotherapy"},
		{name: "dietary from description", description: "Dietary consultation", tags: nil, expected: "dietary"},
		{name: "dietary from nutrition", description: "Nutritional assessment", tags: []string{"nutrition"}, expected: "dietary"},
		{name: "endoscopy from description", description: "Upper GI Endoscopy", tags: nil, expected: "endoscopy"},
		{name: "endoscopy from colonoscopy", description: "Colonoscopy screening", tags: nil, expected: "endoscopy"},
		{name: "urology from description", description: "Urology consultation", tags: nil, expected: "urology"},
		{name: "urology from catheter", description: "Change of catheter", tags: nil, expected: "urology"},
		{name: "oncology from chemo", description: "Chemo admission fee", tags: nil, expected: "oncology"},
		{name: "oncology from radiotherapy", description: "Radiotherapy session", tags: nil, expected: "oncology"},
		{name: "orthopaedics from fracture", description: "Fracture management", tags: nil, expected: "orthopaedics"},
		{name: "orthopaedics from tag", description: "POP application", tags: []string{"orthopaedics"}, expected: "orthopaedics"},
		{name: "neurology from stroke", description: "Stroke ward admission", tags: nil, expected: "neurology"},
		{name: "neurology from eeg", description: "EEG recording", tags: nil, expected: "neurology"},
		{name: "accommodation from ward", description: "General ward stay", tags: nil, expected: "accommodation"},
		{name: "emergency from ambulance", description: "Ambulance transport", tags: nil, expected: "emergency"},
		{name: "administrative from certificate", description: "Death certificate", tags: nil, expected: "administrative"},
		{name: "therapeutic from therapy", description: "Oxygen therapy", tags: nil, expected: "therapeutic"},
		{name: "preventive from checkup", description: "Annual checkup", tags: nil, expected: "preventive"},
		{name: "empty for unknown", description: "Miscellaneous item", tags: nil, expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inferProcedureCategory(tt.description, tt.tags)
			if result != tt.expected {
				t.Errorf("inferProcedureCategory(%q, %v) = %q, want %q", tt.description, tt.tags, result, tt.expected)
			}
		})
	}
}

func TestApplyGeocodedAddress_NilInputs(t *testing.T) {
	t.Run("should handle nil facility gracefully", func(t *testing.T) {
		geo := &providers.GeocodedAddress{
			Street: "Test St",
		}
		// Should not panic
		applyGeocodedAddress(nil, geo)
	})

	t.Run("should handle nil geocoded address gracefully", func(t *testing.T) {
		facility := &entities.Facility{
			ID:   "test",
			Name: "Test",
		}
		// Should not panic
		applyGeocodedAddress(facility, nil)
	})
}
