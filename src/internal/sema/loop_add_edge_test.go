package sema

import (
	"testing"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
)

func TestCheckLoopRangePropagatesGuaranteedBodyState(t *testing.T) {
	updatedRead := nttVariable("updated", true, 7)
	createdRead := nttVariable("created", true, 8)
	loop := &ast.LoopRangeStmt{
		NodeBase: nttBase(3),
		Start:    nttInt(1, 3),
		End:      nttInt(1, 3),
		Body: ast.Block{Statements: []ast.Statement{
			localAssignment("updated", nttString("after", 4), 4),
			localAssignment("created", nttInt(2, 5), 5),
		}},
	}

	result := Check(nttProject(nttModule("main.puff", nttEvent("load",
		localAssignment("updated", nttInt(1, 2), 2),
		loop,
		nttExprStmt(updatedRead, 7),
		nttExprStmt(createdRead, 8),
	))))

	nttAssertNoDiagnostics(t, result.Diagnostics)
	if typ := result.Project.Modules[0].ExpressionTypes[updatedRead]; typ.Kind != TypeString {
		t.Fatalf("expected range body to update local to string, got %s", typ.String())
	}
	if typ := result.Project.Modules[0].ExpressionTypes[createdRead]; typ.Kind != TypeInt {
		t.Fatalf("expected range body to define int local, got %s", typ.String())
	}
}

func TestCheckLoopRangePropagatesGuaranteedReturn(t *testing.T) {
	function := efFunction("value", efType("int", 1), []ast.Statement{
		&ast.LoopRangeStmt{
			NodeBase: nttBase(2),
			Start:    nttInt(1, 2),
			End:      nttInt(1, 2),
			Body: ast.Block{Statements: []ast.Statement{
				&ast.ReturnStmt{NodeBase: nttBase(3), Value: nttInt(1, 3)},
			}},
		},
	}, 1)

	result := Check(nttProject(nttModule("main.puff", function)))

	nttAssertNoDiagnostics(t, result.Diagnostics)
}

func TestCheckNonGuaranteedLoopsKeepConservativeLocalState(t *testing.T) {
	tests := map[string]func(ast.Block) ast.Statement{
		"times": func(body ast.Block) ast.Statement {
			return &ast.LoopTimesStmt{NodeBase: nttBase(3), Count: nttInt(1, 3), Body: body}
		},
		"players": func(body ast.Block) ast.Statement {
			return &ast.LoopPlayersStmt{NodeBase: nttBase(3), Body: body}
		},
		"entities": func(body ast.Block) ast.Statement {
			return &ast.LoopEntitiesStmt{
				NodeBase: nttBase(3),
				Radius:   nttInt(10, 3),
				Around:   nttString("spawn", 3),
				Body:     body,
			}
		},
	}

	for name, makeLoop := range tests {
		t.Run(name, func(t *testing.T) {
			stableRead := nttVariable("stable", true, 7)
			createdRead := nttVariable("created", true, 8)
			body := ast.Block{Statements: []ast.Statement{
				localAssignment("stable", nttString("changed", 4), 4),
				localAssignment("created", nttInt(2, 5), 5),
			}}

			result := Check(nttProject(nttModule("main.puff", nttEvent("load",
				localAssignment("stable", nttInt(1, 2), 2),
				makeLoop(body),
				nttExprStmt(stableRead, 7),
				nttExprStmt(createdRead, 8),
			))))

			nttAssertDiagnostic(t, result.Diagnostics, undefinedLocalDiagnostic(createdRead))
			if typ := result.Project.Modules[0].ExpressionTypes[stableRead]; typ.Kind != TypeInt {
				t.Fatalf("expected conservative loop state to preserve int, got %s", typ.String())
			}
		})
	}
}

func TestCheckAddToEmptyIndexRequiresKnownListTarget(t *testing.T) {
	t.Run("rejects known non-list target", func(t *testing.T) {
		value := nttInt(2, 3)
		result := Check(nttProject(nttModule("main.puff", nttEvent("load",
			localAssignment("values", nttInt(1, 2), 2),
			addStatement(value, listTarget("values", 3), 3),
		))))

		nttAssertDiagnostic(t, result.Diagnostics, diagnostic.Diagnostic{
			Code:     diagnostic.CodeTypeMismatch,
			Phase:    diagnostic.PhaseSemantics,
			Severity: diagnostic.SeverityError,
			Message:  "Type mismatch: cannot add int to int[].",
			Hint:     "Convert one value or use compatible types.",
			File:     "main.puff",
			Span:     value.Span(),
		})
	})

	t.Run("suppresses cascade for unknown target", func(t *testing.T) {
		target := listTarget("missing", 3)
		result := Check(nttProject(nttModule("main.puff", nttEvent("load",
			addStatement(nttInt(2, 3), target, 3),
		))))

		nttAssertDiagnostic(t, result.Diagnostics, undefinedLocalDiagnostic(target))
	})

	t.Run("suppresses cascade for unknown value", func(t *testing.T) {
		value := nttVariable("missing", true, 3)
		result := Check(nttProject(nttModule("main.puff", nttEvent("load",
			localAssignment("values", nttInt(1, 2), 2),
			addStatement(value, listTarget("values", 3), 3),
		))))

		nttAssertDiagnostic(t, result.Diagnostics, undefinedLocalDiagnostic(value))
	})
}

func TestCheckAddRejectsIncompatibleMergedBranchType(t *testing.T) {
	value := nttInt(1, 8)
	statement := &ast.IfStmt{
		NodeBase:  nttBase(3),
		Condition: &ast.BoolLiteral{NodeBase: nttBase(3), Value: true},
		Then: ast.Block{Statements: []ast.Statement{
			localAssignment("value", nttInt(1, 4), 4),
		}},
		Else: &ast.Block{Statements: []ast.Statement{
			localAssignment("value", nttString("wrong", 6), 6),
		}},
	}

	result := Check(nttProject(nttModule("main.puff", nttEvent("load",
		statement,
		addStatement(value, nttVariable("value", true, 8), 8),
	))))

	nttAssertDiagnostic(t, result.Diagnostics, diagnostic.Diagnostic{
		Code:     diagnostic.CodeTypeMismatch,
		Phase:    diagnostic.PhaseSemantics,
		Severity: diagnostic.SeverityError,
		Message:  "Type mismatch: cannot add int to unknown.",
		Hint:     "Convert one value or use compatible types.",
		File:     "main.puff",
		Span:     value.Span(),
	})
}
