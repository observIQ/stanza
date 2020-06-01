package builtin

import (
	"context"
	"testing"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/bluemedora/bplogagent/plugin/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTimestampPlugin(t *testing.T) {
	cfg := &TimestampConfig{
		TransformerConfig: helper.TransformerConfig{
			BasicConfig: helper.BasicConfig{
				PluginID:   "test_plugin_id",
				PluginType: "timestamp",
			},
			OutputID: "testoutput",
		},
		CopyFrom: entry.Field{
			Keys: []string{"timestamp"},
		},
		RemoveField: true,
	}

	buildContext := testutil.NewTestBuildContext(t)
	newPlugin, err := cfg.Build(buildContext)
	require.NoError(t, err)

	timestampPlugin := newPlugin.(*TimestampPlugin)
	mockOutput := &testutil.Plugin{}
	timestampPlugin.Output = mockOutput

	entryChan := make(chan *entry.Entry, 1)
	mockOutput.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		entryChan <- args.Get(1).(*entry.Entry)
	}).Return(nil)

	ts, err := time.Parse(time.UnixDate, "Mon Jan 2 15:04:05 MST 2006")
	entry := &entry.Entry{
		Timestamp: time.Now(),
		Record: map[string]interface{}{
			"timestamp": ts,
		},
	}
	err = timestampPlugin.Process(context.Background(), entry)
	require.NoError(t, err)

	select {
	case e := <-entryChan:
		require.Equal(t, ts, e.Timestamp)
	case <-time.After(time.Second):
		require.FailNow(t, "Timed out waiting for entry to be parsed")
	}

}
