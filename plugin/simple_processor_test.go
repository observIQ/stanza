package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimpleProcessorAdapterImplementsProcessor(t *testing.T) {
	assert.Implements(t, (*Outputter)(nil), new(SimpleProcessorAdapter))
	assert.Implements(t, (*Inputter)(nil), new(SimpleProcessorAdapter))
}
