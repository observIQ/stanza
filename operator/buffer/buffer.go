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

type BufferConfig struct {
	Type string `json:"type" yaml:"type"`
	BufferBuilder
}

type BufferBuilder interface {
	Build(context *operator.BuildContext) Buffer
}

func (bc *BufferConfig) UnmarshalJSON(data []byte) error {
	return bc.unmarshal(func(dst interface{}) error {
		return json.Unmarshal(data, dst)
	})
}

func (bc *BufferConfig) UnmarshalYAML(f func(interface{}) error) error {
	return bc.unmarshal(f)
}

func (bc *BufferConfig) unmarshal(unmarshal func(interface{}) error) error {
	err := unmarshal(bc)
	if err != nil {
		return err
	}

	switch bc.Type {
	case "memory":
		mbc := NewMemoryBufferConfig()
		err := unmarshal(mbc)
		if err != nil {
			return err
		}
		bc.BufferBuilder = mbc
	case "disk":
		dbc := NewMemoryBufferConfig()
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
