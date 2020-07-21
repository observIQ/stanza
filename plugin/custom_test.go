package plugin

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"
)

func NewTempDir(t *testing.T) string {
	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	return tempDir
}

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
		Register("copy", func() Builder { return nil })
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
		tempDir := NewTempDir(t)
		pluginPath := filepath.Join(tempDir, "copy.yaml")
		err := ioutil.WriteFile(pluginPath, []byte("pipeline:\n"), 0755)
		require.NoError(t, err)

		reg := CustomRegistry{}
		err = reg.LoadAll(tempDir, "*.yaml")
		require.Error(t, err)
	})
}

func TestCustomPluginMetadata(t *testing.T) {

	testCases := []struct {
		name      string
		expectErr bool
		template  string
	}{
		{
			name:      "no_meta",
			expectErr: false,
			template: `pipeline:
`,
		},
		{
			name:      "full_meta",
			expectErr: false,
			template: `version: 0.0.0
title: My Super Plugin
description: This is the best plugin ever
parameters:
  path:
    label: Path
    description: The path to a thing
    type: string
  other:
    label: Other Thing
    description: Another parameter
    type: integer
pipeline:
`,
		},
		{
			name:      "bad_version",
			expectErr: true,
			template: `version: []
title: My Super Plugin
description: This is the best plugin ever
parameters:
  path:
    label: Path
    description: The path to a thing
    type: string
  other:
    label: Other Thing
    description: Another parameter
    type: integer
pipeline:
`,
		},
		{
			name:      "bad_title",
			expectErr: true,
			template: `version: 0.0.0
title: []
description: This is the best plugin ever
parameters:
  path:
    label: Path
    description: The path to a thing
    type: string
  other:
    label: Other Thing
    description: Another parameter
    type: integer
pipeline:
`,
		},
		{
			name:      "bad_description",
			expectErr: true,
			template: `version: 0.0.0
title: My Super Plugin
description: []
parameters:
  path:
    label: Path
    description: The path to a thing
    type: string
  other:
    label: Other Thing
    description: Another parameter
    type: integer
pipeline:
`,
		},
		{
			name:      "bad_parameters_type",
			expectErr: true,
			template: `version: 0.0.0
title: My Super Plugin
description: This is the best plugin ever
parameters: hello
`,
		},
		{
			name:      "bad_parameter_structure",
			expectErr: true,
			template: `version: 0.0.0
title: My Super Plugin
description: This is the best plugin ever
parameters:
  path: this used to be supported
pipeline:
`,
		},
		{
			name:      "bad_parameter_label",
			expectErr: true,
			template: `version: 0.0.0
title: My Super Plugin
description: This is the best plugin ever
parameters:
  path:
    label: []
    description: The path to a thing
    type: string
pipeline:
`,
		},
		{
			name:      "bad_parameter_description",
			expectErr: true,
			template: `version: 0.0.0
title: My Super Plugin
description: This is the best plugin ever
parameters:
  path:
    label: Path
    description: []
    type: string
pipeline:
`,
		},
		{
			name:      "bad_parameter_type",
			expectErr: true,
			template: `version: 0.0.0
title: My Super Plugin
description: This is the best plugin ever
parameters:
  path:
    label: Path
    description: The path to a thing
    type: []
pipeline:
`,
		},
		{
			name:      "empty_parameter",
			expectErr: false,
			template: `version: 0.0.0
title: My Super Plugin
description: This is the best plugin ever
parameters:
  path:
pipeline:
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reg := CustomRegistry{}
			err := reg.Add(tc.name, tc.template)
			require.NoError(t, err)
			_, err = reg.Render(tc.name, map[string]interface{}{})
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
