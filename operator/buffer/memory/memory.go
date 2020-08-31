package memory

import (
	"context"
	"fmt"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator/buffer"
)

var _ buffer.Buffer = &MemoryBuffer{}

type MemoryBuffer struct {
	buf chan *entry.Entry
}

func (m *MemoryBuffer) Add(ctx context.Context, e *entry.Entry) error {
	select {
	case m.buf <- e:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("context cancelled")
	}
}

func (m *MemoryBuffer) Read(dst []*entry.Entry) (func(), int, error) {
	for i := 0; i < len(dst); i++ {
		select {
		case e := <-m.buf:
			dst[i] = e
		default:
			return func() {}, 0, nil
		}
	}

	return
}
