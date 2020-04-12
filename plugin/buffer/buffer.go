package buffer

import "context"

type Buffer interface {
	Flush(context.Context) error
	Add(interface{}, int) error
	AddWait(context.Context, interface{}, int) error
}
