package evaluation

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGuardrails_RejectLowConfidence(t *testing.T) {
	config := GuardrailConfig{
		MinIntentConfidence: 0.6,
	}
	g := NewGuardrails(config)

	assert.False(t, g.ShouldProcess(0.5))
	assert.True(t, g.ShouldProcess(0.6))
	assert.True(t, g.ShouldProcess(0.9))
}

func TestGuardrails_LimitExpansion(t *testing.T) {
	config := GuardrailConfig{
		MaxExpansionTerms: 3,
	}
	g := NewGuardrails(config)

	terms := []string{"a", "b", "c", "d", "e"}
	limited := g.LimitExpansion(terms)

	assert.Equal(t, 3, len(limited))
	assert.Equal(t, []string{"a", "b", "c"}, limited)
}
