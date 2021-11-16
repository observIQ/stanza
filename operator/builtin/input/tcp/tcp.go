package tcp

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jpillora/backoff"
	"github.com/observiq/stanza/v2/operator"
	"github.com/observiq/stanza/v2/operator/helper"
	"go.uber.org/zap"
)

const (
	// minBufferSize is the initial size used for buffering
	// TCP input
	minBufferSize = 64 * 1024

	// DefaultMaxBufferSize is the max buffer sized used
	// if MaxBufferSize is not set
	DefaultMaxBufferSize = 1024 * 1024
)

func init() {
	operator.Register("tcp_input", func() operator.Builder { return NewTCPInputConfig("") })
}

// NewTCPInputConfig creates a new TCP input config with default values
func NewTCPInputConfig(operatorID string) *TCPInputConfig {
	return &TCPInputConfig{
		InputConfig: helper.NewInputConfig(operatorID, "tcp_input"),
	}
}

// TCPInputConfig is the configuration of a tcp input operator.
type TCPInputConfig struct {
	helper.InputConfig `yaml:",inline"`

	MaxBufferSize helper.ByteSize `json:"max_buffer_size,omitempty" yaml:"max_buffer_size,omitempty"`
	ListenAddress string          `json:"listen_address,omitempty" yaml:"listen_address,omitempty"`
	TLS           TLSConfig       `json:"tls,omitempty" yaml:"tls,omitempty"`
	AddLabels     bool            `json:"add_labels,omitempty" yaml:"add_labels,omitempty"`
}

// TLSConfig is the configuration for a TLS listener
type TLSConfig struct {
	// Enable forces the user of TLS
	Enable bool `json:"enable,omitempty" yaml:"enable,omitempty"`

	// Certificate is the file path for the certificate
	Certificate string `json:"certificate,omitempty" yaml:"certificate,omitempty"`

	// PrivateKey is the file path for the private key
	PrivateKey string `json:"private_key,omitempty" yaml:"private_key,omitempty"`

	// MinVersion is the minimum tls version
	MinVersion float32 `json:"min_version,omitempty" yaml:"min_version,omitempty"`
}

// Build will build a tcp input operator.
func (c TCPInputConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	inputOperator, err := c.InputConfig.Build(context)
	if err != nil {
		return nil, err
	}

	// If MaxBufferSize not set, set sane default in order to remain
	// backwards compatible with existing plugins and configurations
	if c.MaxBufferSize == 0 {
		c.MaxBufferSize = DefaultMaxBufferSize
	}

	if c.MaxBufferSize < minBufferSize {
		return nil, fmt.Errorf("invalid value for parameter 'max_buffer_size', must be equal to or greater than %d bytes", minBufferSize)
	}

	if c.ListenAddress == "" {
		return nil, fmt.Errorf("missing required parameter 'listen_address'")
	}

	// validate the input address
	if _, err := net.ResolveTCPAddr("tcp", c.ListenAddress); err != nil {
		return nil, fmt.Errorf("failed to resolve listen_address: %s", err)
	}

	cert := tls.Certificate{}
	if c.TLS.Enable {
		if c.TLS.Certificate == "" {
			return nil, fmt.Errorf("missing required parameter 'certificate', required when TLS is enabled")
		}

		if c.TLS.PrivateKey == "" {
			return nil, fmt.Errorf("missing required parameter 'private_key', required when TLS is enabled")
		}

		c, err := tls.LoadX509KeyPair(c.TLS.Certificate, c.TLS.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to load tls certificate: %w", err)
		}
		cert = c
	}

	var tlsMinVersion uint16
	switch c.TLS.MinVersion {
	case 0, 1.0:
		// TLS 1.0 is the default version implemented by cypto/tls https://pkg.go.dev/crypto/tls#Config
		tlsMinVersion = tls.VersionTLS10
	case 1.1:
		tlsMinVersion = tls.VersionTLS11
	case 1.2:
		tlsMinVersion = tls.VersionTLS12
	case 1.3:
		tlsMinVersion = tls.VersionTLS13
	default:
		return nil, fmt.Errorf("unsupported tls version: %f", c.TLS.MinVersion)
	}

	tcpInput := &TCPInput{
		InputOperator: inputOperator,
		address:       c.ListenAddress,
		maxBufferSize: int(c.MaxBufferSize),
		addLabels:     c.AddLabels,
		tlsEnable:     c.TLS.Enable,
		tlsKeyPair:    cert,
		tlsMinVersion: tlsMinVersion,
		backoff: backoff.Backoff{
			Min:    100 * time.Millisecond,
			Max:    3 * time.Second,
			Factor: 2,
			Jitter: false,
		},
	}
	return []operator.Operator{tcpInput}, nil
}

// TCPInput is an operator that listens for log entries over tcp.
type TCPInput struct {
	helper.InputOperator
	address       string
	maxBufferSize int
	addLabels     bool
	tlsEnable     bool
	tlsKeyPair    tls.Certificate
	tlsMinVersion uint16
	backoff       backoff.Backoff

	listener net.Listener
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// Start will start listening for log entries over tcp.
func (t *TCPInput) Start() error {
	if err := t.configureListener(); err != nil {
		return fmt.Errorf("failed to listen on interface: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel
	t.goListen(ctx)
	return nil
}

func (t *TCPInput) configureListener() error {
	if !t.tlsEnable {
		listener, err := net.Listen("tcp", t.address)
		if err != nil {
			return fmt.Errorf("failed to configure tcp listener: %w", err)
		}
		t.listener = listener
		return nil
	}

	// #nosec - User to specify tls minimum version
	config := tls.Config{
		Certificates: []tls.Certificate{t.tlsKeyPair},
		MinVersion:   t.tlsMinVersion,
	}
	config.Time = func() time.Time { return time.Now() }
	config.Rand = rand.Reader

	listener, err := tls.Listen("tcp", t.address, &config)
	if err != nil {
		return fmt.Errorf("failed to configure tls listener: %w", err)
	}

	t.listener = listener
	return nil
}

// goListenn will listen for tcp connections.
func (t *TCPInput) goListen(ctx context.Context) {
	t.wg.Add(1)

	go func() {
		defer t.wg.Done()

		for {
			conn, err := t.listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					t.Debugw("Listener accept error", zap.Error(err))
					time.Sleep(t.backoff.Duration())
					continue
				}
			}
			t.backoff.Reset()

			t.Debugf("Received connection: %s", conn.RemoteAddr().String())
			subctx, cancel := context.WithCancel(ctx)
			t.goHandleClose(subctx, conn)
			t.goHandleMessages(subctx, conn, cancel)
		}
	}()
}

// goHandleClose will wait for the context to finish before closing a connection.
func (t *TCPInput) goHandleClose(ctx context.Context, conn net.Conn) {
	t.wg.Add(1)

	go func() {
		defer t.wg.Done()
		<-ctx.Done()
		t.Debugf("Closing connection: %s", conn.RemoteAddr().String())
		if err := conn.Close(); err != nil {
			t.Errorf("Failed to close connection: %s", err)
		}
	}()
}

// goHandleMessages will handles messages from a tcp connection.
func (t *TCPInput) goHandleMessages(ctx context.Context, conn net.Conn, cancel context.CancelFunc) {
	t.wg.Add(1)

	go func() {
		defer t.wg.Done()
		defer cancel()

		// Initial buffer size is 64k
		buf := make([]byte, 0, 64*1024)
		scanner := bufio.NewScanner(conn)
		scanner.Buffer(buf, t.maxBufferSize*1024)
		for scanner.Scan() {
			entry, err := t.NewEntry(scanner.Text())
			if err != nil {
				t.Errorw("Failed to create entry", zap.Error(err))
				continue
			}

			if t.addLabels {
				entry.AddLabel("net.transport", "IP.TCP")
				if addr, ok := conn.RemoteAddr().(*net.TCPAddr); ok {
					entry.AddLabel("net.peer.ip", addr.IP.String())
					entry.AddLabel("net.peer.port", strconv.FormatInt(int64(addr.Port), 10))
				}

				if addr, ok := conn.LocalAddr().(*net.TCPAddr); ok {
					entry.AddLabel("net.host.ip", addr.IP.String())
					entry.AddLabel("net.host.port", strconv.FormatInt(int64(addr.Port), 10))
				}
			}

			t.Write(ctx, entry)
		}
		if err := scanner.Err(); err != nil {
			// Use of closed network connection is expected if the context is canceled
			if strings.Contains(err.Error(), "use of closed network connection") {
				select {
				case <-ctx.Done():
					return
				default:
				}
			}
			t.Errorw("Scanner error", zap.Error(err))
		}
	}()
}

// Stop will stop listening for log entries over TCP.
func (t *TCPInput) Stop() error {
	t.cancel()

	if t.listener != nil {
		if err := t.listener.Close(); err != nil {
			return err
		}
	}

	t.wg.Wait()
	return nil
}
