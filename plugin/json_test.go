package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSONImplementations(t *testing.T) {
	assert.Implements(t, (*Outputter)(nil), new(JSONPlugin))
	assert.Implements(t, (*Inputter)(nil), new(JSONPlugin))
	assert.Implements(t, (*Plugin)(nil), new(JSONPlugin))
}
