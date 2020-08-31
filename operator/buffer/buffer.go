package buffer

import (
	"context"

	"github.com/observiq/stanza/entry"
)

type Buffer interface {
	Add(context.Context, *entry.Entry) error
	Read([]*entry.Entry) (func(), int, error)
}
