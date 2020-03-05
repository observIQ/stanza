package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoopImplementations(t *testing.T) {
	assert.Implements(t, (*Plugin)(nil), new(NoopOutput))
	assert.Implements(t, (*Inputter)(nil), new(NoopOutput))
	assert.Implements(t, (*Outputter)(nil), new(NoopOutput))
}
