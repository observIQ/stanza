package helper

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCopyRecord(t *testing.T) {
	cases := []struct {
		name  string
		input interface{}
	}{
		{
			"String",
			"testmessage",
		},
		{
			"MapStringInterface",
			map[string]interface{}{
				"message": "testmessage",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			newRecord := CopyRecord(tc.input)
			require.Equal(t, tc.input, newRecord)
		})
	}
}
