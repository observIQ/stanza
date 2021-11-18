package stdin

import (
	"os"
	"testing"

	"github.com/observiq/stanza/v2/operator"
	"github.com/observiq/stanza/v2/testutil"
	"github.com/stretchr/testify/require"
)

func TestStdin(t *testing.T) {
	cfg := NewStdinInputConfig("")
	cfg.OutputIDs = []string{"fake"}

	op, err := cfg.Build(testutil.NewBuildContext(t))
	require.NoError(t, err)

	fake := testutil.NewFakeOutput(t)
	op[0].SetOutputs([]operator.Operator{fake})

	r, w, err := os.Pipe()
	require.NoError(t, err)

	stdin := op[0].(*StdinInput)
	stdin.stdin = r

	require.NoError(t, stdin.Start())
	defer stdin.Stop()

	w.WriteString("test")
	w.Close()
	fake.ExpectBody(t, "test")
}
