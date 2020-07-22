package output

import (
	"github.com/observiq/carbon/plugin/helper"
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
