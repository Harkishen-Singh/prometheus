package prettier

import (
	"fmt"
	"io/ioutil"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"
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
	Type       uint
}

// New returns a new prettier over the given slice of files.
func New(Type uint, content interface{}) (*Prettier, error) {
	var (
		ok         bool
		files      []string
		expression string
		typ        uint
	)
	switch Type {
	case PrettifyRules:
		files, ok = content.([]string)
		typ = PrettifyRules
	case PrettifyExpression:
		expression, ok = content.(string)
		typ = PrettifyExpression
	}
	if !ok {
		return nil, errors.Errorf("invalid type: %T", reflect.TypeOf(content))
	}
	return &Prettier{
		files:      files,
		expression: expression,
		Type:       typ,
	}, nil
}

var i = 0

// Prettify implements the formatting of the expressions.
// TODO: Add support for indetation via tabs/spaces as choices.
func (p *Prettier) Prettify(expr parser.Expr, prevType reflect.Type, indent int, init string) (string, error) {
	var format string
	switch n := expr.(type) {
	case *parser.AggregateExpr:
		if prevType.String() == "*parser.AggregateExpr" {
			indent--
		}
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

		format = padding(indent) + op
		if !containsParam {
			if without {
				format += " without "
			} else {
				format += " by "
			}
			format += grps + " ("
		} else {
			format += "(\n" + padding(indent+1) + paramVal + ",\n" + padding(indent+1)
		}
		s, err := p.Prettify(n.Expr, reflect.TypeOf(expr), indent+1, format)
		if err != nil {
			return "", err
		}
		s += "\n" + padding(indent) + ")"
	case *parser.BinaryExpr:
		var (
			indentChild = indent + 1
			isFirst     = true
		)
		if prevType.String() == "*parser.BinaryExpr" {
			indentChild--
			isFirst = false
		}

		op, ok := (*parser.ItemTyp)[n.Op]
		if !ok {
			return "", errors.New("invalid item-type")
		}
		lhs, err := p.Prettify(n.LHS, reflect.TypeOf(expr), indentChild, "")
		if err != nil {
			return "", errors.Wrap(err, "unable to prettify2")
		}
		rhs, err := p.Prettify(n.RHS, reflect.TypeOf(expr), indentChild, "")
		if err != nil {
			return "", errors.Wrap(err, "unable to prettify3")
		}

		format = ""
		if isFirst {
			indent++
			format += padding(indent)
		}
		format += lhs
		format += "\n" + padding(indent) + op + "\n" + padding(indent)
		if n.ReturnBool {
			format += "bool"
		}
		format += rhs
	case *parser.VectorSelector:
		var containsLabels bool
		metricName, err := getMetricName(n.LabelMatchers)
		if err != nil {
			return "", err
		}
		if len(n.LabelMatchers) > 1 {
			containsLabels = true
		}
		format = padding(indent) + metricName
		if containsLabels {
			format += "{\n"
			// apply labels
			labelMatchers := sortLabels(n.LabelMatchers)
			for _, m := range labelMatchers {
				format += padding(indent + 2)
				format += m.Name + "=\"" + m.Value + "\",\n"
			}
			format += padding(indent+1) + "}"
		}
		if n.Offset.String() != "0s" {
			t, err := getTimeValueStringified(n.Offset)
			if err != nil {
				return "", errors.Wrap(err, "invalid time")
			}
			format += " offset " + t
		}
	}
	return format, nil
}

type ruleGroupFiles struct {
	filename   string
	ruleGroups *rulefmt.RuleGroups
}

// Run executes the prettier over the rules files or expression.
func (p *Prettier) Run() []error {
	var (
		groupFiles []*rulefmt.RuleGroups
		errs       []error
	)
	switch p.Type {
	case PrettifyRules:
		for _, f := range p.files {
			ruleGroups, err := p.parseFile(f)
			if err != nil {
				for _, e := range err {
					errs = append(errs, errors.Wrapf(e, "file: %s", f))
				}
			}
			groupFiles = append(groupFiles, ruleGroups)
		}
		if errs != nil {
			return errs
		}
		for _, rgs := range groupFiles {
			for _, grps := range rgs.Groups {
				for _, rules := range grps.Rules {
					exprStr := rules.Expr.Value
					expr, err := p.parseExpr(exprStr)
					if err != nil {
						return []error{errors.Wrap(err, "parse error")}
					}
					fmt.Printf("%v\n", expr)
					formattedExpr, err := p.Prettify(expr, reflect.TypeOf(""), 0, "")
					if err != nil {
						return []error{errors.Wrap(err, "prettier error")}
					}
					fmt.Println("raw\n", formattedExpr)

					rules.Expr.SetString(formattedExpr)
				}
			}
		}

	}
	return nil
}

func (p *Prettier) parseExpr(expression string) (parser.Expr, error) {
	expr, err := parser.ParseExpr(expression)
	if err != nil {
		return nil, errors.Wrap(err, "parse expr")
	}
	return expr, nil
}

func (p *Prettier) parseFile(name string) (*rulefmt.RuleGroups, []error) {
	b, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, []error{errors.Wrap(err, "unable to read file")}
	}
	groups, errs := rulefmt.Parse(b)
	if errs != nil {
		return nil, errs
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

func getMetricName(l []*labels.Matcher) (string, error) {
	for _, m := range l {
		if m.Name == model.MetricNameLabel {
			return m.Value, nil
		}
	}
	return "", errors.New("metric_name not found")
}

func parseFloat(v float64) string {
	return strconv.FormatFloat(v, 'E', -1, 64)
}

func sortLabels(l []*labels.Matcher) (sortedLabels []*labels.Matcher) {
	var labelName []string
	for _, m := range l {
		if m.Name == model.MetricNameLabel {
			continue
		}
		labelName = append(labelName, m.Name)
	}
	sort.Strings(labelName)
	for _, n := range labelName {
		for _, m := range l {
			if n == m.Name {
				sortedLabels = append(sortedLabels, m)
			}
		}
	}
	return
}

func getTimeValueStringified(d time.Duration) (string, error) {
	units := []string{"y", "w", "d", "h", "m", "s"}
	for _, u := range units {
		if i := strings.Index(d.String(), u); i != -1 {
			return d.String()[:i+1], nil
		}
	}
	return "", fmt.Errorf("%s", d.String())
}

func padding(itr int) string {
	if itr == 0 {
		return ""
	}
	pad := " "
	for i := 1; i <= itr; i++ {
		pad += pad
	}
	return pad
}
