package builtin

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
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
	Host                     string `yaml:",omitempty"`
	Port                     int    `yaml:",omitempty"`
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

	address, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", c.Host, c.Port))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve address: %s", err)
	}

	tcpInput := &TCPInput{
		BasicPlugin: basicPlugin,
		BasicInput:  basicInput,
		address:     address,
		connections: make(map[string]net.Conn),
	}
	return tcpInput, nil
}

// TCPInput is a plugin that listens for log entries over tcp.
type TCPInput struct {
	helper.BasicPlugin
	helper.BasicInput
	address *net.TCPAddr

	listener    net.Listener
	connections map[string]net.Conn

	cancel    context.CancelFunc
	context   context.Context
	waitGroup *sync.WaitGroup
}

// Start will start listening for log entries over tcp.
func (t *TCPInput) Start() error {
	listener, err := net.ListenTCP("tcp", t.address)
	if err != nil {
		return fmt.Errorf("failed to listen on interface: %w", err)
	}

	t.listener = listener
	t.context, t.cancel = context.WithCancel(context.Background())
	t.waitGroup = &sync.WaitGroup{}
	t.goAcceptConnections()
	return nil
}

// goAcceptConnections will listen for tcp connections in a go routine.
func (t *TCPInput) goAcceptConnections() {
	t.waitGroup.Add(1)

	go func() {
		defer t.waitGroup.Done()

		for {
			conn, err := t.listener.Accept()
			if err != nil && t.isExpectedClose(err) {
				break
			} else if err != nil {
				t.Errorf("Failed to accept connection: %s", err)
			}

			t.addConnection(conn)
			t.goHandleConnection(conn)
		}
	}()
}

// goHandleConnection will read logs from a tcp connection in a go routine.
func (t *TCPInput) goHandleConnection(conn net.Conn) {
	t.waitGroup.Add(1)

	go func() {
		defer t.waitGroup.Done()
		defer t.closeConnection(conn)

		for {
			entry, err := t.readEntry(conn)
			if err != nil && t.isExpectedClose(err) {
				break
			} else if err != nil {
				t.Errorf("Failed to read from connection: %s", err)
			}

			if err := t.Output.Process(entry); err != nil {
				t.Debugf("Output %s failed to process entry: %s", t.OutputID, err)
			}
		}
	}()
}

// readEntry will read a log entry from a TCP connection.
func (t *TCPInput) readEntry(conn net.Conn) (*entry.Entry, error) {
	reader := bufio.NewReaderSize(conn, 1024*64)
	message, err := reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	entry := entry.CreateBasicEntry()
	entry.Record["message"] = string(message)
	entry.Record["address"] = conn.RemoteAddr().String()
	return entry, nil
}

// addConnection will add a connection to the registry.
func (t *TCPInput) addConnection(conn net.Conn) {
	t.connections[conn.RemoteAddr().String()] = conn
}

// isExpectedClose will determine if an error was the result of a closed connection.
func (t *TCPInput) isExpectedClose(err error) bool {
	return strings.Contains(err.Error(), "closed network connection")
}

// closeConnection will close a connection and remove it from the registry.
func (t *TCPInput) closeConnection(conn net.Conn) {
	delete(t.connections, conn.RemoteAddr().String())
	_ = conn.Close()
}

// Stop will stop listening for log entries over TCP.
func (t *TCPInput) Stop() error {
	t.cancel()

	for _, conn := range t.connections {
		t.closeConnection(conn)
	}

	if err := t.listener.Close(); err != nil {
		return err
	}

	t.waitGroup.Wait()
	return nil
}
