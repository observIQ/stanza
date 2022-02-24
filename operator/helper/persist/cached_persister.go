package persist

import (
	"context"
	"fmt"
	"sync"

	"github.com/open-telemetry/opentelemetry-log-collection/operator"
)

// CachedPersister implements a cache on top of a persister
type CachedPersister struct {
	base     operator.Persister
	cache    map[string][]byte
	cacheMux sync.RWMutex
}

// NewCachedPersister creates a Cache Persister that wraps the supplied Persister in a cache
func NewCachedPersister(p operator.Persister) *CachedPersister {
	return &CachedPersister{
		base:  p,
		cache: make(map[string][]byte),
	}

}

// Get retrieves data from cache. If not found falls through to underlying persister to retrieve data
func (p *CachedPersister) Get(ctx context.Context, key string) ([]byte, error) {
	p.cacheMux.RLock()
	value, ok := p.cache[key]
	p.cacheMux.RUnlock()

	// Data found return it
	if ok {
		return value, nil
	}

	// Data not found load from base persister
	p.cacheMux.Lock()
	defer p.cacheMux.Unlock()
	baseVal, err := p.base.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("error retrieving %s key from base persister: %w", key, err)
	}

	// Store in cache before returning
	p.cache[key] = baseVal

	return baseVal, nil
}

// Set sets value for key in cache and falls through to the underlying Persister
func (p *CachedPersister) Set(ctx context.Context, key string, value []byte) error {
	p.cacheMux.Lock()
	defer p.cacheMux.Unlock()

	// First set in base persister. We don't want to put it in the cache without knowing it's in the underlying persister
	if err := p.base.Set(ctx, key, value); err != nil {
		return fmt.Errorf("error while setting %s key in base persister: %w", key, err)
	}

	// set in cache
	p.cache[key] = value

	return nil
}

// Delete removes a key from the persister
func (p *CachedPersister) Delete(ctx context.Context, key string) error {
	p.cacheMux.Lock()
	defer p.cacheMux.Unlock()

	// First delete in base persister.
	if err := p.base.Delete(ctx, key); err != nil {
		return fmt.Errorf("error while deleting %s key from base persister: %w", key, err)
	}

	// Remove from cache
	delete(p.cache, key)

	return nil
}
