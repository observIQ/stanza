package builtin

import (
	"testing"

	"github.com/bluemedora/bplogagent/plugin"
	"github.com/stretchr/testify/assert"
)

func TestNoopImplementations(t *testing.T) {
	assert.Implements(t, (*plugin.Plugin)(nil), new(NoopPlugin))

}
