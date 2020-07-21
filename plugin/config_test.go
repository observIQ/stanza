package plugin

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v2"
)

func TestStubDatabase(t *testing.T) {
	stub := &StubDatabase{}

	err := stub.Close()
	require.NoError(t, err)

	err = stub.Sync()
	require.NoError(t, err)

	err = stub.Update(nil)
	require.NoError(t, err)

	err = stub.View(nil)
	require.NoError(t, err)
}

type FakeBuilder struct {
	PluginID   string   `json:"id" yaml:"id"`
	PluginType string   `json:"type" yaml:"type"`
	Array      []string `json:"array" yaml:"array"`
}

func (f *FakeBuilder) SetNamespace(s string, e ...string)         {}
func (f *FakeBuilder) Build(context BuildContext) (Plugin, error) { return nil, nil }
func (f *FakeBuilder) ID() string                                 { return "custom" }
func (f *FakeBuilder) Type() string                               { return "custom" }

func TestUnmarshalJSONErrors(t *testing.T) {
	t.Run("InvalidJSON", func(t *testing.T) {
		raw := `{}}`
		var cfg Config
		err := json.Unmarshal([]byte(raw), &cfg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid")
	})

	t.Run("MissingType", func(t *testing.T) {
		raw := `{"id":"stdout"}`
		var cfg Config
		err := json.Unmarshal([]byte(raw), &cfg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing required field")
	})

	t.Run("UnknownType", func(t *testing.T) {
		raw := `{"id":"stdout","type":"nonexist"}`
		var cfg Config
		err := json.Unmarshal([]byte(raw), &cfg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported type")
	})

	t.Run("TypeSpecificUnmarshal", func(t *testing.T) {
		raw := `{"id":"custom","type":"custom","array":"non-array-value"}`
		Register("custom", func() Builder { return &FakeBuilder{} })
		var cfg Config
		err := json.Unmarshal([]byte(raw), &cfg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot unmarshal string into")
	})
}

func TestMarshalJSON(t *testing.T) {
	cfg := Config{
		Builder: &FakeBuilder{
			PluginID:   "custom",
			PluginType: "custom",
			Array:      []string{"test"},
		},
	}
	out, err := json.Marshal(cfg)
	require.NoError(t, err)
	expected := `{"id":"custom","type":"custom","array":["test"]}`
	require.Equal(t, expected, string(out))
}

func TestUnmarshalYAMLErrors(t *testing.T) {
	t.Run("InvalidYAML", func(t *testing.T) {
		raw := `-- - \n||\\`
		var cfg Config
		err := yaml.Unmarshal([]byte(raw), &cfg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed ")
	})

	t.Run("MissingType", func(t *testing.T) {
		raw := "id: custom\n"
		var cfg Config
		err := yaml.Unmarshal([]byte(raw), &cfg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "missing required field")
	})

	t.Run("NonStringType", func(t *testing.T) {
		raw := "id: custom\ntype: 123"
		var cfg Config
		err := yaml.Unmarshal([]byte(raw), &cfg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "non-string type")
	})

	t.Run("UnknownType", func(t *testing.T) {
		raw := "id: custom\ntype: unknown\n"
		var cfg Config
		err := yaml.Unmarshal([]byte(raw), &cfg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported type")
	})

	t.Run("TypeSpecificUnmarshal", func(t *testing.T) {
		raw := "id: custom\ntype: custom\narray: nonarray"
		Register("custom", func() Builder { return &FakeBuilder{} })
		var cfg Config
		err := yaml.Unmarshal([]byte(raw), &cfg)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot unmarshal !!str")
	})
}

func TestMarshalYAML(t *testing.T) {
	cfg := Config{
		Builder: &FakeBuilder{
			PluginID:   "custom",
			PluginType: "custom",
			Array:      []string{"test"},
		},
	}
	out, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	expected := "id: custom\ntype: custom\narray:\n- test\n"
	require.Equal(t, expected, string(out))
}
