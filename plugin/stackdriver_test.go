package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStackdriverImplementations(t *testing.T) {
	assert.Implements(t, (*Inputter)(nil), new(StackdriverPlugin))
	assert.Implements(t, (*Plugin)(nil), new(StackdriverPlugin))
}
