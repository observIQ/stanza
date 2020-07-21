package transformer

import (
	"testing"

	"github.com/observiq/carbon/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestNoopPluginBuild(t *testing.T) {
	cfg := NewNoopPluginConfig("test_plugin_id")
	cfg.OutputIDs = []string{"output"}

	_, err := cfg.Build(testutil.NewBuildContext(t))
	require.NoError(t, err)
}
