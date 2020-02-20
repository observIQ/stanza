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
	Output  string
	Message map[string]interface{}
	Rate    float64
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

	go func() {
		ticker := time.NewTicker(time.Duration(1.0/p.config.Rate) * time.Second)
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
	}()

	return nil
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
