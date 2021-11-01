package string_replace

import (
	"context"
	"testing"

	"github.com/observiq/stanza/entry"
	"github.com/observiq/stanza/operator"
	"github.com/observiq/stanza/testutil"
	"github.com/stretchr/testify/require"
)

type testCase struct {
	name      string
	op        *AddOperatorConfig
	input     func() *entry.Entry
	output    func() *entry.Entry
	expectErr bool
}

func TestProcessAndBuild(t *testing.T) {
	
}
