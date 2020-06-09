package plugin

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCustomRegistry_LoadAll(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	test1 := []byte(`
id: my_generator
type: generator
output: testoutput
record:
  message1: {{ .message }}
`)

	test2 := []byte(`
id: my_generator
type: generator
output: testoutput
record:
  message2: {{ .message }}
`)

	err = ioutil.WriteFile(filepath.Join(tempDir, "test1.yaml"), test1, 0666)
	require.NoError(t, err)
	err = ioutil.WriteFile(filepath.Join(tempDir, "test2.yaml"), test2, 0666)
	require.NoError(t, err)

	customRegistry := CustomRegistry{}
	err = customRegistry.LoadAll(tempDir, "*.yaml")
	require.NoError(t, err)

	require.Equal(t, 2, len(customRegistry))
}
