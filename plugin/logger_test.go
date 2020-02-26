package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoggerImplementations(t *testing.T) {
	assert.Implements(t, (*Plugin)(nil), new(LoggerPlugin))
	assert.Implements(t, (*Inputter)(nil), new(LoggerPlugin))
}
