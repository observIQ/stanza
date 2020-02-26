package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCopyImplementations(t *testing.T) {
	assert.Implements(t, (*Outputter)(nil), new(CopyPlugin))
	assert.Implements(t, (*Inputter)(nil), new(CopyPlugin))
	assert.Implements(t, (*Plugin)(nil), new(CopyPlugin))
}
