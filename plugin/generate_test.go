package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateImplementations(t *testing.T) {
	assert.Implements(t, (*Stopper)(nil), new(GeneratePlugin))
	assert.Implements(t, (*Outputter)(nil), new(GeneratePlugin))
	assert.Implements(t, (*Plugin)(nil), new(GeneratePlugin))
}
