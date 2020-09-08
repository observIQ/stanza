package drop

import (
	"github.com/observiq/stanza/operator/helper"
)

func newFakeNullOutput() *DropOutput {
	return &DropOutput{
		OutputOperator: helper.OutputOperator{
			BasicOperator: helper.BasicOperator{
				OperatorID:   "testnull",
				OperatorType: "drop_output",
			},
		},
	}
}
