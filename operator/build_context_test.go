package operator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildContext(t *testing.T) {
	t.Run("PrependNamespace", func(t *testing.T) {
		bc := BuildContext{
			Namespace: "$.test",
		}

		t.Run("Standard", func(t *testing.T) {
			id := bc.PrependNamespace("testid")
			require.Equal(t, "$.test.testid", id)
		})

		t.Run("AlreadyPrefixed", func(t *testing.T) {
			id := bc.PrependNamespace("$.myns.testid")
			require.Equal(t, "$.myns.testid", id)
		})
	})

	t.Run("WithSubNamespace", func(t *testing.T) {
		bc := BuildContext{
			Namespace: "$.ns",
		}
		bc2 := bc.WithSubNamespace("subns")
		require.Equal(t, "$.ns.subns", bc2.Namespace)
		require.Equal(t, "$.ns", bc.Namespace)
	})

	t.Run("WithDefaultOutputIDs", func(t *testing.T) {
		bc := BuildContext{
			DefaultOutputIDs: []string{"orig"},
		}
		bc2 := bc.WithDefaultOutputIDs([]string{"id1", "id2"})
		require.Equal(t, []string{"id1", "id2"}, bc2.DefaultOutputIDs)
		require.Equal(t, []string{"orig"}, bc.DefaultOutputIDs)

	})
}
