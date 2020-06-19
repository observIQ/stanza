package plugin

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"text/template"

	"github.com/bluemedora/bplogagent/internal/testutil"
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

func TestCustomRegistryRender(t *testing.T) {
	t.Run("ErrorTypeDoesNotExist", func(t *testing.T) {
		reg := CustomRegistry{}
		_, err := reg.Render("unknown", map[string]interface{}{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not exist")
	})

	t.Run("ErrorExecFailure", func(t *testing.T) {
		tmpl, err := template.New("customtype").Parse(`{{ .panicker }}`)
		require.NoError(t, err)

		reg := CustomRegistry{
			"customtype": tmpl,
		}
		params := map[string]interface{}{
			"panicker": func() {
				panic("testpanic")
			},
		}
		_, err = reg.Render("customtype", params)
		require.Contains(t, err.Error(), "failed to render")
	})
}

func TestCustomRegistryLoad(t *testing.T) {
	t.Run("LoadAllBadGlob", func(t *testing.T) {
		reg := CustomRegistry{}
		err := reg.LoadAll("", `[]`)
		require.Error(t, err)
		require.Contains(t, err.Error(), "with glob pattern")
	})

	t.Run("AddDuplicate", func(t *testing.T) {
		reg := CustomRegistry{}
		var b Builder
		Register("copy", b)
		err := reg.Add("copy", "pipeline:\n")
		require.Error(t, err)
		require.Contains(t, err.Error(), "already exists")
	})

	t.Run("AddBadTemplate", func(t *testing.T) {
		reg := CustomRegistry{}
		err := reg.Add("new", "{{ nofunc }")
		require.Error(t, err)
		require.Contains(t, err.Error(), "as a custom template")
	})

	t.Run("LoadAllWithFailures", func(t *testing.T) {
		tempDir := testutil.NewTempDir(t)
		pluginPath := filepath.Join(tempDir, "copy.yaml")
		err := ioutil.WriteFile(pluginPath, []byte("pipeline:\n"), 0755)
		require.NoError(t, err)

		reg := CustomRegistry{}
		err = reg.LoadAll(tempDir, "*.yaml")
		require.Error(t, err)
	})
}
