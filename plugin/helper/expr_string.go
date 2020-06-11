package helper

import (
	"reflect"
	"strings"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/conf"
	"github.com/antonmedv/expr/vm"
	"github.com/bluemedora/bplogagent/errors"
)

type ExprStringConfig string

func (e ExprStringConfig) Build() (*ExprString, error) {
	s := string(e)
	begin := 0

	subStrings := make([]string, 0)
	subExprStrings := make([]string, 0)

LOOP:
	for {
		indexStart := strings.Index(s[begin:], "{{")
		indexEnd := strings.Index(s[begin:], "}}")
		switch {
		case indexStart == -1 || indexEnd == -1:
			fallthrough
		case indexStart > indexEnd:
			// if we don't have a "{{" followed by a "}}" in the remainder
			// of the string, treat the remainder as a string literal
			subStrings = append(subStrings, s[begin:])
			break LOOP
		default:
			// make indexes relative to whole string again
			indexStart += begin
			indexEnd += begin
		}
		subStrings = append(subStrings, s[begin:indexStart])
		subExprStrings = append(subExprStrings, s[indexStart+2:indexEnd])
		begin = indexEnd + 2
	}

	subExprs := make([]*vm.Program, 0, len(subExprStrings))
	for _, subExprString := range subExprStrings {
		program, err := expr.Compile(subExprString, expr.AllowUndefinedVariables(), asStringOption())
		if err != nil {
			return nil, errors.Wrap(err, "compile embedded expression")
		}
		subExprs = append(subExprs, program)
	}

	return &ExprString{
		SubStrings: subStrings,
		SubExprs:   subExprs,
	}, nil
}

// An ExprString is made up of a list of string literals
// interleaved with expressions. len(SubStrings) == len(SubExprs) + 1
type ExprString struct {
	SubStrings []string
	SubExprs   []*vm.Program
}

func (e *ExprString) Render(env map[string]interface{}) (string, error) {
	var b strings.Builder
	for i := 0; i < len(e.SubExprs); i++ {
		b.WriteString(e.SubStrings[i])
		out, err := vm.Run(e.SubExprs[i], env)
		if err != nil {
			return "", errors.Wrap(err, "render embedded expression")
		}
		b.WriteString(out.(string)) // guaranteed string by asStringOption()
	}
	b.WriteString(e.SubStrings[len(e.SubStrings)-1])

	return b.String(), nil
}

func asStringOption() expr.Option {
	return func(c *conf.Config) {
		c.Expect = reflect.String
	}
}
