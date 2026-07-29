package sema

import (
	"testing"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
)

func TestCheckCollectionAssignmentRequiresListValue(t *testing.T) {
	t.Run("accepts list", func(t *testing.T) {
		value := &ast.ListExpr{
			NodeBase: nttBase(3),
			Elements: []ast.Expression{nttInt(1, 3)},
		}

		result := Check(nttProject(nttModule("main.puff",
			nttEvent("load", collectionAssignment(value, 3)),
		)))

		nttAssertNoDiagnostics(t, result.Diagnostics)
	})

	t.Run("rejects non-list", func(t *testing.T) {
		value := nttInt(1, 4)

		result := Check(nttProject(nttModule("main.puff",
			nttEvent("load", collectionAssignment(value, 4)),
		)))

		nttAssertDiagnostic(t, result.Diagnostics, diagnostic.Diagnostic{
			Code:     diagnostic.CodeTypeMismatch,
			Phase:    diagnostic.PhaseSemantics,
			Severity: diagnostic.SeverityError,
			Message:  "Type mismatch: cannot assign int to $players[].",
			Hint:     "Convert one value or use compatible types.",
			File:     "main.puff",
			Span:     value.Span(),
		})
	})

	t.Run("does not cascade for unknown value", func(t *testing.T) {
		value := nttVariable("missing", false, 5)

		result := Check(nttProject(nttModule("main.puff",
			nttEvent("load", collectionAssignment(value, 5)),
		)))

		nttAssertDiagnostic(t, result.Diagnostics, diagnostic.Diagnostic{
			Code:     diagnostic.CodeUndefinedVariable,
			Phase:    diagnostic.PhaseSemantics,
			Severity: diagnostic.SeverityError,
			Message:  "Undefined variable: $missing",
			Hint:     "Declare it before using it: $missing = 0",
			File:     "main.puff",
			Span:     value.Span(),
		})
	})
}

func collectionAssignment(value ast.Expression, line int) *ast.AssignmentStmt {
	return &ast.AssignmentStmt{
		NodeBase: nttBase(line),
		Target: &ast.VariableExpr{
			NodeBase: nttBase(line),
			Name:     nttIdentifier("players", line),
			Accesses: []ast.VariableAccess{&ast.EmptyIndexAccess{NodeBase: nttBase(line)}},
		},
		Value: value,
	}
}
