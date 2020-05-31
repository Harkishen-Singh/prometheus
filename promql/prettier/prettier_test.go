package prettier

import (
	"reflect"
	"testing"

	"github.com/prometheus/prometheus/util/testutil"
)

type prettierTest struct {
	expr     string
	expected string
}

var exprs = []prettierTest{
	{
		expr: "first + second + third",
		expected: `    first
  +
    second
  +
    third`,
	},
	{
		expr: `first{foo="bar",a="b", c="d"}`,
		expected: `first{
  a="b",
  c="d",
  foo="bar",
}`,
	},
	{
		expr: `first{c="d",
			foo="bar",a="b",}`,
		expected: `first{
  a="b",
  c="d",
  foo="bar",
}`,
	},
	{
		expr: `first{foo="bar",a="b", c="d"} + second{foo="bar", c="d"}`,
		expected: `    first{
    a="b",
    c="d",
    foo="bar",
  }
  +
    second{
    c="d",
    foo="bar",
  }`,
	},
	{
		expr: `(first)`,
		expected: `(
  first
)`,
	},

	{
		expr: `((((first))))`,
		expected: `(
  (
    (
      (
        first
      )
    )
  )
)`,
	},
}

func TestPrettify(t *testing.T) {
	for _, expr := range exprs {
		p, err := New(PrettifyExpression, expr.expr)
		testutil.Ok(t, err)
		expression, err := p.parseExpr(expr.expr)
		testutil.Ok(t, err)
		formatted, err := p.Prettify(expression, reflect.TypeOf(""), 0, "")
		testutil.Ok(t, err)
		testutil.Equals(t, expr.expected, formatted, "formatting does not match")
	}
}
