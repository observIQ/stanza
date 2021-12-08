package buffer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/open-telemetry/opentelemetry-log-collection/entry"
)

// ErrBufferClosed is an error to indicate an operation was attempt on a buffer after it was closed
var ErrBufferClosed = errors.New("buffer is closed")

// ErrEntryTooLarge is an error to indicate an entry is too large to ever fit into the buffer.
// That is, the serialized entry is larger than the maximum size of the buffer
var ErrEntryTooLarge = errors.New("entry is too large to fit in buffer")

// Buffer is an interface for an entry buffer
type Buffer interface {
	// Add adds an entry onto the buffer.
	// Is a blocking call if the buffer is full
	Add(context.Context, *entry.Entry) error

	// Read reads from the buffer.
	// Read can be a blocking call depending on the underlying implementation.
	Read(context.Context) ([]*entry.Entry, error)

	// Close runs cleanup code for buffer and may return entries left in the buffer
	// depending on the underlying implementation
	Close() ([]*entry.Entry, error)
}

// Config is a struct that wraps a Builder
type Config struct {
	Builder
}

// NewConfig returns a default Config
func NewConfig() Config {
	return Config{
		Builder: NewMemoryBufferConfig(),
	}
}

// Builder builds a Buffer given build context
type Builder interface {
	Build() (Buffer, error)
}

// UnmarshalJSON unmarshals JSON
func (bc *Config) UnmarshalJSON(data []byte) error {
	return bc.unmarshal(func(dst interface{}) error {
		return json.Unmarshal(data, dst)
	})
}

// UnmarshalYAML unmarshals YAML
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
		bc.Builder = NewMemoryBufferConfig()
		return unmarshal(bc.Builder)
	case "disk":
		bc.Builder = NewDiskBufferConfig()
		return unmarshal(bc.Builder)
	default:
		return fmt.Errorf("unknown buffer type '%s'", m["type"])
	}
}

func (bc Config) MarshalYAML() (interface{}, error) {
	return bc.Builder, nil
}

func (bc Config) MarshalJSON() ([]byte, error) {
	return json.Marshal(bc.Builder)
}
