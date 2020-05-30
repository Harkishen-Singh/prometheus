package prettier

import (
	"reflect"
	"testing"
	// "github.com/prometheus/prometheus/promql/parser"
	"github.com/prometheus/prometheus/util/testutil"
)

type prettierTest struct {
	Expr string
	Expected string
}

var exprs = []prettierTest{
	{
		Expr: "first + second + third",
		Expected: `
    first
  +
    second
  +
    third
		`,
	},
}

func TestPrettify(t *testing.T) {
	for _, expr := range exprs {
		p, err := New(PrettifyExpression, expr.Expr)
		testutil.Ok(t, err)
		expression, err := p.parseExpr(expr.Expr)
		testutil.Ok(t, err)
		formatted, err := p.Prettify(expression, reflect.TypeOf(""), 0, "")
		testutil.Ok(t, err)
		testutil.Equals(t, expr.Expected, formatted, "formatting does not match")
	}
}