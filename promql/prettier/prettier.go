package prettier

import (
	"io/ioutil"
	"reflect"

	"github.com/pkg/errors"
	"github.com/prometheus/prometheus/promql/parser"
	"github.com/prometheus/prometheus/pkg/rulefmt"
)

const (
	// PrettifyRules for prettifying rules files along with the
	// expressions in them.
	PrettifyRules = iota
	// PrettifyExpression for prettifying instantaneous expressions.
	PrettifyExpression
)

// Prettier handles the prettifying and formatting operation over a
// list of rules files or a single expression.
type Prettier struct {
	files []string
	expression string
}

// New returns a new prettier over the given slice of files.
func New(Type uint, content interface{}) (*Prettier, error) {
	var (
		ok bool
		files []string
		expression string
	)
	switch Type {
	case PrettifyRules:
		files, ok = content.([]string)
	case PrettifyExpression:
		expression, ok = content.(string)
	}
	if !ok {
		return nil, errors.Errorf("invalid type: %T", reflect.TypeOf(content))
	}
	return &Prettier{
		files: files,
		expression: expression,
	}, nil
}

func (p *Prettier) parseExpr(expression string) (parser.Expr, error) {
	expr, err := parser.ParseExpr(expression)
	if err != nil {
		return nil, errors.Wrap(err, "parse expr")
	}
	return expr, nil
}

func (p *Prettier) parseFile(name string) (*rulefmt.RuleGroups, error) {
	b, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read file")
	}
	groups, errs := rulefmt.Parse(b)
	if errs != nil {
		return nil, errors.New("invalid rule files. consider checking rules for errors before prettifying")
	}
	return groups, nil
}