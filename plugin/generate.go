package plugin

import (
	"context"
	"time"

	bpla "github.com/bluemedora/bplogagent"
	"go.uber.org/zap"
)

func init() {
	bpla.RegisterConfig("generate", &GenerateConfig{})
}

type GenerateConfig struct {
	Output   string
	Message  map[string]interface{}
	Interval float64
	Count    int
}

type GeneratePlugin struct {
	config GenerateConfig
	output chan<- bpla.Entry

	cancel context.CancelFunc
	*zap.SugaredLogger
}

func (p *GeneratePlugin) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	if p.config.Interval == 0 {
		go p.untimedGenerator(ctx)
	} else {
		go p.timedGenerator(ctx)
	}

	return nil
}

func (p *GeneratePlugin) timedGenerator(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(p.config.Interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case t := <-ticker.C:
			p.output <- bpla.Entry{
				Timestamp: t,
				Record:    copyMap(p.config.Message),
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

		p.output <- bpla.Entry{
			Timestamp: time.Now(),
			Record:    copyMap(p.config.Message),
		}
	}
}

func (p *GeneratePlugin) Stop() {
	p.cancel()
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
