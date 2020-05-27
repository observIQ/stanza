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

func TestTimeParser(t *testing.T) {
	cfg := &TimeParserConfig{
		BasicPluginConfig: helper.BasicPluginConfig{
			PluginID:   "test_plugin_id",
			PluginType: "time_parser",
		},
		BasicParserConfig: helper.BasicParserConfig{
			OutputID: "output1",
		},
		Layout: time.UnixDate,
	}

	buildContext := testutil.NewTestBuildContext(t)
	newPlugin, err := cfg.Build(buildContext)
	require.NoError(t, err)

	mockOutput := &testutil.Plugin{}
	entryChan := make(chan *entry.Entry, 1)
	mockOutput.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		entryChan <- args.Get(1).(*entry.Entry)
	}).Return(nil)

	timeParser := newPlugin.(*TimeParser)
	timeParser.Output = mockOutput
	entry := entry.New()
	entry.Record = "Mon Jan 2 15:04:05 MST 2006"
	err = timeParser.Process(context.Background(), entry)
	require.NoError(t, err)

	select {
	case e := <-entryChan:
		expectedParsed, _ := time.Parse(time.UnixDate, "Mon Jan 2 15:04:05 MST 2006")
		require.Equal(t, expectedParsed, e.Record)
	case <-time.After(time.Second):
		require.FailNow(t, "Timed out waiting for entry to be processed")
	}

}
