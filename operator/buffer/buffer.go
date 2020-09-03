package buffer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
)

type Buffer interface {
	Add(context.Context, *entry.Entry) error
	Read([]*entry.Entry) (FlushFunc, int, error)
	ReadWait(context.Context, []*entry.Entry) (FlushFunc, int, error)
	Close() error
}

type Config struct {
	BufferBuilder
}

func NewConfig() Config {
	return Config{
		BufferBuilder: NewMemoryBufferConfig(),
	}
}

type BufferBuilder interface {
	Build(context operator.BuildContext, pluginID string) (Buffer, error)
}

func (bc *Config) UnmarshalJSON(data []byte) error {
	return bc.unmarshal(func(dst interface{}) error {
		return json.Unmarshal(data, dst)
	})
}

func (bc *Config) UnmarshalYAML(f func(interface{}) error) error {
	return bc.unmarshal(f)
}

func (bc *Config) unmarshal(unmarshal func(interface{}) error) error {
	var m map[string]interface{}
	err := unmarshal(&m)
	if err != nil {
		return err
	}

	switch m["type"] {
	case "memory":
		bc.BufferBuilder = NewMemoryBufferConfig()
		return unmarshal(bc.BufferBuilder)
	case "disk":
		bc.BufferBuilder = NewDiskBufferConfig()
		return unmarshal(bc.BufferBuilder)
	default:
		return fmt.Errorf("unknown buffer type '%s'", m["type"])
	}
}

type FlushFunc func() error
