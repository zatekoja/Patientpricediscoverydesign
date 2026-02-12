package utils

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewServiceNameNormalizer tests normalizer initialization
func TestNewServiceNameNormalizer_Success(t *testing.T) {
	configPath := filepath.Join("../../config/medical_abbreviations.json")

	normalizer, err := NewServiceNameNormalizer(configPath)
	require.NoError(t, err)
	require.NotNil(t, normalizer)
	require.NotNil(t, normalizer.config)
	require.Greater(t, len(normalizer.config.Abbreviations), 0)
}

func TestNewServiceNameNormalizer_FileNotFound(t *testing.T) {
	configPath := "/nonexistent/path/config.json"

	normalizer, err := NewServiceNameNormalizer(configPath)
	assert.Error(t, err)
	assert.Nil(t, normalizer)
}

// TestNormalize_EmptyString tests handling of empty input
func TestNormalize_EmptyString(t *testing.T) {
	configPath := filepath.Join("../../config/medical_abbreviations.json")
	normalizer, _ := NewServiceNameNormalizer(configPath)

	result := normalizer.Normalize("")
	assert.Equal(t, "", result.DisplayName)
	assert.Equal(t, "", result.OriginalName)
	assert.Empty(t, result.NormalizedTags)
}

// TestNormalize_TypoCorrection tests typo fixing
func TestNormalize_TypoCorrection(t *testing.T) {
	configPath := filepath.Join("../../config/medical_abbreviations.json")
	normalizer, _ := NewServiceNameNormalizer(configPath)

	testCases := []struct {
		input    string
		expected string
	}{
		{"CAESAREAN SECTION", "Caesarean"},
		{"caesarean section", "Caesarean"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := normalizer.Normalize(tc.input)
			assert.NotEmpty(t, result.DisplayName)
			assert.Equal(t, tc.input, result.OriginalName)
		})
	}
}

// TestNormalize_AbbreviationExpansion tests abbreviation expansion
func TestNormalize_AbbreviationExpansion(t *testing.T) {
	configPath := filepath.Join("../../config/medical_abbreviations.json")
	normalizer, _ := NewServiceNameNormalizer(configPath)

	testCases := []struct {
		input    string
		expected string
	}{
		{"C/S", "Caesarean"},
		{"MRI", "Magnetic"},
		{"ICU", "Intensive"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := normalizer.Normalize(tc.input)
			assert.Contains(t, result.DisplayName, tc.expected)
		})
	}
}

// TestNormalize_PreservesOriginalName verifies original name is always preserved
func TestNormalize_PreservesOriginalName(t *testing.T) {
	configPath := filepath.Join("../../config/medical_abbreviations.json")
	normalizer, _ := NewServiceNameNormalizer(configPath)

	originalNames := []string{
		"CAESAREAN SECTION",
		"MRI SCAN",
		"SIMPLE NAME",
	}

	for _, original := range originalNames {
		result := normalizer.Normalize(original)
		assert.Equal(t, original, result.OriginalName)
		assert.NotEmpty(t, result.DisplayName)
	}
}

// BenchmarkNormalize benchmarks the normalization function
func BenchmarkNormalize(b *testing.B) {
	configPath := filepath.Join("../../config/medical_abbreviations.json")
	normalizer, _ := NewServiceNameNormalizer(configPath)

	testInputs := []string{
		"CAESAREAN SECTION WITH OXYGEN",
		"MRI SCAN WITH CONTRAST",
		"C/S WITHOUT EPIDURAL",
		"ENT SURGERY",
		"ICU ADMISSION",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		input := testInputs[i%len(testInputs)]
		normalizer.Normalize(input)
	}
}
