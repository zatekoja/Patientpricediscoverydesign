package search

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zatekoja/Patientpricediscoverydesign/backend/internal/domain/entities"
)

func TestBuildFacilityTags(t *testing.T) {
	facility := &entities.Facility{
		Name:         " General Hospital ",
		FacilityType: "Teaching Hospital",
		Address: entities.Address{
			City:    "Ikeja",
			State:   "Lagos",
			Country: "Nigeria",
		},
		AcceptedInsurance: []string{"NHIS", "Reliance HMO", "NHIS"},
	}

	tags := buildFacilityTags(facility)

	assert.ElementsMatch(t, []string{
		"general hospital",
		"teaching hospital",
		"ikeja",
		"lagos",
		"nigeria",
		"nhis",
		"reliance hmo",
	}, tags)
}

func TestBuildFacilityTagsNil(t *testing.T) {
	assert.Nil(t, buildFacilityTags(nil))
}
