package bundle

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBundleDefinitionRender(t *testing.T) {
	def := newFakeBundleDefinition()

	params := map[string]interface{}{
		"enabled": true,
		"value":   "testval",
	}

	configReader, err := def.Render(params)
	assert.NoError(t, err)

	config, err := ioutil.ReadAll(configReader)
	assert.NoError(t, err)

	expected := `
plugins:
- id: mygenerator
  type: generate
  count: 3
  record:
    testkey: testval`
	assert.YAMLEq(t, expected, string(config))

}
