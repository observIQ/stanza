package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRateLimitImplementations(t *testing.T) {
	assert.Implements(t, (*Outputter)(nil), new(RateLimitPlugin))
	assert.Implements(t, (*Inputter)(nil), new(RateLimitPlugin))
	assert.Implements(t, (*Plugin)(nil), new(RateLimitPlugin))
}
