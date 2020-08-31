package buffer

import (
	"context"
	"time"

	"github.com/observiq/stanza/entry"
)

type Buffer interface {
	Add(context.Context, *entry.Entry) error
	Read([]*entry.Entry) (func(), int, error)
	ReadWait([]*entry.Entry, <-chan time.Time) (func(), int, error)
}
