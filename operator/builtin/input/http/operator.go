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

func init() {
	operator.Register("http_input", func() operator.Builder { return NewHTTPInputConfig("") })
}

// HTTPInput is an operator that listens for log entries over http.
type HTTPInput struct {
	helper.InputOperator

	server      http.Server
	json        jsoniter.API
	maxBodySize int64

	auth authMiddleware

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

	entryCreateMethods := []string{"POST", "PUT"}

	m := mux.NewRouter()
	m.HandleFunc("/", t.goHandleMessages).Methods(entryCreateMethods...)

	if t.auth != nil {
		t.Debugf("using authentication middleware: %s", t.auth.name())
		m.Use(t.auth.auth)
	}

	m.HandleFunc("/health", t.health).Methods("GET")

	t.server.Handler = m

	// shutdown go routine waits for a canceled context before stopping the server
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		for {
			select {
			case <-ctx.Done():
				t.Debugf("Triggering http server shutdown")
				ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
				defer cancel()
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
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		t.Debugf("Starting http server on socket %s", t.server.Addr)
		if err := t.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Errorf("http server failed: %s", err)
			return
		}
		t.Debugf("Http server shutdown finished")
	}()
}

// goHandleMessages will handles messages from a http connection by reading the request
// body and returning http status codes.
func (t *HTTPInput) goHandleMessages(w http.ResponseWriter, req *http.Request) {
	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()

	req.Body = http.MaxBytesReader(nil, req.Body, t.maxBodySize)
	decoder := t.json.NewDecoder(req.Body)
	body := make(map[string]interface{})
	if err := decoder.Decode(&body); err != nil {
		if strings.Contains(err.Error(), "too large") {
			t.Errorf("failed to decode http %s request from %s: %s", req.Method, req.RemoteAddr, err)
			w.WriteHeader(http.StatusRequestEntityTooLarge)
			return
		}
		t.Errorf(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	entry, err := t.parse(body, req)
	if err != nil {
		t.Errorf("failed to create entry from http %s request from %s: %s", req.Method, req.RemoteAddr, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	t.Write(ctx, entry)
	w.WriteHeader(http.StatusCreated)
}

// parse will parse an http request's body into an entry
func (t *HTTPInput) parse(body map[string]interface{}, req *http.Request) (*entry.Entry, error) {
	if body == nil || req == nil {
		return nil, fmt.Errorf("payload and http request must be set")
	}

	payload := make(map[string]interface{})

	const msgKey = "message"
	const bodyKey = "http_body"
	if m, ok := body[msgKey]; ok {
		switch m := m.(type) {
		case string:
			payload[msgKey] = m
			delete(body, msgKey)
		}
	}
	if len(body) > 0 {
		payload[bodyKey] = body
	}

	e, err := t.NewEntry(payload)
	if err != nil {
		return nil, err
	}

	t.addLabels(req, e)

	return e, nil
}

func (t *HTTPInput) addLabels(req *http.Request, entry *entry.Entry) {
	if err := addPeerLabels(req.RemoteAddr, entry); err != nil {
		t.Errorf("failed to set net.peer labels: %s", err)
	}
	if err := addHostLabels(req.Host, entry); err != nil {
		t.Errorf("failed to set net.host labels: %s", err)
	}
	if err := addProtoLabels(req.Proto, entry); err != nil {
		t.Errorf("failed to set protocol and protocol_version labels: %s", err)
	}
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

func (t *HTTPInput) health(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
}
