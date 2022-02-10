package persist

import "context"

// NoopShutdownFunc is a PersisterShutdownFunc that returns nil.
// To be used with persisters that have no shutdown operation
var NoopShutdownFunc PersisterShutdownFunc = func() error {
	return nil
}

// NoopPersister is a persister that does nothing and returns no errors
type NoopPersister struct{}

// Get does nothing and returns no error
func (p *NoopPersister) Get(_ context.Context, _ string) ([]byte, error) {
	return nil, nil
}

// Set does nothing and returns no error
func (p *NoopPersister) Set(_ context.Context, _ string, _ []byte) error {
	return nil
}

// Delete does nothing and returns no error
func (p *NoopPersister) Delete(_ context.Context, _ string) error {
	return nil
}
