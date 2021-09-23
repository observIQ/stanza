package httpevents

import (
	"context"
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

// Stop will stop listening for log entries over http.
func (t *HTTPInput) Stop() error {
	t.cancel()
	t.wg.Wait()
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

// goHandleMessages will handles messages from a http connection.
func (t *HTTPInput) goHandleMessages(w http.ResponseWriter, req *http.Request) {
	t.wg.Add(1)

	ctx, cancel := context.WithCancel(req.Context())

	defer t.wg.Done()
	defer cancel()

	req.Body = http.MaxBytesReader(w, req.Body, t.maxBodySize)
	decoder := t.json.NewDecoder(req.Body)
	m := make(map[string]interface{})
	if err := decoder.Decode(&m); err != nil {
		t.Errorf("failed to decode http %s request from %s: %s", req.Method, req.RemoteAddr, err)
		if strings.Contains(err.Error(), "too large") {
			w.Write([]byte("request body too large"))
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid request, failed to decode json payload"))
		return
	}

	entry, err := t.NewEntry(m)
	if err != nil {
		t.Errorf("failed to create entry from http %s request from %s: %s", req.Method, req.RemoteAddr, err)
		w.WriteHeader(http.StatusInternalServerError)
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
	w.WriteHeader(http.StatusCreated)
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
