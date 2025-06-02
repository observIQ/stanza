package pipeline

import (
	"testing"

	_ "github.com/observiq/stanza/operator/builtin/input/generate"
	_ "github.com/observiq/stanza/operator/builtin/transformer/noop"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
)

func TestNodeDOTID(t *testing.T) {
	operator := testutil.NewMockOperator("test")
	operator.On("Outputs").Return(nil)
	node := createOperatorNode(operator)
	require.Equal(t, operator.ID(), node.DOTID())
}

func TestCreateNodeID(t *testing.T) {
	nodeID := createNodeID("test_id")
	require.Equal(t, int64(5795108767401590291), nodeID)
}

func TestCreateNodeIDOverflow(t *testing.T) {
	var overflownValue string
	var result int64
	for i := 0; i < 1000000; i++ {
		s := "test-string-" + string(rune(i))
		id := createNodeID(s)
		if id < 0 {
			overflownValue = s
			result = id
			break
		}
	}

	if overflownValue == "" {
		t.Logf("did not find any input causing overflow")
		return
	} else {
		t.Logf("Found overflow: input = %q, nodeID = %d", overflownValue, result)
		t.Fail()
	}
}

func TestCreateNodeIDWithSimpleString(t *testing.T) {
	simpleString := "simple"
	nodeID := createNodeID(simpleString)
	require.NotZero(t, nodeID)
}

func TestCreateNodeIDWithEmptyString(t *testing.T) {
	emptyString := ""
	nodeID := createNodeID(emptyString)
	require.NotZero(t, nodeID)
}

func TestCreateNodeIDWithLongString(t *testing.T) {
	longString := "a very long string that exceeds normal length expectations"
	nodeID := createNodeID(longString)
	require.NotZero(t, nodeID)
}

func TestCreateNodeIDWithSpecialCharacters(t *testing.T) {
	specialChars := "!@#$%^&*()_+"
	nodeID := createNodeID(specialChars)
	require.NotZero(t, nodeID)
}
