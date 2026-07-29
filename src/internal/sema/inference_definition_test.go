package sema

import (
	"testing"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/token"
)

func TestCheckReportsGlobalReadBeforeDefinitionWithoutCascade(t *testing.T) {
	read := nttVariable("later", false, 1)
	expression := &ast.BinaryExpr{
		NodeBase: nttBase(1),
		Left:     read,
		Operator: token.Plus,
		Right:    nttString("value", 1),
	}
	module := nttModule("main.puff",
		&ast.GlobalAssignment{
			NodeBase: nttBase(1),
			Target:   nttVariable("copy", false, 1),
			Value:    expression,
		},
		&ast.GlobalAssignment{
			NodeBase: nttBase(2),
			Target:   nttVariable("later", false, 2),
			Value:    nttInt(1, 2),
		},
	)

	result := Check(nttProject(module))

	nttAssertDiagnostic(t, result.Diagnostics, diagnostic.Diagnostic{
		Code:     diagnostic.CodeUndefinedVariable,
		Phase:    diagnostic.PhaseSemantics,
		Severity: diagnostic.SeverityError,
		Message:  "Undefined variable: $later",
		Hint:     "Declare it before using it: $later = 0",
		File:     "main.puff",
		Span:     read.Span(),
	})
	if typ := module.ExpressionTypes[expression]; !typ.IsUnknown() {
		t.Fatalf("expected invalid initializer type to remain unknown, got %#v", typ)
	}
	if symbol := module.ResolvedVariables[read]; symbol != nil {
		t.Fatalf("read before definition must not resolve, got %#v", symbol)
	}
}

func TestCheckAllowsFunctionForwardReferenceFromGlobalInitializer(t *testing.T) {
	call := nttCall("calculate", true, 1)
	module := nttModule("main.puff",
		&ast.GlobalAssignment{
			NodeBase: nttBase(1),
			Target:   nttVariable("result", false, 1),
			Value:    call,
		},
		&ast.FunctionDecl{
			NodeBase:   nttBase(2),
			Name:       nttIdentifier("calculate", 2),
			ReturnType: nttType("int", 2),
			Body: ast.Block{Statements: []ast.Statement{
				&ast.ReturnStmt{NodeBase: nttBase(3), Value: nttInt(42, 3)},
			}},
		},
	)

	result := Check(nttProject(module))

	nttAssertNoDiagnostics(t, result.Diagnostics)
	if symbol := module.ResolvedCalls[call]; symbol == nil || symbol.Name != "calculate" {
		t.Fatalf("expected forward function call to resolve, got %#v", symbol)
	}
	if typ := module.ExpressionTypes[call]; typ.Kind != TypeInt {
		t.Fatalf("expected forward function call type int, got %#v", typ)
	}
}

func TestCheckInfersNumericListAndMapLiteralsIndependentlyOfOrder(t *testing.T) {
	tests := []struct {
		name       string
		expression ast.Expression
		want       Type
	}{
		{
			name: "list int then float",
			expression: &ast.ListExpr{Elements: []ast.Expression{
				nttInt(1, 1),
				nttFloat(2.5, 1),
			}},
			want: Type{Kind: TypeList, Arguments: []Type{{Kind: TypeFloat}}},
		},
		{
			name: "list float then int",
			expression: &ast.ListExpr{Elements: []ast.Expression{
				nttFloat(2.5, 1),
				nttInt(1, 1),
			}},
			want: Type{Kind: TypeList, Arguments: []Type{{Kind: TypeFloat}}},
		},
		{
			name: "map int then float",
			expression: &ast.MapExpr{Entries: []ast.MapEntry{
				{Key: nttInt(1, 1), Value: nttFloat(1.5, 1)},
				{Key: nttFloat(2.5, 1), Value: nttInt(2, 1)},
			}},
			want: Type{Kind: TypeMap, Arguments: []Type{
				{Kind: TypeFloat},
				{Kind: TypeFloat},
			}},
		},
		{
			name: "map float then int",
			expression: &ast.MapExpr{Entries: []ast.MapEntry{
				{Key: nttFloat(2.5, 1), Value: nttInt(2, 1)},
				{Key: nttInt(1, 1), Value: nttFloat(1.5, 1)},
			}},
			want: Type{Kind: TypeMap, Arguments: []Type{
				{Kind: TypeFloat},
				{Kind: TypeFloat},
			}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			module := nttModule("main.puff", &ast.GlobalAssignment{
				NodeBase: nttBase(1),
				Target:   nttVariable("value", false, 1),
				Value:    test.expression,
			})

			result := Check(nttProject(module))

			nttAssertNoDiagnostics(t, result.Diagnostics)
			if got := module.ExpressionTypes[test.expression]; got.String() != test.want.String() {
				t.Fatalf("expected inferred type %s, got %s", test.want.String(), got.String())
			}
		})
	}
}
