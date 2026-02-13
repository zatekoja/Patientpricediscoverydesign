package services

import (
	"os"
)

type FeatureFlags struct {
	contextualSearchEnabled bool
	shadowModeEnabled       bool
}

func NewFeatureFlags() *FeatureFlags {
	enabled := os.Getenv("FEATURE_CONTEXTUAL_SEARCH") == "true"
	shadow := os.Getenv("FEATURE_CONTEXTUAL_SEARCH_SHADOW") == "true"

	return &FeatureFlags{
		contextualSearchEnabled: enabled,
		shadowModeEnabled:       shadow,
	}
}

func (f *FeatureFlags) ContextualSearchEnabled() bool {
	return f.contextualSearchEnabled
}

func (f *FeatureFlags) ShadowModeEnabled() bool {
	return f.shadowModeEnabled
}
