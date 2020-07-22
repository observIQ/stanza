package input

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/observiq/carbon/operator"
	"github.com/observiq/carbon/operator/helper"
)

func init() {
	operator.Register("udp_input", func() operator.Builder { return NewUDPInputConfig("") })
}

func NewUDPInputConfig(operatorID string) *UDPInputConfig {
	return &UDPInputConfig{
		InputConfig: helper.NewInputConfig(operatorID, "udp_input"),
	}
}

// UDPInputConfig is the configuration of a udp input operator.
type UDPInputConfig struct {
	helper.InputConfig `yaml:",inline"`

	ListenAddress string `json:"listen_address,omitempty" yaml:"listen_address,omitempty"`
}

// Build will build a udp input operator.
func (c UDPInputConfig) Build(context operator.BuildContext) (operator.Operator, error) {
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
	}
	return udpInput, nil
}

// UDPInput is an operator that listens to a socket for log entries.
type UDPInput struct {
	helper.InputOperator
	address *net.UDPAddr

	connection net.PacketConn
	cancel     context.CancelFunc
	waitGroup  *sync.WaitGroup
}

// Start will start listening for messages on a socket.
func (u *UDPInput) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	u.cancel = cancel
	u.waitGroup = &sync.WaitGroup{}

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
	u.waitGroup.Add(1)

	go func() {
		defer u.waitGroup.Done()

		for {
			message, err := u.readMessage()
			if err != nil && u.isExpectedClose(err) {
				u.Debugf("Exiting message handler: %s", err)
				break
			}

			entry := u.NewEntry(message)
			u.Write(ctx, entry)
		}
	}()
}

// readMessage will read log messages from the connection.
func (u *UDPInput) readMessage() (string, error) {
	buffer := make([]byte, 1024)
	n, _, err := u.connection.ReadFrom(buffer)
	if err != nil {
		return "", err
	}

	// Remove trailing characters and NULs
	for ; (n > 0) && (buffer[n-1] < 32); n-- {
	}

	return string(buffer[:n]), nil
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
