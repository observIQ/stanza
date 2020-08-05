package input

import (
	"net"
	"testing"
	"time"

	"github.com/observiq/carbon/entry"
	"github.com/observiq/carbon/internal/testutil"
	"github.com/observiq/carbon/operator"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func udpInputTest(input []byte, expected []string) func(t *testing.T) {
	return func(t *testing.T) {
		t.Parallel()
		cfg := NewUDPInputConfig("test_input")
		address := newRandListenAddress()
		cfg.ListenAddress = address

		newOperator, err := cfg.Build(testutil.NewBuildContext(t))
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

		_, err = conn.Write(input)
		require.NoError(t, err)

		for _, expectedRecord := range expected {
			select {
			case entry := <-entryChan:
				require.Equal(t, expectedRecord, entry.Record)
			case <-time.After(time.Second):
				require.FailNow(t, "Timed out waiting for message to be written")
			}
		}

		select {
		case entry := <-entryChan:
			require.FailNow(t, "Unexpected entry: %s", entry)
		case <-time.After(100 * time.Millisecond):
			return
		}
	}
}

func TestUDPInput(t *testing.T) {
	t.Run("Simple", udpInputTest([]byte("message1"), []string{"message1"}))
	t.Run("TrailingNewlines", udpInputTest([]byte("message1\n"), []string{"message1"}))
	t.Run("TrailingCRNewlines", udpInputTest([]byte("message1\r\n"), []string{"message1"}))
	t.Run("NewlineInMessage", udpInputTest([]byte("message1\nmessage2\n"), []string{"message1\nmessage2"}))
}

func BenchmarkUdpInput(b *testing.B) {
	cfg := NewUDPInputConfig("test_id")
	address := newRandListenAddress()
	cfg.ListenAddress = address

	newOperator, err := cfg.Build(testutil.NewBuildContext(b))
	require.NoError(b, err)

	fakeOutput := testutil.NewFakeOutput(b)
	udpInput := newOperator.(*UDPInput)
	udpInput.InputOperator.OutputOperators = []operator.Operator{fakeOutput}

	err = udpInput.Start()
	require.NoError(b, err)

	done := make(chan struct{})
	go func() {
		conn, err := net.Dial("udp", address)
		require.NoError(b, err)
		defer udpInput.Stop()
		defer conn.Close()
		message := []byte("message\n")
		for {
			select {
			case <-done:
				return
			default:
				_, err := conn.Write(message)
				require.NoError(b, err)
			}
		}
	}()

	for i := 0; i < b.N; i++ {
		<-fakeOutput.Received
	}

	defer close(done)
}
