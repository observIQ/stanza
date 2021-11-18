package stanza

import (
	"testing"
	"time"

	"github.com/observiq/stanza/v2/operator"
	"github.com/observiq/stanza/v2/testutil"
	"github.com/stretchr/testify/require"
)

func TestStanzaOperator(t *testing.T) {
	cfg := NewInputConfig("test")
	cfg.OutputIDs = []string{"fake"}

	bc := testutil.NewBuildContext(t)
	ops, err := cfg.Build(bc)
	require.NoError(t, err)
	op := ops[0]

	fake := testutil.NewFakeOutput(t)
	op.SetOutputs([]operator.Operator{fake})

	require.NoError(t, op.Start())
	defer op.Stop()

	bc.Logger.Errorw("test failure", "key", "value")

	expectedBody := map[string]interface{}{
		"message": "test failure",
		"key":     "value",
	}

	select {
	case e := <-fake.Received:
		require.Equal(t, expectedBody, e.Body)

	case <-time.After(time.Second):
		require.FailNow(t, "timed out")
	}
}

func TestStanzaOperatorBUildFailure(t *testing.T) {
	cfg := NewInputConfig("")
	cfg.OperatorType = ""
	bc := testutil.NewBuildContext(t)
	_, err := cfg.Build(bc)
	require.Error(t, err)
}
