package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/bluemedora/bplogagent/entry"
	"go.uber.org/zap"
)

func init() {
	RegisterConfig("generate", &GenerateConfig{})
}

// TODO should this be split into a generator and a rate limiter to be more orthogonal?
type GenerateConfig struct {
	DefaultSourceConfig `mapstructure:",squash"`
	Record              map[string]interface{}
	Interval            float64
	Count               int
}

func (c GenerateConfig) Build(logger *zap.SugaredLogger) (Plugin, error) {
	defaultSource, err := c.DefaultSourceConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build default source: %s", err)
	}

	plugin := &GeneratePlugin{
		config:        c,
		SugaredLogger: logger.With("plugin_type", "generate", "plugin_id", c.ID()),
		DefaultSource: defaultSource,
	}
	return plugin, nil
}

type GeneratePlugin struct {
	DefaultSource
	config GenerateConfig

	cancel context.CancelFunc
	*zap.SugaredLogger
}

func (p *GeneratePlugin) Start(wg *sync.WaitGroup) error {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	p.Infow("Starting generate plugin")
	go func() {
		defer wg.Done()
		defer p.Infow("Stopping generate plugin")
		if p.config.Interval == 0 {
			p.untimedGenerator(ctx)
		} else {
			p.timedGenerator(ctx)

		}
	}()

	return nil
}

func (p *GeneratePlugin) Stop() {
	// TODO should this block until exit?
	p.cancel()
}

func (p *GeneratePlugin) timedGenerator(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(p.config.Interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case t := <-ticker.C:
			p.output <- entry.Entry{
				Timestamp: t,
				Record:    copyMap(p.config.Record),
			}
		case <-ctx.Done():
			return
		}
	}
}

func (p *GeneratePlugin) untimedGenerator(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		p.output <- entry.Entry{
			Timestamp: time.Now(),
			Record:    copyMap(p.config.Record),
		}
	}
}

// TODO should this do something different wiht pointers or arrays?
func copyMap(m map[string]interface{}) map[string]interface{} {
	cp := make(map[string]interface{})
	for k, v := range m {
		vm, ok := v.(map[string]interface{})
		if ok {
			cp[k] = copyMap(vm)
		} else {
			cp[k] = v
		}
	}

	return cp
}
