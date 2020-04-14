package builtin

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/bluemedora/bplogagent/entry"
	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
)

func init() {
	plugin.Register("tcp_input", &TCPInputConfig{})
}

// TCPInputConfig is the configuration of a tcp input plugin.
type TCPInputConfig struct {
	helper.BasicPluginConfig `mapstructure:",squash" yaml:",inline"`
	helper.BasicInputConfig  `mapstructure:",squash" yaml:",inline"`
	ListenAddress            string `mapstructure:"listen_address" yaml:"listen_address,omitempty"`
	MessageField             entry.FieldSelector
	SourceField              entry.FieldSelector
}

// Build will build a tcp input plugin.
func (c TCPInputConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	basicPlugin, err := c.BasicPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, err
	}

	basicInput, err := c.BasicInputConfig.Build()
	if err != nil {
		return nil, err
	}

	if c.ListenAddress == "" {
		return nil, fmt.Errorf("missing field 'listen_address'")
	}

	if c.MessageField == nil {
		fs := entry.FieldSelector([]string{"message"})
		c.MessageField = fs
	}

	address, err := net.ResolveTCPAddr("tcp", c.ListenAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve listen_address: %s", err)
	}

	tcpInput := &TCPInput{
		BasicPlugin:  basicPlugin,
		BasicInput:   basicInput,
		address:      address,
		messageField: c.MessageField,
		sourceField:  c.SourceField,
	}
	return tcpInput, nil
}

// TCPInput is a plugin that listens for log entries over tcp.
type TCPInput struct {
	helper.BasicPlugin
	helper.BasicInput
	address *net.TCPAddr

	listener  net.Listener
	cancel    context.CancelFunc
	waitGroup *sync.WaitGroup

	messageField entry.FieldSelector
	sourceField  entry.FieldSelector
}

// Start will start listening for log entries over tcp.
func (t *TCPInput) Start() error {
	listener, err := net.ListenTCP("tcp", t.address)
	if err != nil {
		return fmt.Errorf("failed to listen on interface: %w", err)
	}

	t.listener = listener
	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel
	t.waitGroup = &sync.WaitGroup{}
	t.goListen(ctx)
	return nil
}

// goListenn will listen for tcp connections.
func (t *TCPInput) goListen(ctx context.Context) {
	t.waitGroup.Add(1)

	go func() {
		defer t.waitGroup.Done()

		for {
			conn, err := t.listener.Accept()
			if err != nil {
				t.Debugf("Exiting listener: %s", err)
				break
			}

			t.Debugf("Received connection: %s", conn.RemoteAddr().String())
			subctx, cancel := context.WithCancel(ctx)
			t.goHandleClose(subctx, conn)
			t.goHandleMessages(conn, cancel)
		}
	}()
}

// goHandleClose will wait for the context to finish before closing a connection.
func (t *TCPInput) goHandleClose(ctx context.Context, conn net.Conn) {
	go func() {
		<-ctx.Done()
		t.Debugf("Closing connection: %s", conn.RemoteAddr().String())
		if err := conn.Close(); err != nil {
			t.Errorf("Failed to close connection: %s", err)
		}
	}()
}

// goHandleMessages will handles messages from a tcp connection.
func (t *TCPInput) goHandleMessages(conn net.Conn, cancel context.CancelFunc) {
	t.waitGroup.Add(1)

	go func() {
		defer t.waitGroup.Done()
		defer cancel()

		reader := bufio.NewReaderSize(conn, 1024*64)
		for {
			entry, err := t.readEntry(conn, reader)
			if err != nil {
				t.Debugf("Exiting message handler: %s", err)
				break
			}

			if err := t.Output.Process(entry); err != nil {
				t.Errorf("Output %s failed to process entry: %s", t.OutputID, err)
			}
		}
	}()
}

// readEntry will read a log entry from a TCP connection.
func (t *TCPInput) readEntry(conn net.Conn, reader *bufio.Reader) (*entry.Entry, error) {
	message, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	entry := entry.NewEntry()
	entry.Set(t.messageField, message)
	if t.sourceField != nil {
		entry.Set(t.sourceField, conn.RemoteAddr().String())
	}
	return entry, nil
}

// Stop will stop listening for log entries over TCP.
func (t *TCPInput) Stop() error {
	t.cancel()

	if err := t.listener.Close(); err != nil {
		return err
	}

	t.waitGroup.Wait()
	return nil
}
