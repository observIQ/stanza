package input

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/observiq/carbon/operator"
	"github.com/observiq/carbon/operator/helper"
	"go.uber.org/zap"
)

func init() {
	operator.Register("tcp_input", func() operator.Builder { return NewTCPInputConfig("") })
}

func NewTCPInputConfig(operatorID string) *TCPInputConfig {
	return &TCPInputConfig{
		InputConfig: helper.NewInputConfig(operatorID, "tcp_input"),
	}
}

// TCPInputConfig is the configuration of a tcp input operator.
type TCPInputConfig struct {
	helper.InputConfig `yaml:",inline"`

	ListenAddress string `json:"listen_address,omitempty" yaml:"listen_address,omitempty"`
}

// Build will build a tcp input operator.
func (c TCPInputConfig) Build(context operator.BuildContext) (operator.Operator, error) {
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

// TCPInput is an operator that listens for log entries over tcp.
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
				select {
				case <-ctx.Done():
					return
				default:
					t.Debugw("Listener accept error", zap.Error(err))
				}
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

		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			entry := t.NewEntry(scanner.Text())
			t.Write(ctx, entry)
		}
		if err := scanner.Err(); err != nil {
			t.Errorw("Scanner error", zap.Error(err))
		}
	}()
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
