package builtin

import (
	"testing"

	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/bluemedora/bplogagent/plugin/testutil"
	"github.com/stretchr/testify/require"
)

func TestNoopPluginBuild(t *testing.T) {
	cfg := &NoopPluginConfig{
		TransformerConfig: helper.TransformerConfig{
			BasicConfig: helper.BasicConfig{
				PluginID:   "test_plugin_id",
				PluginType: "noop",
			},
			OutputID: "output",
		},
	}

	buildContext := testutil.NewTestBuildContext(t)
	_, err := cfg.Build(buildContext)
	require.NoError(t, err)
}
