package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimpleProcessorAdapterImplementsProcessor(t *testing.T) {
	assert.Implements(t, (*Processor)(nil), new(SimpleProcessorAdapter))
}
