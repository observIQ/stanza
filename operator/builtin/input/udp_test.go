package input

import (
	"fmt"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/internal/testutil"
	"github.com/observiq/carbon/operator"
	"github.com/observiq/carbon/operator/helper"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestUDPInput(t *testing.T) {
	basicUDPInputConfig := func() *UDPInputConfig {
		return &UDPInputConfig{
			InputConfig: helper.InputConfig{
				WriteTo: entry.NewRecordField(),
				WriterConfig: helper.WriterConfig{
					BasicConfig: helper.BasicConfig{
						OperatorID:   "test_id",
						OperatorType: "udp_input",
					},
					OutputIDs: []string{"test_output_id"},
				},
			},
		}
	}

	t.Run("Simple", func(t *testing.T) {
		port := rand.Int()%16000 + 49152
    address := fmt.Sprintf("127.0.0.1:%d", port)
		cfg := basicUDPInputConfig()
		cfg.ListenAddress = address

		buildContext := testutil.NewBuildContext(t)
		newOperator, err := cfg.Build(buildContext)
		require.NoError(t, err)

		mockOutput := testutil.Operator{}
		udpInput, ok := newOperator.(*UDPInput)
		require.True(t, ok)

		udpInput.InputOperator.OutputOperators = []operator.Operator{&mockOutput}

		entryChan := make(chan *entry.Entry, 1)
		mockOutput.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			entryChan <- args.Get(1).(*entry.Entry)
		}).Return(nil)

		err = udpInput.Start()
		require.NoError(t, err)
		defer udpInput.Stop()

		conn, err := net.Dial("udp", address)
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
