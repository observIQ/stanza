package httpevents

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	jsoniter "github.com/json-iterator/go"
	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/operator/builtin/input/tcp"
	"github.com/observiq/stanza/operator/helper"
)

const (
	DefaultTimeout     = time.Second * 20
	DefaultIdleTimeout = time.Second * 60
	DefaultMaxBodySize = 10000000 // 10 megabyte
)

func init() {
	operator.Register("http_input", func() operator.Builder { return NewHTTPInputConfig("") })
}

// NewHTTPInputConfig creates a new HTTP input config with default values
func NewHTTPInputConfig(operatorID string) *HTTPInputConfig {
	return &HTTPInputConfig{
		InputConfig:   helper.NewInputConfig(operatorID, "http_input"),
		IdleTimeout:   helper.NewDuration(DefaultIdleTimeout),
		ReadTimeout:   helper.NewDuration(DefaultTimeout),
		WriteTimeout:  helper.NewDuration(DefaultTimeout),
		MaxHeaderSize: helper.ByteSize(http.DefaultMaxHeaderBytes),
		MaxBodySize:   helper.ByteSize(DefaultMaxBodySize),
	}
}

// HTTPInputConfig is the configuration of a http input operator.
type HTTPInputConfig struct {
	helper.InputConfig `yaml:",inline"`

	ListenAddress string          `json:"listen_address,omitempty"  yaml:"listen_address,omitempty"`
	TLS           tcp.TLSConfig   `json:"tls,omitempty"             yaml:"tls,omitempty"`
	IdleTimeout   helper.Duration `json:"idle_timeout,omitempty"    yaml:"idle_timeout,omitempty"`
	ReadTimeout   helper.Duration `json:"read_timeout,omitempty"    yaml:"read_timeout,omitempty"`
	WriteTimeout  helper.Duration `json:"write_timeout,omitempty"   yaml:"write_timeout,omitempty"`
	MaxHeaderSize helper.ByteSize `json:"max_header_size,omitempty" yaml:"max_header_size,omitempty"`
	MaxBodySize   helper.ByteSize `json:"max_body_size,omitempty"   yaml:"max_body_size,omitempty"`
}

// Build will build a http input operator.
func (c HTTPInputConfig) Build(context operator.BuildContext) ([]operator.Operator, error) {
	inputOperator, err := c.InputConfig.Build(context)
	if err != nil {
		return nil, err
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

	// Allow user to configure 0 for timeout values as this is the default behavior
	if c.IdleTimeout.Seconds() < 0 {
		return nil, fmt.Errorf("idle_timeout cannot be less than 0")
	}
	if c.ReadTimeout.Seconds() < 0 {
		return nil, fmt.Errorf("read_timeout cannot be less than 0")
	}
	if c.WriteTimeout.Seconds() < 0 {
		return nil, fmt.Errorf("write_timeout cannot be less than 0")
	}

	// Allow user to configure 0 for max header size as this is the default behavior
	if c.MaxHeaderSize < 0 {
		return nil, fmt.Errorf("max_header_size cannot be less than 0")
	}

	if c.MaxBodySize < 1 {
		return nil, fmt.Errorf("max_body_size cannot be less than 1 byte")
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

	httpInput := &HTTPInput{
		InputOperator: inputOperator,
		server: http.Server{
			Addr: c.ListenAddress,
			TLSConfig: &tls.Config{
				MinVersion:   tlsMinVersion,
				Certificates: []tls.Certificate{cert},
			},
			ReadTimeout:       c.ReadTimeout.Raw(),
			ReadHeaderTimeout: c.ReadTimeout.Raw(),
			WriteTimeout:      c.WriteTimeout.Raw(),
			IdleTimeout:       c.IdleTimeout.Raw(),
			/*
				 This value is padded with 4096 bytes
				 https://cs.opensource.google/go/go/+/refs/tags/go1.17.1:src/net/http/server.go;l=865

				func (srv *Server) initialReadLimitSize() int64 {
					return int64(srv.maxHeaderBytes()) + 4096 // bufio slop
				}
			*/
			MaxHeaderBytes: int(c.MaxHeaderSize),
			TLSNextProto:   nil, // This should be configured if we want HTTP/2 support
			ConnState:      nil,
			ErrorLog:       nil, // TODO: logger logs http server errors
			BaseContext:    nil,
			ConnContext:    nil,
		},
		maxBodySize: int64(c.MaxBodySize),
		json:        jsoniter.ConfigFastest,
	}
	return []operator.Operator{httpInput}, nil
}

// HTTPInput is an operator that listens for log entries over http.
type HTTPInput struct {
	helper.InputOperator
	server      http.Server
	json        jsoniter.API
	maxBodySize int64

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// Start will start listening for log entries over http.
func (t *HTTPInput) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel
	t.goListen(ctx)
	return nil
}

// goListenn will listen for http connections.
func (t *HTTPInput) goListen(ctx context.Context) {
	t.Debugf("using server config: %d", t.server.MaxHeaderBytes)

	t.wg.Add(1)

	entryCreateMethods := []string{"POST", "PUT"}

	m := mux.NewRouter()
	m.HandleFunc("/", t.goHandleMessages).Methods(entryCreateMethods...)
	m.HandleFunc("/health", t.health).Methods("GET")
	t.server.Handler = m

	// TODO: Provide http server with a cancelable context so we dont need this go routine
	go func() {
		defer t.wg.Done()
		for {
			select {
			case <-ctx.Done():
				t.Debugf("Triggering http server shutdown")
				ctx, _ := context.WithTimeout(context.Background(), time.Second*10)
				if err := t.server.Shutdown(ctx); err != nil {
					t.Errorf("error while shutting down http server: %s", err)
				}
				return
			default:
				time.Sleep(time.Second * 2)
			}
		}
	}()

	// server go routine runs the http server
	go func() {
		t.Debugf("Starting http server on socket %s", t.server.Addr)
		if err := t.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Errorf("http server failed: %s", err)
			return
		}
		t.Debugf("Http server shutdown finished")
	}()
}

// Stop will stop listening for log entries over http.
func (t *HTTPInput) Stop() error {
	t.cancel()
	t.wg.Wait()
	return nil
}

// goHandleMessages will handles messages from a http connection.
func (t *HTTPInput) goHandleMessages(w http.ResponseWriter, req *http.Request) {
	t.Debugf("Handling incoming entry %s request from %s", req.Method, req.RemoteAddr)

	t.wg.Add(1)

	ctx, cancel := context.WithCancel(req.Context())

	defer t.wg.Done()
	defer cancel()

	req.Body = http.MaxBytesReader(w, req.Body, t.maxBodySize)
	decoder := t.json.NewDecoder(req.Body) // TODO: limit the size of this payload
	m := make(map[string]interface{})
	if err := decoder.Decode(&m); err != nil {
		t.Errorf("failed to decode http %s request from %s", req.Method, req.RemoteAddr)
		w.WriteHeader(http.StatusBadRequest) // TODO: IS this a valid status code for an invalid map?
		w.Write([]byte("invalid payload"))
		return
	}

	entry, err := t.NewEntry(m)
	if err != nil {
		t.Errorf("failed to create entry from http %s request from %s", req.Method, req.RemoteAddr)
		w.WriteHeader(http.StatusInternalServerError) // Stanza should have no trouble creating an entry from map[string]interface{}
		w.Write([]byte("internal server error"))
		return
	}

	if err := addPeerLabels(req.RemoteAddr, entry); err != nil {
		t.Errorf("failed to set net.peer labels: %s", err)
	}

	if err := addHostLabels(req.Host, entry); err != nil {
		t.Errorf("failed to set net.host labels: %s", err)
	}

	if err := addProtoLabels(req.Proto, entry); err != nil {
		t.Errorf("failed to set protocol and protocol_version labels: %s", err)
	}

	addHeaderLabels(req.Header, entry)

	t.Write(ctx, entry)
	w.WriteHeader(http.StatusCreated) // http status 201
}

func addPeerLabels(remoteAddr string, entry *entry.Entry) error {
	ip, port, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return fmt.Errorf("failed to parse %s into ip and port: %s", remoteAddr, err)
	}
	entry.AddLabel("net.peer.ip", ip)
	entry.AddLabel("net.peer.port", port)
	return nil
}

func addHostLabels(host string, entry *entry.Entry) error {
	ip, port, err := net.SplitHostPort(host)
	if err != nil {
		return fmt.Errorf("failed to parse %s into ip and port: %s", host, err)
	}
	entry.AddLabel("net.host.ip", ip)
	entry.AddLabel("net.host.port", port)
	return nil
}

func addProtoLabels(proto string, entry *entry.Entry) error {
	p := strings.Split(proto, "/")
	if len(p) != 2 {
		return fmt.Errorf("failed to parse %s", proto)
	}
	entry.AddLabel("protocol", p[0])

	if _, err := strconv.ParseFloat(p[1], 32); err != nil {
		return fmt.Errorf("failed to parse %s as protocol_version", p[1])
	}
	entry.AddLabel("protocol_version", p[1])

	return nil
}

func addHeaderLabels(headers http.Header, entry *entry.Entry) {
	for k, v := range headers {
		k = strings.ToLower(k)
		entry.AddLabel(k, strings.Join(v, ","))
	}
}

func (t *HTTPInput) health(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
}
