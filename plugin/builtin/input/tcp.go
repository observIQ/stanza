package input

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/observiq/carbon/plugin"
	"github.com/observiq/carbon/plugin/helper"
)

func init() {
	plugin.Register("tcp_input", func() plugin.Builder { return NewTCPInputConfig("") })
}

func NewTCPInputConfig(pluginID string) *TCPInputConfig {
	return &TCPInputConfig{
		InputConfig: helper.NewInputConfig(pluginID, "tcp_input"),
	}
}

// TCPInputConfig is the configuration of a tcp input plugin.
type TCPInputConfig struct {
	helper.InputConfig `yaml:",inline"`

	ListenAddress string `json:"listen_address,omitempty" yaml:"listen_address,omitempty"`
}

// Build will build a tcp input plugin.
func (c TCPInputConfig) Build(context plugin.BuildContext) (plugin.Operator, error) {
	inputOperator, err := c.InputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	if c.ListenAddress == "" {
		return nil, fmt.Errorf("missing required parameter 'listen_address'")
	}

	address, err := net.ResolveTCPAddr("tcp", c.ListenAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve listen_address: %s", err)
	}

	tcpInput := &TCPInput{
		InputOperator: inputOperator,
		address:       address,
	}
	return tcpInput, nil
}

// TCPInput is a plugin that listens for log entries over tcp.
type TCPInput struct {
	helper.InputOperator
	address *net.TCPAddr

	listener  *net.TCPListener
	cancel    context.CancelFunc
	waitGroup *sync.WaitGroup
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
			conn, err := t.listener.AcceptTCP()
			if err != nil {
				t.Debugf("Exiting listener: %s", err)
				break
			}

			t.Debugf("Received connection: %s", conn.RemoteAddr().String())
			subctx, cancel := context.WithCancel(ctx)
			t.goHandleClose(subctx, conn)
			t.goHandleMessages(subctx, conn, cancel)
		}
	}()
}

// goHandleClose will wait for the context to finish before closing a connection.
func (t *TCPInput) goHandleClose(ctx context.Context, conn net.Conn) {
	t.waitGroup.Add(1)

	go func() {
		defer t.waitGroup.Done()
		<-ctx.Done()
		t.Debugf("Closing connection: %s", conn.RemoteAddr().String())
		if err := conn.Close(); err != nil {
			t.Errorf("Failed to close connection: %s", err)
		}
	}()
}

// goHandleMessages will handles messages from a tcp connection.
func (t *TCPInput) goHandleMessages(ctx context.Context, conn net.Conn, cancel context.CancelFunc) {
	t.waitGroup.Add(1)

	go func() {
		defer t.waitGroup.Done()
		defer cancel()

		reader := bufio.NewReaderSize(conn, 1024*64)
		for {
			message, err := t.readMessage(conn, reader)
			if err != nil {
				t.Debugf("Exiting message handler: %s", err)
				break
			}

			entry := t.NewEntry(message)
			t.Write(ctx, entry)
		}
	}()
}

// readMessage will read a log message from a TCP connection.
func (t *TCPInput) readMessage(conn net.Conn, reader *bufio.Reader) (string, error) {
	message, err := reader.ReadBytes('\n')
	if err != nil {
		if err == io.EOF {
			return string(message), err
		}
		return "", err
	}

	return string(message[:len(message)-1]), nil
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
