package cloudwatch

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"

	"github.com/open-telemetry/opentelemetry-log-collection/operator"
)

type Persister struct {
	base operator.Persister
}

// Helper function to get persisted data
func (p *Persister) Read(ctx context.Context, key string) (int64, error) {
	val, err := p.base.Get(ctx, key)
	if err != nil {
		return 0, err
	}

	var startTime int64
	buffer := bytes.NewBuffer(val)
	err = binary.Read(buffer, binary.BigEndian, &startTime)
	if err != nil && errors.Is(err, io.EOF) {
		return 0, err
	}
	return startTime, nil
}

// Helper function to set persisted data
func (p *Persister) Write(ctx context.Context, key string, value int64) error {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(value))
	return p.base.Set(ctx, key, buf)
}
