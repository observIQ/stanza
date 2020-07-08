package output

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/internal/testutil"
	"github.com/observiq/carbon/plugin/helper"
	"github.com/stretchr/testify/require"
)

func TestStdoutPlugin(t *testing.T) {
	cfg := StdoutConfig{
		OutputConfig: helper.OutputConfig{
			BasicConfig: helper.BasicConfig{
				PluginID:   "test_plugin_id",
				PluginType: "stdout",
			},
		},
	}

	plugin, err := cfg.Build(testutil.NewBuildContext(t))
	require.NoError(t, err)

	var buf bytes.Buffer
	plugin.(*StdoutPlugin).encoder = json.NewEncoder(&buf)

	ts := time.Unix(1591042864, 0)
	e := &entry.Entry{
		Timestamp: ts,
		Record:    "test record",
	}
	err = plugin.Process(context.Background(), e)
	require.NoError(t, err)

	marshalledTimestamp, err := json.Marshal(ts)
	require.NoError(t, err)

	expected := `{"timestamp":` + string(marshalledTimestamp) + `,"severity":0,"record":"test record"}` + "\n"
	require.Equal(t, expected, buf.String())
}
