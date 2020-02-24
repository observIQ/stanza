package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateImplementsSource(t *testing.T) {
	assert.Implements(t, (*Source)(nil), new(GeneratePlugin))
}
