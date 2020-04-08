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
	plugin.Register("socket_input", &GenerateInputConfig{})
}

// SocketInputConfig is the configuration of a socket input plugin.
type SocketInputConfig struct {
	helper.BasicPluginConfig `mapstructure:",squash" yaml:",inline"`
	helper.BasicInputConfig  `mapstructure:",squash" yaml:",inline"`
	Mode                     string
	Address                  string
}

// Build will build a socket input plugin.
func (c SocketInputConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	basicPlugin, err := c.BasicPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, err
	}

	basicInput, err := c.BasicInputConfig.Build()
	if err != nil {
		return nil, err
	}

	if err := c.validateParams(); err != nil {
		return nil, err
	}

	socketInput := &SocketInput{
		BasicPlugin: basicPlugin,
		BasicInput:  basicInput,
		mode:        c.Mode,
		address:     c.Address,
	}
	return socketInput, nil
}

// validateParams will validate the mode and address parameters.
func (c SocketInputConfig) validateParams() error {
	switch c.Mode {
	case "udp":
		return c.validateUDPAddr()
	case "tcp":
		return c.validateTCPAddr()
	case "unix":
		return c.validateUnixAddr()
	default:
		return fmt.Errorf("invalid mode %s", c.Mode)
	}
}

// validateUDPAddr will determine if the address is valid for udp.
func (c SocketInputConfig) validateUDPAddr() error {
	if _, err := net.ResolveUDPAddr("udp", c.Address); err != nil {
		return fmt.Errorf("failed to resolve %s udp address: %s", c.Address, err)
	}
	return nil
}

// validateUDPAddr will determine if the address is valid for tcp.
func (c SocketInputConfig) validateTCPAddr() error {
	if _, err := net.ResolveTCPAddr("tcp", c.Address); err != nil {
		return fmt.Errorf("failed to resolve %s tcp address: %s", c.Address, err)
	}
	return nil
}

// validateUDPAddr will determine if the address is valid for unix.
func (c SocketInputConfig) validateUnixAddr() error {
	if _, err := net.ResolveUnixAddr("unix", c.Address); err != nil {
		return fmt.Errorf("failed to resolve %s unix address: %s", c.Address, err)
	}
	return nil
}

// SocketInput is a plugin that listens to a socket for log entries.
type SocketInput struct {
	helper.BasicPlugin
	helper.BasicInput
	mode    string
	address string

	listener    net.Listener
	connections map[string]net.Conn

	cancel    context.CancelFunc
	context   context.Context
	waitGroup *sync.WaitGroup
}

// Start will start listening for messages on a socket.
func (s *SocketInput) Start() error {
	listener, err := net.Listen(s.mode, s.address)
	if err != nil {
		return fmt.Errorf("failed to listen on interface %s: %w", s.address, err)
	}

	s.listener = listener
	s.context, s.cancel = context.WithCancel(context.Background())
	s.waitGroup = &sync.WaitGroup{}
	s.goAcceptConnections()
	return nil
}

// goAcceptConnections will listen for incoming connections in a go routine.
func (s *SocketInput) goAcceptConnections() {
	s.waitGroup.Add(1)

	go func() {
		defer s.waitGroup.Done()
		pluginStopped := s.context.Done()

		for {
			select {
			case <-pluginStopped:
				// Stop accepting connections.
				break
			default:
				// Continue listening for connections.
			}

			// Blocks until a connection is accepted or an error is thrown.
			conn, err := s.listener.Accept()
			if err != nil {
				// Debug only, because this will return an error when closing the listener.
				s.Debugf("failed to accept connection: %s", err)
			} else {
				s.addConnection(conn)
				s.goHandleConnection(conn)
			}
		}
	}()
}

// goHandleConnections will handle messages from a connection in a go routine.
func (s *SocketInput) goHandleConnection(conn net.Conn) {
	s.waitGroup.Add(1)

	go func() {
		defer s.waitGroup.Done()
		defer s.closeConnection(conn)
		reader := bufio.NewReaderSize(conn, 1024*64)

		for {
			message, err := reader.ReadBytes('\n')
			if err != nil {
				// Debug only, because this will return an error when the connection closes.
				s.Debugf("failed to handle connection: %s", err)
				return
			}
			s.sendToOutput(message)
		}
	}()
}

// sendToOutput will send a socket message to the connected output.
func (s *SocketInput) sendToOutput(message []byte) {
	entry := entry.CreateBasicEntry(message)
	if err := s.Output.Process(&entry); err != nil {
		s.Errorf("output %s failed to process entry: %s", err)
	}
}

func (s *SocketInput) addConnection(conn net.Conn) {
	s.connections[conn.RemoteAddr().String()] = conn
}

func (s *SocketInput) closeConnection(conn net.Conn) {
	delete(s.connections, conn.RemoteAddr().String())
	_ = conn.Close()
}

// Stop will stop listening for messages on a socket.
func (s *SocketInput) Stop() error {
	s.cancel()

	for _, conn := range s.connections {
		s.closeConnection(conn)
	}

	if err := s.listener.Close(); err != nil {
		return err
	}

	s.waitGroup.Wait()
	return nil
}
