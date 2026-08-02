package sema

import (
	"testing"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
)

func TestCheckTopLevelCollectionAssignmentRequiresListValue(t *testing.T) {
	value := nttInt(1, 1)
	target := nttVariableWithFields("players", 1)
	target.Accesses = append(target.Accesses, &ast.EmptyIndexAccess{NodeBase: nttBase(1)})
	module := nttModule("main.puff", &ast.GlobalAssignment{
		NodeBase: nttBase(1),
		Target:   target,
		Value:    value,
	})

	result := Check(nttProject(module))

	nttAssertDiagnostic(t, result.Diagnostics, diagnostic.Diagnostic{
		Code:     diagnostic.CodeTypeMismatch,
		Phase:    diagnostic.PhaseSemantics,
		Severity: diagnostic.SeverityError,
		Message:  "Type mismatch: cannot assign int to $players[].",
		Hint:     "Convert one value or use compatible types.",
		File:     "main.puff",
		Span:     value.Span(),
	})
}

func TestCheckGlobalStaticPathVisibilityIsOrderIndependent(t *testing.T) {
	for _, publicFirst := range []bool{true, false} {
		name := "private path first"
		if publicFirst {
			name = "public path first"
		}
		t.Run(name, func(t *testing.T) {
			public := globalPathDeclaration("config", "name", true, nttString("Shop", 1), 1)
			private := globalPathDeclaration("config", "secret", false, nttInt(7, 2), 2)
			declarations := []ast.Declaration{private, public}
			if publicFirst {
				declarations = []ast.Declaration{public, private}
			}
			declarations = append(declarations, &ast.GlobalAssignment{
				NodeBase: nttBase(3),
				Target:   nttVariable("coins", false, 3),
				Value:    nttInt(10, 3),
			})
			library := nttModule("lib/config.puff", declarations...)

			publicRead := importedVariableWithFields("config", "config", 4, "name")
			privateRead := importedVariableWithFields("config", "config", 5, "secret")
			main := nttModule("main.puff", nttEvent("load",
				nttExprStmt(publicRead, 4),
				nttExprStmt(privateRead, 5),
			))
			main.Imports["config"] = &Import{Prefix: "config", Target: library}

			result := Check(nttProject(main, library))

			nttAssertDiagnostic(t, result.Diagnostics, diagnostic.Diagnostic{
				Code:     diagnostic.CodeUndefinedVariable,
				Phase:    diagnostic.PhaseSemantics,
				Severity: diagnostic.SeverityError,
				Message:  "Undefined variable: config.$config.secret",
				Hint:     "Declare it before using it: config.$config.secret = 0",
				File:     "main.puff",
				Span:     privateRead.Span(),
			})
			if typ := main.ExpressionTypes[publicRead]; typ.Kind != TypeString {
				t.Fatalf("expected public path type string, got %s", typ.String())
			}
			if library.Symbols.Globals["config.name"] == nil || !library.Symbols.Globals["config.name"].Public {
				t.Fatalf("expected public path symbol, got %#v", library.Symbols.Globals)
			}
			if library.Symbols.Globals["config.secret"] == nil || library.Symbols.Globals["config.secret"].Public {
				t.Fatalf("expected private path symbol, got %#v", library.Symbols.Globals)
			}
			if library.Symbols.Globals["coins"] == nil {
				t.Fatalf("expected simple root symbol to remain under Globals[\"coins\"]")
			}
		})
	}

	t.Run("public imported path is read only", func(t *testing.T) {
		library := nttModule(
			"lib/config.puff",
			globalPathDeclaration("config", "name", true, nttString("Shop", 1), 1),
		)
		target := importedVariableWithFields("config", "config", 3, "name")
		main := nttModule("main.puff", nttEvent("load",
			&ast.AssignmentStmt{NodeBase: nttBase(3), Target: target, Value: nttString("Other", 3)},
		))
		main.Imports["config"] = &Import{Prefix: "config", Target: library}

		result := Check(nttProject(main, library))

		nttAssertDiagnostic(t, result.Diagnostics, diagnostic.Diagnostic{
			Code:     diagnostic.CodeAssignToImportedPublicVar,
			Phase:    diagnostic.PhaseSemantics,
			Severity: diagnostic.SeverityError,
			Message:  "Cannot assign to imported public variable: config.$config.name",
			Hint:     "Use a public function like config.setTax(0.2).",
			File:     "main.puff",
			Span:     target.Span(),
		})
	})
}

func TestCheckHeterogeneousCollectionsDoNotMatchGenericTypes(t *testing.T) {
	tests := []struct {
		name       string
		returnType *ast.TypeRef
		value      ast.Expression
	}{
		{
			name:       "list",
			returnType: nttGenericType("list", 1, nttType("int", 1)),
			value: &ast.ListExpr{NodeBase: nttBase(2), Elements: []ast.Expression{
				nttInt(1, 2),
				nttString("wrong", 2),
			}},
		},
		{
			name:       "map",
			returnType: nttGenericType("map", 1, nttType("string", 1), nttType("int", 1)),
			value: &ast.MapExpr{NodeBase: nttBase(2), Entries: []ast.MapEntry{
				{Key: nttString("right", 2), Value: nttInt(1, 2)},
				{Key: nttString("wrong", 2), Value: nttString("wrong", 2)},
			}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			function := returningFunction("values", test.returnType, test.value)
			result := Check(nttProject(nttModule("main.puff", function)))

			if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != diagnostic.CodeTypeMismatch {
				t.Fatalf("expected one TYPE_MISMATCH, got %#v", result.Diagnostics)
			}
		})
	}

	t.Run("unknown from prior error suppresses return cascade", func(t *testing.T) {
		missing := nttVariable("missing", false, 2)
		value := &ast.ListExpr{NodeBase: nttBase(2), Elements: []ast.Expression{
			nttInt(1, 2),
			nttString("incompatible", 2),
			missing,
		}}
		function := returningFunction("values", nttGenericType("list", 1, nttType("int", 1)), value)

		result := Check(nttProject(nttModule("main.puff", function)))

		if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != diagnostic.CodeUndefinedVariable {
			t.Fatalf("expected only UNDEFINED_VARIABLE, got %#v", result.Diagnostics)
		}
	})
}

func TestCheckImportedGlobalInitializerTypeIsModuleOrderIndependent(t *testing.T) {
	for _, importerFirst := range []bool{true, false} {
		name := "dependency first"
		if importerFirst {
			name = "importer first"
		}
		t.Run(name, func(t *testing.T) {
			call := nttCall("calculate", true, 1)
			library := nttModule("lib/config.puff",
				globalPathDeclaration("config", "value", true, call, 1),
				&ast.FunctionDecl{
					NodeBase:   nttBase(2),
					Name:       nttIdentifier("calculate", 2),
					ReturnType: nttType("int", 2),
					Body: ast.Block{Statements: []ast.Statement{
						&ast.ReturnStmt{NodeBase: nttBase(3), Value: nttInt(42, 3)},
					}},
				},
			)
			read := importedVariableWithFields("config", "config", 1, "value")
			main := nttModule("main.puff", &ast.GlobalAssignment{
				NodeBase: nttBase(1),
				Target:   nttVariable("copy", false, 1),
				Value:    read,
			})
			main.Imports["config"] = &Import{Prefix: "config", Target: library}

			modules := []*Module{library, main}
			if importerFirst {
				modules = []*Module{main, library}
			}
			result := Check(nttProject(modules...))

			nttAssertNoDiagnostics(t, result.Diagnostics)
			if got := main.Symbols.Globals["copy"].Type.Kind; got != TypeInt {
				t.Fatalf("expected imported initializer type int, got %s", got)
			}
			if library.ResolvedCalls[call] == nil {
				t.Fatal("expected forward function reference to resolve")
			}
		})
	}

	t.Run("local global read before definition remains invalid", func(t *testing.T) {
		read := nttVariable("later", false, 1)
		module := nttModule("main.puff",
			&ast.GlobalAssignment{NodeBase: nttBase(1), Target: nttVariable("copy", false, 1), Value: read},
			&ast.GlobalAssignment{NodeBase: nttBase(2), Target: nttVariable("later", false, 2), Value: nttInt(1, 2)},
		)

		result := Check(nttProject(module))

		if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != diagnostic.CodeUndefinedVariable {
			t.Fatalf("expected one UNDEFINED_VARIABLE, got %#v", result.Diagnostics)
		}
	})
}

func globalPathDeclaration(
	root string,
	field string,
	public bool,
	value ast.Expression,
	line int,
) *ast.GlobalAssignment {
	return &ast.GlobalAssignment{
		NodeBase: nttBase(line),
		Public:   public,
		Target:   nttVariableWithFields(root, line, field),
		Value:    value,
	}
}

func nttVariableWithFields(root string, line int, fields ...string) *ast.VariableExpr {
	variable := nttVariable(root, false, line)
	for _, field := range fields {
		variable.Accesses = append(variable.Accesses, &ast.FieldAccess{
			NodeBase: nttBase(line),
			Field:    nttIdentifier(field, line),
		})
	}
	return variable
}

func importedVariableWithFields(prefix string, root string, line int, fields ...string) *ast.VariableExpr {
	variable := nttImportedVariable(prefix, root, line)
	for _, field := range fields {
		variable.Accesses = append(variable.Accesses, &ast.FieldAccess{
			NodeBase: nttBase(line),
			Field:    nttIdentifier(field, line),
		})
	}
	return variable
}

func returningFunction(name string, returnType *ast.TypeRef, value ast.Expression) *ast.FunctionDecl {
	return &ast.FunctionDecl{
		NodeBase:   nttBase(1),
		Name:       nttIdentifier(name, 1),
		ReturnType: returnType,
		Body: ast.Block{Statements: []ast.Statement{
			&ast.ReturnStmt{NodeBase: nttBase(2), Value: value},
		}},
	}
}
