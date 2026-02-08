package services

import (
	"testing"
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
