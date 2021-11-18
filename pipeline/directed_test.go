package pipeline

import (
	"fmt"
	"testing"

	"github.com/observiq/stanza/v2/operator"
	"github.com/observiq/stanza/v2/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
	"gonum.org/v1/gonum/graph/topo"
)

func TestUnorderableToCycles(t *testing.T) {
	t.Run("SingleCycle", func(t *testing.T) {
		mockOperator1 := testutil.NewMockOperator("operator1")
		mockOperator2 := testutil.NewMockOperator("operator2")
		mockOperator3 := testutil.NewMockOperator("operator3")
		mockOperator1.On("Outputs").Return([]operator.Operator{mockOperator2})
		mockOperator2.On("Outputs").Return([]operator.Operator{mockOperator3})
		mockOperator3.On("Outputs").Return([]operator.Operator{mockOperator1})

		err := topo.Unorderable([][]graph.Node{{
			createOperatorNode(mockOperator1),
			createOperatorNode(mockOperator2),
			createOperatorNode(mockOperator3),
		}})

		output := unorderableToCycles(err)
		expected := `(operator1 -> operator2 -> operator3 -> operator1)`

		require.Equal(t, expected, output)
	})

	t.Run("MultipleCycles", func(t *testing.T) {
		mockOperator1 := testutil.NewMockOperator("operator1")
		mockOperator2 := testutil.NewMockOperator("operator2")
		mockOperator3 := testutil.NewMockOperator("operator3")
		mockOperator1.On("Outputs").Return([]operator.Operator{mockOperator2})
		mockOperator2.On("Outputs").Return([]operator.Operator{mockOperator3})
		mockOperator3.On("Outputs").Return([]operator.Operator{mockOperator1})

		mockOperator4 := testutil.NewMockOperator("operator4")
		mockOperator5 := testutil.NewMockOperator("operator5")
		mockOperator6 := testutil.NewMockOperator("operator6")
		mockOperator4.On("Outputs").Return([]operator.Operator{mockOperator5})
		mockOperator5.On("Outputs").Return([]operator.Operator{mockOperator6})
		mockOperator6.On("Outputs").Return([]operator.Operator{mockOperator4})

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
		expected := `(operator1 -> operator2 -> operator3 -> operator1),(operator4 -> operator5 -> operator6 -> operator4)`

		require.Equal(t, expected, output)
	})
}

func TestPipeline(t *testing.T) {
	t.Run("MultipleStart", func(t *testing.T) {
		pipeline, err := NewDirectedPipeline([]operator.Operator{})
		require.NoError(t, err)

		err = pipeline.Start()
		require.NoError(t, err)

		err = pipeline.Start()
		require.NoError(t, err)

		pipeline.Stop()
	})

	t.Run("MultipleStop", func(t *testing.T) {
		pipeline, err := NewDirectedPipeline([]operator.Operator{})
		require.NoError(t, err)

		err = pipeline.Start()
		require.NoError(t, err)

		pipeline.Stop()
		pipeline.Stop()
	})

	t.Run("DuplicateNodeIDs", func(t *testing.T) {
		operator1 := testutil.NewMockOperator("operator1")
		operator1.On("SetOutputs", mock.Anything).Return(nil)
		operator1.On("Outputs").Return(nil)
		operator2 := testutil.NewMockOperator("operator1")
		operator2.On("SetOutputs", mock.Anything).Return(nil)
		operator2.On("Outputs").Return(nil)

		_, err := NewDirectedPipeline([]operator.Operator{operator1, operator2})
		require.Error(t, err)
		require.Contains(t, err.Error(), "already exists")
	})

	t.Run("OutputNotExist", func(t *testing.T) {
		operator1 := testutil.NewMockOperator("operator1")
		operator1.On("SetOutputs", mock.Anything).Return(nil)
		operator1.On("Outputs").Return()

		operator2 := testutil.NewMockOperator("operator2")
		operator2.On("SetOutputs", mock.Anything).Return(nil)
		operator2.On("Outputs").Return([]operator.Operator{operator1})

		_, err := NewDirectedPipeline([]operator.Operator{operator2})
		require.Error(t, err)
		require.Contains(t, err.Error(), "does not exist")
	})

	t.Run("OutputNotProcessor", func(t *testing.T) {
		operator1 := &testutil.Operator{}
		operator1.On("ID").Return("operator1")
		operator1.On("CanProcess").Return(false)
		operator1.On("CanOutput").Return(true)
		operator1.On("SetOutputs", mock.Anything).Return(nil)
		operator1.On("Outputs").Return(nil)

		operator2 := testutil.NewMockOperator("operator2")
		operator2.On("SetOutputs", mock.Anything).Return(nil)
		operator2.On("Outputs").Return([]operator.Operator{operator1})

		_, err := NewDirectedPipeline([]operator.Operator{operator1, operator2})
		require.Error(t, err)
		require.Contains(t, err.Error(), "can not process")
	})

	t.Run("DuplicateEdges", func(t *testing.T) {
		operator1 := testutil.NewMockOperator("operator1")
		operator1.On("SetOutputs", mock.Anything).Return(nil)
		operator1.On("Outputs").Return(nil)

		operator2 := testutil.NewMockOperator("operator2")
		operator2.On("SetOutputs", mock.Anything).Return(nil)
		operator2.On("Outputs").Return([]operator.Operator{operator1, operator1}, nil)

		node1 := createOperatorNode(operator1)
		node2 := createOperatorNode(operator2)

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
		mockOperator1 := testutil.NewMockOperator("operator1")
		mockOperator2 := testutil.NewMockOperator("operator2")
		mockOperator3 := testutil.NewMockOperator("operator3")
		mockOperator1.On("Outputs").Return([]operator.Operator{mockOperator2})
		mockOperator1.On("SetOutputs", mock.Anything).Return(nil)
		mockOperator2.On("Outputs").Return([]operator.Operator{mockOperator3})
		mockOperator2.On("SetOutputs", mock.Anything).Return(nil)
		mockOperator3.On("Outputs").Return([]operator.Operator{mockOperator1})
		mockOperator3.On("SetOutputs", mock.Anything).Return(nil)

		_, err := NewDirectedPipeline([]operator.Operator{mockOperator1, mockOperator2, mockOperator3})
		require.Error(t, err)
		require.Contains(t, err.Error(), "circular dependency")
	})
}

func TestPipelineStartOrder(t *testing.T) {
	var mock2Started bool
	var mock3Started bool

	mockOperator1 := testutil.NewMockOperator("operator1")
	mockOperator2 := testutil.NewMockOperator("operator2")
	mockOperator3 := testutil.NewMockOperator("operator3")

	mockOperator1.On("Outputs").Return([]operator.Operator{mockOperator2})
	mockOperator2.On("Outputs").Return([]operator.Operator{mockOperator3})
	mockOperator3.On("Outputs").Return(nil)

	mockOperator1.On("SetOutputs", mock.Anything).Return(nil)
	mockOperator2.On("SetOutputs", mock.Anything).Return(nil)
	mockOperator3.On("SetOutputs", mock.Anything).Return(nil)

	mockOperator1.On("Logger", mock.Anything).Return(zap.NewNop().Sugar())
	mockOperator2.On("Logger", mock.Anything).Return(zap.NewNop().Sugar())
	mockOperator3.On("Logger", mock.Anything).Return(zap.NewNop().Sugar())

	mockOperator1.On("Start").Return(fmt.Errorf("operator 1 failed to start"))
	mockOperator2.On("Start").Run(func(mock.Arguments) { mock2Started = true }).Return(nil)
	mockOperator3.On("Start").Run(func(mock.Arguments) { mock3Started = true }).Return(nil)

	pipeline, err := NewDirectedPipeline([]operator.Operator{mockOperator1, mockOperator2, mockOperator3})
	require.NoError(t, err)

	err = pipeline.Start()
	require.Error(t, err)
	require.Contains(t, err.Error(), "operator 1 failed to start")
	require.True(t, mock2Started)
	require.True(t, mock3Started)
}

func TestPipelineStopOrder(t *testing.T) {
	stopOrder := []int{}

	mockOperator1 := testutil.NewMockOperator("operator1")
	mockOperator2 := testutil.NewMockOperator("operator2")
	mockOperator3 := testutil.NewMockOperator("operator3")

	mockOperator1.On("Outputs").Return([]operator.Operator{mockOperator2})
	mockOperator2.On("Outputs").Return([]operator.Operator{mockOperator3})
	mockOperator3.On("Outputs").Return(nil)

	mockOperator1.On("SetOutputs", mock.Anything).Return(nil)
	mockOperator2.On("SetOutputs", mock.Anything).Return(nil)
	mockOperator3.On("SetOutputs", mock.Anything).Return(nil)

	mockOperator1.On("Logger", mock.Anything).Return(zap.NewNop().Sugar())
	mockOperator2.On("Logger", mock.Anything).Return(zap.NewNop().Sugar())
	mockOperator3.On("Logger", mock.Anything).Return(zap.NewNop().Sugar())

	mockOperator1.On("Start").Return(nil)
	mockOperator2.On("Start").Return(nil)
	mockOperator3.On("Start").Return(nil)

	mockOperator1.On("Stop").Run(func(mock.Arguments) { stopOrder = append(stopOrder, 1) }).Return(nil)
	mockOperator2.On("Stop").Run(func(mock.Arguments) { stopOrder = append(stopOrder, 2) }).Return(nil)
	mockOperator3.On("Stop").Run(func(mock.Arguments) { stopOrder = append(stopOrder, 3) }).Return(nil)

	pipeline, err := NewDirectedPipeline([]operator.Operator{mockOperator1, mockOperator2, mockOperator3})
	require.NoError(t, err)

	err = pipeline.Start()
	require.NoError(t, err)

	err = pipeline.Stop()
	require.NoError(t, err)
	require.Equal(t, []int{1, 2, 3}, stopOrder)
}

func TestPipelineRender(t *testing.T) {
	mockOperator1 := testutil.NewMockOperator("operator1")
	mockOperator2 := testutil.NewMockOperator("operator2")
	mockOperator3 := testutil.NewMockOperator("operator3")

	mockOperator1.On("Outputs").Return([]operator.Operator{mockOperator2})
	mockOperator2.On("Outputs").Return([]operator.Operator{mockOperator3})
	mockOperator3.On("Outputs").Return(nil)

	mockOperator1.On("SetOutputs", mock.Anything).Return(nil)
	mockOperator2.On("SetOutputs", mock.Anything).Return(nil)
	mockOperator3.On("SetOutputs", mock.Anything).Return(nil)

	pipeline, err := NewDirectedPipeline([]operator.Operator{mockOperator1, mockOperator2, mockOperator3})
	require.NoError(t, err)

	dotGraph, err := pipeline.Render()
	require.NoError(t, err)
	expected := `strict digraph G {
 // Node definitions.
 operator1;
 operator3;
 operator2;

 // Edge definitions.
 operator1 -> operator2;
 operator2 -> operator3;
}`
	require.Equal(t, expected, string(dotGraph))
}

func TestPipelineOperators(t *testing.T) {
	mockOperator1 := testutil.NewMockOperator("operator1")
	mockOperator2 := testutil.NewMockOperator("operator2")
	mockOperator3 := testutil.NewMockOperator("operator3")

	mockOperator1.On("Outputs").Return([]operator.Operator{mockOperator2})
	mockOperator2.On("Outputs").Return([]operator.Operator{mockOperator3})
	mockOperator3.On("Outputs").Return(nil)

	mockOperator1.On("SetOutputs", mock.Anything).Return(nil)
	mockOperator2.On("SetOutputs", mock.Anything).Return(nil)
	mockOperator3.On("SetOutputs", mock.Anything).Return(nil)

	pipeline, err := NewDirectedPipeline([]operator.Operator{mockOperator1, mockOperator2, mockOperator3})
	require.NoError(t, err)

	operators := pipeline.Operators()
	require.ElementsMatch(t, []operator.Operator{mockOperator1, mockOperator2, mockOperator3}, operators)
}
