package parser

import (
	"fmt"
	"strings"
	"testing"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/token"
)

func TestParseExpressionPrecedence(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		{source: "1 + 2 * 3", want: "(+ 1 (* 2 3))"},
		{source: "(1 + 2) * 3", want: "(* (group (+ 1 2)) 3)"},
		{source: "10 - 3 - 2", want: "(- (- 10 3) 2)"},
		{source: "-1 * 2", want: "(* (unary - 1) 2)"},
		{source: "not false or true", want: "(or (unary not false) true)"},
		{source: "1 + 2 >= 3 == true", want: "(== (>= (+ 1 2) 3) true)"},
		{source: "true or false and false", want: "(or true (and false false))"},
		{source: "20 / 5 % 3", want: "(% (/ 20 5) 3)"},
	}

	for _, test := range tests {
		t.Run(test.source, func(t *testing.T) {
			result := parseTestSource("expression.puff", "$value = "+test.source+"\n")
			if len(result.Diagnostics) != 0 {
				t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
			}
			value := result.File.Declarations[0].(*ast.GlobalAssignment).Value
			if got := expressionShape(value); got != test.want {
				t.Fatalf("expected %s, got %s", test.want, got)
			}
		})
	}
}

func TestParseVariablesCallsCollectionsAndRange(t *testing.T) {
	result := parseTestSource("expressions.puff", `
$variable = $player.stats[$index][]
$imported = shop.$tax
$call = shop.finalPrice(100)
$list = [1, 2, 3,]
$map = {"coins": 100, "kills": 5,}
$range = 1..10
`)

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
	values := make([]ast.Expression, len(result.File.Declarations))
	for index, declaration := range result.File.Declarations {
		values[index] = declaration.(*ast.GlobalAssignment).Value
	}

	variable := values[0].(*ast.VariableExpr)
	if variable.Name.Name != "player" || len(variable.Accesses) != 3 {
		t.Fatalf("unexpected variable: %#v", variable)
	}
	if variable.Accesses[1].(*ast.IndexAccess).Index.(*ast.VariableExpr).Name.Name != "index" {
		t.Fatalf("unexpected variable index: %#v", variable.Accesses[1])
	}
	if _, ok := variable.Accesses[2].(*ast.EmptyIndexAccess); !ok {
		t.Fatalf("expected empty index access, got %T", variable.Accesses[2])
	}

	imported := values[1].(*ast.VariableExpr)
	if imported.Qualifier.Name != "shop" || imported.Name.Name != "tax" {
		t.Fatalf("unexpected imported variable: %#v", imported)
	}
	call := values[2].(*ast.CallExpr)
	if !call.ExplicitParens || len(call.Callee.Parts) != 2 || len(call.Arguments) != 1 {
		t.Fatalf("unexpected call: %#v", call)
	}
	if len(values[3].(*ast.ListExpr).Elements) != 3 {
		t.Fatalf("unexpected list: %#v", values[3])
	}
	if len(values[4].(*ast.MapExpr).Entries) != 2 {
		t.Fatalf("unexpected map: %#v", values[4])
	}
	rangeExpression := values[5].(*ast.RangeExpr)
	if rangeExpression.Start.(*ast.IntLiteral).Value != 1 || rangeExpression.End.(*ast.IntLiteral).Value != 10 {
		t.Fatalf("unexpected range: %#v", rangeExpression)
	}
}

func TestParseStringInterpolationUsesExpressionParser(t *testing.T) {
	result := parseTestSource("string.puff", `$message = "Total: {$coins + 10}; User: {shop.$name}"`+"\n")
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}

	stringExpression := result.File.Declarations[0].(*ast.GlobalAssignment).Value.(*ast.StringExpr)
	if len(stringExpression.Parts) != 4 {
		t.Fatalf("expected four string parts, got %#v", stringExpression.Parts)
	}
	first := stringExpression.Parts[1].(*ast.StringInterpolation).Expression
	if expressionShape(first) != "(+ $coins 10)" {
		t.Fatalf("unexpected first interpolation: %s", expressionShape(first))
	}
	second := stringExpression.Parts[3].(*ast.StringInterpolation).Expression.(*ast.VariableExpr)
	if second.Qualifier.Name != "shop" || second.Name.Name != "name" {
		t.Fatalf("unexpected imported interpolation: %#v", second)
	}
}

func TestParseExpressionSeparatorErrors(t *testing.T) {
	tests := []struct {
		source  string
		message string
	}{
		{source: "$x = [1 2]\n", message: `Expected "\",\" or \"]\"".`},
		{source: "$x = call(1 2)\n", message: `Expected "\",\" or \")\"".`},
		{source: `$x = {"a" 1}` + "\n", message: `Expected ":".`},
		{source: "$x = 1 2\n", message: "Expected newline."},
		{source: "$x = [1,,2]\n", message: `Expected "expression".`},
		{source: "$x = call(,1)\n", message: `Expected "expression".`},
		{source: "$x = call(,)\n", message: `Expected "expression".`},
		{source: "$x = ()\n", message: `Expected "expression".`},
		{source: "$x = +\n", message: `Expected "expression".`},
		{source: "$x = 1..2..3\n", message: `Unexpected token: ..`},
		{source: `$x = "value: {1 2}"` + "\n", message: `Expected "}".`},
	}

	for _, test := range tests {
		result := parseTestSource("separator.puff", test.source)
		if len(result.Diagnostics) != 1 {
			t.Fatalf("source %q: expected one diagnostic, got %#v", test.source, result.Diagnostics)
		}
		if result.Diagnostics[0].Message != test.message {
			t.Fatalf("source %q: expected %q, got %#v", test.source, test.message, result.Diagnostics)
		}
	}
}

func TestParseImportedVariableSpanIncludesQualifier(t *testing.T) {
	result := parseTestSource("span.puff", "$value = shop.$tax\n")
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
	variable := result.File.Declarations[0].(*ast.GlobalAssignment).Value.(*ast.VariableExpr)
	if variable.Span().StartOffset != 9 || variable.Span().EndOffset != 18 {
		t.Fatalf("expected imported variable span 9..18, got %#v", variable.Span())
	}
}

func TestParseDoesNotDuplicateLexerExpressionDiagnostics(t *testing.T) {
	tests := []struct {
		source string
		code   diagnostic.Code
	}{
		{source: `$x = "value: {}"` + "\n", code: diagnostic.CodeEmptyInterpolation},
		{source: "$x = 1abc\n", code: diagnostic.CodeInvalidNumber},
		{source: "$x = 1abc # comment\n", code: diagnostic.CodeInvalidNumber},
		{source: "$x = @ # comment\r\n", code: diagnostic.CodeInvalidCharacter},
	}

	for _, test := range tests {
		result := parseTestSource("lexer-error.puff", test.source)
		if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != test.code {
			t.Fatalf("source %q: expected only %s, got %#v", test.source, test.code, result.Diagnostics)
		}
	}
}

func TestParseInvalidVariableForms(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		code    diagnostic.Code
		message string
	}{
		{
			name:    "local at top level",
			source:  "$_price = 50\n",
			code:    diagnostic.CodeInvalidTopLevelStatement,
			message: "Executable statements are not allowed at the top level.",
		},
		{
			name:    "missing variable name",
			source:  "$ = 1\n",
			code:    diagnostic.CodeExpectedToken,
			message: `Expected "variable name".`,
		},
		{
			name:    "missing local variable name",
			source:  "$_ = 1\n",
			code:    diagnostic.CodeExpectedToken,
			message: `Expected "variable name".`,
		},
		{
			name:    "missing assignment value",
			source:  "fun f\n$_value =\nend\n",
			code:    diagnostic.CodeExpectedToken,
			message: `Expected "expression".`,
		},
		{
			name:    "imported local variable",
			source:  "fun f\nshop.$_price\nend\n",
			code:    diagnostic.CodeExpectedToken,
			message: `Expected "global variable name".`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := parseTestSource("variable.puff", test.source)
			if len(result.Diagnostics) != 1 {
				t.Fatalf("expected one diagnostic, got %#v", result.Diagnostics)
			}
			got := result.Diagnostics[0]
			if got.Code != test.code || got.Message != test.message {
				t.Fatalf("expected %s %q, got %#v", test.code, test.message, got)
			}
		})
	}
}

func expressionShape(expression ast.Expression) string {
	switch node := expression.(type) {
	case *ast.NilLiteral:
		return "nil"
	case *ast.BoolLiteral:
		return fmt.Sprintf("%t", node.Value)
	case *ast.IntLiteral:
		return fmt.Sprintf("%d", node.Value)
	case *ast.FloatLiteral:
		return fmt.Sprintf("%g", node.Value)
	case *ast.UnaryExpr:
		return fmt.Sprintf("(unary %s %s)", operatorShape(node.Operator), expressionShape(node.Operand))
	case *ast.BinaryExpr:
		return fmt.Sprintf("(%s %s %s)", operatorShape(node.Operator), expressionShape(node.Left), expressionShape(node.Right))
	case *ast.GroupExpr:
		return "(group " + expressionShape(node.Expression) + ")"
	case *ast.RangeExpr:
		return "(range " + expressionShape(node.Start) + " " + expressionShape(node.End) + ")"
	case *ast.VariableExpr:
		var builder strings.Builder
		if node.Qualifier != nil {
			builder.WriteString(node.Qualifier.Name)
			builder.WriteByte('.')
		}
		builder.WriteByte('$')
		if node.Local {
			builder.WriteByte('_')
		}
		builder.WriteString(node.Name.Name)
		return builder.String()
	case *ast.CallExpr:
		parts := make([]string, len(node.Callee.Parts))
		for index, part := range node.Callee.Parts {
			parts[index] = part.Name
		}
		return strings.Join(parts, ".")
	default:
		return fmt.Sprintf("%T", expression)
	}
}

func operatorShape(operator token.Type) string {
	switch operator {
	case token.Plus:
		return "+"
	case token.Minus:
		return "-"
	case token.Star:
		return "*"
	case token.Slash:
		return "/"
	case token.Percent:
		return "%"
	case token.EqualEqual:
		return "=="
	case token.BangEqual:
		return "!="
	case token.Greater:
		return ">"
	case token.GreaterEq:
		return ">="
	case token.Less:
		return "<"
	case token.LessEq:
		return "<="
	case token.And:
		return "and"
	case token.Or:
		return "or"
	case token.Not:
		return "not"
	default:
		return string(operator)
	}
}
