package sema

import (
	"testing"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
)

func TestCheckInfersNestedCollectionsAcrossEmptyPlaceholders(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		value := nestedList(
			&ast.ListExpr{NodeBase: nttBase(2)},
			nestedList(nttInt(1, 2)),
		)
		function := returningFunction(
			"values",
			nttGenericType("list", 1, nttGenericType("list", 1, nttType("int", 1))),
			value,
		)
		module := nttModule("main.puff", function)

		result := Check(nttProject(module))

		nttAssertNoDiagnostics(t, result.Diagnostics)
		if got := module.ExpressionTypes[value].String(); got != "list<list<int>>" {
			t.Fatalf("expected list<list<int>>, got %s", got)
		}
	})

	t.Run("map values", func(t *testing.T) {
		value := &ast.MapExpr{
			NodeBase: nttBase(2),
			Entries: []ast.MapEntry{
				{Key: nttString("empty", 2), Value: &ast.ListExpr{NodeBase: nttBase(2)}},
				{Key: nttString("values", 2), Value: nestedList(nttInt(1, 2))},
			},
		}
		function := returningFunction(
			"values",
			nttGenericType(
				"map",
				1,
				nttType("string", 1),
				nttGenericType("list", 1, nttType("int", 1)),
			),
			value,
		)
		module := nttModule("main.puff", function)

		result := Check(nttProject(module))

		nttAssertNoDiagnostics(t, result.Diagnostics)
		if got := module.ExpressionTypes[value].String(); got != "map<string, list<int>>" {
			t.Fatalf("expected map<string, list<int>>, got %s", got)
		}
	})
}

func TestCheckRejectsNestedCollectionTypeMismatchAfterEmptyPlaceholder(t *testing.T) {
	value := nestedList(
		&ast.ListExpr{NodeBase: nttBase(2)},
		nestedList(nttInt(1, 2)),
	)
	function := returningFunction(
		"values",
		nttGenericType("list", 1, nttGenericType("list", 1, nttType("string", 1))),
		value,
	)

	result := Check(nttProject(nttModule("main.puff", function)))

	nttAssertDiagnostic(t, result.Diagnostics, diagnostic.Diagnostic{
		Code:     diagnostic.CodeTypeMismatch,
		Phase:    diagnostic.PhaseSemantics,
		Severity: diagnostic.SeverityError,
		Message:  "Type mismatch: cannot return list<list<int>> as list<list<string>>.",
		Hint:     "Return a value compatible with list<list<string>>.",
		File:     "main.puff",
		Span:     value.Span(),
	})
}

func TestCheckSuppressesNestedCollectionCascadeAfterUndefinedVariable(t *testing.T) {
	missing := nttVariable("missing", false, 2)
	value := nestedList(
		&ast.ListExpr{NodeBase: nttBase(2)},
		nestedList(missing),
	)
	function := returningFunction(
		"values",
		nttGenericType("list", 1, nttGenericType("list", 1, nttType("string", 1))),
		value,
	)
	module := nttModule("main.puff", function)

	result := Check(nttProject(module))

	nttAssertDiagnostic(t, result.Diagnostics, diagnostic.Diagnostic{
		Code:     diagnostic.CodeUndefinedVariable,
		Phase:    diagnostic.PhaseSemantics,
		Severity: diagnostic.SeverityError,
		Message:  "Undefined variable: $missing",
		Hint:     "Declare it before using it: $missing = 0",
		File:     "main.puff",
		Span:     missing.Span(),
	})
	if got := module.ExpressionTypes[value].String(); got != "list<list<unknown>>" {
		t.Fatalf("expected list<list<unknown>>, got %s", got)
	}
}

func TestCheckPropagatesKnownNestedCollectionIncompatibility(t *testing.T) {
	value := nestedList(
		&ast.ListExpr{NodeBase: nttBase(2)},
		nestedList(nttInt(1, 2)),
		nestedList(nttString("wrong", 2)),
	)
	function := returningFunction(
		"values",
		nttGenericType("list", 1, nttGenericType("list", 1, nttType("int", 1))),
		value,
	)

	result := Check(nttProject(nttModule("main.puff", function)))

	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != diagnostic.CodeTypeMismatch {
		t.Fatalf("expected one TYPE_MISMATCH, got %#v", result.Diagnostics)
	}
}

func nestedList(elements ...ast.Expression) *ast.ListExpr {
	return &ast.ListExpr{NodeBase: nttBase(2), Elements: elements}
}
