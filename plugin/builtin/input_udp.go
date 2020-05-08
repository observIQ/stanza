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
	"go.uber.org/zap"
)

func init() {
	plugin.Register("udp_input", &UDPInputConfig{})
}

// UDPInputConfig is the configuration of a udp input plugin.
type UDPInputConfig struct {
	helper.BasicPluginConfig `mapstructure:",squash" yaml:",inline"`
	helper.BasicInputConfig  `mapstructure:",squash" yaml:",inline"`

	ListenAddress string              `mapstructure:"listen_address" json:"listen_address,omitempty" yaml:"listen_address,omitempty"`
	MessageField  entry.FieldSelector `mapstructure:"message_field"  json:"message_field,omitempty"  yaml:"message_field,omitempty,flow"`
	SourceField   entry.FieldSelector `mapstructure:"source_field"   json:"source_field,omitempty"   yaml:"source_field,omitempty,flow"`
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

	if c.ListenAddress == "" {
		return nil, fmt.Errorf("missing field 'listen_address'")
	}

	if c.MessageField == nil {
		fs := entry.FieldSelector([]string{"message"})
		c.MessageField = fs
	}

	address, err := net.ResolveUDPAddr("udp", c.ListenAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve listen_address: %s", err)
	}

	udpInput := &UDPInput{
		BasicPlugin:  basicPlugin,
		BasicInput:   basicInput,
		address:      address,
		messageField: c.MessageField,
		sourceField:  c.SourceField,
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
	waitGroup  *sync.WaitGroup

	messageField entry.FieldSelector
	sourceField  entry.FieldSelector
}

// Start will start listening for messages on a socket.
func (u *UDPInput) Start() error {
	_, u.cancel = context.WithCancel(context.Background())
	u.waitGroup = &sync.WaitGroup{}

	conn, err := net.ListenUDP("udp", u.address)
	if err != nil {
		return fmt.Errorf("failed to open connection: %s", err)
	}
	u.connection = conn

	u.goHandleMessages()
	return nil
}

// goHandleMessages will handle messages from a udp connection.
func (u *UDPInput) goHandleMessages() {
	u.waitGroup.Add(1)

	go func() {
		defer u.waitGroup.Done()

		for {
			entry, err := u.readEntry()
			if err != nil && u.isExpectedClose(err) {
				u.Debugf("Exiting message handler: %s", err)
				break
			}

			if err := u.Output.Process(entry); err != nil {
				u.Errorw("Output failed to process entry", zap.Any("error", err))
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

	entry := entry.NewEntry()
	entry.Set(u.messageField, buffer[:n])
	if u.sourceField != nil {
		entry.Set(u.sourceField, address.String())
	}
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
