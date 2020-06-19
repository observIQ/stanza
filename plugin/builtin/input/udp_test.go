package input

import (
	"net"
	"testing"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/internal/testutil"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"github.com/bluemedora/bplogagent/plugin/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUDPInput(t *testing.T) {
	basicUDPInputConfig := func() *UDPInputConfig {
		return &UDPInputConfig{
			InputConfig: helper.InputConfig{
				BasicConfig: helper.BasicConfig{
					PluginID:   "test_id",
					PluginType: "tcp_input",
				},
				WriteTo: entry.Field{
					Keys: []string{},
				},
				OutputID: "test_output_id",
			},
		}
	}

	t.Run("Simple", func(t *testing.T) {
		cfg := basicUDPInputConfig()
		cfg.ListenAddress = "127.0.0.1:63001"

		buildContext := testutil.NewTestBuildContext(t)
		newPlugin, err := cfg.Build(buildContext)
		require.NoError(t, err)

		mockOutput := mocks.Plugin{}
		tcpInput := newPlugin.(*UDPInput)
		tcpInput.InputPlugin.Output = &mockOutput

		entryChan := make(chan *entry.Entry, 1)
		mockOutput.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			entryChan <- args.Get(1).(*entry.Entry)
		}).Return(nil)

		err = tcpInput.Start()
		require.NoError(t, err)
		defer tcpInput.Stop()

		conn, err := net.Dial("udp", "127.0.0.1:63001")
		require.NoError(t, err)
		defer conn.Close()

		_, err = conn.Write([]byte("message1\n"))
		require.NoError(t, err)

		expectedRecord := "message1"
		select {
		case entry := <-entryChan:
			require.Equal(t, expectedRecord, entry.Record)
		case <-time.After(time.Second):
			require.FailNow(t, "Timed out waiting for message to be written")
		}
	})

}
