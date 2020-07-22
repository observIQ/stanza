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
		mockOperator1 := testutil.NewMockOperator("plugin1")
		mockOperator2 := testutil.NewMockOperator("plugin2")
		mockOperator3 := testutil.NewMockOperator("plugin3")
		mockOperator1.On("Outputs").Return([]plugin.Operator{mockOperator2})
		mockOperator2.On("Outputs").Return([]plugin.Operator{mockOperator3})
		mockOperator3.On("Outputs").Return([]plugin.Operator{mockOperator1})

		err := topo.Unorderable([][]graph.Node{{
			createOperatorNode(mockOperator1),
			createOperatorNode(mockOperator2),
			createOperatorNode(mockOperator3),
		}})

		output := unorderableToCycles(err)
		expected := `(plugin1 -> plugin2 -> plugin3 -> plugin1)`

		require.Equal(t, expected, output)
	})

	t.Run("MultipleCycles", func(t *testing.T) {
		mockOperator1 := testutil.NewMockOperator("plugin1")
		mockOperator2 := testutil.NewMockOperator("plugin2")
		mockOperator3 := testutil.NewMockOperator("plugin3")
		mockOperator1.On("Outputs").Return([]plugin.Operator{mockOperator2})
		mockOperator2.On("Outputs").Return([]plugin.Operator{mockOperator3})
		mockOperator3.On("Outputs").Return([]plugin.Operator{mockOperator1})

		mockOperator4 := testutil.NewMockOperator("plugin4")
		mockOperator5 := testutil.NewMockOperator("plugin5")
		mockOperator6 := testutil.NewMockOperator("plugin6")
		mockOperator4.On("Outputs").Return([]plugin.Operator{mockOperator5})
		mockOperator5.On("Outputs").Return([]plugin.Operator{mockOperator6})
		mockOperator6.On("Outputs").Return([]plugin.Operator{mockOperator4})

		err := topo.Unorderable([][]graph.Node{{
			createOperatorNode(mockOperator1),
			createOperatorNode(mockOperator2),
			createOperatorNode(mockOperator3),
		}, {
			createOperatorNode(mockOperator4),
			createOperatorNode(mockOperator5),
			createOperatorNode(mockOperator6),
		}})

		output := unorderableToCycles(err)
		expected := `(plugin1 -> plugin2 -> plugin3 -> plugin1),(plugin4 -> plugin5 -> plugin6 -> plugin4)`

		require.Equal(t, expected, output)
	})
}

func TestPipeline(t *testing.T) {
	t.Run("MultipleStart", func(t *testing.T) {
		pipeline, err := NewPipeline([]plugin.Operator{})
		require.NoError(t, err)

		err = pipeline.Start()
		require.NoError(t, err)

		err = pipeline.Start()
		require.NoError(t, err)

		pipeline.Stop()
	})

	t.Run("MultipleStop", func(t *testing.T) {
		pipeline, err := NewPipeline([]plugin.Operator{})
		require.NoError(t, err)

		err = pipeline.Start()
		require.NoError(t, err)

		pipeline.Stop()
		pipeline.Stop()
	})

	t.Run("DuplicateNodeIDs", func(t *testing.T) {
		plugin1 := testutil.NewMockOperator("plugin1")
		plugin1.On("SetOutputs", mock.Anything).Return(nil)
		plugin1.On("Outputs").Return(nil)
		plugin2 := testutil.NewMockOperator("plugin1")
		plugin2.On("SetOutputs", mock.Anything).Return(nil)
		plugin2.On("Outputs").Return(nil)

		_, err := NewPipeline([]plugin.Operator{plugin1, plugin2})
		require.Error(t, err)
		require.Contains(t, err.Error(), "already exists")
	})

	t.Run("OutputNotExist", func(t *testing.T) {
		plugin1 := testutil.NewMockOperator("plugin1")
		plugin1.On("SetOutputs", mock.Anything).Return(nil)
		plugin1.On("Outputs").Return()

		plugin2 := testutil.NewMockOperator("plugin2")
		plugin2.On("SetOutputs", mock.Anything).Return(nil)
		plugin2.On("Outputs").Return([]plugin.Operator{plugin1})

		_, err := NewPipeline([]plugin.Operator{plugin2})
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not exist")
	})

	t.Run("OutputNotProcessor", func(t *testing.T) {
		plugin1 := &testutil.Operator{}
		plugin1.On("ID").Return("plugin1")
		plugin1.On("CanProcess").Return(false)
		plugin1.On("CanOutput").Return(true)
		plugin1.On("SetOutputs", mock.Anything).Return(nil)
		plugin1.On("Outputs").Return(nil)

		plugin2 := testutil.NewMockOperator("plugin2")
		plugin2.On("SetOutputs", mock.Anything).Return(nil)
		plugin2.On("Outputs").Return([]plugin.Operator{plugin1})

		_, err := NewPipeline([]plugin.Operator{plugin1, plugin2})
		require.Error(t, err)
		require.Contains(t, err.Error(), "can not process")
	})

	t.Run("DuplicateEdges", func(t *testing.T) {
		plugin1 := testutil.NewMockOperator("plugin1")
		plugin1.On("SetOutputs", mock.Anything).Return(nil)
		plugin1.On("Outputs").Return(nil)

		plugin2 := testutil.NewMockOperator("plugin2")
		plugin2.On("SetOutputs", mock.Anything).Return(nil)
		plugin2.On("Outputs").Return([]plugin.Operator{plugin1, plugin1})

		node1 := createOperatorNode(plugin1)
		node2 := createOperatorNode(plugin2)

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
		mockOperator1 := testutil.NewMockOperator("plugin1")
		mockOperator2 := testutil.NewMockOperator("plugin2")
		mockOperator3 := testutil.NewMockOperator("plugin3")
		mockOperator1.On("Outputs").Return([]plugin.Operator{mockOperator2})
		mockOperator1.On("SetOutputs", mock.Anything).Return(nil)
		mockOperator2.On("Outputs").Return([]plugin.Operator{mockOperator3})
		mockOperator2.On("SetOutputs", mock.Anything).Return(nil)
		mockOperator3.On("Outputs").Return([]plugin.Operator{mockOperator1})
		mockOperator3.On("SetOutputs", mock.Anything).Return(nil)

		_, err := NewPipeline([]plugin.Operator{mockOperator1, mockOperator2, mockOperator3})
		require.Error(t, err)
		require.Contains(t, err.Error(), "circular dependency")
	})
}
