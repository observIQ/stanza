package builtin

import (
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
	plugin.Register("udp_input", &UDPInputConfig{})
}

// UDPInputConfig is the configuration of a udp input plugin.
type UDPInputConfig struct {
	helper.BasicPluginConfig `mapstructure:",squash" yaml:",inline"`
	helper.BasicInputConfig  `mapstructure:",squash" yaml:",inline"`
	Host                     string `yaml:",omitempty"`
	Port                     int    `yaml:",omitempty"`
}

// Build will build a udp input plugin.
func (c UDPInputConfig) Build(context plugin.BuildContext) (plugin.Plugin, error) {
	basicPlugin, err := c.BasicPluginConfig.Build(context.Logger)
	if err != nil {
		return nil, err
	}

	basicInput, err := c.BasicInputConfig.Build()
	if err != nil {
		return nil, err
	}

	if c.Port == 0 {
		return nil, fmt.Errorf("missing field 'port'")
	}

	address, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", c.Host, c.Port))
	if err != nil {
		return nil, fmt.Errorf("failed to resolve address: %s", err)
	}

	udpInput := &UDPInput{
		BasicPlugin: basicPlugin,
		BasicInput:  basicInput,
		address:     address,
	}
	return udpInput, nil
}

// UDPInput is a plugin that listens to a socket for log entries.
type UDPInput struct {
	helper.BasicPlugin
	helper.BasicInput
	address *net.UDPAddr

	connection net.PacketConn
	cancel     context.CancelFunc
	context    context.Context
	waitGroup  *sync.WaitGroup
}

// Start will start listening for messages on a socket.
func (u *UDPInput) Start() error {
	u.context, u.cancel = context.WithCancel(context.Background())
	u.waitGroup = &sync.WaitGroup{}

	conn, err := net.ListenUDP("udp", u.address)
	if err != nil {
		return fmt.Errorf("failed to open connection: %s", err)
	}
	u.connection = conn

	u.goRead()
	return nil
}

// goRead will read entries from a connection in a go routine.
func (u *UDPInput) goRead() {
	u.waitGroup.Add(1)

	go func() {
		defer u.waitGroup.Done()

		for {
			entry, err := u.readEntry()
			if err != nil && u.isExpectedClose(err) {
				break
			} else if err != nil {
				u.Errorf("Failed to read from connection: %s", err)
			}

			if err := u.Output.Process(entry); err != nil {
				u.Debugf("Output %s failed to process entry: %s", u.OutputID, err)
			}
		}
	}()
}

// readEntry will read log entries from the connection.
func (u *UDPInput) readEntry() (*entry.Entry, error) {
	buffer := make([]byte, 1024)
	n, address, err := u.connection.ReadFrom(buffer)
	if err != nil {
		return nil, err
	}

	// Remove trailing characters and NULs
	for ; (n > 0) && (buffer[n-1] < 32); n-- {
	}

	entry := entry.CreateBasicEntry()
	entry.Record["message"] = string(buffer[:n])
	entry.Record["address"] = address.String()
	return entry, nil
}

// isExpectedClose will determine if an error was the result of a closed connection.
func (u *UDPInput) isExpectedClose(err error) bool {
	return strings.Contains(err.Error(), "closed network connection")
}

// Stop will stop listening for udp messages.
func (u *UDPInput) Stop() error {
	u.cancel()
	u.connection.Close()
	u.waitGroup.Wait()
	return nil
}
