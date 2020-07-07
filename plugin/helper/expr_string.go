package helper

import (
	"fmt"
	"strings"

	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/observiq/bplogagent/errors"
)

type ExprStringConfig string

const (
	exprStartTag = "EXPR("
	exprEndTag   = ")"
)

func (e ExprStringConfig) Build() (*ExprString, error) {
	s := string(e)
	begin := 0

	subStrings := make([]string, 0)
	subExprStrings := make([]string, 0)

LOOP:
	for {
		indexStart := strings.Index(s[begin:], exprStartTag)
		indexEnd := strings.Index(s[begin:], exprEndTag)
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
		subExprStrings = append(subExprStrings, s[indexStart+len(exprStartTag):indexEnd])
		begin = indexEnd + len(exprEndTag)
	}

	subExprs := make([]*vm.Program, 0, len(subExprStrings))
	for _, subExprString := range subExprStrings {
		program, err := expr.Compile(subExprString, expr.AllowUndefinedVariables())
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
		outString, ok := out.(string)
		if !ok {
			return "", fmt.Errorf("embedded expression returned non-string %v", out)
		}
		b.WriteString(outString)
	}
	b.WriteString(e.SubStrings[len(e.SubStrings)-1])

	return b.String(), nil
}
