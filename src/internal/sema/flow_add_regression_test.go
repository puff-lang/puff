package sema

import (
	"testing"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
)

func TestCheckIfUsesIndependentLocalSnapshots(t *testing.T) {
	leakedRead := nttVariable("value", true, 6)
	statement := &ast.IfStmt{
		NodeBase:  nttBase(3),
		Condition: &ast.BoolLiteral{NodeBase: nttBase(3), Value: true},
		Then: ast.Block{Statements: []ast.Statement{
			localAssignment("value", nttInt(1, 4), 4),
		}},
		Else: &ast.Block{Statements: []ast.Statement{
			nttExprStmt(leakedRead, 6),
		}},
	}

	result := Check(nttProject(nttModule("main.puff", nttEvent("load", statement))))

	nttAssertDiagnostic(t, result.Diagnostics, undefinedLocalDiagnostic(leakedRead))
}

func TestCheckIfKeepsOnlyDefinitelyDefinedLocals(t *testing.T) {
	priorRead := nttVariable("prior", true, 10)
	sharedRead := nttVariable("shared", true, 11)
	oneBranchRead := nttVariable("oneBranch", true, 12)
	statement := &ast.IfStmt{
		NodeBase:  nttBase(4),
		Condition: &ast.BoolLiteral{NodeBase: nttBase(4), Value: true},
		Then: ast.Block{Statements: []ast.Statement{
			localAssignment("shared", nttInt(1, 5), 5),
			localAssignment("oneBranch", nttInt(1, 6), 6),
		}},
		Else: &ast.Block{Statements: []ast.Statement{
			localAssignment("shared", nttFloat(2, 8), 8),
		}},
	}

	result := Check(nttProject(nttModule("main.puff", nttEvent("load",
		localAssignment("prior", nttInt(1, 2), 2),
		statement,
		nttExprStmt(priorRead, 10),
		nttExprStmt(sharedRead, 11),
		nttExprStmt(oneBranchRead, 12),
	))))

	nttAssertDiagnostic(t, result.Diagnostics, undefinedLocalDiagnostic(oneBranchRead))
	if typ := result.Project.Modules[0].ExpressionTypes[priorRead]; typ.Kind != TypeInt {
		t.Fatalf("expected prior local to remain int, got %s", typ.String())
	}
	if typ := result.Project.Modules[0].ExpressionTypes[sharedRead]; typ.Kind != TypeFloat {
		t.Fatalf("expected numeric branch types to merge to float, got %s", typ.String())
	}
}

func TestCheckLoopDoesNotDefineLocalAfterBody(t *testing.T) {
	loopLocalRead := nttVariable("inside", true, 6)
	loop := &ast.LoopTimesStmt{
		NodeBase: nttBase(3),
		Count:    nttInt(1, 3),
		Body: ast.Block{Statements: []ast.Statement{
			localAssignment("inside", nttInt(1, 4), 4),
		}},
	}

	result := Check(nttProject(nttModule("main.puff", nttEvent("load",
		loop,
		nttExprStmt(loopLocalRead, 6),
	))))

	nttAssertDiagnostic(t, result.Diagnostics, undefinedLocalDiagnostic(loopLocalRead))
}

func TestCheckAddValidatesTargetCompatibility(t *testing.T) {
	t.Run("accepts numeric scalar promotion", func(t *testing.T) {
		result := Check(nttProject(nttModule("main.puff", nttEvent("load",
			localAssignment("amount", nttInt(1, 2), 2),
			addStatement(nttFloat(2.5, 3), nttVariable("amount", true, 3), 3),
		))))

		nttAssertNoDiagnostics(t, result.Diagnostics)
	})

	t.Run("rejects incompatible scalar without cascade", func(t *testing.T) {
		value := nttString("wrong", 3)
		result := Check(nttProject(nttModule("main.puff", nttEvent("load",
			localAssignment("amount", nttInt(1, 2), 2),
			addStatement(value, nttVariable("amount", true, 3), 3),
		))))

		nttAssertDiagnostic(t, result.Diagnostics, diagnostic.Diagnostic{
			Code:     diagnostic.CodeTypeMismatch,
			Phase:    diagnostic.PhaseSemantics,
			Severity: diagnostic.SeverityError,
			Message:  "Type mismatch: cannot add string to int.",
			Hint:     "Convert one value or use compatible types.",
			File:     "main.puff",
			Span:     value.Span(),
		})
	})

	t.Run("checks known list element type", func(t *testing.T) {
		value := nttString("wrong", 4)
		target := nttVariable("values", true, 4)
		target.Accesses = []ast.VariableAccess{&ast.EmptyIndexAccess{NodeBase: nttBase(4)}}
		result := Check(nttProject(nttModule("main.puff", nttEvent("load",
			localAssignment("values", &ast.ListExpr{
				NodeBase: nttBase(2),
				Elements: []ast.Expression{nttInt(1, 2)},
			}, 2),
			addStatement(nttInt(2, 3), listTarget("values", 3), 3),
			addStatement(value, target, 4),
		))))

		nttAssertDiagnostic(t, result.Diagnostics, diagnostic.Diagnostic{
			Code:     diagnostic.CodeTypeMismatch,
			Phase:    diagnostic.PhaseSemantics,
			Severity: diagnostic.SeverityError,
			Message:  "Type mismatch: cannot add string to int.",
			Hint:     "Convert one value or use compatible types.",
			File:     "main.puff",
			Span:     value.Span(),
		})
	})
}

func localAssignment(name string, value ast.Expression, line int) *ast.AssignmentStmt {
	return &ast.AssignmentStmt{
		NodeBase: nttBase(line),
		Target:   nttVariable(name, true, line),
		Value:    value,
	}
}

func addStatement(value ast.Expression, target ast.Assignable, line int) *ast.AddStmt {
	return &ast.AddStmt{NodeBase: nttBase(line), Value: value, Target: target}
}

func listTarget(name string, line int) *ast.VariableExpr {
	target := nttVariable(name, true, line)
	target.Accesses = []ast.VariableAccess{&ast.EmptyIndexAccess{NodeBase: nttBase(line)}}
	return target
}

func undefinedLocalDiagnostic(variable *ast.VariableExpr) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Code:     diagnostic.CodeUndefinedVariable,
		Phase:    diagnostic.PhaseSemantics,
		Severity: diagnostic.SeverityError,
		Message:  "Undefined variable: $_" + variable.Name.Name,
		Hint:     "Declare it before using it: $_" + variable.Name.Name + " = 0",
		File:     "main.puff",
		Span:     variable.Span(),
	}
}
