package plugins

import (
	"testing"

	pg "github.com/bluemedora/bplogagent/plugin"
	"github.com/stretchr/testify/assert"
)

func TestGenerateImplementations(t *testing.T) {
	assert.Implements(t, (*pg.Stopper)(nil), new(GenerateSource))
	assert.Implements(t, (*pg.Outputter)(nil), new(GenerateSource))
	assert.Implements(t, (*pg.Plugin)(nil), new(GenerateSource))
}
