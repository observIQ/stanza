package elastic

import (
	"testing"

	"github.com/observiq/stanza/entry"
	"github.com/stretchr/testify/require"
)

func TestFindIndex(t *testing.T) {
	indexField := entry.NewRecordField("bar")
	output := &ElasticOutput{
		indexField: &indexField,
	}

	t.Run("StringValue", func(t *testing.T) {
		entry := entry.New()
		entry.Set(indexField, "testval")
		idx, err := output.FindIndex(entry)
		require.NoError(t, err)
		require.Equal(t, "testval", idx)
	})

	t.Run("ByteValue", func(t *testing.T) {
		entry := entry.New()
		entry.Set(indexField, []byte("testval"))
		idx, err := output.FindIndex(entry)
		require.NoError(t, err)
		require.Equal(t, "testval", idx)
	})

	t.Run("NoValue", func(t *testing.T) {
		entry := entry.New()
		_, err := output.FindIndex(entry)
		require.Error(t, err)
	})

	t.Run("IndexFieldUnset", func(t *testing.T) {
		entry := entry.New()
		output := &ElasticOutput{}
		idx, err := output.FindIndex(entry)
		require.NoError(t, err)
		require.Equal(t, "default", idx)
	})
}

func TestFindID(t *testing.T) {
	idField := entry.NewRecordField("foo")
	output := &ElasticOutput{
		idField: &idField,
	}

	t.Run("StringValue", func(t *testing.T) {
		entry := entry.New()
		entry.Set(idField, "testval")
		idx, err := output.FindID(entry)
		require.NoError(t, err)
		require.Equal(t, "testval", idx)
	})

	t.Run("ByteValue", func(t *testing.T) {
		entry := entry.New()
		entry.Set(idField, []byte("testval"))
		idx, err := output.FindID(entry)
		require.NoError(t, err)
		require.Equal(t, "testval", idx)
	})

	t.Run("NoValue", func(t *testing.T) {
		entry := entry.New()
		_, err := output.FindID(entry)
		require.Error(t, err)
	})

	t.Run("IDFieldUnset", func(t *testing.T) {
		entry := entry.New()
		output := &ElasticOutput{}
		idx, err := output.FindID(entry)
		require.NoError(t, err)
		require.NotEmpty(t, idx)
	})
}
