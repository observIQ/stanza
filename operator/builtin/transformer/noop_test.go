package transformer

import (
	"testing"

	"github.com/observiq/carbon/testutil"
	"github.com/stretchr/testify/require"
)

func TestNoopOperatorBuild(t *testing.T) {
	cfg := NewNoopOperatorConfig("test_operator_id")
	cfg.OutputIDs = []string{"output"}

	_, err := cfg.Build(testutil.NewBuildContext(t))
	require.NoError(t, err)
}
