package persist

import (
	"context"
	"errors"
	"testing"

	"github.com/observiq/stanza/v2/testutil"
	"github.com/open-telemetry/opentelemetry-log-collection/operator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewCachedPersister(t *testing.T) {
	var mockPersister operator.Persister = &testutil.MockPersister{}

	cachedPersister := NewCachedPersister(mockPersister)

	assert.Equal(t, mockPersister, cachedPersister.base)
	assert.Len(t, cachedPersister.cache, 0)
}

func TestCachePersisterGet(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Value exists in cache",
			testFunc: func(t *testing.T) {
				mockPersister := &testutil.MockPersister{}
				cachedPersister := NewCachedPersister(mockPersister)

				key, expectedVal := "key", []byte("value")
				// Inject key/value into cache
				cachedPersister.cache[key] = expectedVal

				val, err := cachedPersister.Get(context.Background(), key)
				assert.NoError(t, err)
				assert.Equal(t, expectedVal, val)
			},
		},
		{
			desc: "Value does not exist in cache, base errors on get",
			testFunc: func(t *testing.T) {
				key := "key"
				expectedError := errors.New("bad")

				// Setup Mock
				mockPersister := &testutil.MockPersister{}
				mockPersister.On("Get", mock.Anything, key).Return(nil, expectedError)

				cachedPersister := NewCachedPersister(mockPersister)

				val, err := cachedPersister.Get(context.Background(), key)
				assert.ErrorIs(t, err, expectedError)
				assert.Nil(t, val)
			},
		},
		{
			desc: "Value does not exist in cache, fall through to base",
			testFunc: func(t *testing.T) {
				key, expectedVal := "key", []byte("value")

				// Setup Mock
				mockPersister := &testutil.MockPersister{}
				mockPersister.On("Get", mock.Anything, key).Return(expectedVal, nil)

				cachedPersister := NewCachedPersister(mockPersister)

				val, err := cachedPersister.Get(context.Background(), key)
				assert.NoError(t, err)
				assert.Equal(t, expectedVal, val)

				// Verify value was stored in cache
				_, ok := cachedPersister.cache[key]
				assert.True(t, ok)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestCachePersisterSet(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Value sets in cache and base persister",
			testFunc: func(t *testing.T) {
				key, val := "key", []byte("value")

				// Setup Mock
				mockPersister := &testutil.MockPersister{}
				mockPersister.On("Set", mock.Anything, key, val).Return(nil)

				cachedPersister := NewCachedPersister(mockPersister)

				err := cachedPersister.Set(context.Background(), key, val)
				assert.NoError(t, err)

				// Verify value was stored in cache
				_, ok := cachedPersister.cache[key]
				assert.True(t, ok)
			},
		},
		{
			desc: "Base errors on set",
			testFunc: func(t *testing.T) {
				key, val := "key", []byte("value")
				expectedError := errors.New("bad")
				// Setup Mock
				mockPersister := &testutil.MockPersister{}
				mockPersister.On("Set", mock.Anything, key, val).Return(expectedError)

				cachedPersister := NewCachedPersister(mockPersister)

				err := cachedPersister.Set(context.Background(), key, val)
				assert.ErrorIs(t, err, expectedError)

				// Verify value was not stored in cache
				_, ok := cachedPersister.cache[key]
				assert.False(t, ok)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}

func TestCachePersisterDelete(t *testing.T) {
	testCases := []struct {
		desc     string
		testFunc func(*testing.T)
	}{
		{
			desc: "Value deletes in cache and base persister",
			testFunc: func(t *testing.T) {
				key, val := "key", []byte("value")

				// Setup Mock
				mockPersister := &testutil.MockPersister{}
				mockPersister.On("Delete", mock.Anything, key).Return(nil)

				cachedPersister := NewCachedPersister(mockPersister)

				// seed data
				cachedPersister.cache[key] = val

				err := cachedPersister.Delete(context.Background(), key)
				assert.NoError(t, err)

				// Verify value is removed from cache
				_, ok := cachedPersister.cache[key]
				assert.False(t, ok)
			},
		},
		{
			desc: "Base errors on delete",
			testFunc: func(t *testing.T) {
				key, val := "key", []byte("value")
				expectedError := errors.New("bad")
				// Setup Mock
				mockPersister := &testutil.MockPersister{}
				mockPersister.On("Delete", mock.Anything, key).Return(expectedError)

				cachedPersister := NewCachedPersister(mockPersister)

				// seed data
				cachedPersister.cache[key] = val

				err := cachedPersister.Delete(context.Background(), key)
				assert.ErrorIs(t, err, expectedError)

				// Verify value was not deleted in cache
				_, ok := cachedPersister.cache[key]
				assert.True(t, ok)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, tc.testFunc)
	}
}
