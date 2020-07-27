package operator

import (
	"fmt"

	"github.com/observiq/carbon/errors"
)

// PluginParameter is a basic description of a plugin's parameter.
type PluginParameter struct {
	Label       string
	Description string
	Required    bool
	Type        interface{} // "string", "int", "bool" or array of strings
	Default     interface{} // Must be valid according to Type
}

func (param PluginParameter) validate() error {
	if param.Required && param.Default != nil {
		return errors.NewError(
			"required parameter cannot have a default value",
			"ensure that required parameters do not have default values",
		)
	}

	switch t := param.Type.(type) {
	case string:
		switch t {
		case "string", "int", "bool": // ok
		default:
			return errors.NewError(
				"invalid type for parameter",
				"ensure that the type is one of 'string', 'int', 'bool', or an array containing only strings",
			)
		}

		if param.Default == nil {
			return nil
		}

		// Validate default corresponds to type
		switch param.Default.(type) {
		case string:
			if param.Type != "string" {
				return errors.NewError(
					fmt.Sprintf("default value is a string but parameter type is %s", param.Type),
					"ensure that the default value is a string",
				)
			}
		case int, int32, int64:
			if param.Type != "int" {
				return errors.NewError(
					fmt.Sprintf("default value is an int but parameter type is %s", param.Type),
					"ensure that the default value is an int",
				)
			}
		case bool:
			if param.Type != "bool" {
				return errors.NewError(
					fmt.Sprintf("default value is a bool but parameter type is %s", param.Type),
					"ensure that the default value is a bool",
				)
			}
		default:
			return errors.NewError(
				"invalid default value",
				"ensure that the default value corresponds to parameter type",
			)
		}

		return nil
	case []interface{}: // array represents enumerated values
		for _, e := range t {
			if _, ok := e.(string); !ok {
				return errors.NewError(
					"invalid value for enumerated parameter",
					"ensure that all enumerated values are strings",
				)
			}
		}

		if param.Default == nil {
			return nil
		}

		// Validate that the default value is included in the enumeration
		def, ok := param.Default.(string)
		if !ok {
			return errors.NewError(
				"invalid default for enumerated parameter",
				"ensure that the default value is a string",
			)
		}

		validDef := false
		for _, e := range t {
			if str, ok := e.(string); ok && str == def {
				validDef = true
			}
		}

		if !validDef {
			return errors.NewError(
				"invalid default value for enumerated parameter",
				"ensure default value is listed as a valid value",
			)
		}

		return nil

	default:
		return errors.NewError(
			"invalid type for parameter",
			"supported types are 'string', 'int', 'bool', and array of strings",
		)
	}
}
