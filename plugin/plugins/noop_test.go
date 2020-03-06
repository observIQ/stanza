package plugins

import (
	"testing"

	pg "github.com/bluemedora/bplogagent/plugin"
	"github.com/stretchr/testify/assert"
)

func TestNoopImplementations(t *testing.T) {
	assert.Implements(t, (*pg.Plugin)(nil), new(NoopParser))
	assert.Implements(t, (*pg.Inputter)(nil), new(NoopParser))
	assert.Implements(t, (*pg.Outputter)(nil), new(NoopParser))
}
