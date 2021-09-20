package udp

import (
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func udpInputTest(input []byte, expected []string) func(t *testing.T) {
	return func(t *testing.T) {
		cfg := NewUDPInputConfig("test_input")
		cfg.ListenAddress = ":0"

		ops, err := cfg.Build(testutil.NewBuildContext(t))
		require.NoError(t, err)
		op := ops[0]

		mockOutput := testutil.Operator{}
		udpInput, ok := op.(*UDPInput)
		require.True(t, ok)

		udpInput.InputOperator.OutputOperators = []operator.Operator{&mockOutput}

		entryChan := make(chan *entry.Entry, 1)
		mockOutput.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			entryChan <- args.Get(1).(*entry.Entry)
		}).Return(nil)

		err = udpInput.Start()
		require.NoError(t, err)
		defer udpInput.Stop()

		conn, err := net.Dial("udp", udpInput.connection.LocalAddr().String())
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

func udpInputAttributesTest(input []byte, expected []string) func(t *testing.T) {
	return func(t *testing.T) {
		cfg := NewUDPInputConfig("test_input")
		cfg.ListenAddress = ":0"
		cfg.AddAttributes = true

		ops, err := cfg.Build(testutil.NewBuildContext(t))
		require.NoError(t, err)
		op := ops[0]

		mockOutput := testutil.Operator{}
		udpInput, ok := op.(*UDPInput)
		require.True(t, ok)

		udpInput.InputOperator.OutputOperators = []operator.Operator{&mockOutput}

		entryChan := make(chan *entry.Entry, 1)
		mockOutput.On("Process", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			entryChan <- args.Get(1).(*entry.Entry)
		}).Return(nil)

		err = udpInput.Start()
		require.NoError(t, err)
		defer udpInput.Stop()

		conn, err := net.Dial("udp", udpInput.connection.LocalAddr().String())
		require.NoError(t, err)
		defer conn.Close()

		_, err = conn.Write(input)
		require.NoError(t, err)

		for _, expectedRecord := range expected {
			select {
			case entry := <-entryChan:
				expectedAttributes := map[string]string{
					"net.transport": "IP.UDP",
				}
				// LocalAddr for udpInput.connection is a server address
				if addr, ok := udpInput.connection.LocalAddr().(*net.UDPAddr); ok {
					expectedAttributes["net.host.ip"] = addr.IP.String()
					expectedAttributes["net.host.port"] = strconv.FormatInt(int64(addr.Port), 10)
				}
				// LocalAddr for conn is a client (peer) address
				if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok {
					expectedAttributes["net.peer.ip"] = addr.IP.String()
					expectedAttributes["net.peer.port"] = strconv.FormatInt(int64(addr.Port), 10)
				}
				require.Equal(t, expectedRecord, entry.Record)
				require.Equal(t, expectedAttributes, entry.Attributes)
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

func TestUDPInputAttributes(t *testing.T) {
	t.Run("Simple", udpInputAttributesTest([]byte("message1"), []string{"message1"}))
	t.Run("TrailingNewlines", udpInputAttributesTest([]byte("message1\n"), []string{"message1"}))
	t.Run("TrailingCRNewlines", udpInputAttributesTest([]byte("message1\r\n"), []string{"message1"}))
	t.Run("NewlineInMessage", udpInputAttributesTest([]byte("message1\nmessage2\n"), []string{"message1\nmessage2"}))
}

func BenchmarkUdpInput(b *testing.B) {
	cfg := NewUDPInputConfig("test_id")
	cfg.ListenAddress = ":0"

	ops, err := cfg.Build(testutil.NewBuildContext(b))
	require.NoError(b, err)
	op := ops[0]

	fakeOutput := testutil.NewFakeOutput(b)
	udpInput := op.(*UDPInput)
	udpInput.InputOperator.OutputOperators = []operator.Operator{fakeOutput}

	err = udpInput.Start()
	require.NoError(b, err)

	done := make(chan struct{})
	go func() {
		conn, err := net.Dial("udp", udpInput.connection.LocalAddr().String())
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
