package builtin

import (
	"testing"

	"github.com/bluemedora/bplogagent/plugin"
	"github.com/stretchr/testify/assert"
)

func TestNoopImplementations(t *testing.T) {
	assert.Implements(t, (*plugin.Plugin)(nil), new(NoopFilter))
	assert.Implements(t, (*plugin.Producer)(nil), new(NoopFilter))
	assert.Implements(t, (*plugin.Consumer)(nil), new(NoopFilter))
}
