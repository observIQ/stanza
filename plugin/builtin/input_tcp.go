package builtin

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/helper"
	"go.uber.org/zap"
)

func init() {
	plugin.Register("tcp_input", &TCPInputConfig{})
}

// TCPInputConfig is the configuration of a tcp input plugin.
type TCPInputConfig struct {
	helper.BasicPluginConfig `mapstructure:",squash" yaml:",inline"`
	helper.BasicInputConfig  `mapstructure:",squash" yaml:",inline"`

	ListenAddress string `mapstructure:"listen_address" json:"listen_address,omitempty" yaml:"listen_address,omitempty"`
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

	address, err := net.ResolveTCPAddr("tcp", c.ListenAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve listen_address: %s", err)
	}

	tcpInput := &TCPInput{
		BasicPlugin: basicPlugin,
		BasicInput:  basicInput,
		address:     address,
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
			t.goHandleMessages(subctx, conn, cancel)
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

			if err := t.Write(ctx, message); err != nil {
				t.Errorw("Failed to write entry", zap.Any("error", err))
			}
		}
	}()
}

// readMessage will read a log message from a TCP connection.
func (t *TCPInput) readMessage(conn net.Conn, reader *bufio.Reader) (string, error) {
	message, err := reader.ReadBytes('\n')
	if err != nil {
		return "", err
	}

	return string(message), nil
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
