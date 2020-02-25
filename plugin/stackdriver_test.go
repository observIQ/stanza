package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStackdriverImplementsInputter(t *testing.T) {
	assert.Implements(t, (*Inputter)(nil), new(StackdriverPlugin))
}

func TestStackdriverImplementsPlugin(t *testing.T) {
	assert.Implements(t, (*Plugin)(nil), new(StackdriverPlugin))
}
