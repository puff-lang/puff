package sema

import (
	"testing"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/token"
)

func TestCheckArithmeticRejectsIncompatibleBranchType(t *testing.T) {
	value := nttVariable("value", true, 8)
	sum := &ast.BinaryExpr{
		NodeBase: nttBase(8),
		Left:     value,
		Operator: token.Plus,
		Right:    nttInt(1, 8),
	}
	branch := &ast.IfStmt{
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
		branch,
		nttExprStmt(sum, 8),
	))))

	nttAssertDiagnostic(t, result.Diagnostics, diagnostic.Diagnostic{
		Code:     diagnostic.CodeTypeMismatch,
		Phase:    diagnostic.PhaseSemantics,
		Severity: diagnostic.SeverityError,
		Message:  "Type mismatch: cannot add unknown and int.",
		Hint:     "Convert one value or use compatible types.",
		File:     "main.puff",
		Span:     sum.Span(),
	})
}

func TestCheckDynamicTopLevelIndexDoesNotRedefineRoot(t *testing.T) {
	root := nttVariable("stats", false, 1)
	key := nttVariable("key", false, 2)
	dynamicTarget := indexedVariable("stats", key, 3)
	module := nttModule("main.puff",
		&ast.GlobalAssignment{
			NodeBase: nttBase(1),
			Public:   true,
			Target:   root,
			Value: &ast.MapExpr{NodeBase: nttBase(1), Entries: []ast.MapEntry{{
				Key: nttString("coins", 1), Value: nttInt(1, 1),
			}}},
		},
		&ast.GlobalAssignment{
			NodeBase: nttBase(2),
			Target:   nttVariable("key", false, 2),
			Value:    nttString("coins", 2),
		},
		&ast.GlobalAssignment{
			NodeBase: nttBase(3),
			Target:   dynamicTarget,
			Value:    nttInt(2, 3),
		},
	)

	result := Check(nttProject(module))

	nttAssertNoDiagnostics(t, result.Diagnostics)
	symbol := module.Symbols.Globals["stats"]
	if symbol == nil || !symbol.Public || symbol.Type.Kind != TypeMap {
		t.Fatalf("expected public map root to remain intact, got %#v", symbol)
	}
	if module.ResolvedVariables[dynamicTarget] != symbol {
		t.Fatalf("expected dynamic assignment to resolve the root symbol")
	}
}

func TestCheckConditionalRuntimeGlobalRequiresDefinitionOnEveryPath(t *testing.T) {
	read := nttVariable("coins", false, 6)
	branch := &ast.IfStmt{
		NodeBase:  nttBase(2),
		Condition: &ast.BoolLiteral{NodeBase: nttBase(2), Value: true},
		Then: ast.Block{Statements: []ast.Statement{
			&ast.AssignmentStmt{
				NodeBase: nttBase(3),
				Target:   nttVariable("coins", false, 3),
				Value:    nttInt(1, 3),
			},
		}},
	}

	result := Check(nttProject(nttModule("main.puff", nttEvent("load",
		branch,
		nttExprStmt(read, 6),
	))))

	nttAssertDiagnostic(t, result.Diagnostics, diagnostic.Diagnostic{
		Code:     diagnostic.CodeUndefinedVariable,
		Phase:    diagnostic.PhaseSemantics,
		Severity: diagnostic.SeverityError,
		Message:  "Undefined variable: $coins",
		Hint:     "Declare it before using it: $coins = 0",
		File:     "main.puff",
		Span:     read.Span(),
	})
}

func TestCheckConditionalRuntimeGlobalMergesAllPaths(t *testing.T) {
	read := nttVariable("coins", false, 7)
	branch := &ast.IfStmt{
		NodeBase:  nttBase(2),
		Condition: &ast.BoolLiteral{NodeBase: nttBase(2), Value: true},
		Then: ast.Block{Statements: []ast.Statement{
			&ast.AssignmentStmt{
				NodeBase: nttBase(3),
				Target:   nttVariable("coins", false, 3),
				Value:    nttInt(1, 3),
			},
		}},
		Else: &ast.Block{Statements: []ast.Statement{
			&ast.AssignmentStmt{
				NodeBase: nttBase(5),
				Target:   nttVariable("coins", false, 5),
				Value:    nttFloat(2.5, 5),
			},
		}},
	}
	module := nttModule("main.puff", nttEvent("load", branch, nttExprStmt(read, 7)))

	result := Check(nttProject(module))

	nttAssertNoDiagnostics(t, result.Diagnostics)
	if typ := module.ExpressionTypes[read]; typ.Kind != TypeFloat {
		t.Fatalf("expected runtime global paths to merge to float, got %s", typ.String())
	}
}

func TestCheckReportsEachIndependentReadBeforeDefinition(t *testing.T) {
	first := nttVariable("later", false, 1)
	second := nttVariable("later", false, 2)
	module := nttModule("main.puff",
		&ast.GlobalAssignment{NodeBase: nttBase(1), Target: nttVariable("a", false, 1), Value: first},
		&ast.GlobalAssignment{NodeBase: nttBase(2), Target: nttVariable("b", false, 2), Value: second},
		&ast.GlobalAssignment{NodeBase: nttBase(3), Target: nttVariable("later", false, 3), Value: nttInt(1, 3)},
	)

	result := Check(nttProject(module))

	if len(result.Diagnostics) != 2 {
		t.Fatalf("expected one diagnostic per invalid read, got %#v", result.Diagnostics)
	}
	assertDiagnosticCount(t, result.Diagnostics, diagnostic.CodeUndefinedVariable, 2)
	if result.Diagnostics[0].Span != first.Span() || result.Diagnostics[1].Span != second.Span() {
		t.Fatalf("expected diagnostics at both reads, got %#v", result.Diagnostics)
	}
}
