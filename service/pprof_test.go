package service

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestDefaultPProfProfiler(t *testing.T) {
	prof := newPProfProfiler(context.Background(), zaptest.NewLogger(t).Sugar(), *DefaultPProfConfig())

	// By default, the PprofProfiler doesn't do anything or start any goroutines
	require.NoError(t, prof.Start())

	time.Sleep(time.Second)

	prof.Stop()
}

func TestHTTPPProfProfiler(t *testing.T) {
	conf := *DefaultPProfConfig()
	conf.HTTP.Enabled = true
	conf.HTTP.Port = 0

	prof := newPProfProfiler(context.Background(), zaptest.NewLogger(t).Sugar(), conf)

	srv := newTestHttpServer()

	prof.newServer = func(port int) httpServer {
		assert.Equal(t, 0, port)
		return srv
	}

	require.NoError(t, prof.Start())
	defer prof.Stop()

	var port int
	select {
	case port = <-srv.port:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "Timed out waiting for http server to start!")
	}

	// server should be started, see if we can GET /debug/pprof
	resp, err := http.DefaultClient.Get(fmt.Sprintf("http://localhost:%d/debug/pprof", port))
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, 200, resp.StatusCode)
}

func TestMemPProfProfiler(t *testing.T) {
	f, err := ioutil.TempFile("", "mem-pprof-test")
	require.NoError(t, err)

	filePath := f.Name()
	require.NoError(t, f.Close())
	defer os.RemoveAll(filePath)

	conf := *DefaultPProfConfig()
	conf.Mem.Enabled = true
	conf.Mem.Delay = 10 * time.Millisecond
	conf.Mem.Path = filePath

	prof := newPProfProfiler(context.Background(), zaptest.NewLogger(t).Sugar(), conf)
	require.NoError(t, prof.Start())
	defer prof.Stop()

	time.Sleep(500 * time.Millisecond)

	// Check that the file exists, and has had bytes written to it
	fi, err := os.Stat(filePath)
	require.NoError(t, err)

	require.True(t, fi.Size() > 0, "heap dump size was not greater than 0!")
}

func TestCPUPprofProfiler(t *testing.T) {
	f, err := ioutil.TempFile("", "cpu-pprof-test")
	require.NoError(t, err)

	filePath := f.Name()
	require.NoError(t, f.Close())
	defer os.RemoveAll(filePath)

	conf := *DefaultPProfConfig()
	conf.CPU.Enabled = true
	conf.CPU.Duration = 250 * time.Millisecond
	conf.CPU.Path = filePath

	prof := newPProfProfiler(context.Background(), zaptest.NewLogger(t).Sugar(), conf)
	require.NoError(t, prof.Start())
	defer prof.Stop()

	time.Sleep(500 * time.Millisecond)

	// Check that the file exists, and has had bytes written to it
	fi, err := os.Stat(filePath)
	require.NoError(t, err)
	require.True(t, fi.Size() > 0, "cpu profile size was not greater than 0!")
}

func newTestHttpServer() *testHttpServer {
	return &testHttpServer{
		port: make(chan int),
	}
}

type testHttpServer struct {
	port chan int // If this isn't atomic, the race detector will throw a fit
	srv  http.Server
}

func (t *testHttpServer) ListenAndServe() error {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return err
	}

	t.port <- listener.Addr().(*net.TCPAddr).Port
	return t.srv.Serve(listener)
}

func (t *testHttpServer) Shutdown(ctx context.Context) error {
	t.srv.Shutdown(ctx)
	return nil
}
