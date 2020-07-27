package operator

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

func TestPluginRegistry_LoadAll(t *testing.T) {
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

	pluginRegistry := PluginRegistry{}
	err = pluginRegistry.LoadAll(tempDir, "*.yaml")
	require.NoError(t, err)

	require.Equal(t, 2, len(pluginRegistry))
}

func TestPluginRegistryRender(t *testing.T) {
	t.Run("ErrorTypeDoesNotExist", func(t *testing.T) {
		reg := PluginRegistry{}
		_, err := reg.Render("unknown", map[string]interface{}{})
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not exist")
	})

	t.Run("ErrorExecFailure", func(t *testing.T) {
		tmpl, err := template.New("plugintype").Parse(`{{ .panicker }}`)
		require.NoError(t, err)

		reg := PluginRegistry{
			"plugintype": tmpl,
		}
		params := map[string]interface{}{
			"panicker": func() {
				panic("testpanic")
			},
		}
		_, err = reg.Render("plugintype", params)
		require.Contains(t, err.Error(), "failed to render")
	})
}

func TestPluginRegistryLoad(t *testing.T) {
	t.Run("LoadAllBadGlob", func(t *testing.T) {
		reg := PluginRegistry{}
		err := reg.LoadAll("", `[]`)
		require.Error(t, err)
		require.Contains(t, err.Error(), "with glob pattern")
	})

	t.Run("AddDuplicate", func(t *testing.T) {
		reg := PluginRegistry{}
		Register("copy", func() Builder { return nil })
		err := reg.Add("copy", "pipeline:\n")
		require.Error(t, err)
		require.Contains(t, err.Error(), "already exists")
	})

	t.Run("AddBadTemplate", func(t *testing.T) {
		reg := PluginRegistry{}
		err := reg.Add("new", "{{ nofunc }")
		require.Error(t, err)
		require.Contains(t, err.Error(), "as a plugin template")
	})

	t.Run("LoadAllWithFailures", func(t *testing.T) {
		tempDir := NewTempDir(t)
		pluginPath := filepath.Join(tempDir, "copy.yaml")
		err := ioutil.WriteFile(pluginPath, []byte("pipeline:\n"), 0755)
		require.NoError(t, err)

		reg := PluginRegistry{}
		err = reg.LoadAll(tempDir, "*.yaml")
		require.Error(t, err)
	})
}

func TestPluginMetadata(t *testing.T) {

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
    type: int
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
    type: int
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
    type: int
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
    type: int
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
    type: {}
pipeline:
`,
		},
		{
			name:      "empty_parameter",
			expectErr: true,
			template: `version: 0.0.0
title: My Super Plugin
description: This is the best plugin ever
parameters:
  path:
pipeline:
`,
		},
		{
			name:      "unknown_parameter_type",
			expectErr: true,
			template: `version: 0.0.0
title: My Super Plugin
description: This is the best plugin ever
parameters:
  path:
    label: Parameter
    description: The thing of the thing
    type: custom
pipeline:
`,
		},
		{
			name:      "string_parameter_type",
			expectErr: false,
			template: `version: 0.0.0
title: My Super Plugin
description: This is the best plugin ever
parameters:
  path:
    label: Parameter
    description: The thing of the thing
    type: string
pipeline:
`,
		},
		{
			name:      "string_parameter_default",
			expectErr: false,
			template: `version: 0.0.0
title: My Super Plugin
description: This is the best plugin ever
parameters:
  path:
    label: Parameter
    description: The thing of the thing
    type: string
    default: hello
pipeline:
`,
		},
		{
			name:      "string_parameter_default_invalid",
			expectErr: true,
			template: `version: 0.0.0
title: My Super Plugin
description: This is the best plugin ever
parameters:
  path:
    label: Parameter
    description: The thing of the thing
    type: string
    default: 123
pipeline:
`,
		},
		{
			name:      "int_parameter_type",
			expectErr: false,
			template: `version: 0.0.0
title: My Super Plugin
description: This is the best plugin ever
parameters:
  path:
    label: Parameter
    description: The thing of the thing
    type: int
pipeline:
`,
		},
		{
			name:      "int_parameter_default",
			expectErr: false,
			template: `version: 0.0.0
title: My Super Plugin
description: This is the best plugin ever
parameters:
  path:
    label: Parameter
    description: The thing of the thing
    type: int
    default: 123
pipeline:
`,
		},
		{
			name:      "int_parameter_default_invalid",
			expectErr: true,
			template: `version: 0.0.0
title: My Super Plugin
description: This is the best plugin ever
parameters:
  path:
    label: Parameter
    description: The thing of the thing
    type: int
    default: hello
pipeline:
`,
		},
		{
			name:      "bool_parameter_type",
			expectErr: false,
			template: `version: 0.0.0
title: My Super Plugin
description: This is the best plugin ever
parameters:
  path:
    label: Parameter
    description: The thing of the thing
    type: bool
pipeline:
`,
		},
		{
			name:      "bool_parameter_default_true",
			expectErr: false,
			template: `version: 0.0.0
title: My Super Plugin
description: This is the best plugin ever
parameters:
  path:
    label: Parameter
    description: The thing of the thing
    type: bool
    default: true
pipeline:
`,
		},
		{
			name:      "bool_parameter_default_false",
			expectErr: false,
			template: `version: 0.0.0
title: My Super Plugin
description: This is the best plugin ever
parameters:
  path:
    label: Parameter
    description: The thing of the thing
    type: bool
    default: false
pipeline:
`,
		},
		{
			name:      "bool_parameter_default_invalid",
			expectErr: true,
			template: `version: 0.0.0
title: My Super Plugin
description: This is the best plugin ever
parameters:
  path:
    label: Parameter
    description: The thing of the thing
    type: bool
    default: 123
pipeline:
`,
		},
		{
			name:      "enum_parameter_type",
			expectErr: false,
			template: `version: 0.0.0
title: My Super Plugin
description: This is the best plugin ever
parameters:
  path:
    label: Parameter
    description: The thing of the thing
    type: ["one", "two"]
pipeline:
`,
		},
		{
			name:      "enum_parameter_type_alternate",
			expectErr: false,
			template: `version: 0.0.0
title: My Super Plugin
description: This is the best plugin ever
parameters:
  path:
    label: Parameter
    description: The thing of the thing
    type:
     - one
     - two
pipeline:
`,
		},
		{
			name:      "enum_parameter_type_default",
			expectErr: false,
			template: `version: 0.0.0
title: My Super Plugin
description: This is the best plugin ever
parameters:
  path:
    label: Parameter
    description: The thing of the thing
    type:
     - one
     - two
    default: one
pipeline:
`,
		},
		{
			name:      "enum_parameter_type_default_invalid",
			expectErr: true,
			template: `version: 0.0.0
title: My Super Plugin
description: This is the best plugin ever
parameters:
  path:
    label: Parameter
    description: The thing of the thing
    type:
     - one
     - two
    default: three
pipeline:
`,
		},
		{
			name:      "default_invalid_type",
			expectErr: true,
			template: `version: 0.0.0
title: My Super Plugin
description: This is the best plugin ever
parameters:
  path:
    label: Parameter
    description: The thing of the thing
    type: int
    default: {}
pipeline:
`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reg := PluginRegistry{}
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
