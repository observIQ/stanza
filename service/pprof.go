package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	_ "net/http/pprof" // importing this installs pprof http handlers into the default mux
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"

	"go.uber.org/zap"
)

type PProfConfig struct {
	HTTP PProfHTTPConfig          `yaml:"http"`
	CPU  PProfCPUProfileConfig    `yaml:"cpu"`
	Mem  PProfMemoryProfileConfig `yaml:"mem"`
}

type PProfHTTPConfig struct {
	Enabled bool `yaml:"enabled"`
	Port    int  `yaml:"port"`
}

type PProfCPUProfileConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Path     string        `yaml:"path"`
	Duration time.Duration `yaml:"duration"`
}

type PProfMemoryProfileConfig struct {
	Enabled bool          `yaml:"enabled"`
	Path    string        `yaml:"path"`
	Delay   time.Duration `yaml:"delay"`
}

func DefaultPProfConfig() *PProfConfig {
	return &PProfConfig{
		HTTP: PProfHTTPConfig{
			Port: 6060,
		},
		CPU: PProfCPUProfileConfig{
			Path:     "cpu.prof",
			Duration: time.Minute,
		},
		Mem: PProfMemoryProfileConfig{
			Path:  "mem.prof",
			Delay: 10 * time.Second,
		},
	}
}

type httpServer interface {
	ListenAndServe() error
	Shutdown(context.Context) error
}

type newServerFunc func(port int) httpServer

func defaultNewServerFunc(port int) httpServer {
	srv := &http.Server{}
	srv.Addr = fmt.Sprintf(":%d", port)
	return srv
}

type pprofProfiler struct {
	config    PProfConfig
	logger    *zap.SugaredLogger
	wg        *sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	newServer newServerFunc // Function to create a new server; Used for testing
}

func newPProfProfiler(ctx context.Context, logger *zap.SugaredLogger, config PProfConfig) *pprofProfiler {
	cancelCtx, cancel := context.WithCancel(ctx)
	return &pprofProfiler{
		config:    config,
		wg:        &sync.WaitGroup{},
		logger:    logger,
		cancel:    cancel,
		ctx:       cancelCtx,
		newServer: defaultNewServerFunc,
	}
}

// Start the profiler; Conditionally starts parts depending on user config.
// With the default config, no component is started, and this function is effectively a noop.
func (p *pprofProfiler) Start() error {
	if p.config.HTTP.Enabled {
		p.startHttp()
	}

	if p.config.CPU.Enabled {
		if err := p.startCPUProfile(); err != nil {
			return err
		}
	}

	if p.config.Mem.Enabled {
		p.startMemProfile()
	}

	return nil
}

func (p *pprofProfiler) Stop() {
	p.cancel()
	p.wg.Wait()
}

func (p pprofProfiler) startHttp() {
	p.logger.Debugw("Starting pprof http server", zap.Int("port", p.config.HTTP.Port))

	// pprof endpoints registered by importing net/pprof
	srv := p.newServer(p.config.HTTP.Port)

	// Goroutine to handle serving
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		err := srv.ListenAndServe()
		// ErrServerClosed is the expected error from a Shutdown
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			p.logger.Errorw("Error starting pprof server", zap.Error(err))
		}
	}()

	// Goroutine for shutting down the server when context is cancelled
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		<-p.ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		err := srv.Shutdown(shutdownCtx)
		if err != nil {
			p.logger.Warnw("Error shutting down pprof server", zap.Error(err))
		}
	}()
}

func (p pprofProfiler) startCPUProfile() error {
	p.logger.Debugw("Starting pprof cpu profile", zap.String("path", p.config.CPU.Path), zap.Stringer("duration", p.config.CPU.Duration))

	f, err := os.Create(p.config.CPU.Path)
	if err != nil {
		return err
	}

	if err := pprof.StartCPUProfile(f); err != nil {
		return err
	}

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()

		select {
		case <-p.ctx.Done():
		case <-time.After(p.config.CPU.Duration):
		}

		pprof.StopCPUProfile()
		p.logger.Debugw("Stopped pprof cpu profile", zap.String("path", p.config.CPU.Path))

		if err := f.Close(); err != nil {
			p.logger.Errorw("Failed closing cpu profile file", zap.Error(err))
		}
	}()

	return nil
}

func (p pprofProfiler) startMemProfile() {
	p.logger.Debugw("Setting up pprof memory profile", zap.String("path", p.config.Mem.Path), zap.Stringer("delay", p.config.Mem.Delay))

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()

		select {
		case <-p.ctx.Done(): // We take the heap profile on context cancel if we don't hit the delay
		case <-time.After(p.config.Mem.Delay):
		}

		p.logger.Debugw("Writing heap profile", zap.String("path", p.config.Mem.Path))

		f, err := os.Create(p.config.Mem.Path)
		if err != nil {
			p.logger.Errorw("Failed to create memory profile", zap.Error(err))
			return
		}

		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			p.logger.Errorw("Failed to write memory profile", zap.Error(err))
		}

		if err := f.Close(); err != nil {
			p.logger.Errorw("Failed to close file", zap.Error(err))
		}
	}()
}
