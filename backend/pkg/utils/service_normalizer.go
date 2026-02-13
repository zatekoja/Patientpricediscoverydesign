package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

// NormalizationConfig holds medical abbreviation and typo correction data
type NormalizationConfig struct {
	Abbreviations            map[string]AbbreviationEntry `json:"abbreviations"`
	Typos                    map[string]string            `json:"typos"`
	QualifierPatterns        map[string][]string          `json:"qualifierPatterns"`
	CompoundQualifierMapping map[string]string            `json:"compoundQualifierMapping"`
}

// AbbreviationEntry represents a medical abbreviation
type AbbreviationEntry struct {
	Expanded   string   `json:"expanded"`
	Alternates []string `json:"alternates"`
	Category   string   `json:"category"`
	Tags       []string `json:"tags"`
}

// NormalizedServiceName contains all normalized output
type NormalizedServiceName struct {
	DisplayName    string
	NormalizedTags []string
	OriginalName   string
}

// ServiceNameNormalizer handles service name normalization
type ServiceNameNormalizer struct {
	config *NormalizationConfig
}

// NewServiceNameNormalizer creates and initializes a new normalizer
func NewServiceNameNormalizer(configPath string) (*ServiceNameNormalizer, error) {
	configFile, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config NormalizationConfig
	if err := json.Unmarshal(configFile, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &ServiceNameNormalizer{
		config: &config,
	}, nil
}

// Normalize performs all normalization steps on a service name
func (sn *ServiceNameNormalizer) Normalize(originalName string) *NormalizedServiceName {
	if originalName == "" {
		return &NormalizedServiceName{
			DisplayName:    "",
			NormalizedTags: []string{},
			OriginalName:   originalName,
		}
	}

	// Step 1: Correct typos
	corrected := sn.correctTypos(originalName)

	// Step 2: Extract qualifiers and clean the name
	cleanedName, qualifierTags := sn.extractAndStandardizeQualifiers(corrected)

	// Step 3: Expand abbreviations
	displayName := sn.expandAbbreviationsInText(cleanedName)
	displayName = sn.titleCase(strings.TrimSpace(displayName))

	// Step 4: Generate deduplicated tags
	tags := sn.generateDedupTags(originalName, displayName, qualifierTags)

	return &NormalizedServiceName{
		OriginalName:   originalName,
		DisplayName:    displayName,
		NormalizedTags: tags,
	}
}

// correctTypos fixes known spelling errors
func (sn *ServiceNameNormalizer) correctTypos(text string) string {
	result := text
	for typo, correct := range sn.config.Typos {
		re := regexp.MustCompile("(?i)" + regexp.QuoteMeta(typo))
		result = re.ReplaceAllStringFunc(result, func(matched string) string {
			return correct
		})
	}
	return result
}

// extractAndStandardizeQualifiers extracts parenthetical qualifiers and returns clean name + tags
func (sn *ServiceNameNormalizer) extractAndStandardizeQualifiers(text string) (string, []string) {
	var tags []string
	cleaned := text

	// Extract content within parentheses
	parenRe := regexp.MustCompile(`\s*\(([^)]+)\)`)
	matches := parenRe.FindAllStringSubmatch(text, -1)

	for _, match := range matches {
		if len(match) > 1 {
			qualifier := strings.TrimSpace(match[1])
			tag := sn.standardizeQualifier(qualifier)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}

	// Remove parenthetical content from name
	cleaned = parenRe.ReplaceAllString(cleaned, "")
	cleaned = strings.TrimSpace(cleaned)

	return cleaned, tags
}

// standardizeQualifier converts a qualifier to standardized tag format
func (sn *ServiceNameNormalizer) standardizeQualifier(qualifier string) string {
	qualifier = strings.TrimSpace(strings.ToLower(qualifier))

	// Handle compound qualifiers (with/without patterns)
	if strings.Contains(qualifier, "with") && strings.Contains(qualifier, "without") {
		parts := strings.FieldsFunc(qualifier, func(r rune) bool {
			return r == '/' || r == ' '
		})
		if len(parts) >= 2 {
			obj := parts[len(parts)-1]
			tag := fmt.Sprintf("optional_%s", obj)
			if mapped, ok := sn.config.CompoundQualifierMapping[tag]; ok {
				return mapped
			}
			return tag
		}
	}

	// Handle individual with/without patterns
	if strings.HasPrefix(qualifier, "with ") {
		obj := strings.TrimPrefix(qualifier, "with ")
		obj = strings.TrimSpace(obj)
		tag := fmt.Sprintf("includes_%s", strings.ReplaceAll(obj, " ", "_"))
		if mapped, ok := sn.config.CompoundQualifierMapping[tag]; ok {
			return mapped
		}
		return tag
	}

	if strings.HasPrefix(qualifier, "without ") {
		obj := strings.TrimPrefix(qualifier, "without ")
		obj = strings.TrimSpace(obj)
		withTag := fmt.Sprintf("with_%s", strings.ReplaceAll(obj, " ", "_"))
		if mapped, ok := sn.config.CompoundQualifierMapping[withTag]; ok {
			return mapped
		}
		tag := fmt.Sprintf("excludes_%s", strings.ReplaceAll(obj, " ", "_"))
		if mapped, ok := sn.config.CompoundQualifierMapping[tag]; ok {
			return mapped
		}
		return tag
	}

	if strings.HasPrefix(qualifier, "excluding ") {
		obj := strings.TrimPrefix(qualifier, "excluding ")
		obj = strings.TrimSpace(obj)
		tag := fmt.Sprintf("excludes_%s", strings.ReplaceAll(obj, " ", "_"))
		if mapped, ok := sn.config.CompoundQualifierMapping[tag]; ok {
			return mapped
		}
		return tag
	}

	if strings.HasPrefix(qualifier, "including ") {
		obj := strings.TrimPrefix(qualifier, "including ")
		obj = strings.TrimSpace(obj)
		tag := fmt.Sprintf("includes_%s", strings.ReplaceAll(obj, " ", "_"))
		if mapped, ok := sn.config.CompoundQualifierMapping[tag]; ok {
			return mapped
		}
		return tag
	}

	// Normalize known qualifier patterns to tags
	normalized := strings.ReplaceAll(qualifier, " ", "_")
	return normalized
}

// expandAbbreviationsInText expands abbreviations and returns the expanded text
func (sn *ServiceNameNormalizer) expandAbbreviationsInText(text string) string {
	result := text

	// Sort by length (longest first) to match longer abbreviations first
	var abbrevs []string
	for abbr := range sn.config.Abbreviations {
		abbrevs = append(abbrevs, abbr)
	}
	sort.Slice(abbrevs, func(i, j int) bool {
		return len(abbrevs[i]) > len(abbrevs[j])
	})

	// Replace abbreviations in text
	for _, abbr := range abbrevs {
		entry := sn.config.Abbreviations[abbr]

		// Case-insensitive word-boundary matching
		pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(abbr))
		re := regexp.MustCompile("(?i)" + pattern)

		if re.MatchString(result) {
			result = re.ReplaceAllString(result, entry.Expanded)
		}

		// Also check alternates
		for _, alt := range entry.Alternates {
			altPattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(alt))
			altRe := regexp.MustCompile("(?i)" + altPattern)
			if altRe.MatchString(result) {
				result = altRe.ReplaceAllString(result, entry.Expanded)
			}
		}
	}

	return result
}

// titleCase performs title casing on text
func (sn *ServiceNameNormalizer) titleCase(text string) string {
	if text == "" {
		return ""
	}

	words := strings.Fields(strings.ToLower(text))
	result := make([]string, len(words))

	for i, word := range words {
		// Don't capitalize small connecting words
		if i > 0 && (word == "and" || word == "or" || word == "the" || word == "a" || word == "an" || word == "of" || word == "in" || word == "at" || word == "by" || word == "to" || word == "for" || word == "with" || word == "without") {
			result[i] = word
		} else {
			// Capitalize first letter
			if len(word) > 0 {
				result[i] = strings.ToUpper(word[:1]) + word[1:]
			} else {
				result[i] = word
			}
		}
	}

	return strings.Join(result, " ")
}

// generateDedupTags creates deduplicated tags from original and normalized names
func (sn *ServiceNameNormalizer) generateDedupTags(originalName, displayName string, qualifierTags []string) []string {
	tagSet := make(map[string]bool)

	// Add normalized original name as tag
	origTag := NormalizeIdentifier(originalName)
	if origTag != "" {
		tagSet[origTag] = true
	}

	// Add normalized display name (deduplicated expanded form)
	dispTag := NormalizeIdentifier(displayName)
	if dispTag != "" && dispTag != origTag {
		tagSet[dispTag] = true
	}

	// Add qualifier tags
	for _, qualTag := range qualifierTags {
		if qualTag != "" {
			tagSet[qualTag] = true
		}
	}

	// Convert map to sorted slice
	var tags []string
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	return tags
}

// NormalizeIdentifier converts a string to a normalized identifier
func NormalizeIdentifier(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return ""
	}

	var out string
	lastUnderscore := false

	for _, ch := range trimmed {
		isAlphaNum := (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
		if isAlphaNum {
			out += string(ch)
			lastUnderscore = false
		} else if !lastUnderscore {
			out += "_"
			lastUnderscore = true
		}
	}

	// Clean up leading/trailing underscores
	out = strings.Trim(out, "_")
	return out
}

// GetConfigPath returns the default config path
func GetConfigPath() string {
	// Check environment variable first
	if configPath := os.Getenv("MEDICAL_ABBREV_CONFIG"); configPath != "" {
		return configPath
	}

	// Default path
	return "config/medical_abbreviations.json"
}
