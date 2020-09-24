package newrelic

import (
	"compress/gzip"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator/helper"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
)

func TestNewRelicOutput(t *testing.T) {
	cases := []struct {
		name     string
		input    *entry.Entry
		expected string
	}{
		{
			"Simple",
			&entry.Entry{
				Timestamp: time.Date(2016, 10, 10, 8, 58, 52, 0, time.UTC),
				Record:    "test",
			},
			`[{"common":{"attributes":{"plugin":{"type":"stanza","version":"unknown"}}},"logs":[{"timestamp":1476089932000,"attributes":{"labels":null,"resource":null,"severity":"default"},"message":"test"}]}]` + "\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ln := newListener()
			addr, err := ln.start()
			require.NoError(t, err)
			defer ln.stop()

			cfg := NewNewRelicOutputConfig("test")
			cfg.BaseURI = fmt.Sprintf("http://%s/log/v1", addr)
			cfg.FlusherConfig.MaxWait = helper.NewDuration(time.Millisecond)
			cfg.APIKey = "testkey"

			op, err := cfg.Build(testutil.NewBuildContext(t))
			require.NoError(t, err)
			require.NoError(t, op.Start())
			require.NoError(t, op.Process(context.Background(), tc.input))
			defer op.Stop()

			testConnection := `[{"common":{"attributes":{"plugin":{"type":"stanza","version":"unknown"}}},"logs":[]}]` + "\n"
			select {
			case body := <-ln.requestBodies:
				require.Equal(t, testConnection, string(body))
			case <-time.After(time.Minute):
				require.FailNow(t, "Timed out waiting for test connection")
			}

			select {
			case body := <-ln.requestBodies:
				require.Equal(t, tc.expected, string(body))
			case <-time.After(time.Minute):
				require.FailNow(t, "Timed out waiting for request")
			}
		})
	}
}

type listener struct {
	server        *http.Server
	requestBodies chan []byte
}

func newListener() *listener {
	requests := make(chan []byte, 100)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handle(requests))

	return &listener{
		server: &http.Server{
			Handler: mux,
		},
		requestBodies: requests,
	}
}

func (l *listener) start() (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}

	go func() {
		l.server.Serve(ln)
	}()

	// Wait for the server to start to avoid an error log
	time.Sleep(100 * time.Millisecond)

	return ln.Addr().String(), nil
}

func (l *listener) stop() {
	l.server.Shutdown(context.Background())
}

func handle(ch chan []byte) func(rw http.ResponseWriter, req *http.Request) {
	return func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(200)
		rw.Write([]byte(`{}`))

		rd, err := gzip.NewReader(req.Body)
		if err != nil {
			panic(err)
		}
		body, err := ioutil.ReadAll(rd)
		if err != nil {
			panic(err)
		}
		ch <- body
	}
}
