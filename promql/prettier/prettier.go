package prettier

import (
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/prometheus/prometheus/pkg/rulefmt"
	"github.com/prometheus/prometheus/promql/parser"
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
	files      []string
	expression string
}

// New returns a new prettier over the given slice of files.
func New(Type uint, content interface{}) (*Prettier, error) {
	var (
		ok         bool
		files      []string
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
		files:      files,
		expression: expression,
	}, nil
}

// Prettify implements the formatting of the expressions.
// TODO: Add support for indetation via tabs/spaces as choices.
func (p *Prettier) Prettify(expr parser.Expr, indent int, init string) (string, error) {
	switch n := expr.(type) {
	case *parser.AggregateExpr:
		op, err := getItemStringified(n.Op)
		if err != nil {
			return "", errors.Wrap(err, "unable to prettify")
		}
		// param
		var paramVal, grps string
		var containsParam bool
		without := n.Without
		if n.Param != nil {
			containsParam = true
			sp, ok := n.Param.(*parser.StringLiteral)
			if ok {
				paramVal = sp.Val
			}
			np, ok := n.Param.(*parser.NumberLiteral)
			if ok {
				paramVal = parseFloat(np.Val)
			}
		}
		// groups
		if len(n.Grouping) != 0 {
			grps = "("
			grps += strings.Join(n.Grouping, ", ")
			grps = ")"
		}

		format := op
		if !containsParam {
			if without {
				format += " without "
			} else {
				format += " by "
			}
			format += grps + " ("
		} else {
			format += "(\n" + padding(indent+2) + paramVal + ",\n" + padding(indent+2)
		}
		s, err := p.Prettify(n.Expr, indent+2, format)
		if err != nil {
			return "", err
		}
		s += "\n" + padding(indent) + ")"
		return s, nil
	}
	return init, nil
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

func getItemStringified(typ parser.ItemType) (string, error) {
	for k, v := range *parser.Key {
		if v == typ {
			return k, nil
		}
	}
	return "", errors.New("invalid item-type")
}

func parseFloat(v float64) string {
	return strconv.FormatFloat(v, 'E', -1, 64)
}

func padding(itr int) string {
	if itr < 1 {
		return ""
	}
	pad := "  " // 2 spaces
	for i := 1; i < itr; i++ {
		pad += pad
	}
	return pad
}
