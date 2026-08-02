package sema

import (
	"testing"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/token"
)

func TestCheckUnseededImportedGlobalCycleReportsDependenciesOnce(t *testing.T) {
	for _, reverseModules := range []bool{false, true} {
		name := "module order"
		if reverseModules {
			name = "reverse module order"
		}
		t.Run(name, func(t *testing.T) {
			fromB := nttImportedVariable("b", "y", 1)
			moduleA := nttModule("a.puff", &ast.GlobalAssignment{
				NodeBase: nttBase(1),
				Public:   true,
				Target:   nttVariable("x", false, 1),
				Value:    fromB,
			})
			fromA := nttImportedVariable("a", "x", 1)
			moduleB := nttModule("b.puff", &ast.GlobalAssignment{
				NodeBase: nttBase(1),
				Public:   true,
				Target:   nttVariable("y", false, 1),
				Value:    fromA,
			})
			moduleA.Imports["b"] = &Import{Prefix: "b", Target: moduleB}
			moduleB.Imports["a"] = &Import{Prefix: "a", Target: moduleA}

			modules := []*Module{moduleA, moduleB}
			if reverseModules {
				modules = []*Module{moduleB, moduleA}
			}
			result := Check(nttProject(modules...))

			if len(result.Diagnostics) != 1 {
				t.Fatalf("expected one root diagnostic for the unresolved cycle, got %#v", result.Diagnostics)
			}
			assertUndefinedGlobalDependency(t, result.Diagnostics[0], "a.puff", "b.$y", fromB)
			if moduleA.Symbols.Globals["x"].resolution != globalUnresolved ||
				moduleB.Symbols.Globals["y"].resolution != globalUnresolved {
				t.Fatal("expected the unseeded cycle to remain explicitly unresolved")
			}
			if !moduleA.Symbols.Globals["x"].reported || !moduleB.Symbols.Globals["y"].reported {
				t.Fatal("expected the root diagnostic to suppress duplicate cycle reports")
			}
		})
	}
}

func TestCheckDeferredUnknownGlobalIsNotAnUnresolvedDependency(t *testing.T) {
	deferred := &ast.PatternExpr{NodeBase: nttBase(1)}
	library := nttModule("lib/deferred.puff", &ast.GlobalAssignment{
		NodeBase: nttBase(1),
		Public:   true,
		Target:   nttVariable("value", false, 1),
		Value:    deferred,
	})
	read := nttImportedVariable("deferred", "value", 1)
	main := nttModule("main.puff", &ast.GlobalAssignment{
		NodeBase: nttBase(1),
		Target:   nttVariable("copy", false, 1),
		Value:    read,
	})
	main.Imports["deferred"] = &Import{Prefix: "deferred", Target: library}

	result := Check(nttProject(main, library))

	nttAssertNoDiagnostics(t, result.Diagnostics)
	if library.Symbols.Globals["value"].resolution != globalResolved ||
		main.Symbols.Globals["copy"].resolution != globalResolved {
		t.Fatal("expected deferred unknown types to remain resolved dependencies")
	}
	if !main.Symbols.Globals["copy"].Type.IsUnknown() {
		t.Fatalf("expected deferred type to stay unknown, got %s", main.Symbols.Globals["copy"].Type)
	}
}

func TestCheckNegativeStaticGlobalIndexVisibilityIsOrderIndependent(t *testing.T) {
	for _, reverseDeclarations := range []bool{false, true} {
		name := "declaration order"
		if reverseDeclarations {
			name = "reverse declaration order"
		}
		t.Run(name, func(t *testing.T) {
			declarations := []ast.Declaration{
				indexedGlobal("stats", negativeIndex(nttInt(1, 1), 1), true, nttString("visible", 1), 1),
				indexedGlobal("stats", negativeIndex(nttInt(2, 2), 2), false, nttString("hidden", 2), 2),
				indexedGlobal("stats", negativeIndex(nttFloat(1.5, 3), 3), true, nttInt(15, 3), 3),
				indexedGlobal("stats", negativeIndex(nttFloat(2.5, 4), 4), false, nttInt(25, 4), 4),
			}
			if reverseDeclarations {
				for left, right := 0, len(declarations)-1; left < right; left, right = left+1, right-1 {
					declarations[left], declarations[right] = declarations[right], declarations[left]
				}
			}
			library := nttModule("lib/stats.puff", declarations...)

			publicInt := importedIndexedVariable("stats", "stats", negativeIndex(nttInt(1, 10), 10), 10)
			privateInt := importedIndexedVariable("stats", "stats", negativeIndex(nttInt(2, 11), 11), 11)
			publicFloat := importedIndexedVariable("stats", "stats", negativeIndex(nttFloat(1.5, 12), 12), 12)
			privateFloat := importedIndexedVariable("stats", "stats", negativeIndex(nttFloat(2.5, 13), 13), 13)
			main := nttModule("main.puff", nttEvent("load",
				nttExprStmt(publicInt, 10),
				nttExprStmt(privateInt, 11),
				nttExprStmt(publicFloat, 12),
				nttExprStmt(privateFloat, 13),
			))
			main.Imports["stats"] = &Import{Prefix: "stats", Target: library}

			result := Check(nttProject(main, library))

			if len(result.Diagnostics) != 2 {
				t.Fatalf("expected exactly two private-index diagnostics, got %#v", result.Diagnostics)
			}
			assertDiagnosticCount(t, result.Diagnostics, diagnostic.CodeUndefinedVariable, 2)
			assertExpressionKind(t, main, publicInt, TypeString)
			assertExpressionKind(t, main, publicFloat, TypeInt)
			if symbol := library.Symbols.lookupGlobal(publicInt); symbol == nil || !symbol.Public {
				t.Fatalf("expected independent public negative int index, got %#v", symbol)
			}
			if symbol := library.Symbols.lookupGlobal(privateInt); symbol == nil || symbol.Public {
				t.Fatalf("expected independent private negative int index, got %#v", symbol)
			}
			if symbol := library.Symbols.lookupGlobal(publicFloat); symbol == nil || !symbol.Public {
				t.Fatalf("expected independent public negative float index, got %#v", symbol)
			}
			if symbol := library.Symbols.lookupGlobal(privateFloat); symbol == nil || symbol.Public {
				t.Fatalf("expected independent private negative float index, got %#v", symbol)
			}
		})
	}
}

func TestCheckTopLevelReassignmentUsesSequentialTypesAndKeepsLastType(t *testing.T) {
	first := nttInt(1, 1)
	read := nttVariable("x", false, 2)
	sum := &ast.BinaryExpr{
		NodeBase: nttBase(2),
		Left:     read,
		Operator: token.Plus,
		Right:    nttInt(1, 2),
	}
	last := nttString("done", 3)
	module := nttModule("main.puff",
		&ast.GlobalAssignment{NodeBase: nttBase(1), Target: nttVariable("x", false, 1), Value: first},
		&ast.GlobalAssignment{NodeBase: nttBase(2), Target: nttVariable("copy", false, 2), Value: sum},
		&ast.GlobalAssignment{NodeBase: nttBase(3), Target: nttVariable("x", false, 3), Value: last},
	)

	result := Check(nttProject(module))

	nttAssertNoDiagnostics(t, result.Diagnostics)
	assertExpressionKind(t, module, read, TypeInt)
	assertExpressionKind(t, module, sum, TypeInt)
	if got := module.Symbols.Globals["copy"].Type.Kind; got != TypeInt {
		t.Fatalf("expected copy to observe the intermediate int type, got %s", got)
	}
	if got := module.Symbols.Globals["x"].Type.Kind; got != TypeString {
		t.Fatalf("expected x to keep the last assignment type, got %s", got)
	}
}

func assertUndefinedGlobalDependency(
	t *testing.T,
	got diagnostic.Diagnostic,
	file string,
	name string,
	node ast.Node,
) {
	t.Helper()
	if got.Code != diagnostic.CodeUndefinedVariable ||
		got.Phase != diagnostic.PhaseSemantics ||
		got.Severity != diagnostic.SeverityError ||
		got.Message != "Undefined variable: "+name ||
		got.Hint != "Declare it before using it: "+name+" = 0" ||
		got.File != file ||
		got.Span != node.Span() {
		t.Fatalf("unexpected unresolved dependency diagnostic: %#v", got)
	}
}

func negativeIndex(operand ast.Expression, line int) *ast.UnaryExpr {
	return &ast.UnaryExpr{
		NodeBase: nttBase(line),
		Operator: token.Minus,
		Operand:  operand,
	}
}
