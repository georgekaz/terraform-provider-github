package checkers

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

// FloatCompare detects situation like
//
//	assert.Equal(t, 42.42, result)
//	assert.EqualValues(t, 42.42, result)
//	assert.Exactly(t, 42.42, result)
//	assert.True(t, result == 42.42)
//	assert.False(t, result != 42.42)
//
// and requires
//
//	assert.InEpsilon(t, 42.42, result, 0.0001) // Or assert.InDelta
type FloatCompare struct{}

// NewFloatCompare constructs FloatCompare checker.
func NewFloatCompare() FloatCompare { return FloatCompare{} }
func (FloatCompare) Name() string   { return "float-compare" }

func (checker FloatCompare) Check(pass *analysis.Pass, call *CallMeta) *analysis.Diagnostic {
	invalid := func() bool {
		switch call.Fn.NameFTrimmed {
		case "Equal", "EqualValues", "Exactly":
			return len(call.Args) > 1 && (isFloat(pass, call.Args[0]) || isFloat(pass, call.Args[1]))

		case "True":
			return len(call.Args) > 0 && isFloatCompare(pass, call.Args[0], token.EQL)

		case "False":
			return len(call.Args) > 0 && isFloatCompare(pass, call.Args[0], token.NEQ)
		}
		return false
	}()

	if invalid {
		format := "use %s.InEpsilon (or InDelta)"
		if call.Fn.IsFmt {
			format = "use %s.InEpsilonf (or InDeltaf)"
		}
		return newDiagnostic(checker.Name(), call, fmt.Sprintf(format, call.SelectorXStr), nil)
	}
	return nil
}

func isFloat(pass *analysis.Pass, expr ast.Expr) bool {
	t := pass.TypesInfo.TypeOf(expr)
	if t == nil {
		return false
	}

	bt, ok := t.Underlying().(*types.Basic)
	return ok && (bt.Info()&types.IsFloat > 0)
}

func isFloatCompare(p *analysis.Pass, e ast.Expr, op token.Token) bool {
	be, ok := e.(*ast.BinaryExpr)
	if !ok {
		return false
	}
	return be.Op == op && (isFloat(p, be.X) || isFloat(p, be.Y))
}
