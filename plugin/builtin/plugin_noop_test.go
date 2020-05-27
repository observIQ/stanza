package builtin

import (
	"testing"

	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/bluemedora/bplogagent/plugin/testutil"
	"github.com/stretchr/testify/require"
)

func TestNoopPluginBuild(t *testing.T) {
	cfg := &NoopPluginConfig{
		BasicPluginConfig: helper.BasicPluginConfig{
			PluginID:   "test_plugin_id",
			PluginType: "noop",
		},
		BasicTransformerConfig: helper.BasicTransformerConfig{
			OutputID: "output",
		},
	}

	buildContext := testutil.NewTestBuildContext(t)
	_, err := cfg.Build(buildContext)
	require.NoError(t, err)
}
