package stdout

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/observiq/stanza/v2/entry"
	"github.com/observiq/stanza/v2/operator/helper"
	"github.com/observiq/stanza/v2/testutil"
	"github.com/stretchr/testify/require"
)

func TestStdoutOperator(t *testing.T) {
	cfg := StdoutConfig{
		OutputConfig: helper.OutputConfig{
			BasicConfig: helper.BasicConfig{
				OperatorID:   "test_operator_id",
				OperatorType: "stdout",
			},
		},
	}

	ops, err := cfg.Build(testutil.NewBuildContext(t))
	require.NoError(t, err)
	op := ops[0]

	var buf bytes.Buffer
	op.(*StdoutOperator).encoder = json.NewEncoder(&buf)

	ts := time.Unix(1591042864, 0)
	e := &entry.Entry{
		Timestamp: ts,
		Record:    "test record",
	}
	err = op.Process(context.Background(), e)
	require.NoError(t, err)

	marshalledTimestamp, err := json.Marshal(ts)
	require.NoError(t, err)

	expected := `{"timestamp":` + string(marshalledTimestamp) + `,"severity":0,"record":"test record"}` + "\n"
	require.Equal(t, expected, buf.String())
}
