// Copyright 2015 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package parser

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/common/model"

	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/util/testutil"
)

var testExpr = []struct {
	input    string // The input to be parsed.
	expected Expr   // The expected expression AST.
	fail     bool   // Whether parsing is supposed to fail.
	errMsg   string // If not empty the parsing error has to contain this string.
}{
	// Scalars and scalar-to-scalar operations.
	{
		input: "1",
		expected: &NumberLiteral{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57359, Pos: 0, Val: "1"}}},
			Val:            1,
			PosRange:       PositionRange{Start: 0, End: 1},
		},
	}, {
		input: "+Inf",
		expected: &NumberLiteral{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57368, Pos: 0, Val: "+"}, {Typ: 57359, Pos: 1, Val: "Inf"}}},
			Val:            math.Inf(1),
			PosRange:       PositionRange{Start: 0, End: 4},
		},
	}, {
		input: "-Inf",
		expected: &NumberLiteral{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57384, Pos: 0, Val: "-"}, {Typ: 57359, Pos: 1, Val: "Inf"}}},
			Val:            math.Inf(-1),
			PosRange:       PositionRange{Start: 0, End: 4},
		},
	}, {
		input: ".5",
		expected: &NumberLiteral{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57359, Pos: 0, Val: ".5"}}},
			Val:            0.5,
			PosRange:       PositionRange{Start: 0, End: 2},
		},
	}, {
		input: "5.",
		expected: &NumberLiteral{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57359, Pos: 0, Val: "5."}}},
			Val:            5,
			PosRange:       PositionRange{Start: 0, End: 2},
		},
	}, {
		input: "123.4567",
		expected: &NumberLiteral{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57359, Pos: 0, Val: "123.4567"}}},
			Val:            123.4567,
			PosRange:       PositionRange{Start: 0, End: 8},
		},
	}, {
		input: "5e-3",
		expected: &NumberLiteral{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57359, Pos: 0, Val: "5e-3"}}},
			Val:            0.005,
			PosRange:       PositionRange{Start: 0, End: 4},
		},
	}, {
		input: "5e3",
		expected: &NumberLiteral{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57359, Pos: 0, Val: "5e3"}}},
			Val:            5000,
			PosRange:       PositionRange{Start: 0, End: 3},
		},
	}, {
		input: "0xc",
		expected: &NumberLiteral{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57359, Pos: 0, Val: "0xc"}}},
			Val:            12,
			PosRange:       PositionRange{Start: 0, End: 3},
		},
	}, {
		input: "0755",
		expected: &NumberLiteral{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57359, Pos: 0, Val: "0755"}}},
			Val:            493,
			PosRange:       PositionRange{Start: 0, End: 4},
		},
	}, {
		input: "+5.5e-3",
		expected: &NumberLiteral{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57368, Pos: 0, Val: "+"}, {Typ: 57359, Pos: 1, Val: "5.5e-3"}}},
			Val:            0.0055,
			PosRange:       PositionRange{Start: 0, End: 7},
		},
	}, {
		input: "-0755",
		expected: &NumberLiteral{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57384, Pos: 0, Val: "-"}, {Typ: 57359, Pos: 1, Val: "0755"}}},
			Val:            -493,
			PosRange:       PositionRange{Start: 0, End: 5},
		},
	}, {
		input: "1 + 1",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57359, Pos: 0, Val: "1"}, {Typ: 57368, Pos: 2, Val: "+"}, {Typ: 57359, Pos: 4, Val: "1"}}},
			Op:             ADD,
			LHS: &NumberLiteral{
				Val:      1,
				PosRange: PositionRange{Start: 0, End: 1},
			},
			RHS: &NumberLiteral{
				Val:      1,
				PosRange: PositionRange{Start: 4, End: 5},
			},
		},
	}, {
		input: "1 - 1",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57359, Pos: 0, Val: "1"}, {Typ: 57384, Pos: 2, Val: "-"}, {Typ: 57359, Pos: 4, Val: "1"}}},
			Op:             SUB,
			LHS: &NumberLiteral{
				Val:      1,
				PosRange: PositionRange{Start: 0, End: 1},
			},
			RHS: &NumberLiteral{
				Val:      1,
				PosRange: PositionRange{Start: 4, End: 5},
			},
		},
	}, {
		input: "1 * 1",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57359, Pos: 0, Val: "1"}, {Typ: 57380, Pos: 2, Val: "*"}, {Typ: 57359, Pos: 4, Val: "1"}}},
			Op:             MUL,
			LHS: &NumberLiteral{
				Val:      1,
				PosRange: PositionRange{Start: 0, End: 1},
			},
			RHS: &NumberLiteral{
				Val:      1,
				PosRange: PositionRange{Start: 4, End: 5},
			},
		},
	}, {
		input: "1 % 1",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57359, Pos: 0, Val: "1"}, {Typ: 57379, Pos: 2, Val: "%"}, {Typ: 57359, Pos: 4, Val: "1"}}},
			Op:             MOD,
			LHS: &NumberLiteral{
				Val:      1,
				PosRange: PositionRange{Start: 0, End: 1},
			},
			RHS: &NumberLiteral{
				Val:      1,
				PosRange: PositionRange{Start: 4, End: 5},
			},
		},
	}, {
		input: "1 / 1",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57359, Pos: 0, Val: "1"}, {Typ: 57369, Pos: 2, Val: "/"}, {Typ: 57359, Pos: 4, Val: "1"}}},
			Op:             DIV,
			LHS: &NumberLiteral{
				Val:      1,
				PosRange: PositionRange{Start: 0, End: 1},
			},
			RHS: &NumberLiteral{
				Val:      1,
				PosRange: PositionRange{Start: 4, End: 5},
			},
		},
	}, {
		input: "1 == bool 1",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57359, Pos: 0, Val: "1"}, {Typ: 57370, Pos: 2, Val: "=="}, {Typ: 57401, Pos: 5, Val: "bool"}, {Typ: 57359, Pos: 10, Val: "1"}}},
			Op:             EQL,
			LHS: &NumberLiteral{
				Val:      1,
				PosRange: PositionRange{Start: 0, End: 1},
			},
			RHS: &NumberLiteral{
				Val:      1,
				PosRange: PositionRange{Start: 10, End: 11},
			},
			ReturnBool: true,
		},
	}, {
		input: "1 != bool 1",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57359, Pos: 0, Val: "1"}, {Typ: 57381, Pos: 2, Val: "!="}, {Typ: 57401, Pos: 5, Val: "bool"}, {Typ: 57359, Pos: 10, Val: "1"}}},
			Op:             NEQ,
			LHS: &NumberLiteral{
				Val:      1,
				PosRange: PositionRange{Start: 0, End: 1},
			},
			RHS: &NumberLiteral{
				Val:      1,
				PosRange: PositionRange{Start: 10, End: 11},
			},
			ReturnBool: true,
		},
	}, {
		input: "1 > bool 1",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57359, Pos: 0, Val: "1"}, {Typ: 57373, Pos: 2, Val: ">"}, {Typ: 57401, Pos: 4, Val: "bool"}, {Typ: 57359, Pos: 9, Val: "1"}}},
			Op:             GTR,
			LHS: &NumberLiteral{
				Val:      1,
				PosRange: PositionRange{Start: 0, End: 1},
			},
			RHS: &NumberLiteral{
				Val:      1,
				PosRange: PositionRange{Start: 9, End: 10},
			},
			ReturnBool: true,
		},
	},
	{
		input: "1 >= bool 1",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57359, Pos: 0, Val: "1"}, {Typ: 57372, Pos: 2, Val: ">="}, {Typ: 57401, Pos: 5, Val: "bool"}, {Typ: 57359, Pos: 10, Val: "1"}}},
			Op:             GTE,
			LHS: &NumberLiteral{
				Val:      1,
				PosRange: PositionRange{Start: 0, End: 1},
			},
			RHS: &NumberLiteral{
				Val:      1,
				PosRange: PositionRange{Start: 10, End: 11},
			},
			ReturnBool: true,
		},
	}, {
		input: "1 < bool 1",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57359, Pos: 0, Val: "1"}, {Typ: 57376, Pos: 2, Val: "<"}, {Typ: 57401, Pos: 4, Val: "bool"}, {Typ: 57359, Pos: 9, Val: "1"}}},
			Op:             LSS,
			LHS: &NumberLiteral{
				Val:      1,
				PosRange: PositionRange{Start: 0, End: 1},
			},
			RHS: &NumberLiteral{
				Val:      1,
				PosRange: PositionRange{Start: 9, End: 10},
			},
			ReturnBool: true,
		},
	}, {
		input: "1 <= bool 1",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57359, Pos: 0, Val: "1"}, {Typ: 57377, Pos: 2, Val: "<="}, {Typ: 57401, Pos: 5, Val: "bool"}, {Typ: 57359, Pos: 10, Val: "1"}}},
			Op:             LTE,
			LHS: &NumberLiteral{
				Val:      1,
				PosRange: PositionRange{Start: 0, End: 1},
			},
			RHS: &NumberLiteral{
				Val:      1,
				PosRange: PositionRange{Start: 10, End: 11},
			},
			ReturnBool: true,
		},
	}, {
		input: "-1^2",
		expected: &UnaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57384, Pos: 0, Val: "-"}, {Typ: 57359, Pos: 1, Val: "1"}, {Typ: 57383, Pos: 2, Val: "^"}, {Typ: 57359, Pos: 3, Val: "2"}}},
			Op:             SUB,
			Expr: &BinaryExpr{
				Op: POW,
				LHS: &NumberLiteral{
					Val:      1,
					PosRange: PositionRange{Start: 1, End: 2},
				},
				RHS: &NumberLiteral{
					Val:      2,
					PosRange: PositionRange{Start: 3, End: 4},
				},
			},
		},
	}, {
		input: "-1*2",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57384, Pos: 0, Val: "-"}, {Typ: 57359, Pos: 1, Val: "1"}, {Typ: 57380, Pos: 2, Val: "*"}, {Typ: 57359, Pos: 3, Val: "2"}}},
			Op:             MUL,
			LHS: &NumberLiteral{
				Val:      -1,
				PosRange: PositionRange{Start: 0, End: 2},
			},
			RHS: &NumberLiteral{
				Val:      2,
				PosRange: PositionRange{Start: 3, End: 4},
			},
		},
	}, {
		input: "-1+2",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57384, Pos: 0, Val: "-"}, {Typ: 57359, Pos: 1, Val: "1"}, {Typ: 57368, Pos: 2, Val: "+"}, {Typ: 57359, Pos: 3, Val: "2"}}},
			Op:             ADD,
			LHS: &NumberLiteral{
				Val:      -1,
				PosRange: PositionRange{Start: 0, End: 2},
			},
			RHS: &NumberLiteral{
				Val:      2,
				PosRange: PositionRange{Start: 3, End: 4},
			},
		},
	}, {
		input: "-1^-2",
		expected: &UnaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57384, Pos: 0, Val: "-"}, {Typ: 57359, Pos: 1, Val: "1"}, {Typ: 57383, Pos: 2, Val: "^"}, {Typ: 57384, Pos: 3, Val: "-"}, {Typ: 57359, Pos: 4, Val: "2"}}},
			Op:             SUB,
			Expr: &BinaryExpr{
				Op: POW,
				LHS: &NumberLiteral{
					Val:      1,
					PosRange: PositionRange{Start: 1, End: 2},
				},
				RHS: &NumberLiteral{
					Val:      -2,
					PosRange: PositionRange{Start: 3, End: 5},
				},
			},
		},
	}, {
		input: "+1 + -2 * 1",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57368, Pos: 0, Val: "+"}, {Typ: 57359, Pos: 1, Val: "1"}, {Typ: 57368, Pos: 3, Val: "+"}, {Typ: 57384, Pos: 5, Val: "-"}, {Typ: 57359, Pos: 6, Val: "2"}, {Typ: 57380, Pos: 8, Val: "*"}, {Typ: 57359, Pos: 10, Val: "1"}}},
			Op:             ADD,
			LHS: &NumberLiteral{
				Val:      1,
				PosRange: PositionRange{Start: 0, End: 2},
			},
			RHS: &BinaryExpr{
				Op: MUL,
				LHS: &NumberLiteral{
					Val:      -2,
					PosRange: PositionRange{Start: 5, End: 7},
				},
				RHS: &NumberLiteral{
					Val:      1,
					PosRange: PositionRange{Start: 10, End: 11},
				},
			},
		},
	}, {
		input: "1 + 2/(3*1)",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57359, Pos: 0, Val: "1"}, {Typ: 57368, Pos: 2, Val: "+"}, {Typ: 57359, Pos: 4, Val: "2"}, {Typ: 57369, Pos: 5, Val: "/"}, {Typ: 57357, Pos: 6, Val: "("}, {Typ: 57359, Pos: 7, Val: "3"}, {Typ: 57380, Pos: 8, Val: "*"}, {Typ: 57359, Pos: 9, Val: "1"}, {Typ: 57362, Pos: 10, Val: ")"}}},
			Op:             ADD,
			LHS: &NumberLiteral{
				Val:      1,
				PosRange: PositionRange{Start: 0, End: 1},
			},
			RHS: &BinaryExpr{
				Op: DIV,
				LHS: &NumberLiteral{
					Val:      2,
					PosRange: PositionRange{Start: 4, End: 5},
				},
				RHS: &ParenExpr{
					Expr: &BinaryExpr{
						Op: MUL,
						LHS: &NumberLiteral{
							Val:      3,
							PosRange: PositionRange{Start: 7, End: 8},
						},
						RHS: &NumberLiteral{
							Val:      1,
							PosRange: PositionRange{Start: 9, End: 10},
						},
					},
					PosRange: PositionRange{Start: 6, End: 11},
				},
			},
		},
	}, {
		input: "1 < bool 2 - 1 * 2",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57359, Pos: 0, Val: "1"}, Item{Typ: 57376, Pos: 2, Val: "<"}, Item{Typ: 57401, Pos: 4, Val: "bool"}, Item{Typ: 57359, Pos: 9, Val: "2"}, Item{Typ: 57384, Pos: 11, Val: "-"}, Item{Typ: 57359, Pos: 13, Val: "1"}, Item{Typ: 57380, Pos: 15, Val: "*"}, Item{Typ: 57359, Pos: 17, Val: "2"}}},
			Op:             LSS,
			ReturnBool:     true,
			LHS: &NumberLiteral{
				Val:      1,
				PosRange: PositionRange{Start: 0, End: 1},
			},
			RHS: &BinaryExpr{
				Op: SUB,
				LHS: &NumberLiteral{
					Val:      2,
					PosRange: PositionRange{Start: 9, End: 10},
				},
				RHS: &BinaryExpr{
					Op: MUL,
					LHS: &NumberLiteral{
						Val:      1,
						PosRange: PositionRange{Start: 13, End: 14},
					},
					RHS: &NumberLiteral{
						Val:      2,
						PosRange: PositionRange{Start: 17, End: 18},
					},
				},
			},
		},
	}, {
		input: "-some_metric",
		expected: &UnaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57384, Pos: 0, Val: "-"}, {Typ: 57354, Pos: 1, Val: "some_metric"}}},
			Op:             SUB,
			Expr: &VectorSelector{
				Name: "some_metric",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, model.MetricNameLabel, "some_metric"),
				},
				PosRange: PositionRange{
					Start: 1,
					End:   12,
				},
			},
		},
	}, {
		input: "+some_metric",
		expected: &UnaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57368, Pos: 0, Val: "+"}, {Typ: 57354, Pos: 1, Val: "some_metric"}}},
			Op:             ADD,
			Expr: &VectorSelector{
				Name: "some_metric",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, model.MetricNameLabel, "some_metric"),
				},
				PosRange: PositionRange{
					Start: 1,
					End:   12,
				},
			},
		},
	}, {
		input: " +some_metric",
		expected: &UnaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{{Typ: 57368, Pos: 1, Val: "+"}, {Typ: 57354, Pos: 2, Val: "some_metric"}}},
			Op:             ADD,
			Expr: &VectorSelector{
				Name: "some_metric",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "some_metric"),
				},
				PosRange: PositionRange{
					Start: 2,
					End:   13,
				},
			},
			StartPos: 1,
		},
	},
	{
		input:  "",
		fail:   true,
		errMsg: "no expression found in input",
	}, {
		input:  "# just a comment\n\n",
		fail:   true,
		errMsg: "no expression found in input",
	}, {
		input:  "1+",
		fail:   true,
		errMsg: "unexpected end of input",
	}, {
		input:  ".",
		fail:   true,
		errMsg: "unexpected character: '.'",
	}, {
		input:  "2.5.",
		fail:   true,
		errMsg: "unexpected character: '.'",
	}, {
		input:  "100..4",
		fail:   true,
		errMsg: `unexpected number ".4"`,
	}, {
		input:  "0deadbeef",
		fail:   true,
		errMsg: "bad number or duration syntax: \"0de\"",
	}, {
		input:  "1 /",
		fail:   true,
		errMsg: "unexpected end of input",
	}, {
		input:  "*1",
		fail:   true,
		errMsg: "unexpected <op:*>",
	}, {
		input:  "(1))",
		fail:   true,
		errMsg: "unexpected right parenthesis ')'",
	}, {
		input:  "((1)",
		fail:   true,
		errMsg: "unclosed left parenthesis",
	}, {
		input:  "999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999999",
		fail:   true,
		errMsg: "out of range",
	}, {
		input:  "(",
		fail:   true,
		errMsg: "unclosed left parenthesis",
	}, {
		input:  "1 and 1",
		fail:   true,
		errMsg: "set operator \"and\" not allowed in binary scalar expression",
	}, {
		input:  "1 == 1",
		fail:   true,
		errMsg: "1:3: parse error: comparisons between scalars must use BOOL modifier",
	}, {
		input:  "1 or 1",
		fail:   true,
		errMsg: "set operator \"or\" not allowed in binary scalar expression",
	}, {
		input:  "1 unless 1",
		fail:   true,
		errMsg: "set operator \"unless\" not allowed in binary scalar expression",
	}, {
		input:  "1 !~ 1",
		fail:   true,
		errMsg: `unexpected character after '!': '~'`,
	}, {
		input:  "1 =~ 1",
		fail:   true,
		errMsg: `unexpected character after '=': '~'`,
	}, {
		input:  `-"string"`,
		fail:   true,
		errMsg: `unary expression only allowed on expressions of type scalar or instant vector, got "string"`,
	}, {
		input:  `-test[5m]`,
		fail:   true,
		errMsg: `unary expression only allowed on expressions of type scalar or instant vector, got "range vector"`,
	}, {
		input:  `*test`,
		fail:   true,
		errMsg: "unexpected <op:*>",
	}, {
		input:  "1 offset 1d",
		fail:   true,
		errMsg: "offset modifier must be preceded by an instant or range selector",
	}, {
		input:  "foo offset 1s offset 2s",
		fail:   true,
		errMsg: "offset may not be set multiple times",
	}, {
		input:  "a - on(b) ignoring(c) d",
		fail:   true,
		errMsg: "1:11: parse error: unexpected <ignoring>",
	},
	// Vector binary operations.
	{
		input: "foo * bar",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57380, Pos: 4, Val: "*"}, Item{Typ: 57354, Pos: 6, Val: "bar"}}},
			Op:             MUL,
			LHS: &VectorSelector{
				Name: "foo",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, model.MetricNameLabel, "foo"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   3,
				},
			},
			RHS: &VectorSelector{
				Name: "bar",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "bar"),
				},
				PosRange: PositionRange{
					Start: 6,
					End:   9,
				},
			},
			VectorMatching: &VectorMatching{Card: CardOneToOne},
		},
	}, {
		input: "foo * sum",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57380, Pos: 4, Val: "*"}, Item{Typ: 57397, Pos: 6, Val: "sum"}}},
			Op:             MUL,
			LHS: &VectorSelector{
				Name: "foo",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   3,
				},
			},
			RHS: &VectorSelector{
				Name: "sum",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "sum"),
				},
				PosRange: PositionRange{
					Start: 6,
					End:   9,
				},
			},
			VectorMatching: &VectorMatching{Card: CardOneToOne},
		},
	}, {
		input: "foo == 1",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57370, Pos: 4, Val: "=="}, Item{Typ: 57359, Pos: 7, Val: "1"}}},
			Op:             EQL,
			LHS: &VectorSelector{
				Name: "foo",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   3,
				},
			},
			RHS: &NumberLiteral{
				Val:      1,
				PosRange: PositionRange{Start: 7, End: 8},
			},
		},
	}, {
		input: "foo == bool 1",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57370, Pos: 4, Val: "=="}, Item{Typ: 57401, Pos: 7, Val: "bool"}, Item{Typ: 57359, Pos: 12, Val: "1"}}},
			Op:             EQL,
			LHS: &VectorSelector{
				Name: "foo",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   3,
				},
			},
			RHS: &NumberLiteral{
				Val:      1,
				PosRange: PositionRange{Start: 12, End: 13},
			},
			ReturnBool: true,
		},
	}, {
		input: "2.5 / bar",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57359, Pos: 0, Val: "2.5"}, Item{Typ: 57369, Pos: 4, Val: "/"}, Item{Typ: 57354, Pos: 6, Val: "bar"}}},
			Op:             DIV,
			LHS: &NumberLiteral{
				Val:      2.5,
				PosRange: PositionRange{Start: 0, End: 3},
			},
			RHS: &VectorSelector{
				Name: "bar",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "bar"),
				},
				PosRange: PositionRange{
					Start: 6,
					End:   9,
				},
			},
		},
	}, {
		input: "foo and bar",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57374, Pos: 4, Val: "and"}, Item{Typ: 57354, Pos: 8, Val: "bar"}}},
			Op:             LAND,
			LHS: &VectorSelector{
				Name: "foo",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   3,
				},
			},
			RHS: &VectorSelector{
				Name: "bar",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "bar"),
				},
				PosRange: PositionRange{
					Start: 8,
					End:   11,
				},
			},
			VectorMatching: &VectorMatching{Card: CardManyToMany},
		},
	}, {
		input: "foo or bar",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57375, Pos: 4, Val: "or"}, Item{Typ: 57354, Pos: 7, Val: "bar"}}},
			Op:             LOR,
			LHS: &VectorSelector{
				Name: "foo",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   3,
				},
			},
			RHS: &VectorSelector{
				Name: "bar",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "bar"),
				},
				PosRange: PositionRange{
					Start: 7,
					End:   10,
				},
			},
			VectorMatching: &VectorMatching{Card: CardManyToMany},
		},
	}, {
		input: "foo unless bar",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57378, Pos: 4, Val: "unless"}, Item{Typ: 57354, Pos: 11, Val: "bar"}}},
			Op:             LUNLESS,
			LHS: &VectorSelector{
				Name: "foo",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   3,
				},
			},
			RHS: &VectorSelector{
				Name: "bar",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "bar"),
				},
				PosRange: PositionRange{
					Start: 11,
					End:   14,
				},
			},
			VectorMatching: &VectorMatching{Card: CardManyToMany},
		},
	}, {
		// Test and/or precedence and reassigning of operands.
		input: "foo + bar or bla and blub",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57368, Pos: 4, Val: "+"}, Item{Typ: 57354, Pos: 6, Val: "bar"}, Item{Typ: 57375, Pos: 10, Val: "or"}, Item{Typ: 57354, Pos: 13, Val: "bla"}, Item{Typ: 57374, Pos: 17, Val: "and"}, Item{Typ: 57354, Pos: 21, Val: "blub"}}},
			Op:             LOR,
			LHS: &BinaryExpr{
				Op: ADD,
				LHS: &VectorSelector{
					Name: "foo",
					LabelMatchers: []*labels.Matcher{
						mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
					},
					PosRange: PositionRange{
						Start: 0,
						End:   3,
					},
				},
				RHS: &VectorSelector{
					Name: "bar",
					LabelMatchers: []*labels.Matcher{
						mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "bar"),
					},
					PosRange: PositionRange{
						Start: 6,
						End:   9,
					},
				},
				VectorMatching: &VectorMatching{Card: CardOneToOne},
			},
			RHS: &BinaryExpr{
				Op: LAND,
				LHS: &VectorSelector{
					Name: "bla",
					LabelMatchers: []*labels.Matcher{
						mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "bla"),
					},
					PosRange: PositionRange{
						Start: 13,
						End:   16,
					},
				},
				RHS: &VectorSelector{
					Name: "blub",
					LabelMatchers: []*labels.Matcher{
						mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "blub"),
					},
					PosRange: PositionRange{
						Start: 21,
						End:   25,
					},
				},
				VectorMatching: &VectorMatching{Card: CardManyToMany},
			},
			VectorMatching: &VectorMatching{Card: CardManyToMany},
		},
	}, {
		// Test and/or/unless precedence.
		input: "foo and bar unless baz or qux",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57374, Pos: 4, Val: "and"}, Item{Typ: 57354, Pos: 8, Val: "bar"}, Item{Typ: 57378, Pos: 12, Val: "unless"}, Item{Typ: 57354, Pos: 19, Val: "baz"}, Item{Typ: 57375, Pos: 23, Val: "or"}, Item{Typ: 57354, Pos: 26, Val: "qux"}}},
			Op:             LOR,
			LHS: &BinaryExpr{
				Op: LUNLESS,
				LHS: &BinaryExpr{
					Op: LAND,
					LHS: &VectorSelector{
						Name: "foo",
						LabelMatchers: []*labels.Matcher{
							mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
						},
						PosRange: PositionRange{
							Start: 0,
							End:   3,
						},
					},
					RHS: &VectorSelector{
						Name: "bar",
						LabelMatchers: []*labels.Matcher{
							mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "bar"),
						},
						PosRange: PositionRange{
							Start: 8,
							End:   11,
						},
					},
					VectorMatching: &VectorMatching{Card: CardManyToMany},
				},
				RHS: &VectorSelector{
					Name: "baz",
					LabelMatchers: []*labels.Matcher{
						mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "baz"),
					},
					PosRange: PositionRange{
						Start: 19,
						End:   22,
					},
				},
				VectorMatching: &VectorMatching{Card: CardManyToMany},
			},
			RHS: &VectorSelector{
				Name: "qux",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "qux"),
				},
				PosRange: PositionRange{
					Start: 26,
					End:   29,
				},
			},
			VectorMatching: &VectorMatching{Card: CardManyToMany},
		},
	}, {
		// Test precedence and reassigning of operands.
		input: "bar + on(foo) bla / on(baz, buz) group_right(test) blub",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "bar"}, Item{Typ: 57368, Pos: 4, Val: "+"}, Item{Typ: 57407, Pos: 6, Val: "on"}, Item{Typ: 57357, Pos: 8, Val: "("}, Item{Typ: 57354, Pos: 9, Val: "foo"}, Item{Typ: 57362, Pos: 12, Val: ")"}, Item{Typ: 57354, Pos: 14, Val: "bla"}, Item{Typ: 57369, Pos: 18, Val: "/"}, Item{Typ: 57407, Pos: 20, Val: "on"}, Item{Typ: 57357, Pos: 22, Val: "("}, Item{Typ: 57354, Pos: 23, Val: "baz"}, Item{Typ: 57349, Pos: 26, Val: ","}, Item{Typ: 57354, Pos: 28, Val: "buz"}, Item{Typ: 57362, Pos: 31, Val: ")"}, Item{Typ: 57404, Pos: 33, Val: "group_right"}, Item{Typ: 57357, Pos: 44, Val: "("}, Item{Typ: 57354, Pos: 45, Val: "test"}, Item{Typ: 57362, Pos: 49, Val: ")"}, Item{Typ: 57354, Pos: 51, Val: "blub"}}},
			Op:             ADD,
			LHS: &VectorSelector{
				Name: "bar",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "bar"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   3,
				},
			},
			RHS: &BinaryExpr{
				Op: DIV,
				LHS: &VectorSelector{
					Name: "bla",
					LabelMatchers: []*labels.Matcher{
						mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "bla"),
					},
					PosRange: PositionRange{
						Start: 14,
						End:   17,
					},
				},
				RHS: &VectorSelector{
					Name: "blub",
					LabelMatchers: []*labels.Matcher{
						mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "blub"),
					},
					PosRange: PositionRange{
						Start: 51,
						End:   55,
					},
				},
				VectorMatching: &VectorMatching{
					Card:           CardOneToMany,
					MatchingLabels: []string{"baz", "buz"},
					On:             true,
					Include:        []string{"test"},
				},
			},
			VectorMatching: &VectorMatching{
				Card:           CardOneToOne,
				MatchingLabels: []string{"foo"},
				On:             true,
			},
		},
	}, {
		input: "foo * on(test,blub) bar",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57380, Pos: 4, Val: "*"}, Item{Typ: 57407, Pos: 6, Val: "on"}, Item{Typ: 57357, Pos: 8, Val: "("}, Item{Typ: 57354, Pos: 9, Val: "test"}, Item{Typ: 57349, Pos: 13, Val: ","}, Item{Typ: 57354, Pos: 14, Val: "blub"}, Item{Typ: 57362, Pos: 18, Val: ")"}, Item{Typ: 57354, Pos: 20, Val: "bar"}}},
			Op:             MUL,
			LHS: &VectorSelector{
				Name: "foo",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   3,
				},
			},
			RHS: &VectorSelector{
				Name: "bar",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "bar"),
				},
				PosRange: PositionRange{
					Start: 20,
					End:   23,
				},
			},
			VectorMatching: &VectorMatching{
				Card:           CardOneToOne,
				MatchingLabels: []string{"test", "blub"},
				On:             true,
			},
		},
	}, {
		input: "foo * on(test,blub) group_left bar",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57380, Pos: 4, Val: "*"}, Item{Typ: 57407, Pos: 6, Val: "on"}, Item{Typ: 57357, Pos: 8, Val: "("}, Item{Typ: 57354, Pos: 9, Val: "test"}, Item{Typ: 57349, Pos: 13, Val: ","}, Item{Typ: 57354, Pos: 14, Val: "blub"}, Item{Typ: 57362, Pos: 18, Val: ")"}, Item{Typ: 57403, Pos: 20, Val: "group_left"}, Item{Typ: 57354, Pos: 31, Val: "bar"}}},
			Op:             MUL,
			LHS: &VectorSelector{
				Name: "foo",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   3,
				},
			},
			RHS: &VectorSelector{
				Name: "bar",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "bar"),
				},
				PosRange: PositionRange{
					Start: 31,
					End:   34,
				},
			},
			VectorMatching: &VectorMatching{
				Card:           CardManyToOne,
				MatchingLabels: []string{"test", "blub"},
				On:             true,
			},
		},
	}, {
		input: "foo and on(test,blub) bar",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57374, Pos: 4, Val: "and"}, Item{Typ: 57407, Pos: 8, Val: "on"}, Item{Typ: 57357, Pos: 10, Val: "("}, Item{Typ: 57354, Pos: 11, Val: "test"}, Item{Typ: 57349, Pos: 15, Val: ","}, Item{Typ: 57354, Pos: 16, Val: "blub"}, Item{Typ: 57362, Pos: 20, Val: ")"}, Item{Typ: 57354, Pos: 22, Val: "bar"}}},
			Op:             LAND,
			LHS: &VectorSelector{
				Name: "foo",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   3,
				},
			},
			RHS: &VectorSelector{
				Name: "bar",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "bar"),
				},
				PosRange: PositionRange{
					Start: 22,
					End:   25,
				},
			},
			VectorMatching: &VectorMatching{
				Card:           CardManyToMany,
				MatchingLabels: []string{"test", "blub"},
				On:             true,
			},
		},
	}, {
		input: "foo and on() bar",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57374, Pos: 4, Val: "and"}, Item{Typ: 57407, Pos: 8, Val: "on"}, Item{Typ: 57357, Pos: 10, Val: "("}, Item{Typ: 57362, Pos: 11, Val: ")"}, Item{Typ: 57354, Pos: 13, Val: "bar"}}},
			Op:             LAND,
			LHS: &VectorSelector{
				Name: "foo",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   3,
				},
			},
			RHS: &VectorSelector{
				Name: "bar",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "bar"),
				},
				PosRange: PositionRange{
					Start: 13,
					End:   16,
				},
			},
			VectorMatching: &VectorMatching{
				Card:           CardManyToMany,
				MatchingLabels: []string{},
				On:             true,
			},
		},
	}, {
		input: "foo and ignoring(test,blub) bar",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57374, Pos: 4, Val: "and"}, Item{Typ: 57405, Pos: 8, Val: "ignoring"}, Item{Typ: 57357, Pos: 16, Val: "("}, Item{Typ: 57354, Pos: 17, Val: "test"}, Item{Typ: 57349, Pos: 21, Val: ","}, Item{Typ: 57354, Pos: 22, Val: "blub"}, Item{Typ: 57362, Pos: 26, Val: ")"}, Item{Typ: 57354, Pos: 28, Val: "bar"}}},
			Op:             LAND,
			LHS: &VectorSelector{
				Name: "foo",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   3,
				},
			},
			RHS: &VectorSelector{
				Name: "bar",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "bar"),
				},
				PosRange: PositionRange{
					Start: 28,
					End:   31,
				},
			},
			VectorMatching: &VectorMatching{
				Card:           CardManyToMany,
				MatchingLabels: []string{"test", "blub"},
			},
		},
	}, {
		input: "foo and ignoring() bar",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57374, Pos: 4, Val: "and"}, Item{Typ: 57405, Pos: 8, Val: "ignoring"}, Item{Typ: 57357, Pos: 16, Val: "("}, Item{Typ: 57362, Pos: 17, Val: ")"}, Item{Typ: 57354, Pos: 19, Val: "bar"}}},
			Op:             LAND,
			LHS: &VectorSelector{
				Name: "foo",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   3,
				},
			},
			RHS: &VectorSelector{
				Name: "bar",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "bar"),
				},
				PosRange: PositionRange{
					Start: 19,
					End:   22,
				},
			},
			VectorMatching: &VectorMatching{
				Card:           CardManyToMany,
				MatchingLabels: []string{},
			},
		},
	}, {
		input: "foo unless on(bar) baz",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57378, Pos: 4, Val: "unless"}, Item{Typ: 57407, Pos: 11, Val: "on"}, Item{Typ: 57357, Pos: 13, Val: "("}, Item{Typ: 57354, Pos: 14, Val: "bar"}, Item{Typ: 57362, Pos: 17, Val: ")"}, Item{Typ: 57354, Pos: 19, Val: "baz"}}},
			Op:             LUNLESS,
			LHS: &VectorSelector{
				Name: "foo",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   3,
				},
			},
			RHS: &VectorSelector{
				Name: "baz",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "baz"),
				},
				PosRange: PositionRange{
					Start: 19,
					End:   22,
				},
			},
			VectorMatching: &VectorMatching{
				Card:           CardManyToMany,
				MatchingLabels: []string{"bar"},
				On:             true,
			},
		},
	}, {
		input: "foo / on(test,blub) group_left(bar) bar",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57369, Pos: 4, Val: "/"}, Item{Typ: 57407, Pos: 6, Val: "on"}, Item{Typ: 57357, Pos: 8, Val: "("}, Item{Typ: 57354, Pos: 9, Val: "test"}, Item{Typ: 57349, Pos: 13, Val: ","}, Item{Typ: 57354, Pos: 14, Val: "blub"}, Item{Typ: 57362, Pos: 18, Val: ")"}, Item{Typ: 57403, Pos: 20, Val: "group_left"}, Item{Typ: 57357, Pos: 30, Val: "("}, Item{Typ: 57354, Pos: 31, Val: "bar"}, Item{Typ: 57362, Pos: 34, Val: ")"}, Item{Typ: 57354, Pos: 36, Val: "bar"}}},
			Op:             DIV,
			LHS: &VectorSelector{
				Name: "foo",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   3,
				},
			},
			RHS: &VectorSelector{
				Name: "bar",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "bar"),
				},
				PosRange: PositionRange{
					Start: 36,
					End:   39,
				},
			},
			VectorMatching: &VectorMatching{
				Card:           CardManyToOne,
				MatchingLabels: []string{"test", "blub"},
				On:             true,
				Include:        []string{"bar"},
			},
		},
	}, {
		input: "foo / ignoring(test,blub) group_left(blub) bar",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57369, Pos: 4, Val: "/"}, Item{Typ: 57405, Pos: 6, Val: "ignoring"}, Item{Typ: 57357, Pos: 14, Val: "("}, Item{Typ: 57354, Pos: 15, Val: "test"}, Item{Typ: 57349, Pos: 19, Val: ","}, Item{Typ: 57354, Pos: 20, Val: "blub"}, Item{Typ: 57362, Pos: 24, Val: ")"}, Item{Typ: 57403, Pos: 26, Val: "group_left"}, Item{Typ: 57357, Pos: 36, Val: "("}, Item{Typ: 57354, Pos: 37, Val: "blub"}, Item{Typ: 57362, Pos: 41, Val: ")"}, Item{Typ: 57354, Pos: 43, Val: "bar"}}},
			Op:             DIV,
			LHS: &VectorSelector{
				Name: "foo",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   3,
				},
			},
			RHS: &VectorSelector{
				Name: "bar",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "bar"),
				},
				PosRange: PositionRange{
					Start: 43,
					End:   46,
				},
			},
			VectorMatching: &VectorMatching{
				Card:           CardManyToOne,
				MatchingLabels: []string{"test", "blub"},
				Include:        []string{"blub"},
			},
		},
	}, {
		input: "foo / ignoring(test,blub) group_left(bar) bar",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57369, Pos: 4, Val: "/"}, Item{Typ: 57405, Pos: 6, Val: "ignoring"}, Item{Typ: 57357, Pos: 14, Val: "("}, Item{Typ: 57354, Pos: 15, Val: "test"}, Item{Typ: 57349, Pos: 19, Val: ","}, Item{Typ: 57354, Pos: 20, Val: "blub"}, Item{Typ: 57362, Pos: 24, Val: ")"}, Item{Typ: 57403, Pos: 26, Val: "group_left"}, Item{Typ: 57357, Pos: 36, Val: "("}, Item{Typ: 57354, Pos: 37, Val: "bar"}, Item{Typ: 57362, Pos: 40, Val: ")"}, Item{Typ: 57354, Pos: 42, Val: "bar"}}},
			Op:             DIV,
			LHS: &VectorSelector{
				Name: "foo",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   3,
				},
			},
			RHS: &VectorSelector{
				Name: "bar",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "bar"),
				},
				PosRange: PositionRange{
					Start: 42,
					End:   45,
				},
			},
			VectorMatching: &VectorMatching{
				Card:           CardManyToOne,
				MatchingLabels: []string{"test", "blub"},
				Include:        []string{"bar"},
			},
		},
	}, {
		input: "foo - on(test,blub) group_right(bar,foo) bar",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57384, Pos: 4, Val: "-"}, Item{Typ: 57407, Pos: 6, Val: "on"}, Item{Typ: 57357, Pos: 8, Val: "("}, Item{Typ: 57354, Pos: 9, Val: "test"}, Item{Typ: 57349, Pos: 13, Val: ","}, Item{Typ: 57354, Pos: 14, Val: "blub"}, Item{Typ: 57362, Pos: 18, Val: ")"}, Item{Typ: 57404, Pos: 20, Val: "group_right"}, Item{Typ: 57357, Pos: 31, Val: "("}, Item{Typ: 57354, Pos: 32, Val: "bar"}, Item{Typ: 57349, Pos: 35, Val: ","}, Item{Typ: 57354, Pos: 36, Val: "foo"}, Item{Typ: 57362, Pos: 39, Val: ")"}, Item{Typ: 57354, Pos: 41, Val: "bar"}}},
			Op:             SUB,
			LHS: &VectorSelector{
				Name: "foo",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   3,
				},
			},
			RHS: &VectorSelector{
				Name: "bar",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "bar"),
				},
				PosRange: PositionRange{
					Start: 41,
					End:   44,
				},
			},
			VectorMatching: &VectorMatching{
				Card:           CardOneToMany,
				MatchingLabels: []string{"test", "blub"},
				Include:        []string{"bar", "foo"},
				On:             true,
			},
		},
	}, {
		input: "foo - ignoring(test,blub) group_right(bar,foo) bar",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57384, Pos: 4, Val: "-"}, Item{Typ: 57405, Pos: 6, Val: "ignoring"}, Item{Typ: 57357, Pos: 14, Val: "("}, Item{Typ: 57354, Pos: 15, Val: "test"}, Item{Typ: 57349, Pos: 19, Val: ","}, Item{Typ: 57354, Pos: 20, Val: "blub"}, Item{Typ: 57362, Pos: 24, Val: ")"}, Item{Typ: 57404, Pos: 26, Val: "group_right"}, Item{Typ: 57357, Pos: 37, Val: "("}, Item{Typ: 57354, Pos: 38, Val: "bar"}, Item{Typ: 57349, Pos: 41, Val: ","}, Item{Typ: 57354, Pos: 42, Val: "foo"}, Item{Typ: 57362, Pos: 45, Val: ")"}, Item{Typ: 57354, Pos: 47, Val: "bar"}}},
			Op:             SUB,
			LHS: &VectorSelector{
				Name: "foo",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   3,
				},
			},
			RHS: &VectorSelector{
				Name: "bar",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "bar"),
				},
				PosRange: PositionRange{
					Start: 47,
					End:   50,
				},
			},
			VectorMatching: &VectorMatching{
				Card:           CardOneToMany,
				MatchingLabels: []string{"test", "blub"},
				Include:        []string{"bar", "foo"},
			},
		},
	}, {
		input:  "foo and 1",
		fail:   true,
		errMsg: "set operator \"and\" not allowed in binary scalar expression",
	}, {
		input:  "1 and foo",
		fail:   true,
		errMsg: "set operator \"and\" not allowed in binary scalar expression",
	}, {
		input:  "foo or 1",
		fail:   true,
		errMsg: "set operator \"or\" not allowed in binary scalar expression",
	}, {
		input:  "1 or foo",
		fail:   true,
		errMsg: "set operator \"or\" not allowed in binary scalar expression",
	}, {
		input:  "foo unless 1",
		fail:   true,
		errMsg: "set operator \"unless\" not allowed in binary scalar expression",
	}, {
		input:  "1 unless foo",
		fail:   true,
		errMsg: "set operator \"unless\" not allowed in binary scalar expression",
	}, {
		input:  "1 or on(bar) foo",
		fail:   true,
		errMsg: "vector matching only allowed between instant vectors",
	}, {
		input:  "foo == on(bar) 10",
		fail:   true,
		errMsg: "vector matching only allowed between instant vectors",
	}, {
		input:  "foo + group_left(baz) bar",
		fail:   true,
		errMsg: "unexpected <group_left>",
	}, {
		input:  "foo and on(bar) group_left(baz) bar",
		fail:   true,
		errMsg: "no grouping allowed for \"and\" operation",
	}, {
		input:  "foo and on(bar) group_right(baz) bar",
		fail:   true,
		errMsg: "no grouping allowed for \"and\" operation",
	}, {
		input:  "foo or on(bar) group_left(baz) bar",
		fail:   true,
		errMsg: "no grouping allowed for \"or\" operation",
	}, {
		input:  "foo or on(bar) group_right(baz) bar",
		fail:   true,
		errMsg: "no grouping allowed for \"or\" operation",
	}, {
		input:  "foo unless on(bar) group_left(baz) bar",
		fail:   true,
		errMsg: "no grouping allowed for \"unless\" operation",
	}, {
		input:  "foo unless on(bar) group_right(baz) bar",
		fail:   true,
		errMsg: "no grouping allowed for \"unless\" operation",
	}, {
		input:  `http_requests{group="production"} + on(instance) group_left(job,instance) cpu_count{type="smp"}`,
		fail:   true,
		errMsg: "label \"instance\" must not occur in ON and GROUP clause at once",
	}, {
		input:  "foo + bool bar",
		fail:   true,
		errMsg: "bool modifier can only be used on comparison operators",
	}, {
		input:  "foo + bool 10",
		fail:   true,
		errMsg: "bool modifier can only be used on comparison operators",
	}, {
		input:  "foo and bool 10",
		fail:   true,
		errMsg: "bool modifier can only be used on comparison operators",
	},
	// Test Vector selector.
	{
		input: "foo",
		expected: &VectorSelector{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}}},
			Name:           "foo",
			Offset:         0,
			LabelMatchers: []*labels.Matcher{
				mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
			},
			PosRange: PositionRange{
				Start: 0,
				End:   3,
			},
		},
	}, {
		input: "min",
		expected: &VectorSelector{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57393, Pos: 0, Val: "min"}}},
			Name:           "min",
			Offset:         0,
			LabelMatchers: []*labels.Matcher{
				mustLabelMatcher(labels.MatchEqual, model.MetricNameLabel, "min"),
			},
			PosRange: PositionRange{
				Start: 0,
				End:   3,
			},
		},
	}, {
		input: "foo offset 5m",
		expected: &VectorSelector{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57406, Pos: 4, Val: "offset"}, Item{Typ: 57351, Pos: 11, Val: "5m"}}},
			Name:           "foo",
			Offset:         5 * time.Minute,
			LabelMatchers: []*labels.Matcher{
				mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
			},
			PosRange: PositionRange{
				Start: 0,
				End:   13,
			},
		},
	}, {
		input: `foo OFFSET 1h30m`,
		expected: &VectorSelector{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57406, Pos: 4, Val: "OFFSET"}, Item{Typ: 57351, Pos: 11, Val: "1h30m"}}},
			Name:           "foo",
			Offset:         90 * time.Minute,
			LabelMatchers: []*labels.Matcher{
				mustLabelMatcher(labels.MatchEqual, model.MetricNameLabel, "foo"),
			},
			PosRange: PositionRange{
				Start: 0,
				End:   16,
			},
		},
	}, {
		input: `foo OFFSET 1m30ms`,
		expected: &VectorSelector{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57406, Pos: 4, Val: "OFFSET"}, Item{Typ: 57351, Pos: 11, Val: "1m30ms"}}},
			Name:           "foo",
			Offset:         time.Minute + 30*time.Millisecond,
			LabelMatchers: []*labels.Matcher{
				mustLabelMatcher(labels.MatchEqual, model.MetricNameLabel, "foo"),
			},
			PosRange: PositionRange{
				Start: 0,
				End:   17,
			},
		},
	}, {
		input: `foo:bar{a="bc"}`,
		expected: &VectorSelector{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57358, Pos: 0, Val: "foo:bar"}, Item{Typ: 57355, Pos: 7, Val: "{"}, Item{Typ: 57354, Pos: 8, Val: "a"}, Item{Typ: 57370, Pos: 9, Val: "="}, Item{Typ: 57365, Pos: 10, Val: "\"bc\""}, Item{Typ: 57360, Pos: 14, Val: "}"}}},
			Name:           "foo:bar",
			Offset:         0,
			LabelMatchers: []*labels.Matcher{
				mustLabelMatcher(labels.MatchEqual, "a", "bc"),
				mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo:bar"),
			},
			PosRange: PositionRange{
				Start: 0,
				End:   15,
			},
		},
	}, {
		input: `foo{NaN='bc'}`,
		expected: &VectorSelector{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57355, Pos: 3, Val: "{"}, Item{Typ: 57354, Pos: 4, Val: "NaN"}, Item{Typ: 57370, Pos: 7, Val: "="}, Item{Typ: 57365, Pos: 8, Val: "'bc'"}, Item{Typ: 57360, Pos: 12, Val: "}"}}},
			Name:           "foo",
			Offset:         0,
			LabelMatchers: []*labels.Matcher{
				mustLabelMatcher(labels.MatchEqual, "NaN", "bc"),
				mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
			},
			PosRange: PositionRange{
				Start: 0,
				End:   13,
			},
		},
	}, {
		input: `foo{bar='}'}`,
		expected: &VectorSelector{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57355, Pos: 3, Val: "{"}, Item{Typ: 57354, Pos: 4, Val: "bar"}, Item{Typ: 57370, Pos: 7, Val: "="}, Item{Typ: 57365, Pos: 8, Val: "'}'"}, Item{Typ: 57360, Pos: 11, Val: "}"}}},
			Name:           "foo",
			Offset:         0,
			LabelMatchers: []*labels.Matcher{
				mustLabelMatcher(labels.MatchEqual, "bar", "}"),
				mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
			},
			PosRange: PositionRange{
				Start: 0,
				End:   12,
			},
		},
	}, {
		input: `foo{a="b", foo!="bar", test=~"test", bar!~"baz"}`,
		expected: &VectorSelector{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57355, Pos: 3, Val: "{"}, Item{Typ: 57354, Pos: 4, Val: "a"}, Item{Typ: 57370, Pos: 5, Val: "="}, Item{Typ: 57365, Pos: 6, Val: "\"b\""}, Item{Typ: 57349, Pos: 9, Val: ","}, Item{Typ: 57354, Pos: 11, Val: "foo"}, Item{Typ: 57381, Pos: 14, Val: "!="}, Item{Typ: 57365, Pos: 16, Val: "\"bar\""}, Item{Typ: 57349, Pos: 21, Val: ","}, Item{Typ: 57354, Pos: 23, Val: "test"}, Item{Typ: 57371, Pos: 27, Val: "=~"}, Item{Typ: 57365, Pos: 29, Val: "\"test\""}, Item{Typ: 57349, Pos: 35, Val: ","}, Item{Typ: 57354, Pos: 37, Val: "bar"}, Item{Typ: 57382, Pos: 40, Val: "!~"}, Item{Typ: 57365, Pos: 42, Val: "\"baz\""}, Item{Typ: 57360, Pos: 47, Val: "}"}}},
			Name:           "foo",
			Offset:         0,
			LabelMatchers: []*labels.Matcher{
				mustLabelMatcher(labels.MatchEqual, "a", "b"),
				mustLabelMatcher(labels.MatchNotEqual, "foo", "bar"),
				mustLabelMatcher(labels.MatchRegexp, "test", "test"),
				mustLabelMatcher(labels.MatchNotRegexp, "bar", "baz"),
				mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
			},
			PosRange: PositionRange{
				Start: 0,
				End:   48,
			},
		},
	}, {
		input: `foo{a="b", foo!="bar", test=~"test", bar!~"baz",}`,
		expected: &VectorSelector{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57355, Pos: 3, Val: "{"}, Item{Typ: 57354, Pos: 4, Val: "a"}, Item{Typ: 57370, Pos: 5, Val: "="}, Item{Typ: 57365, Pos: 6, Val: "\"b\""}, Item{Typ: 57349, Pos: 9, Val: ","}, Item{Typ: 57354, Pos: 11, Val: "foo"}, Item{Typ: 57381, Pos: 14, Val: "!="}, Item{Typ: 57365, Pos: 16, Val: "\"bar\""}, Item{Typ: 57349, Pos: 21, Val: ","}, Item{Typ: 57354, Pos: 23, Val: "test"}, Item{Typ: 57371, Pos: 27, Val: "=~"}, Item{Typ: 57365, Pos: 29, Val: "\"test\""}, Item{Typ: 57349, Pos: 35, Val: ","}, Item{Typ: 57354, Pos: 37, Val: "bar"}, Item{Typ: 57382, Pos: 40, Val: "!~"}, Item{Typ: 57365, Pos: 42, Val: "\"baz\""}, Item{Typ: 57349, Pos: 47, Val: ","}, Item{Typ: 57360, Pos: 48, Val: "}"}}},
			Name:           "foo",
			Offset:         0,
			LabelMatchers: []*labels.Matcher{
				mustLabelMatcher(labels.MatchEqual, "a", "b"),
				mustLabelMatcher(labels.MatchNotEqual, "foo", "bar"),
				mustLabelMatcher(labels.MatchRegexp, "test", "test"),
				mustLabelMatcher(labels.MatchNotRegexp, "bar", "baz"),
				mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
			},
			PosRange: PositionRange{
				Start: 0,
				End:   49,
			},
		},
	}, {
		input:  `{`,
		fail:   true,
		errMsg: "unexpected end of input inside braces",
	}, {
		input:  `}`,
		fail:   true,
		errMsg: "unexpected character: '}'",
	}, {
		input:  `some{`,
		fail:   true,
		errMsg: "unexpected end of input inside braces",
	}, {
		input:  `some}`,
		fail:   true,
		errMsg: "unexpected character: '}'",
	}, {
		input:  `some_metric{a=b}`,
		fail:   true,
		errMsg: "unexpected identifier \"b\" in label matching, expected string",
	}, {
		input:  `some_metric{a:b="b"}`,
		fail:   true,
		errMsg: "unexpected character inside braces: ':'",
	}, {
		input:  `foo{a*"b"}`,
		fail:   true,
		errMsg: "unexpected character inside braces: '*'",
	}, {
		input: `foo{a>="b"}`,
		fail:  true,
		// TODO(fabxc): willingly lexing wrong tokens allows for more precise error
		// messages from the parser - consider if this is an option.
		errMsg: "unexpected character inside braces: '>'",
	}, {
		input:  "some_metric{a=\"\xff\"}",
		fail:   true,
		errMsg: "1:15: parse error: invalid UTF-8 rune",
	}, {
		input:  `foo{gibberish}`,
		fail:   true,
		errMsg: `unexpected "}" in label matching, expected label matching operator`,
	}, {
		input:  `foo{1}`,
		fail:   true,
		errMsg: "unexpected character inside braces: '1'",
	}, {
		input:  `{}`,
		fail:   true,
		errMsg: "vector selector must contain at least one non-empty matcher",
	}, {
		input:  `{x=""}`,
		fail:   true,
		errMsg: "vector selector must contain at least one non-empty matcher",
	}, {
		input:  `{x=~".*"}`,
		fail:   true,
		errMsg: "vector selector must contain at least one non-empty matcher",
	}, {
		input:  `{x!~".+"}`,
		fail:   true,
		errMsg: "vector selector must contain at least one non-empty matcher",
	}, {
		input:  `{x!="a"}`,
		fail:   true,
		errMsg: "vector selector must contain at least one non-empty matcher",
	}, {
		input:  `foo{__name__="bar"}`,
		fail:   true,
		errMsg: `metric name must not be set twice: "foo" or "bar"`,
	}, {
		input:  `foo{__name__= =}`,
		fail:   true,
		errMsg: "unexpected <op:=> in label matching, expected string",
	}, {
		input:  `foo{,}`,
		fail:   true,
		errMsg: `unexpected "," in label matching, expected identifier or "}"`,
	}, {
		input:  `foo{__name__ == "bar"}`,
		fail:   true,
		errMsg: "unexpected <op:=> in label matching, expected string",
	}, {
		input:  `foo{__name__="bar" lol}`,
		fail:   true,
		errMsg: `unexpected identifier "lol" in label matching, expected "," or "}"`,
	},
	// Test matrix selector.
	{
		input: "test[5s]",
		expected: &MatrixSelector{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "test"}, Item{Typ: 57356, Pos: 4, Val: "["}, Item{Typ: 57351, Pos: 5, Val: "5s"}, Item{Typ: 57361, Pos: 7, Val: "]"}}},
			VectorSelector: &VectorSelector{
				Name:   "test",
				Offset: 0,
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "test"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   4,
				},
			},
			Range:  5 * time.Second,
			EndPos: 8,
		},
	}, {
		input: "test[5m]",
		expected: &MatrixSelector{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "test"}, Item{Typ: 57356, Pos: 4, Val: "["}, Item{Typ: 57351, Pos: 5, Val: "5m"}, Item{Typ: 57361, Pos: 7, Val: "]"}}},
			VectorSelector: &VectorSelector{
				Name:   "test",
				Offset: 0,
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "test"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   4,
				},
			},
			Range:  5 * time.Minute,
			EndPos: 8,
		},
	}, {
		input: `foo[5m30s]`,
		expected: &MatrixSelector{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57356, Pos: 3, Val: "["}, Item{Typ: 57351, Pos: 4, Val: "5m30s"}, Item{Typ: 57361, Pos: 9, Val: "]"}}},
			VectorSelector: &VectorSelector{
				Name:   "foo",
				Offset: 0,
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   3,
				},
			},
			Range:  5*time.Minute + 30*time.Second,
			EndPos: 10,
		},
	}, {
		input: "test[5h] OFFSET 5m",
		expected: &MatrixSelector{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "test"}, Item{Typ: 57356, Pos: 4, Val: "["}, Item{Typ: 57351, Pos: 5, Val: "5h"}, Item{Typ: 57361, Pos: 7, Val: "]"}, Item{Typ: 57406, Pos: 9, Val: "OFFSET"}, Item{Typ: 57351, Pos: 16, Val: "5m"}}},
			VectorSelector: &VectorSelector{
				Name:   "test",
				Offset: 5 * time.Minute,
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "test"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   4,
				},
			},
			Range:  5 * time.Hour,
			EndPos: 18,
		},
	}, {
		input: "test[5d] OFFSET 10s",
		expected: &MatrixSelector{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "test"}, Item{Typ: 57356, Pos: 4, Val: "["}, Item{Typ: 57351, Pos: 5, Val: "5d"}, Item{Typ: 57361, Pos: 7, Val: "]"}, Item{Typ: 57406, Pos: 9, Val: "OFFSET"}, Item{Typ: 57351, Pos: 16, Val: "10s"}}},
			VectorSelector: &VectorSelector{
				Name:   "test",
				Offset: 10 * time.Second,
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "test"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   4,
				},
			},
			Range:  5 * 24 * time.Hour,
			EndPos: 19,
		},
	}, {
		input: "test[5w] offset 2w",
		expected: &MatrixSelector{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "test"}, Item{Typ: 57356, Pos: 4, Val: "["}, Item{Typ: 57351, Pos: 5, Val: "5w"}, Item{Typ: 57361, Pos: 7, Val: "]"}, Item{Typ: 57406, Pos: 9, Val: "offset"}, Item{Typ: 57351, Pos: 16, Val: "2w"}}},
			VectorSelector: &VectorSelector{
				Name:   "test",
				Offset: 14 * 24 * time.Hour,
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "test"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   4,
				},
			},
			Range:  5 * 7 * 24 * time.Hour,
			EndPos: 18,
		},
	},
	{
		input: `test{a="b"}[5y] OFFSET 3d`,
		expected: &MatrixSelector{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "test"}, Item{Typ: 57355, Pos: 4, Val: "{"}, Item{Typ: 57354, Pos: 5, Val: "a"}, Item{Typ: 57370, Pos: 6, Val: "="}, Item{Typ: 57365, Pos: 7, Val: "\"b\""}, Item{Typ: 57360, Pos: 10, Val: "}"}, Item{Typ: 57356, Pos: 11, Val: "["}, Item{Typ: 57351, Pos: 12, Val: "5y"}, Item{Typ: 57361, Pos: 14, Val: "]"}, Item{Typ: 57406, Pos: 16, Val: "OFFSET"}, Item{Typ: 57351, Pos: 23, Val: "3d"}}},
			VectorSelector: &VectorSelector{
				Name:   "test",
				Offset: 3 * 24 * time.Hour,
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, "a", "b"),
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "test"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   11,
				},
			},
			Range:  5 * 365 * 24 * time.Hour,
			EndPos: 25,
		},
	}, {
		input:  `foo[5mm]`,
		fail:   true,
		errMsg: "bad duration syntax: \"5mm\"",
	}, {
		input:  `foo[5m1]`,
		fail:   true,
		errMsg: "bad duration syntax: \"5m1\"",
	}, {
		input:  `foo[5m:1m1]`,
		fail:   true,
		errMsg: "bad number or duration syntax: \"1m1\"",
	}, {
		input:  `foo[5y1hs]`,
		fail:   true,
		errMsg: "not a valid duration string: \"5y1hs\"",
	}, {
		input:  `foo[5m1h]`,
		fail:   true,
		errMsg: "not a valid duration string: \"5m1h\"",
	}, {
		input:  `foo[5m1m]`,
		fail:   true,
		errMsg: "not a valid duration string: \"5m1m\"",
	}, {
		input:  `foo[0m]`,
		fail:   true,
		errMsg: "duration must be greater than 0",
	}, {
		input: `foo["5m"]`,
		fail:  true,
	}, {
		input:  `foo[]`,
		fail:   true,
		errMsg: "missing unit character in duration",
	}, {
		input:  `foo[1]`,
		fail:   true,
		errMsg: "missing unit character in duration",
	}, {
		input:  `some_metric[5m] OFFSET 1`,
		fail:   true,
		errMsg: "unexpected number \"1\" in offset, expected duration",
	}, {
		input:  `some_metric[5m] OFFSET 1mm`,
		fail:   true,
		errMsg: "bad number or duration syntax: \"1mm\"",
	}, {
		input:  `some_metric[5m] OFFSET`,
		fail:   true,
		errMsg: "unexpected end of input in offset, expected duration",
	}, {
		input:  `some_metric OFFSET 1m[5m]`,
		fail:   true,
		errMsg: "1:22: parse error: no offset modifiers allowed before range",
	}, {
		input:  `(foo + bar)[5m]`,
		fail:   true,
		errMsg: "1:12: parse error: ranges only allowed for vector selectors",
	},
	// Test aggregation.
	{
		input: "sum by (foo)(some_metric)",
		expected: &AggregateExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57397, Pos: 0, Val: "sum"}, Item{Typ: 57402, Pos: 4, Val: "by"}, Item{Typ: 57357, Pos: 7, Val: "("}, Item{Typ: 57354, Pos: 8, Val: "foo"}, Item{Typ: 57362, Pos: 11, Val: ")"}, Item{Typ: 57357, Pos: 12, Val: "("}, Item{Typ: 57354, Pos: 13, Val: "some_metric"}, Item{Typ: 57362, Pos: 24, Val: ")"}}},
			Op:             SUM,
			Expr: &VectorSelector{
				Name: "some_metric",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "some_metric"),
				},
				PosRange: PositionRange{
					Start: 13,
					End:   24,
				},
			},
			Grouping: []string{"foo"},
			PosRange: PositionRange{
				Start: 0,
				End:   25,
			},
		},
	}, {
		input: "avg by (foo)(some_metric)",
		expected: &AggregateExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57387, Pos: 0, Val: "avg"}, Item{Typ: 57402, Pos: 4, Val: "by"}, Item{Typ: 57357, Pos: 7, Val: "("}, Item{Typ: 57354, Pos: 8, Val: "foo"}, Item{Typ: 57362, Pos: 11, Val: ")"}, Item{Typ: 57357, Pos: 12, Val: "("}, Item{Typ: 57354, Pos: 13, Val: "some_metric"}, Item{Typ: 57362, Pos: 24, Val: ")"}}},
			Op:             AVG,
			Expr: &VectorSelector{
				Name: "some_metric",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "some_metric"),
				},
				PosRange: PositionRange{
					Start: 13,
					End:   24,
				},
			},
			Grouping: []string{"foo"},
			PosRange: PositionRange{
				Start: 0,
				End:   25,
			},
		},
	}, {
		input: "max by (foo)(some_metric)",
		expected: &AggregateExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57392, Pos: 0, Val: "max"}, Item{Typ: 57402, Pos: 4, Val: "by"}, Item{Typ: 57357, Pos: 7, Val: "("}, Item{Typ: 57354, Pos: 8, Val: "foo"}, Item{Typ: 57362, Pos: 11, Val: ")"}, Item{Typ: 57357, Pos: 12, Val: "("}, Item{Typ: 57354, Pos: 13, Val: "some_metric"}, Item{Typ: 57362, Pos: 24, Val: ")"}}},
			Op:             MAX,
			Expr: &VectorSelector{
				Name: "some_metric",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "some_metric"),
				},
				PosRange: PositionRange{
					Start: 13,
					End:   24,
				},
			},
			Grouping: []string{"foo"},
			PosRange: PositionRange{
				Start: 0,
				End:   25,
			},
		},
	}, {
		input: "sum without (foo) (some_metric)",
		expected: &AggregateExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57397, Pos: 0, Val: "sum"}, Item{Typ: 57408, Pos: 4, Val: "without"}, Item{Typ: 57357, Pos: 12, Val: "("}, Item{Typ: 57354, Pos: 13, Val: "foo"}, Item{Typ: 57362, Pos: 16, Val: ")"}, Item{Typ: 57357, Pos: 18, Val: "("}, Item{Typ: 57354, Pos: 19, Val: "some_metric"}, Item{Typ: 57362, Pos: 30, Val: ")"}}},
			Op:             SUM,
			Without:        true,
			Expr: &VectorSelector{
				Name: "some_metric",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "some_metric"),
				},
				PosRange: PositionRange{
					Start: 19,
					End:   30,
				},
			},
			Grouping: []string{"foo"},
			PosRange: PositionRange{
				Start: 0,
				End:   31,
			},
		},
	}, {
		input: "sum (some_metric) without (foo)",
		expected: &AggregateExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57397, Pos: 0, Val: "sum"}, Item{Typ: 57357, Pos: 4, Val: "("}, Item{Typ: 57354, Pos: 5, Val: "some_metric"}, Item{Typ: 57362, Pos: 16, Val: ")"}, Item{Typ: 57408, Pos: 18, Val: "without"}, Item{Typ: 57357, Pos: 26, Val: "("}, Item{Typ: 57354, Pos: 27, Val: "foo"}, Item{Typ: 57362, Pos: 30, Val: ")"}}},
			Op:             SUM,
			Without:        true,
			Expr: &VectorSelector{
				Name: "some_metric",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "some_metric"),
				},
				PosRange: PositionRange{
					Start: 5,
					End:   16,
				},
			},
			Grouping: []string{"foo"},
			PosRange: PositionRange{
				Start: 0,
				End:   31,
			},
		},
	}, {
		input: "stddev(some_metric)",
		expected: &AggregateExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57395, Pos: 0, Val: "stddev"}, Item{Typ: 57357, Pos: 6, Val: "("}, Item{Typ: 57354, Pos: 7, Val: "some_metric"}, Item{Typ: 57362, Pos: 18, Val: ")"}}},
			Op:             STDDEV,
			Expr: &VectorSelector{
				Name: "some_metric",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "some_metric"),
				},
				PosRange: PositionRange{
					Start: 7,
					End:   18,
				},
			},
			PosRange: PositionRange{
				Start: 0,
				End:   19,
			},
		},
	}, {
		input: "stdvar by (foo)(some_metric)",
		expected: &AggregateExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57396, Pos: 0, Val: "stdvar"}, Item{Typ: 57402, Pos: 7, Val: "by"}, Item{Typ: 57357, Pos: 10, Val: "("}, Item{Typ: 57354, Pos: 11, Val: "foo"}, Item{Typ: 57362, Pos: 14, Val: ")"}, Item{Typ: 57357, Pos: 15, Val: "("}, Item{Typ: 57354, Pos: 16, Val: "some_metric"}, Item{Typ: 57362, Pos: 27, Val: ")"}}},
			Op:             STDVAR,
			Expr: &VectorSelector{
				Name: "some_metric",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "some_metric"),
				},
				PosRange: PositionRange{
					Start: 16,
					End:   27,
				},
			},
			Grouping: []string{"foo"},
			PosRange: PositionRange{
				Start: 0,
				End:   28,
			},
		},
	}, {
		input: "sum by ()(some_metric)",
		expected: &AggregateExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57397, Pos: 0, Val: "sum"}, Item{Typ: 57402, Pos: 4, Val: "by"}, Item{Typ: 57357, Pos: 7, Val: "("}, Item{Typ: 57362, Pos: 8, Val: ")"}, Item{Typ: 57357, Pos: 9, Val: "("}, Item{Typ: 57354, Pos: 10, Val: "some_metric"}, Item{Typ: 57362, Pos: 21, Val: ")"}}},
			Op:             SUM,
			Expr: &VectorSelector{
				Name: "some_metric",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "some_metric"),
				},
				PosRange: PositionRange{
					Start: 10,
					End:   21,
				},
			},
			Grouping: []string{},
			PosRange: PositionRange{
				Start: 0,
				End:   22,
			},
		},
	}, {
		input: "sum by (foo,bar,)(some_metric)",
		expected: &AggregateExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57397, Pos: 0, Val: "sum"}, Item{Typ: 57402, Pos: 4, Val: "by"}, Item{Typ: 57357, Pos: 7, Val: "("}, Item{Typ: 57354, Pos: 8, Val: "foo"}, Item{Typ: 57349, Pos: 11, Val: ","}, Item{Typ: 57354, Pos: 12, Val: "bar"}, Item{Typ: 57349, Pos: 15, Val: ","}, Item{Typ: 57362, Pos: 16, Val: ")"}, Item{Typ: 57357, Pos: 17, Val: "("}, Item{Typ: 57354, Pos: 18, Val: "some_metric"}, Item{Typ: 57362, Pos: 29, Val: ")"}}},
			Op:             SUM,
			Expr: &VectorSelector{
				Name: "some_metric",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "some_metric"),
				},
				PosRange: PositionRange{
					Start: 18,
					End:   29,
				},
			},
			Grouping: []string{"foo", "bar"},
			PosRange: PositionRange{
				Start: 0,
				End:   30,
			},
		},
	}, {
		input: "sum by (foo,)(some_metric)",
		expected: &AggregateExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57397, Pos: 0, Val: "sum"}, Item{Typ: 57402, Pos: 4, Val: "by"}, Item{Typ: 57357, Pos: 7, Val: "("}, Item{Typ: 57354, Pos: 8, Val: "foo"}, Item{Typ: 57349, Pos: 11, Val: ","}, Item{Typ: 57362, Pos: 12, Val: ")"}, Item{Typ: 57357, Pos: 13, Val: "("}, Item{Typ: 57354, Pos: 14, Val: "some_metric"}, Item{Typ: 57362, Pos: 25, Val: ")"}}},
			Op:             SUM,
			Expr: &VectorSelector{
				Name: "some_metric",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "some_metric"),
				},
				PosRange: PositionRange{
					Start: 14,
					End:   25,
				},
			},
			Grouping: []string{"foo"},
			PosRange: PositionRange{
				Start: 0,
				End:   26,
			},
		},
	}, {
		input: "topk(5, some_metric)",
		expected: &AggregateExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57398, Pos: 0, Val: "topk"}, Item{Typ: 57357, Pos: 4, Val: "("}, Item{Typ: 57359, Pos: 5, Val: "5"}, Item{Typ: 57349, Pos: 6, Val: ","}, Item{Typ: 57354, Pos: 8, Val: "some_metric"}, Item{Typ: 57362, Pos: 19, Val: ")"}}},
			Op:             TOPK,
			Expr: &VectorSelector{
				Name: "some_metric",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "some_metric"),
				},
				PosRange: PositionRange{
					Start: 8,
					End:   19,
				},
			},
			Param: &NumberLiteral{
				Val: 5,
				PosRange: PositionRange{
					Start: 5,
					End:   6,
				},
			},
			PosRange: PositionRange{
				Start: 0,
				End:   20,
			},
		},
	}, {
		input: `count_values("value", some_metric)`,
		expected: &AggregateExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57390, Pos: 0, Val: "count_values"}, Item{Typ: 57357, Pos: 12, Val: "("}, Item{Typ: 57365, Pos: 13, Val: "\"value\""}, Item{Typ: 57349, Pos: 20, Val: ","}, Item{Typ: 57354, Pos: 22, Val: "some_metric"}, Item{Typ: 57362, Pos: 33, Val: ")"}}},
			Op:             COUNT_VALUES,
			Expr: &VectorSelector{
				Name: "some_metric",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "some_metric"),
				},
				PosRange: PositionRange{
					Start: 22,
					End:   33,
				},
			},
			Param: &StringLiteral{
				Val: "value",
				PosRange: PositionRange{
					Start: 13,
					End:   20,
				},
			},
			PosRange: PositionRange{
				Start: 0,
				End:   34,
			},
		},
	}, {
		// Test usage of keywords as label names.
		input: "sum without(and, by, avg, count, alert, annotations)(some_metric)",
		expected: &AggregateExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57397, Pos: 0, Val: "sum"}, Item{Typ: 57408, Pos: 4, Val: "without"}, Item{Typ: 57357, Pos: 11, Val: "("}, Item{Typ: 57374, Pos: 12, Val: "and"}, Item{Typ: 57349, Pos: 15, Val: ","}, Item{Typ: 57402, Pos: 17, Val: "by"}, Item{Typ: 57349, Pos: 19, Val: ","}, Item{Typ: 57387, Pos: 21, Val: "avg"}, Item{Typ: 57349, Pos: 24, Val: ","}, Item{Typ: 57389, Pos: 26, Val: "count"}, Item{Typ: 57349, Pos: 31, Val: ","}, Item{Typ: 57354, Pos: 33, Val: "alert"}, Item{Typ: 57349, Pos: 38, Val: ","}, Item{Typ: 57354, Pos: 40, Val: "annotations"}, Item{Typ: 57362, Pos: 51, Val: ")"}, Item{Typ: 57357, Pos: 52, Val: "("}, Item{Typ: 57354, Pos: 53, Val: "some_metric"}, Item{Typ: 57362, Pos: 64, Val: ")"}}},
			Op:             SUM,
			Without:        true,
			Expr: &VectorSelector{
				Name: "some_metric",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "some_metric"),
				},
				PosRange: PositionRange{
					Start: 53,
					End:   64,
				},
			},
			Grouping: []string{"and", "by", "avg", "count", "alert", "annotations"},
			PosRange: PositionRange{
				Start: 0,
				End:   65,
			},
		},
	}, {
		input:  "sum without(==)(some_metric)",
		fail:   true,
		errMsg: "unexpected <op:==> in grouping opts, expected label",
	}, {
		input:  "sum without(,)(some_metric)",
		fail:   true,
		errMsg: `unexpected "," in grouping opts, expected label`,
	}, {
		input:  "sum without(foo,,)(some_metric)",
		fail:   true,
		errMsg: `unexpected "," in grouping opts, expected label`,
	}, {
		input:  `sum some_metric by (test)`,
		fail:   true,
		errMsg: "unexpected identifier \"some_metric\"",
	}, {
		input:  `sum (some_metric) by test`,
		fail:   true,
		errMsg: "unexpected identifier \"test\" in grouping opts",
	}, {
		input:  `sum (some_metric) by test`,
		fail:   true,
		errMsg: "unexpected identifier \"test\" in grouping opts",
	}, {
		input:  `sum () by (test)`,
		fail:   true,
		errMsg: "no arguments for aggregate expression provided",
	}, {
		input:  "MIN keep_common (some_metric)",
		fail:   true,
		errMsg: "1:5: parse error: unexpected identifier \"keep_common\"",
	}, {
		input:  "MIN (some_metric) keep_common",
		fail:   true,
		errMsg: `unexpected identifier "keep_common"`,
	}, {
		input:  `sum (some_metric) without (test) by (test)`,
		fail:   true,
		errMsg: "unexpected <by>",
	}, {
		input:  `sum without (test) (some_metric) by (test)`,
		fail:   true,
		errMsg: "unexpected <by>",
	}, {
		input:  `topk(some_metric)`,
		fail:   true,
		errMsg: "wrong number of arguments for aggregate expression provided, expected 2, got 1",
	}, {
		input:  `topk(some_metric,)`,
		fail:   true,
		errMsg: "trailing commas not allowed in function call args",
	}, {
		input:  `topk(some_metric, other_metric)`,
		fail:   true,
		errMsg: "1:6: parse error: expected type scalar in aggregation parameter, got instant vector",
	}, {
		input:  `count_values(5, other_metric)`,
		fail:   true,
		errMsg: "1:14: parse error: expected type string in aggregation parameter, got scalar",
	},
	// Test function calls.
	{
		input: "time()",
		expected: &Call{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "time"}, Item{Typ: 57357, Pos: 4, Val: "("}, Item{Typ: 57362, Pos: 5, Val: ")"}}},
			Func:           mustGetFunction("time"),
			Args:           Expressions{},
			PosRange: PositionRange{
				Start: 0,
				End:   6,
			},
		},
	}, {
		input: `floor(some_metric{foo!="bar"})`,
		expected: &Call{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "floor"}, Item{Typ: 57357, Pos: 5, Val: "("}, Item{Typ: 57354, Pos: 6, Val: "some_metric"}, Item{Typ: 57355, Pos: 17, Val: "{"}, Item{Typ: 57354, Pos: 18, Val: "foo"}, Item{Typ: 57381, Pos: 21, Val: "!="}, Item{Typ: 57365, Pos: 23, Val: "\"bar\""}, Item{Typ: 57360, Pos: 28, Val: "}"}, Item{Typ: 57362, Pos: 29, Val: ")"}}},
			Func:           mustGetFunction("floor"),
			Args: Expressions{
				&VectorSelector{
					Name: "some_metric",
					LabelMatchers: []*labels.Matcher{
						mustLabelMatcher(labels.MatchNotEqual, "foo", "bar"),
						mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "some_metric"),
					},
					PosRange: PositionRange{
						Start: 6,
						End:   29,
					},
				},
			},
			PosRange: PositionRange{
				Start: 0,
				End:   30,
			},
		},
	}, {
		input: "rate(some_metric[5m])",
		expected: &Call{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "rate"}, Item{Typ: 57357, Pos: 4, Val: "("}, Item{Typ: 57354, Pos: 5, Val: "some_metric"}, Item{Typ: 57356, Pos: 16, Val: "["}, Item{Typ: 57351, Pos: 17, Val: "5m"}, Item{Typ: 57361, Pos: 19, Val: "]"}, Item{Typ: 57362, Pos: 20, Val: ")"}}},
			Func:           mustGetFunction("rate"),
			Args: Expressions{
				&MatrixSelector{
					VectorSelector: &VectorSelector{
						Name: "some_metric",
						LabelMatchers: []*labels.Matcher{
							mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "some_metric"),
						},
						PosRange: PositionRange{
							Start: 5,
							End:   16,
						},
					},
					Range:  5 * time.Minute,
					EndPos: 20,
				},
			},
			PosRange: PositionRange{
				Start: 0,
				End:   21,
			},
		},
	}, {
		input: "round(some_metric)",
		expected: &Call{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "round"}, Item{Typ: 57357, Pos: 5, Val: "("}, Item{Typ: 57354, Pos: 6, Val: "some_metric"}, Item{Typ: 57362, Pos: 17, Val: ")"}}},
			Func:           mustGetFunction("round"),
			Args: Expressions{
				&VectorSelector{
					Name: "some_metric",
					LabelMatchers: []*labels.Matcher{
						mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "some_metric"),
					},
					PosRange: PositionRange{
						Start: 6,
						End:   17,
					},
				},
			},
			PosRange: PositionRange{
				Start: 0,
				End:   18,
			},
		},
	}, {
		input: "round(some_metric, 5)",
		expected: &Call{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "round"}, Item{Typ: 57357, Pos: 5, Val: "("}, Item{Typ: 57354, Pos: 6, Val: "some_metric"}, Item{Typ: 57349, Pos: 17, Val: ","}, Item{Typ: 57359, Pos: 19, Val: "5"}, Item{Typ: 57362, Pos: 20, Val: ")"}}},
			Func:           mustGetFunction("round"),
			Args: Expressions{
				&VectorSelector{
					Name: "some_metric",
					LabelMatchers: []*labels.Matcher{
						mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "some_metric"),
					},
					PosRange: PositionRange{
						Start: 6,
						End:   17,
					},
				},
				&NumberLiteral{
					Val: 5,
					PosRange: PositionRange{
						Start: 19,
						End:   20,
					},
				},
			},
			PosRange: PositionRange{
				Start: 0,
				End:   21,
			},
		},
	}, {
		input:  "floor()",
		fail:   true,
		errMsg: "expected 1 argument(s) in call to \"floor\", got 0",
	}, {
		input:  "floor(some_metric, other_metric)",
		fail:   true,
		errMsg: "expected 1 argument(s) in call to \"floor\", got 2",
	}, {
		input:  "floor(some_metric, 1)",
		fail:   true,
		errMsg: "expected 1 argument(s) in call to \"floor\", got 2",
	}, {
		input:  "floor(1)",
		fail:   true,
		errMsg: "expected type instant vector in call to function \"floor\", got scalar",
	}, {
		input:  "hour(some_metric, some_metric, some_metric)",
		fail:   true,
		errMsg: "expected at most 1 argument(s) in call to \"hour\", got 3",
	}, {
		input:  "time(some_metric)",
		fail:   true,
		errMsg: "expected 0 argument(s) in call to \"time\", got 1",
	}, {
		input:  "non_existent_function_far_bar()",
		fail:   true,
		errMsg: "unknown function with name \"non_existent_function_far_bar\"",
	}, {
		input:  "rate(some_metric)",
		fail:   true,
		errMsg: "expected type range vector in call to function \"rate\", got instant vector",
	}, {
		input:  "label_replace(a, `b`, `c\xff`, `d`, `.*`)",
		fail:   true,
		errMsg: "1:23: parse error: invalid UTF-8 rune",
	},
	// Fuzzing regression tests.
	{
		input:  "-=",
		fail:   true,
		errMsg: `unexpected "="`,
	}, {
		input:  "++-++-+-+-<",
		fail:   true,
		errMsg: `unexpected <op:<>`,
	}, {
		input:  "e-+=/(0)",
		fail:   true,
		errMsg: `unexpected "="`,
	}, {
		input:  "a>b()",
		fail:   true,
		errMsg: `unknown function`,
	}, {
		input:  "rate(avg)",
		fail:   true,
		errMsg: `expected type range vector`,
	}, {
		input: "sum(sum)",
		expected: &AggregateExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57397, Pos: 0, Val: "sum"}, Item{Typ: 57357, Pos: 3, Val: "("}, Item{Typ: 57397, Pos: 4, Val: "sum"}, Item{Typ: 57362, Pos: 7, Val: ")"}}},
			Op:             SUM,
			Expr: &VectorSelector{
				Name: "sum",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "sum"),
				},
				PosRange: PositionRange{
					Start: 4,
					End:   7,
				},
			},
			PosRange: PositionRange{
				Start: 0,
				End:   8,
			},
		},
	}, {
		input: "a + sum",
		expected: &BinaryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "a"}, Item{Typ: 57368, Pos: 2, Val: "+"}, Item{Typ: 57397, Pos: 4, Val: "sum"}}},
			Op:             ADD,
			LHS: &VectorSelector{
				Name: "a",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "a"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   1,
				},
			},
			RHS: &VectorSelector{
				Name: "sum",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "sum"),
				},
				PosRange: PositionRange{
					Start: 4,
					End:   7,
				},
			},
			VectorMatching: &VectorMatching{},
		},
	},
	// String quoting and escape sequence interpretation tests.
	{
		input: `"double-quoted string \" with escaped quote"`,
		expected: &StringLiteral{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57365, Pos: 0, Val: "\"double-quoted string \\\" with escaped quote\""}}},
			Val:            "double-quoted string \" with escaped quote",
			PosRange:       PositionRange{Start: 0, End: 44},
		},
	}, {
		input: `'single-quoted string \' with escaped quote'`,
		expected: &StringLiteral{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57365, Pos: 0, Val: "'single-quoted string \\' with escaped quote'"}}},
			Val:            "single-quoted string ' with escaped quote",
			PosRange:       PositionRange{Start: 0, End: 44},
		},
	}, {
		input: "`backtick-quoted string`",
		expected: &StringLiteral{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57365, Pos: 0, Val: "`backtick-quoted string`"}}},
			Val:            "backtick-quoted string",
			PosRange:       PositionRange{Start: 0, End: 24},
		},
	}, {
		input: `"\a\b\f\n\r\t\v\\\" - \xFF\377\u1234\U00010111\U0001011111☺"`,
		expected: &StringLiteral{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57365, Pos: 0, Val: "\"\\a\\b\\f\\n\\r\\t\\v\\\\\\\" - \\xFF\\377\\u1234\\U00010111\\U0001011111☺\""}}},
			Val:            "\a\b\f\n\r\t\v\\\" - \xFF\377\u1234\U00010111\U0001011111☺",
			PosRange:       PositionRange{Start: 0, End: 62},
		},
	}, {
		input: `'\a\b\f\n\r\t\v\\\' - \xFF\377\u1234\U00010111\U0001011111☺'`,
		expected: &StringLiteral{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57365, Pos: 0, Val: "'\\a\\b\\f\\n\\r\\t\\v\\\\\\' - \\xFF\\377\\u1234\\U00010111\\U0001011111☺'"}}},
			Val:            "\a\b\f\n\r\t\v\\' - \xFF\377\u1234\U00010111\U0001011111☺",
			PosRange:       PositionRange{Start: 0, End: 62},
		},
	}, {
		input: "`" + `\a\b\f\n\r\t\v\\\"\' - \xFF\377\u1234\U00010111\U0001011111☺` + "`",
		expected: &StringLiteral{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57365, Pos: 0, Val: "`\\a\\b\\f\\n\\r\\t\\v\\\\\\\"\\' - \\xFF\\377\\u1234\\U00010111\\U0001011111☺`"}}},
			Val:            `\a\b\f\n\r\t\v\\\"\' - \xFF\377\u1234\U00010111\U0001011111☺`,
			PosRange:       PositionRange{Start: 0, End: 64},
		},
	}, {
		input:  "`\\``",
		fail:   true,
		errMsg: "unterminated raw string",
	}, {
		input:  `"\`,
		fail:   true,
		errMsg: "escape sequence not terminated",
	}, {
		input:  `"\c"`,
		fail:   true,
		errMsg: "unknown escape sequence U+0063 'c'",
	}, {
		input:  `"\x."`,
		fail:   true,
		errMsg: "illegal character U+002E '.' in escape sequence",
	},
	// Subquery.
	{
		input: `foo{bar="baz"}[10m:6s]`,
		expected: &SubqueryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57355, Pos: 3, Val: "{"}, Item{Typ: 57354, Pos: 4, Val: "bar"}, Item{Typ: 57370, Pos: 7, Val: "="}, Item{Typ: 57365, Pos: 8, Val: "\"baz\""}, Item{Typ: 57360, Pos: 13, Val: "}"}, Item{Typ: 57356, Pos: 14, Val: "["}, Item{Typ: 57351, Pos: 15, Val: "10m"}, Item{Typ: 57348, Pos: 18, Val: ":"}, Item{Typ: 57351, Pos: 19, Val: "6s"}, Item{Typ: 57361, Pos: 21, Val: "]"}}},
			Expr: &VectorSelector{
				Name: "foo",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, "bar", "baz"),
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   14,
				},
			},
			Range:  10 * time.Minute,
			Step:   6 * time.Second,
			EndPos: 22,
		},
	},
	{
		input: `foo{bar="baz"}[10m5s:1h6ms]`,
		expected: &SubqueryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57355, Pos: 3, Val: "{"}, Item{Typ: 57354, Pos: 4, Val: "bar"}, Item{Typ: 57370, Pos: 7, Val: "="}, Item{Typ: 57365, Pos: 8, Val: "\"baz\""}, Item{Typ: 57360, Pos: 13, Val: "}"}, Item{Typ: 57356, Pos: 14, Val: "["}, Item{Typ: 57351, Pos: 15, Val: "10m5s"}, Item{Typ: 57348, Pos: 20, Val: ":"}, Item{Typ: 57351, Pos: 21, Val: "1h6ms"}, Item{Typ: 57361, Pos: 26, Val: "]"}}},
			Expr: &VectorSelector{
				Name: "foo",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, "bar", "baz"),
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   14,
				},
			},
			Range:  10*time.Minute + 5*time.Second,
			Step:   time.Hour + 6*time.Millisecond,
			EndPos: 27,
		},
	}, {
		input: `foo[10m:]`,
		expected: &SubqueryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "foo"}, Item{Typ: 57356, Pos: 3, Val: "["}, Item{Typ: 57351, Pos: 4, Val: "10m"}, Item{Typ: 57348, Pos: 7, Val: ":"}, Item{Typ: 57361, Pos: 8, Val: "]"}}},
			Expr: &VectorSelector{
				Name: "foo",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   3,
				},
			},
			Range:  10 * time.Minute,
			EndPos: 9,
		},
	}, {
		input: `min_over_time(rate(foo{bar="baz"}[2s])[5m:5s])`,
		expected: &Call{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "min_over_time"}, Item{Typ: 57357, Pos: 13, Val: "("}, Item{Typ: 57354, Pos: 14, Val: "rate"}, Item{Typ: 57357, Pos: 18, Val: "("}, Item{Typ: 57354, Pos: 19, Val: "foo"}, Item{Typ: 57355, Pos: 22, Val: "{"}, Item{Typ: 57354, Pos: 23, Val: "bar"}, Item{Typ: 57370, Pos: 26, Val: "="}, Item{Typ: 57365, Pos: 27, Val: "\"baz\""}, Item{Typ: 57360, Pos: 32, Val: "}"}, Item{Typ: 57356, Pos: 33, Val: "["}, Item{Typ: 57351, Pos: 34, Val: "2s"}, Item{Typ: 57361, Pos: 36, Val: "]"}, Item{Typ: 57362, Pos: 37, Val: ")"}, Item{Typ: 57356, Pos: 38, Val: "["}, Item{Typ: 57351, Pos: 39, Val: "5m"}, Item{Typ: 57348, Pos: 41, Val: ":"}, Item{Typ: 57351, Pos: 42, Val: "5s"}, Item{Typ: 57361, Pos: 44, Val: "]"}, Item{Typ: 57362, Pos: 45, Val: ")"}}},
			Func:           mustGetFunction("min_over_time"),
			Args: Expressions{
				&SubqueryExpr{
					Expr: &Call{
						Func: mustGetFunction("rate"),
						Args: Expressions{
							&MatrixSelector{
								VectorSelector: &VectorSelector{
									Name: "foo",
									LabelMatchers: []*labels.Matcher{
										mustLabelMatcher(labels.MatchEqual, "bar", "baz"),
										mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
									},
									PosRange: PositionRange{
										Start: 19,
										End:   33,
									},
								},
								Range:  2 * time.Second,
								EndPos: 37,
							},
						},
						PosRange: PositionRange{
							Start: 14,
							End:   38,
						},
					},
					Range: 5 * time.Minute,
					Step:  5 * time.Second,

					EndPos: 45,
				},
			},
			PosRange: PositionRange{
				Start: 0,
				End:   46,
			},
		},
	}, {
		input: `min_over_time(rate(foo{bar="baz"}[2s])[5m:])[4m:3s]`,
		expected: &SubqueryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "min_over_time"}, Item{Typ: 57357, Pos: 13, Val: "("}, Item{Typ: 57354, Pos: 14, Val: "rate"}, Item{Typ: 57357, Pos: 18, Val: "("}, Item{Typ: 57354, Pos: 19, Val: "foo"}, Item{Typ: 57355, Pos: 22, Val: "{"}, Item{Typ: 57354, Pos: 23, Val: "bar"}, Item{Typ: 57370, Pos: 26, Val: "="}, Item{Typ: 57365, Pos: 27, Val: "\"baz\""}, Item{Typ: 57360, Pos: 32, Val: "}"}, Item{Typ: 57356, Pos: 33, Val: "["}, Item{Typ: 57351, Pos: 34, Val: "2s"}, Item{Typ: 57361, Pos: 36, Val: "]"}, Item{Typ: 57362, Pos: 37, Val: ")"}, Item{Typ: 57356, Pos: 38, Val: "["}, Item{Typ: 57351, Pos: 39, Val: "5m"}, Item{Typ: 57348, Pos: 41, Val: ":"}, Item{Typ: 57361, Pos: 42, Val: "]"}, Item{Typ: 57362, Pos: 43, Val: ")"}, Item{Typ: 57356, Pos: 44, Val: "["}, Item{Typ: 57351, Pos: 45, Val: "4m"}, Item{Typ: 57348, Pos: 47, Val: ":"}, Item{Typ: 57351, Pos: 48, Val: "3s"}, Item{Typ: 57361, Pos: 50, Val: "]"}}},
			Expr: &Call{
				Func: mustGetFunction("min_over_time"),
				Args: Expressions{
					&SubqueryExpr{
						Expr: &Call{
							Func: mustGetFunction("rate"),
							Args: Expressions{
								&MatrixSelector{
									VectorSelector: &VectorSelector{
										Name: "foo",
										LabelMatchers: []*labels.Matcher{
											mustLabelMatcher(labels.MatchEqual, "bar", "baz"),
											mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
										},
										PosRange: PositionRange{
											Start: 19,
											End:   33,
										},
									},
									Range:  2 * time.Second,
									EndPos: 37,
								},
							},
							PosRange: PositionRange{
								Start: 14,
								End:   38,
							},
						},
						Range:  5 * time.Minute,
						EndPos: 43,
					},
				},
				PosRange: PositionRange{
					Start: 0,
					End:   44,
				},
			},
			Range:  4 * time.Minute,
			Step:   3 * time.Second,
			EndPos: 51,
		},
	}, {
		input: `min_over_time(rate(foo{bar="baz"}[2s])[5m:] offset 4m)[4m:3s]`,
		expected: &SubqueryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "min_over_time"}, Item{Typ: 57357, Pos: 13, Val: "("}, Item{Typ: 57354, Pos: 14, Val: "rate"}, Item{Typ: 57357, Pos: 18, Val: "("}, Item{Typ: 57354, Pos: 19, Val: "foo"}, Item{Typ: 57355, Pos: 22, Val: "{"}, Item{Typ: 57354, Pos: 23, Val: "bar"}, Item{Typ: 57370, Pos: 26, Val: "="}, Item{Typ: 57365, Pos: 27, Val: "\"baz\""}, Item{Typ: 57360, Pos: 32, Val: "}"}, Item{Typ: 57356, Pos: 33, Val: "["}, Item{Typ: 57351, Pos: 34, Val: "2s"}, Item{Typ: 57361, Pos: 36, Val: "]"}, Item{Typ: 57362, Pos: 37, Val: ")"}, Item{Typ: 57356, Pos: 38, Val: "["}, Item{Typ: 57351, Pos: 39, Val: "5m"}, Item{Typ: 57348, Pos: 41, Val: ":"}, Item{Typ: 57361, Pos: 42, Val: "]"}, Item{Typ: 57406, Pos: 44, Val: "offset"}, Item{Typ: 57351, Pos: 51, Val: "4m"}, Item{Typ: 57362, Pos: 53, Val: ")"}, Item{Typ: 57356, Pos: 54, Val: "["}, Item{Typ: 57351, Pos: 55, Val: "4m"}, Item{Typ: 57348, Pos: 57, Val: ":"}, Item{Typ: 57351, Pos: 58, Val: "3s"}, Item{Typ: 57361, Pos: 60, Val: "]"}}},
			Expr: &Call{
				Func: mustGetFunction("min_over_time"),
				Args: Expressions{
					&SubqueryExpr{
						Expr: &Call{
							Func: mustGetFunction("rate"),
							Args: Expressions{
								&MatrixSelector{
									VectorSelector: &VectorSelector{
										Name: "foo",
										LabelMatchers: []*labels.Matcher{
											mustLabelMatcher(labels.MatchEqual, "bar", "baz"),
											mustLabelMatcher(labels.MatchEqual, model.MetricNameLabel, "foo"),
										},
										PosRange: PositionRange{
											Start: 19,
											End:   33,
										},
									},
									Range:  2 * time.Second,
									EndPos: 37,
								},
							},
							PosRange: PositionRange{
								Start: 14,
								End:   38,
							},
						},
						Range:  5 * time.Minute,
						Offset: 4 * time.Minute,
						EndPos: 53,
					},
				},
				PosRange: PositionRange{
					Start: 0,
					End:   54,
				},
			},
			Range:  4 * time.Minute,
			Step:   3 * time.Second,
			EndPos: 61,
		},
	}, {
		input: "sum without(and, by, avg, count, alert, annotations)(some_metric) [30m:10s]",
		expected: &SubqueryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57397, Pos: 0, Val: "sum"}, Item{Typ: 57408, Pos: 4, Val: "without"}, Item{Typ: 57357, Pos: 11, Val: "("}, Item{Typ: 57374, Pos: 12, Val: "and"}, Item{Typ: 57349, Pos: 15, Val: ","}, Item{Typ: 57402, Pos: 17, Val: "by"}, Item{Typ: 57349, Pos: 19, Val: ","}, Item{Typ: 57387, Pos: 21, Val: "avg"}, Item{Typ: 57349, Pos: 24, Val: ","}, Item{Typ: 57389, Pos: 26, Val: "count"}, Item{Typ: 57349, Pos: 31, Val: ","}, Item{Typ: 57354, Pos: 33, Val: "alert"}, Item{Typ: 57349, Pos: 38, Val: ","}, Item{Typ: 57354, Pos: 40, Val: "annotations"}, Item{Typ: 57362, Pos: 51, Val: ")"}, Item{Typ: 57357, Pos: 52, Val: "("}, Item{Typ: 57354, Pos: 53, Val: "some_metric"}, Item{Typ: 57362, Pos: 64, Val: ")"}, Item{Typ: 57356, Pos: 66, Val: "["}, Item{Typ: 57351, Pos: 67, Val: "30m"}, Item{Typ: 57348, Pos: 70, Val: ":"}, Item{Typ: 57351, Pos: 71, Val: "10s"}, Item{Typ: 57361, Pos: 74, Val: "]"}}},
			Expr: &AggregateExpr{
				Op:      SUM,
				Without: true,
				Expr: &VectorSelector{
					Name: "some_metric",
					LabelMatchers: []*labels.Matcher{
						mustLabelMatcher(labels.MatchEqual, model.MetricNameLabel, "some_metric"),
					},
					PosRange: PositionRange{
						Start: 53,
						End:   64,
					},
				},
				Grouping: []string{"and", "by", "avg", "count", "alert", "annotations"},
				PosRange: PositionRange{
					Start: 0,
					End:   65,
				},
			},
			Range:  30 * time.Minute,
			Step:   10 * time.Second,
			EndPos: 75,
		},
	}, {
		input: `some_metric OFFSET 1m [10m:5s]`,
		expected: &SubqueryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57354, Pos: 0, Val: "some_metric"}, Item{Typ: 57406, Pos: 12, Val: "OFFSET"}, Item{Typ: 57351, Pos: 19, Val: "1m"}, Item{Typ: 57356, Pos: 22, Val: "["}, Item{Typ: 57351, Pos: 23, Val: "10m"}, Item{Typ: 57348, Pos: 26, Val: ":"}, Item{Typ: 57351, Pos: 27, Val: "5s"}, Item{Typ: 57361, Pos: 29, Val: "]"}}},
			Expr: &VectorSelector{
				Name: "some_metric",
				LabelMatchers: []*labels.Matcher{
					mustLabelMatcher(labels.MatchEqual, model.MetricNameLabel, "some_metric"),
				},
				PosRange: PositionRange{
					Start: 0,
					End:   21,
				},
				Offset: 1 * time.Minute,
			},
			Range:  10 * time.Minute,
			Step:   5 * time.Second,
			EndPos: 30,
		},
	}, {
		input: `(foo + bar{nm="val"})[5m:]`,
		expected: &SubqueryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57357, Pos: 0, Val: "("}, Item{Typ: 57354, Pos: 1, Val: "foo"}, Item{Typ: 57368, Pos: 5, Val: "+"}, Item{Typ: 57354, Pos: 7, Val: "bar"}, Item{Typ: 57355, Pos: 10, Val: "{"}, Item{Typ: 57354, Pos: 11, Val: "nm"}, Item{Typ: 57370, Pos: 13, Val: "="}, Item{Typ: 57365, Pos: 14, Val: "\"val\""}, Item{Typ: 57360, Pos: 19, Val: "}"}, Item{Typ: 57362, Pos: 20, Val: ")"}, Item{Typ: 57356, Pos: 21, Val: "["}, Item{Typ: 57351, Pos: 22, Val: "5m"}, Item{Typ: 57348, Pos: 24, Val: ":"}, Item{Typ: 57361, Pos: 25, Val: "]"}}},
			Expr: &ParenExpr{
				Expr: &BinaryExpr{
					Op: ADD,
					VectorMatching: &VectorMatching{
						Card: CardOneToOne,
					},
					LHS: &VectorSelector{
						Name: "foo",
						LabelMatchers: []*labels.Matcher{
							mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
						},
						PosRange: PositionRange{
							Start: 1,
							End:   4,
						},
					},
					RHS: &VectorSelector{
						Name: "bar",
						LabelMatchers: []*labels.Matcher{
							mustLabelMatcher(labels.MatchEqual, "nm", "val"),
							mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "bar"),
						},
						PosRange: PositionRange{
							Start: 7,
							End:   20,
						},
					},
				},
				PosRange: PositionRange{
					Start: 0,
					End:   21,
				},
			},
			Range:  5 * time.Minute,
			EndPos: 26,
		},
	}, {
		input: `(foo + bar{nm="val"})[5m:] offset 10m`,
		expected: &SubqueryExpr{
			ExprExtensions: ExprExtensions{LexItems: []Item{Item{Typ: 57357, Pos: 0, Val: "("}, Item{Typ: 57354, Pos: 1, Val: "foo"}, Item{Typ: 57368, Pos: 5, Val: "+"}, Item{Typ: 57354, Pos: 7, Val: "bar"}, Item{Typ: 57355, Pos: 10, Val: "{"}, Item{Typ: 57354, Pos: 11, Val: "nm"}, Item{Typ: 57370, Pos: 13, Val: "="}, Item{Typ: 57365, Pos: 14, Val: "\"val\""}, Item{Typ: 57360, Pos: 19, Val: "}"}, Item{Typ: 57362, Pos: 20, Val: ")"}, Item{Typ: 57356, Pos: 21, Val: "["}, Item{Typ: 57351, Pos: 22, Val: "5m"}, Item{Typ: 57348, Pos: 24, Val: ":"}, Item{Typ: 57361, Pos: 25, Val: "]"}, Item{Typ: 57406, Pos: 27, Val: "offset"}, Item{Typ: 57351, Pos: 34, Val: "10m"}}},
			Expr: &ParenExpr{
				Expr: &BinaryExpr{
					Op: ADD,
					VectorMatching: &VectorMatching{
						Card: CardOneToOne,
					},
					LHS: &VectorSelector{
						Name: "foo",
						LabelMatchers: []*labels.Matcher{
							mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "foo"),
						},
						PosRange: PositionRange{
							Start: 1,
							End:   4,
						},
					},
					RHS: &VectorSelector{
						Name: "bar",
						LabelMatchers: []*labels.Matcher{
							mustLabelMatcher(labels.MatchEqual, "nm", "val"),
							mustLabelMatcher(labels.MatchEqual, string(model.MetricNameLabel), "bar"),
						},
						PosRange: PositionRange{
							Start: 7,
							End:   20,
						},
					},
				},
				PosRange: PositionRange{
					Start: 0,
					End:   21,
				},
			},
			Range:  5 * time.Minute,
			Offset: 10 * time.Minute,
			EndPos: 37,
		},
	}, {
		input:  "test[5d] OFFSET 10s [10m:5s]",
		fail:   true,
		errMsg: "1:1: parse error: subquery is only allowed on instant vector, got matrix in \"test[5d] offset 10s[10m:5s]\"",
	}, {
		input:  `(foo + bar{nm="val"})[5m:][10m:5s]`,
		fail:   true,
		errMsg: `1:1: parse error: subquery is only allowed on instant vector, got matrix in "(foo + bar{nm=\"val\"})[5m:][10m:5s]" instead`,
	},
}

func TestParseExpressions(t *testing.T) {
	for _, test := range testExpr {
		t.Run(test.input, func(t *testing.T) {
			expr, err := ParseExpr(test.input)

			// Unexpected errors are always caused by a bug.
			testutil.Assert(t, err != errUnexpected, "unexpected error occurred")

			if !test.fail {
				testutil.Ok(t, err)
				testutil.Equals(t, test.expected, expr, "error on input '%s'", test.input)
			} else {
				testutil.NotOk(t, err)
				testutil.Assert(t, strings.Contains(err.Error(), test.errMsg), "unexpected error on input '%s', expected '%s', got '%s'", test.input, test.errMsg, err.Error())

				errorList, ok := err.(ParseErrors)

				testutil.Assert(t, ok, "unexpected error type")

				for _, e := range errorList {
					testutil.Assert(t, 0 <= e.PositionRange.Start, "parse error has negative position\nExpression '%s'\nError: %v", test.input, e)
					testutil.Assert(t, e.PositionRange.Start <= e.PositionRange.End, "parse error has negative length\nExpression '%s'\nError: %v", test.input, e)
					testutil.Assert(t, e.PositionRange.End <= Pos(len(test.input)), "parse error is not contained in input\nExpression '%s'\nError: %v", test.input, e)
				}
			}
		})
	}
}

// NaN has no equality. Thus, we need a separate test for it.
func TestNaNExpression(t *testing.T) {
	expr, err := ParseExpr("NaN")
	testutil.Ok(t, err)

	nl, ok := expr.(*NumberLiteral)
	testutil.Assert(t, ok, "expected number literal but got %T", expr)
	testutil.Assert(t, math.IsNaN(float64(nl.Val)), "expected 'NaN' in number literal but got %v", nl.Val)
}

func mustLabelMatcher(mt labels.MatchType, name, val string) *labels.Matcher {
	m, err := labels.NewMatcher(mt, name, val)
	if err != nil {
		panic(err)
	}
	return m
}

func mustGetFunction(name string) *Function {
	f, ok := getFunction(name)
	if !ok {
		panic(errors.Errorf("function %q does not exist", name))
	}
	return f
}

var testSeries = []struct {
	input          string
	expectedMetric labels.Labels
	expectedValues []SequenceValue
	fail           bool
}{
	{
		input:          `{} 1 2 3`,
		expectedMetric: labels.Labels{},
		expectedValues: newSeq(1, 2, 3),
	}, {
		input:          `{a="b"} -1 2 3`,
		expectedMetric: labels.FromStrings("a", "b"),
		expectedValues: newSeq(-1, 2, 3),
	}, {
		input:          `my_metric 1 2 3`,
		expectedMetric: labels.FromStrings(labels.MetricName, "my_metric"),
		expectedValues: newSeq(1, 2, 3),
	}, {
		input:          `my_metric{} 1 2 3`,
		expectedMetric: labels.FromStrings(labels.MetricName, "my_metric"),
		expectedValues: newSeq(1, 2, 3),
	}, {
		input:          `my_metric{a="b"} 1 2 3`,
		expectedMetric: labels.FromStrings(labels.MetricName, "my_metric", "a", "b"),
		expectedValues: newSeq(1, 2, 3),
	}, {
		input:          `my_metric{a="b"} 1 2 3-10x4`,
		expectedMetric: labels.FromStrings(labels.MetricName, "my_metric", "a", "b"),
		expectedValues: newSeq(1, 2, 3, -7, -17, -27, -37),
	}, {
		input:          `my_metric{a="b"} 1 2 3-0x4`,
		expectedMetric: labels.FromStrings(labels.MetricName, "my_metric", "a", "b"),
		expectedValues: newSeq(1, 2, 3, 3, 3, 3, 3),
	}, {
		input:          `my_metric{a="b"} 1 3 _ 5 _x4`,
		expectedMetric: labels.FromStrings(labels.MetricName, "my_metric", "a", "b"),
		expectedValues: newSeq(1, 3, none, 5, none, none, none, none),
	}, {
		input: `my_metric{a="b"} 1 3 _ 5 _a4`,
		fail:  true,
	}, {
		input:          `my_metric{a="b"} 1 -1`,
		expectedMetric: labels.FromStrings(labels.MetricName, "my_metric", "a", "b"),
		expectedValues: newSeq(1, -1),
	}, {
		input:          `my_metric{a="b"} 1 +1`,
		expectedMetric: labels.FromStrings(labels.MetricName, "my_metric", "a", "b"),
		expectedValues: newSeq(1, 1),
	}, {
		input:          `my_metric{a="b"} 1 -1 -3-10x4 7 9 +5`,
		expectedMetric: labels.FromStrings(labels.MetricName, "my_metric", "a", "b"),
		expectedValues: newSeq(1, -1, -3, -13, -23, -33, -43, 7, 9, 5),
	}, {
		input:          `my_metric{a="b"} 1 +1 +4 -6 -2 8`,
		expectedMetric: labels.FromStrings(labels.MetricName, "my_metric", "a", "b"),
		expectedValues: newSeq(1, 1, 4, -6, -2, 8),
	}, {
		// Trailing spaces should be correctly handles.
		input:          `my_metric{a="b"} 1 2 3    `,
		expectedMetric: labels.FromStrings(labels.MetricName, "my_metric", "a", "b"),
		expectedValues: newSeq(1, 2, 3),
	}, {
		input: `my_metric{a="b"} -3-3 -3`,
		fail:  true,
	}, {
		input: `my_metric{a="b"} -3 -3-3`,
		fail:  true,
	}, {
		input: `my_metric{a="b"} -3 _-2`,
		fail:  true,
	}, {
		input: `my_metric{a="b"} -3 3+3x4-4`,
		fail:  true,
	},
}

// For these tests only, we use the smallest float64 to signal an omitted value.
const none = math.SmallestNonzeroFloat64

func newSeq(vals ...float64) (res []SequenceValue) {
	for _, v := range vals {
		if v == none {
			res = append(res, SequenceValue{Omitted: true})
		} else {
			res = append(res, SequenceValue{Value: v})
		}
	}
	return res
}

func TestParseSeries(t *testing.T) {
	for _, test := range testSeries {
		metric, vals, err := ParseSeriesDesc(test.input)

		// Unexpected errors are always caused by a bug.
		testutil.Assert(t, err != errUnexpected, "unexpected error occurred")

		if !test.fail {
			testutil.Ok(t, err)
			testutil.Equals(t, test.expectedMetric, metric, "error on input '%s'", test.input)
			testutil.Equals(t, test.expectedValues, vals, "error in input '%s'", test.input)
		} else {
			testutil.NotOk(t, err)
		}
	}
}

func TestRecoverParserRuntime(t *testing.T) {
	p := newParser("foo bar")
	var err error

	defer func() {
		testutil.Equals(t, errUnexpected, err)
	}()
	defer p.recover(&err)
	// Cause a runtime panic.
	var a []int
	//nolint:govet
	a[123] = 1
}

func TestRecoverParserError(t *testing.T) {
	p := newParser("foo bar")
	var err error

	e := errors.New("custom error")

	defer func() {
		testutil.Equals(t, e.Error(), err.Error())
	}()
	defer p.recover(&err)

	panic(e)
}
