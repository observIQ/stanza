package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSONImplementsSimpleProcessor(t *testing.T) {
	assert.Implements(t, (*SimpleProcessor)(nil), new(JSONPlugin))
}
