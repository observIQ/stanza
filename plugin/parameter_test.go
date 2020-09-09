package plugin

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateDefault(t *testing.T) {
	testCases := []struct {
		name      string
		expectErr bool
		param     Parameter
	}{
		{
			"ValidStringDefault",
			false,
			Parameter{
				Type:    "string",
				Default: "test",
			},
		},
		{
			"InvalidStringDefault",
			true,
			Parameter{
				Type:    "string",
				Default: 5,
			},
		},
		{
			"ValidIntDefault",
			false,
			Parameter{
				Type:    "int",
				Default: 5,
			},
		},
		{
			"InvalidStringDefault",
			true,
			Parameter{
				Type:    "int",
				Default: "test",
			},
		},
		{
			"ValidBoolDefault",
			false,
			Parameter{
				Type:    "bool",
				Default: true,
			},
		},
		{
			"InvalidBoolDefault",
			true,
			Parameter{
				Type:    "bool",
				Default: "test",
			},
		},
		{
			"ValidStringsDefault",
			false,
			Parameter{
				Type:    "strings",
				Default: []interface{}{"test"},
			},
		},
		{
			"InvalidStringsDefault",
			true,
			Parameter{
				Type:    "strings",
				Default: []interface{}{5},
			},
		},
		{
			"ValidEnumDefault",
			false,
			Parameter{
				Type:        "enum",
				ValidValues: []string{"test"},
				Default:     "test",
			},
		},
		{
			"InvalidEnumDefault",
			true,
			Parameter{
				Type:        "enum",
				ValidValues: []string{"test"},
				Default:     "invalid",
			},
		},
		{
			"NonStringEnumDefault",
			true,
			Parameter{
				Type:        "enum",
				ValidValues: []string{"test"},
				Default:     5,
			},
		},
		{
			"InvalidTypeDefault",
			true,
			Parameter{
				Type:    "float",
				Default: 5,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.param.validateDefault()
			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
