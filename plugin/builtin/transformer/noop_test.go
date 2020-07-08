package transformer

import (
	"testing"

	"github.com/observiq/carbon/internal/testutil"
	"github.com/observiq/carbon/plugin/helper"
	"github.com/stretchr/testify/require"
)

func TestNoopPluginBuild(t *testing.T) {
	cfg := &NoopPluginConfig{
		TransformerConfig: helper.TransformerConfig{
			BasicConfig: helper.BasicConfig{
				PluginID:   "test_plugin_id",
				PluginType: "noop",
			},
			WriterConfig: helper.WriterConfig{
				OutputIDs: []string{"output"},
			},
		},
	}

	buildContext := testutil.NewBuildContext(t)
	_, err := cfg.Build(buildContext)
	require.NoError(t, err)
}
