package pipeline

import (
	"testing"

	"github.com/bluemedora/bplogagent/plugin"
	"github.com/bluemedora/bplogagent/plugin/testutil"
	"github.com/stretchr/testify/require"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/topo"
)

func TestUnorderableToCycles(t *testing.T) {
	mockPlugin1 := &testutil.Plugin{}
	mockPlugin1.On("ID").Return("plugin1")
	mockPlugin1.On("CanOutput").Return(true)
	mockPlugin2 := &testutil.Plugin{}
	mockPlugin2.On("ID").Return("plugin2")
	mockPlugin2.On("CanOutput").Return(true)
	mockPlugin3 := &testutil.Plugin{}
	mockPlugin3.On("ID").Return("plugin3")
	mockPlugin3.On("CanOutput").Return(true)
	mockPlugin1.On("Outputs").Return([]plugin.Plugin{mockPlugin2})
	mockPlugin2.On("Outputs").Return([]plugin.Plugin{mockPlugin3})
	mockPlugin3.On("Outputs").Return([]plugin.Plugin{mockPlugin1})

	err := topo.Unorderable([][]graph.Node{[]graph.Node{
		createPluginNode(mockPlugin1),
		createPluginNode(mockPlugin2),
		createPluginNode(mockPlugin3),
	}})

	output := unorderableToCycles(err)
	expected := `(plugin1 -> plugin2 -> plugin3 -> plugin1)`

	require.Equal(t, expected, output)
}
