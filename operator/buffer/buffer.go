package buffer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
)

type Buffer interface {
	Add(context.Context, *entry.Entry) error
	Read([]*entry.Entry) (func(), int, error)
	ReadWait([]*entry.Entry, <-chan time.Time) (func(), int, error)
	Close() error
}

type Config struct {
	Type string `json:"type" yaml:"type"`
	BufferBuilder
}

func NewConfig() Config {
	return Config{
		Type: "memory",
		BufferBuilder: &MemoryBufferConfig{
			MaxEntries: 1 << 20,
		},
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
	var typeStruct struct {
		Type string
	}
	err := unmarshal(&typeStruct)
	if err != nil {
		return err
	}
	bc.Type = typeStruct.Type

	switch bc.Type {
	case "memory":
		mbc := NewMemoryBufferConfig()
		err := unmarshal(mbc)
		if err != nil {
			return err
		}
		bc.BufferBuilder = mbc
	case "disk":
		dbc := NewDiskBufferConfig()
		err := unmarshal(dbc)
		if err != nil {
			return err
		}
		bc.BufferBuilder = dbc
	default:
		return fmt.Errorf("unknown buffer type '%s'", bc.Type)
	}

	return nil
}
