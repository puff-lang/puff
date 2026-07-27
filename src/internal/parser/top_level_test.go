package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/lexer"
	"github.com/puff-lang/puff/internal/source"
	"github.com/puff-lang/puff/internal/token"
)

func TestParseTopLevelGolden(t *testing.T) {
	sourcePath := filepath.Join("testdata", "top_level.puff")
	input, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "top_level.golden"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}

	result := parseTestSource(sourcePath, string(input))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
	if got := renderFile(result.File); got != string(want) {
		t.Fatalf("unexpected AST\nwant:\n%s\ngot:\n%s", want, got)
	}
}

func TestParseFunctionSignaturesAndNestedTypes(t *testing.T) {
	result := parseTestSource("functions.puff", `
fun noParams
end
fun explicit()
end
pub fun transform(value, items: map<string, list<int>>) -> list<string>
end
`)

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
	if len(result.File.Declarations) != 3 {
		t.Fatalf("expected three functions, got %d", len(result.File.Declarations))
	}

	function := result.File.Declarations[2].(*ast.FunctionDecl)
	if !function.Public || function.Name.Name != "transform" || len(function.Parameters) != 2 {
		t.Fatalf("unexpected function: %#v", function)
	}
	if function.Parameters[0].Name.Name != "value" || function.Parameters[0].Type != nil {
		t.Fatalf("unexpected untyped parameter: %#v", function.Parameters[0])
	}
	mapType := function.Parameters[1].Type
	if mapType.Name.Name != "map" || len(mapType.Arguments) != 2 {
		t.Fatalf("unexpected map type: %#v", mapType)
	}
	listType := mapType.Arguments[1]
	if listType.Name.Name != "list" || len(listType.Arguments) != 1 || listType.Arguments[0].Name.Name != "int" {
		t.Fatalf("unexpected nested list type: %#v", listType)
	}
	if function.ReturnType.Name.Name != "list" || function.ReturnType.Arguments[0].Name.Name != "string" {
		t.Fatalf("unexpected return type: %#v", function.ReturnType)
	}
}

func TestParseGlobalsPreservesTargetsAndSimpleValues(t *testing.T) {
	result := parseTestSource("globals.puff", `
$shop.name = "Main Shop"
$players[] = []
$stats[$key] = nil
pub $tax = 0.1
`)

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
	if len(result.File.Declarations) != 4 {
		t.Fatalf("expected four globals, got %d", len(result.File.Declarations))
	}

	shop := result.File.Declarations[0].(*ast.GlobalAssignment)
	if shop.Target.Name.Name != "shop" || shop.Target.Accesses[0].(*ast.FieldAccess).Field.Name != "name" {
		t.Fatalf("unexpected field target: %#v", shop.Target)
	}
	shopName := shop.Value.(*ast.StringExpr)
	if shopName.Parts[0].(*ast.StringText).Value != "Main Shop" {
		t.Fatalf("unexpected string value: %#v", shopName)
	}

	players := result.File.Declarations[1].(*ast.GlobalAssignment)
	if _, ok := players.Target.Accesses[0].(*ast.EmptyIndexAccess); !ok {
		t.Fatalf("expected empty index access, got %T", players.Target.Accesses[0])
	}
	if _, ok := players.Value.(*ast.ListExpr); !ok {
		t.Fatalf("expected empty list, got %T", players.Value)
	}

	stats := result.File.Declarations[2].(*ast.GlobalAssignment)
	index := stats.Target.Accesses[0].(*ast.IndexAccess).Index.(*ast.PatternExpr)
	if len(index.Tokens) != 2 || index.Tokens[0].Type != token.Dollar || index.Tokens[1].Lexeme != "key" {
		t.Fatalf("unexpected index expression: %#v", index)
	}
	if _, ok := stats.Value.(*ast.NilLiteral); !ok {
		t.Fatalf("expected nil value, got %T", stats.Value)
	}

	tax := result.File.Declarations[3].(*ast.GlobalAssignment)
	if !tax.Public || tax.Value.(*ast.FloatLiteral).Value != 0.1 {
		t.Fatalf("unexpected public global: %#v", tax)
	}
}

func TestParseEventsAndBalancedBodies(t *testing.T) {
	result := parseTestSource("events.puff", `
fun nested
if true
end
end
on scoreboard update
end
`)

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}
	if len(result.File.Declarations) != 2 {
		t.Fatalf("expected function and event, got %d declarations", len(result.File.Declarations))
	}
	event := result.File.Declarations[1].(*ast.EventDecl)
	if len(event.Name) != 2 || event.Name[0].Name != "scoreboard" || event.Name[1].Name != "update" {
		t.Fatalf("unexpected event name: %#v", event.Name)
	}
}

func TestParseRequiredDiagnostics(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		code    diagnostic.Code
		message string
		hint    string
	}{
		{
			name:    "invalid top-level statement",
			source:  "send \"Hello\" to player\n",
			code:    diagnostic.CodeInvalidTopLevelStatement,
			message: "Executable statements are not allowed at the top level.",
			hint:    "Move this statement into an event or function.",
		},
		{
			name:    "expected end",
			source:  "on load\nsend \"Loaded\" to player\n",
			code:    diagnostic.CodeExpectedEnd,
			message: `Expected "end" before end of file.`,
			hint:    "Add end to close the block.",
		},
		{
			name:    "expected token",
			source:  "fun add(a: int, b: int -> int\nend\n",
			code:    diagnostic.CodeExpectedToken,
			message: `Expected ")".`,
			hint:    "Close the parameter list before the return type.",
		},
		{
			name:    "unexpected token",
			source:  "fun example\nelse\nend\n",
			code:    diagnostic.CodeUnexpectedToken,
			message: "Unexpected token: else",
			hint:    "else can only appear inside an if block.",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := parseTestSource("invalid.puff", test.source)
			if len(result.Diagnostics) != 1 {
				t.Fatalf("expected one diagnostic, got %#v", result.Diagnostics)
			}
			got := result.Diagnostics[0]
			if got.Code != test.code || got.Phase != diagnostic.PhaseParser || got.Severity != diagnostic.SeverityError {
				t.Fatalf("unexpected diagnostic identity: %#v", got)
			}
			if got.Message != test.message || got.Hint != test.hint {
				t.Fatalf("unexpected diagnostic text: %#v", got)
			}
			if got.File != "invalid.puff" || got.Span.StartOffset > got.Span.EndOffset {
				t.Fatalf("unexpected diagnostic location: %#v", got)
			}
		})
	}
}

func TestParseRecoversAtNextTopLevelDeclaration(t *testing.T) {
	result := parseTestSource("recovery.puff", `
return 1
on tick
end
`)

	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != diagnostic.CodeInvalidTopLevelStatement {
		t.Fatalf("expected invalid top-level diagnostic, got %#v", result.Diagnostics)
	}
	if len(result.File.Declarations) != 1 {
		t.Fatalf("expected parser to recover one declaration, got %d", len(result.File.Declarations))
	}
	event := result.File.Declarations[0].(*ast.EventDecl)
	if len(event.Name) != 1 || event.Name[0].Name != "tick" {
		t.Fatalf("unexpected recovered declaration: %#v", event)
	}
}

func TestParseRejectsLateRequireAsUnexpected(t *testing.T) {
	result := parseTestSource("late-require.puff", `
on load
end
require "abc/shop"
on tick
end
`)

	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != diagnostic.CodeUnexpectedToken {
		t.Fatalf("expected unexpected token, got %#v", result.Diagnostics)
	}
	if result.Diagnostics[0].Message != "Unexpected token: require" {
		t.Fatalf("unexpected message: %q", result.Diagnostics[0].Message)
	}
	if len(result.File.Requirements) != 0 || len(result.File.Declarations) != 2 {
		t.Fatalf("unexpected recovery result: %#v", result.File)
	}
}

func TestParseDeclarationSpansCoverSourceConstructs(t *testing.T) {
	input := "require \"abc/shop\"\nfun setup\nend\n"
	result := parseTestSource("spans.puff", input)
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}

	requirement := result.File.Requirements[0]
	if requirement.Span().StartOffset != 0 || requirement.Span().EndOffset != 18 {
		t.Fatalf("unexpected require span: %#v", requirement.Span())
	}
	function := result.File.Declarations[0].(*ast.FunctionDecl)
	if function.Span().StartOffset != 19 || function.Span().EndOffset != 32 {
		t.Fatalf("unexpected function span: %#v", function.Span())
	}
	if function.Body.Span().StartOffset != 29 || function.Body.Span().EndOffset != 29 {
		t.Fatalf("unexpected empty body span: %#v", function.Body.Span())
	}
}

func parseTestSource(path string, text string) Result {
	file := source.NewFile(path, path, text)
	return Parse(file, lexer.Lex(file))
}

func renderFile(file *ast.File) string {
	var builder strings.Builder
	for _, metadata := range file.Metadata {
		fmt.Fprintf(&builder, "metadata %s=%s\n", metadata.Key, strconv.Quote(metadata.Value))
	}
	for _, requirement := range file.Requirements {
		fmt.Fprintf(&builder, "require %s", strconv.Quote(stringValue(requirement.Path)))
		if requirement.Alias != nil {
			fmt.Fprintf(&builder, " as %s", requirement.Alias.Name)
		}
		builder.WriteByte('\n')
	}
	for _, declaration := range file.Declarations {
		switch node := declaration.(type) {
		case *ast.GlobalAssignment:
			if node.Public {
				builder.WriteString("pub ")
			}
			fmt.Fprintf(&builder, "global %s = %s\n", renderVariable(node.Target), renderExpression(node.Value))
		case *ast.FunctionDecl:
			if node.Public {
				builder.WriteString("pub ")
			}
			fmt.Fprintf(&builder, "fun %s(", node.Name.Name)
			for index, parameter := range node.Parameters {
				if index > 0 {
					builder.WriteString(", ")
				}
				builder.WriteString(parameter.Name.Name)
				if parameter.Type != nil {
					fmt.Fprintf(&builder, ": %s", renderType(parameter.Type))
				}
			}
			builder.WriteByte(')')
			if node.ReturnType != nil {
				fmt.Fprintf(&builder, " -> %s", renderType(node.ReturnType))
			}
			builder.WriteByte('\n')
		case *ast.EventDecl:
			names := make([]string, len(node.Name))
			for index, name := range node.Name {
				names[index] = name.Name
			}
			fmt.Fprintf(&builder, "event %s\n", strings.Join(names, " "))
		}
	}
	return builder.String()
}

func stringValue(expression *ast.StringExpr) string {
	var value strings.Builder
	for _, part := range expression.Parts {
		if text, ok := part.(*ast.StringText); ok {
			value.WriteString(text.Value)
		}
	}
	return value.String()
}

func renderVariable(variable *ast.VariableExpr) string {
	var builder strings.Builder
	builder.WriteByte('$')
	builder.WriteString(variable.Name.Name)
	for _, access := range variable.Accesses {
		switch node := access.(type) {
		case *ast.FieldAccess:
			builder.WriteByte('.')
			builder.WriteString(node.Field.Name)
		case *ast.EmptyIndexAccess:
			builder.WriteString("[]")
		case *ast.IndexAccess:
			builder.WriteByte('[')
			builder.WriteString(renderExpression(node.Index))
			builder.WriteByte(']')
		}
	}
	return builder.String()
}

func renderExpression(expression ast.Expression) string {
	switch node := expression.(type) {
	case *ast.NilLiteral:
		return "nil"
	case *ast.BoolLiteral:
		return fmt.Sprintf("bool(%t)", node.Value)
	case *ast.IntLiteral:
		return fmt.Sprintf("int(%d)", node.Value)
	case *ast.FloatLiteral:
		return fmt.Sprintf("float(%g)", node.Value)
	case *ast.StringExpr:
		return "string(" + strconv.Quote(stringValue(node)) + ")"
	case *ast.ListExpr:
		return "list()"
	case *ast.PatternExpr:
		var builder strings.Builder
		for _, tok := range node.Tokens {
			builder.WriteString(tok.Lexeme)
		}
		return "deferred(" + builder.String() + ")"
	default:
		return fmt.Sprintf("%T", expression)
	}
}

func renderType(typeRef *ast.TypeRef) string {
	if len(typeRef.Arguments) == 0 {
		return typeRef.Name.Name
	}
	arguments := make([]string, len(typeRef.Arguments))
	for index, argument := range typeRef.Arguments {
		arguments[index] = renderType(argument)
	}
	return typeRef.Name.Name + "<" + strings.Join(arguments, ", ") + ">"
}
