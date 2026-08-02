package sema

import (
	"testing"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/token"
)

func TestCheckGlobalStaticIndexVisibilityIsOrderIndependent(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		name := "declaration order"
		if reverse {
			name = "reverse declaration order"
		}
		t.Run(name, func(t *testing.T) {
			declarations := []ast.Declaration{
				indexedGlobal("stats", nttString("coins", 1), true, nttInt(10, 1), 1),
				indexedGlobal("stats", nttString("secret", 2), false, nttString("hidden", 2), 2),
				indexedGlobal("stats", nttInt(1, 3), true, nttString("one", 3), 3),
				indexedGlobal("stats", &ast.BoolLiteral{NodeBase: nttBase(4), Value: true}, true,
					&ast.BoolLiteral{NodeBase: nttBase(4), Value: true}, 4),
				indexedGlobal("stats", &ast.NilLiteral{NodeBase: nttBase(5)}, true, nttFloat(2.5, 5), 5),
			}
			if reverse {
				for left, right := 0, len(declarations)-1; left < right; left, right = left+1, right-1 {
					declarations[left], declarations[right] = declarations[right], declarations[left]
				}
			}
			library := nttModule("lib/stats.puff", declarations...)

			coins := importedIndexedVariable("stats", "stats", nttString("coins", 10), 10)
			secret := importedIndexedVariable("stats", "stats", nttString("secret", 11), 11)
			number := importedIndexedVariable("stats", "stats", nttInt(1, 12), 12)
			flag := importedIndexedVariable("stats", "stats",
				&ast.BoolLiteral{NodeBase: nttBase(13), Value: true}, 13)
			nilValue := importedIndexedVariable("stats", "stats",
				&ast.NilLiteral{NodeBase: nttBase(14)}, 14)
			main := nttModule("main.puff", nttEvent("load",
				nttExprStmt(coins, 10),
				nttExprStmt(secret, 11),
				nttExprStmt(number, 12),
				nttExprStmt(flag, 13),
				nttExprStmt(nilValue, 14),
			))
			main.Imports["stats"] = &Import{Prefix: "stats", Target: library}

			result := Check(nttProject(main, library))

			nttAssertDiagnostic(t, result.Diagnostics, diagnostic.Diagnostic{
				Code:     diagnostic.CodeUndefinedVariable,
				Phase:    diagnostic.PhaseSemantics,
				Severity: diagnostic.SeverityError,
				Message:  "Undefined variable: stats.$stats",
				Hint:     "Declare it before using it: stats.$stats = 0",
				File:     "main.puff",
				Span:     secret.Span(),
			})
			assertExpressionKind(t, main, coins, TypeInt)
			assertExpressionKind(t, main, number, TypeString)
			assertExpressionKind(t, main, flag, TypeBool)
			assertExpressionKind(t, main, nilValue, TypeFloat)
		})
	}
}

func TestCheckGlobalDynamicIndexFallsBackToRootSymbol(t *testing.T) {
	library := nttModule("lib/stats.puff", &ast.GlobalAssignment{
		NodeBase: nttBase(1),
		Public:   true,
		Target:   nttVariable("stats", false, 1),
		Value: &ast.MapExpr{NodeBase: nttBase(1), Entries: []ast.MapEntry{{
			Key:   nttInt(2, 1),
			Value: nttString("two", 1),
		}}},
	})
	index := &ast.BinaryExpr{
		NodeBase: nttBase(3),
		Left:     nttInt(1, 3),
		Operator: token.Plus,
		Right:    nttInt(1, 3),
	}
	read := importedIndexedVariable("stats", "stats", index, 3)
	main := nttModule("main.puff", nttEvent("load", nttExprStmt(read, 3)))
	main.Imports["stats"] = &Import{Prefix: "stats", Target: library}

	result := Check(nttProject(main, library))

	nttAssertNoDiagnostics(t, result.Diagnostics)
	assertExpressionKind(t, main, read, TypeString)
	if main.ResolvedVariables[read] != library.Symbols.Globals["stats"] {
		t.Fatal("expected dynamic index lookup to resolve through root global")
	}
}

func TestCheckGlobalInitializerCycleConvergesWithoutDuplicateDiagnostics(t *testing.T) {
	for _, reverseModules := range []bool{false, true} {
		name := "module order"
		if reverseModules {
			name = "reverse module order"
		}
		t.Run(name, func(t *testing.T) {
			fromB := nttImportedVariable("b", "copy", 2)
			moduleA := nttModule("a.puff",
				&ast.GlobalAssignment{
					NodeBase: nttBase(1),
					Public:   true,
					Target:   nttVariable("seed", false, 1),
					Value:    nttInt(1, 1),
				},
				&ast.GlobalAssignment{
					NodeBase: nttBase(2),
					Public:   true,
					Target:   nttVariable("fromB", false, 2),
					Value:    fromB,
				},
			)

			fromA := nttImportedVariable("a", "seed", 1)
			missing := nttVariable("missing", false, 2)
			moduleB := nttModule("b.puff",
				&ast.GlobalAssignment{
					NodeBase: nttBase(1),
					Public:   true,
					Target:   nttVariable("copy", false, 1),
					Value:    fromA,
				},
				&ast.GlobalAssignment{
					NodeBase: nttBase(2),
					Target:   nttVariable("broken", false, 2),
					Value:    missing,
				},
				returningFunction("value", nttType("string", 3), nttVariable("copy", false, 4)),
			)
			moduleA.Imports["b"] = &Import{Prefix: "b", Target: moduleB}
			moduleB.Imports["a"] = &Import{Prefix: "a", Target: moduleA}

			modules := []*Module{moduleA, moduleB}
			if reverseModules {
				modules = []*Module{moduleB, moduleA}
			}
			result := Check(nttProject(modules...))

			if len(result.Diagnostics) != 2 {
				t.Fatalf("expected two diagnostics without fixpoint duplicates, got %#v", result.Diagnostics)
			}
			assertDiagnosticCount(t, result.Diagnostics, diagnostic.CodeUndefinedVariable, 1)
			assertDiagnosticCount(t, result.Diagnostics, diagnostic.CodeTypeMismatch, 1)
			if got := moduleA.Symbols.Globals["fromB"].Type.Kind; got != TypeInt {
				t.Fatalf("expected cycle to converge from seed to int, got %s", got)
			}
			if got := moduleB.Symbols.Globals["copy"].Type.Kind; got != TypeInt {
				t.Fatalf("expected imported cycle value to converge to int, got %s", got)
			}
			if moduleA.ResolvedVariables[fromB] == nil || moduleB.ResolvedVariables[fromA] == nil {
				t.Fatal("expected converged imported globals to remain resolved")
			}
		})
	}
}

func indexedGlobal(
	root string,
	index ast.Expression,
	public bool,
	value ast.Expression,
	line int,
) *ast.GlobalAssignment {
	return &ast.GlobalAssignment{
		NodeBase: nttBase(line),
		Public:   public,
		Target:   indexedVariable(root, index, line),
		Value:    value,
	}
}

func importedIndexedVariable(prefix string, root string, index ast.Expression, line int) *ast.VariableExpr {
	variable := nttImportedVariable(prefix, root, line)
	variable.Accesses = append(variable.Accesses, &ast.IndexAccess{NodeBase: nttBase(line), Index: index})
	return variable
}

func indexedVariable(root string, index ast.Expression, line int) *ast.VariableExpr {
	variable := nttVariable(root, false, line)
	variable.Accesses = append(variable.Accesses, &ast.IndexAccess{NodeBase: nttBase(line), Index: index})
	return variable
}

func assertExpressionKind(t *testing.T, module *Module, expression ast.Expression, want TypeKind) {
	t.Helper()
	if got := module.ExpressionTypes[expression].Kind; got != want {
		t.Fatalf("expected expression type %s, got %s", want, got)
	}
}

func assertDiagnosticCount(
	t *testing.T,
	diagnostics []diagnostic.Diagnostic,
	code diagnostic.Code,
	want int,
) {
	t.Helper()
	got := 0
	for _, item := range diagnostics {
		if item.Code == code {
			got++
		}
	}
	if got != want {
		t.Fatalf("expected %d %s diagnostics, got %d: %#v", want, code, got, diagnostics)
	}
}
