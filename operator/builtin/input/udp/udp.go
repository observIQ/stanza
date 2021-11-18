package udp

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/observiq/stanza/v2/operator"
	"github.com/observiq/stanza/v2/operator/helper"
	"go.uber.org/zap"
)

func init() {
	operator.Register("udp_input", func() operator.Builder { return NewUDPInputConfig("") })
}

// NewUDPInputConfig creates a new UDP input config with default values
func NewUDPInputConfig(operatorID string) *UDPInputConfig {
	return &UDPInputConfig{
		InputConfig: helper.NewInputConfig(operatorID, "udp_input"),
	}
}

// UDPInputConfig is the configuration of a udp input operator.
type UDPInputConfig struct {
	helper.InputConfig `yaml:",inline"`

	ListenAddress string `json:"listen_address,omitempty" yaml:"listen_address,omitempty"`
	AddAttributes bool   `json:"add_attributes,omitempty" yaml:"add_attributes,omitempty"`
}

// Build will build a udp input operator.
func (c UDPInputConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	inputOperator, err := c.InputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	if c.ListenAddress == "" {
		return nil, fmt.Errorf("missing required parameter 'listen_address'")
	}

	address, err := net.ResolveUDPAddr("udp", c.ListenAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve listen_address: %s", err)
	}

	udpInput := &UDPInput{
		InputOperator: inputOperator,
		address:       address,
		buffer:        make([]byte, 8192),
		addAttributes: c.AddAttributes,
	}
	return []operator.Operator{udpInput}, nil
}

// UDPInput is an operator that listens to a socket for log entries.
type UDPInput struct {
	buffer []byte
	helper.InputOperator
	address       *net.UDPAddr
	addAttributes bool

	connection net.PacketConn
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// Start will start listening for messages on a socket.
func (u *UDPInput) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	u.cancel = cancel

	conn, err := net.ListenUDP("udp", u.address)
	if err != nil {
		return fmt.Errorf("failed to open connection: %s", err)
	}
	u.connection = conn

	u.goHandleMessages(ctx)
	return nil
}

// goHandleMessages will handle messages from a udp connection.
func (u *UDPInput) goHandleMessages(ctx context.Context) {
	u.wg.Add(1)

	go func() {
		defer u.wg.Done()

		for {
			message, remoteAddr, err := u.readMessage()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					u.Errorw("Failed reading messages", zap.Error(err))
				}
				break
			}

			entry, err := u.NewEntry(message)
			if err != nil {
				u.Errorw("Failed to create entry", zap.Error(err))
				continue
			}

			if u.addAttributes {
				entry.AddAttribute("net.transport", "IP.UDP")
				if addr, ok := u.connection.LocalAddr().(*net.UDPAddr); ok {
					entry.AddAttribute("net.host.ip", addr.IP.String())
					entry.AddAttribute("net.host.port", strconv.FormatInt(int64(addr.Port), 10))
				}

				if addr, ok := remoteAddr.(*net.UDPAddr); ok {
					entry.AddAttribute("net.peer.ip", addr.IP.String())
					entry.AddAttribute("net.peer.port", strconv.FormatInt(int64(addr.Port), 10))
				}
			}

			u.Write(ctx, entry)
		}
	}()
}

// readMessage will read log messages from the connection.
func (u *UDPInput) readMessage() (string, net.Addr, error) {
	n, addr, err := u.connection.ReadFrom(u.buffer)
	if err != nil {
		return "", nil, err
	}

	// Remove trailing characters and NULs
	for ; (n > 0) && (u.buffer[n-1] < 32); n-- {
	}

	return string(u.buffer[:n]), addr, nil
}

// Stop will stop listening for udp messages.
func (u *UDPInput) Stop() error {
	u.cancel()
	if u.connection != nil {
		if err := u.connection.Close(); err != nil {
			u.Errorf("failed to close connection, got error: %s", err)
		}
	}
	u.wg.Wait()
	return nil
}
