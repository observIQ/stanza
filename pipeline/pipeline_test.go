package pipeline

import (
	"testing"

	"github.com/observiq/carbon/internal/testutil"
	"github.com/observiq/carbon/plugin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
)

func TestUnorderableToCycles(t *testing.T) {
	t.Run("SingleCycle", func(t *testing.T) {
		mockPlugin1 := testutil.NewMockPlugin("plugin1")
		mockPlugin2 := testutil.NewMockPlugin("plugin2")
		mockPlugin3 := testutil.NewMockPlugin("plugin3")
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
	})

	t.Run("MultipleCycles", func(t *testing.T) {
		mockPlugin1 := testutil.NewMockPlugin("plugin1")
		mockPlugin2 := testutil.NewMockPlugin("plugin2")
		mockPlugin3 := testutil.NewMockPlugin("plugin3")
		mockPlugin1.On("Outputs").Return([]plugin.Plugin{mockPlugin2})
		mockPlugin2.On("Outputs").Return([]plugin.Plugin{mockPlugin3})
		mockPlugin3.On("Outputs").Return([]plugin.Plugin{mockPlugin1})

		mockPlugin4 := testutil.NewMockPlugin("plugin4")
		mockPlugin5 := testutil.NewMockPlugin("plugin5")
		mockPlugin6 := testutil.NewMockPlugin("plugin6")
		mockPlugin4.On("Outputs").Return([]plugin.Plugin{mockPlugin5})
		mockPlugin5.On("Outputs").Return([]plugin.Plugin{mockPlugin6})
		mockPlugin6.On("Outputs").Return([]plugin.Plugin{mockPlugin4})

		err := topo.Unorderable([][]graph.Node{{
			createPluginNode(mockPlugin1),
			createPluginNode(mockPlugin2),
			createPluginNode(mockPlugin3),
		}, {
			createPluginNode(mockPlugin4),
			createPluginNode(mockPlugin5),
			createPluginNode(mockPlugin6),
		}})

		output := unorderableToCycles(err)
		expected := `(plugin1 -> plugin2 -> plugin3 -> plugin1),(plugin4 -> plugin5 -> plugin6 -> plugin4)`

		require.Equal(t, expected, output)
	})
}

func TestPipeline(t *testing.T) {
	t.Run("MultipleStart", func(t *testing.T) {
		pipeline, err := NewPipeline([]plugin.Plugin{})
		require.NoError(t, err)

		err = pipeline.Start()
		require.NoError(t, err)

		err = pipeline.Start()
		require.NoError(t, err)

		pipeline.Stop()
	})

	t.Run("MultipleStop", func(t *testing.T) {
		pipeline, err := NewPipeline([]plugin.Plugin{})
		require.NoError(t, err)

		err = pipeline.Start()
		require.NoError(t, err)

		pipeline.Stop()
		pipeline.Stop()
	})

	t.Run("DuplicateNodeIDs", func(t *testing.T) {
		plugin1 := testutil.NewMockPlugin("plugin1")
		plugin1.On("SetOutputs", mock.Anything).Return(nil)
		plugin1.On("Outputs").Return(nil)
		plugin2 := testutil.NewMockPlugin("plugin1")
		plugin2.On("SetOutputs", mock.Anything).Return(nil)
		plugin2.On("Outputs").Return(nil)

		_, err := NewPipeline([]plugin.Plugin{plugin1, plugin2})
		require.Error(t, err)
		require.Contains(t, err.Error(), "already exists")
	})

	t.Run("OutputNotExist", func(t *testing.T) {
		plugin1 := testutil.NewMockPlugin("plugin1")
		plugin1.On("SetOutputs", mock.Anything).Return(nil)
		plugin1.On("Outputs").Return()

		plugin2 := testutil.NewMockPlugin("plugin2")
		plugin2.On("SetOutputs", mock.Anything).Return(nil)
		plugin2.On("Outputs").Return([]plugin.Plugin{plugin1})

		_, err := NewPipeline([]plugin.Plugin{plugin2})
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not exist")
	})

	t.Run("OutputNotProcessor", func(t *testing.T) {
		plugin1 := &testutil.Plugin{}
		plugin1.On("ID").Return("plugin1")
		plugin1.On("CanProcess").Return(false)
		plugin1.On("CanOutput").Return(true)
		plugin1.On("SetOutputs", mock.Anything).Return(nil)
		plugin1.On("Outputs").Return(nil)

		plugin2 := testutil.NewMockPlugin("plugin2")
		plugin2.On("SetOutputs", mock.Anything).Return(nil)
		plugin2.On("Outputs").Return([]plugin.Plugin{plugin1})

		_, err := NewPipeline([]plugin.Plugin{plugin1, plugin2})
		require.Error(t, err)
		require.Contains(t, err.Error(), "can not process")
	})

	t.Run("DuplicateEdges", func(t *testing.T) {
		plugin1 := testutil.NewMockPlugin("plugin1")
		plugin1.On("SetOutputs", mock.Anything).Return(nil)
		plugin1.On("Outputs").Return(nil)

		plugin2 := testutil.NewMockPlugin("plugin2")
		plugin2.On("SetOutputs", mock.Anything).Return(nil)
		plugin2.On("Outputs").Return([]plugin.Plugin{plugin1, plugin1})

		node1 := createPluginNode(plugin1)
		node2 := createPluginNode(plugin2)

		graph := simple.NewDirectedGraph()
		graph.AddNode(node1)
		graph.AddNode(node2)
		edge := graph.NewEdge(node2, node1)
		graph.SetEdge(edge)

		err := connectNode(graph, node2)
		require.Error(t, err)
		require.Contains(t, err.Error(), "connection already exists")
	})

	t.Run("Cyclical", func(t *testing.T) {
		mockPlugin1 := testutil.NewMockPlugin("plugin1")
		mockPlugin2 := testutil.NewMockPlugin("plugin2")
		mockPlugin3 := testutil.NewMockPlugin("plugin3")
		mockPlugin1.On("Outputs").Return([]plugin.Plugin{mockPlugin2})
		mockPlugin1.On("SetOutputs", mock.Anything).Return(nil)
		mockPlugin2.On("Outputs").Return([]plugin.Plugin{mockPlugin3})
		mockPlugin2.On("SetOutputs", mock.Anything).Return(nil)
		mockPlugin3.On("Outputs").Return([]plugin.Plugin{mockPlugin1})
		mockPlugin3.On("SetOutputs", mock.Anything).Return(nil)

		_, err := NewPipeline([]plugin.Plugin{mockPlugin1, mockPlugin2, mockPlugin3})
		require.Error(t, err)
		require.Contains(t, err.Error(), "circular dependency")
	})
}
