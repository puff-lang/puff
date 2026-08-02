package sema

import (
	"reflect"
	"testing"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/source"
	"github.com/puff-lang/puff/internal/token"
)

func TestCheckResolvesModuleGlobalsLocalsAndParameters(t *testing.T) {
	global := nttVariable("coins", false, 1)
	localTarget := nttVariable("price", true, 4)
	localRead := nttVariable("price", true, 5)
	parameterRead := nttCall("amount", false, 6)
	globalRead := nttVariable("coins", false, 7)
	module := nttModule("main.puff",
		&ast.GlobalAssignment{
			NodeBase: nttBase(1),
			Target:   global,
			Value:    nttInt(100, 1),
		},
		&ast.FunctionDecl{
			NodeBase: nttBase(3),
			Name:     nttIdentifier("calculate", 3),
			Parameters: []ast.Parameter{{
				NodeBase: nttBase(3),
				Name:     nttIdentifier("amount", 3),
			}},
			Body: ast.Block{Statements: []ast.Statement{
				&ast.AssignmentStmt{NodeBase: nttBase(4), Target: localTarget, Value: nttInt(50, 4)},
				&ast.ExprStmt{NodeBase: nttBase(5), Expression: localRead},
				&ast.ExprStmt{NodeBase: nttBase(6), Expression: parameterRead},
				&ast.ExprStmt{NodeBase: nttBase(7), Expression: globalRead},
			}},
		},
	)

	project := nttProject(module)
	result := Check(project)

	nttAssertNoDiagnostics(t, result.Diagnostics)
	if result.Project != project {
		t.Fatal("Check must preserve the input project graph")
	}
	if module.Symbols == nil || module.Symbols.Functions["calculate"] == nil ||
		module.Symbols.Globals["coins"] == nil {
		t.Fatalf("expected module symbols to be indexed, got %#v", module.Symbols)
	}
	if symbol := module.ResolvedVariables[localRead]; symbol == nil || !symbol.Local {
		t.Fatalf("expected local read to resolve to a local symbol, got %#v", symbol)
	}
	if symbol := module.ResolvedVariables[globalRead]; symbol == nil || symbol.Local {
		t.Fatalf("expected global read to resolve to a global symbol, got %#v", symbol)
	}
	if typ := module.ExpressionTypes[localRead]; typ.Kind != TypeInt {
		t.Fatalf("expected local read type int, got %#v", typ)
	}
	if _, resolvedAsFunction := module.ResolvedCalls[parameterRead]; resolvedAsFunction {
		t.Fatal("parameter value must not resolve as a function")
	}
}

func TestCheckKeepsLocalsInsideTheirExecutionScope(t *testing.T) {
	leakedRead := nttVariable("price", true, 8)
	module := nttModule("main.puff",
		&ast.FunctionDecl{
			Name: nttIdentifier("first", 2),
			Body: ast.Block{Statements: []ast.Statement{
				&ast.AssignmentStmt{
					NodeBase: nttBase(3),
					Target:   nttVariable("price", true, 3),
					Value:    nttInt(50, 3),
				},
			}},
		},
		&ast.FunctionDecl{
			Name: nttIdentifier("second", 7),
			Body: ast.Block{Statements: []ast.Statement{
				&ast.ExprStmt{NodeBase: nttBase(8), Expression: leakedRead},
			}},
		},
	)

	result := Check(nttProject(module))

	nttAssertDiagnostic(t, result.Diagnostics, diagnostic.Diagnostic{
		Code:     diagnostic.CodeUndefinedVariable,
		Phase:    diagnostic.PhaseSemantics,
		Severity: diagnostic.SeverityError,
		Message:  "Undefined variable: $_price",
		Hint:     "Declare it before using it: $_price = 0",
		File:     "main.puff",
		Span:     leakedRead.Span(),
	})
}

func TestCheckReportsUndefinedSymbolsWithoutCascades(t *testing.T) {
	tests := []struct {
		name string
		decl ast.Declaration
		want diagnostic.Diagnostic
	}{
		{
			name: "global variable",
			decl: nttEvent("load", &ast.ExprStmt{
				NodeBase:   nttBase(4),
				Expression: nttVariable("coins", false, 4),
			}),
			want: diagnostic.Diagnostic{
				Code:    diagnostic.CodeUndefinedVariable,
				Message: "Undefined variable: $coins",
				Hint:    "Declare it before using it: $coins = 0",
				Span:    nttSpan(4),
			},
		},
		{
			name: "function",
			decl: nttEvent("load", &ast.ExprStmt{
				NodeBase:   nttBase(5),
				Expression: nttCall("setupShop", false, 5),
			}),
			want: diagnostic.Diagnostic{
				Code:    diagnostic.CodeUndefinedFunction,
				Message: "Undefined function: setupShop",
				Hint:    "Declare fun setupShop before using it, or import it from a module.",
				Span:    nttSpan(5),
			},
		},
		{
			name: "context name",
			decl: nttEvent("load", &ast.ExprStmt{
				NodeBase:   nttBase(6),
				Expression: nttQualifiedCall([]string{"loop", "index"}, false, 6),
			}),
			want: diagnostic.Diagnostic{
				Code:    diagnostic.CodeUndefinedName,
				Message: "Undefined name: loop.index",
				Hint:    "Declare the name before using it.",
				Span:    nttSpan(6),
			},
		},
		{
			name: "type",
			decl: &ast.FunctionDecl{
				Name: nttIdentifier("lookup", 7),
				Parameters: []ast.Parameter{{
					NodeBase: nttBase(7),
					Name:     nttIdentifier("value", 7),
					Type:     nttType("UnknownType", 7),
				}},
			},
			want: diagnostic.Diagnostic{
				Code:    diagnostic.CodeUndefinedType,
				Message: "Undefined type: UnknownType",
				Span:    nttSpan(7),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.want.Phase = diagnostic.PhaseSemantics
			test.want.Severity = diagnostic.SeverityError
			test.want.File = "main.puff"

			result := Check(nttProject(nttModule("main.puff", test.decl)))

			nttAssertDiagnostic(t, result.Diagnostics, test.want)
		})
	}
}

func TestCheckAcceptsDocumentedBuiltInAndNestedGenericTypes(t *testing.T) {
	builtIns := []string{
		"nil", "bool", "int", "float", "string",
		"list", "map", "range",
		"Player", "Entity", "Mob", "Item", "Block", "Location", "Vector", "NBT",
		"Identifier", "Score", "Objective", "Tag", "Command", "Predicate",
		"function",
		"Error", "TypeError", "NameError", "SyntaxError", "RuntimeError",
		"IndexError", "KeyError", "ValueError",
	}
	parameters := make([]ast.Parameter, 0, len(builtIns)+1)
	for index, name := range builtIns {
		line := index + 1
		parameters = append(parameters, ast.Parameter{
			NodeBase: nttBase(line),
			Name:     nttIdentifier("value"+name, line),
			Type:     nttType(name, line),
		})
	}
	parameters = append(parameters, ast.Parameter{
		NodeBase: nttBase(40),
		Name:     nttIdentifier("nested", 40),
		Type: nttGenericType("map", 40,
			nttType("string", 40),
			nttGenericType("list", 40, nttType("int", 40)),
		),
	})

	result := Check(nttProject(nttModule("types.puff", &ast.FunctionDecl{
		Name:       nttIdentifier("acceptTypes", 1),
		Parameters: parameters,
	})))

	nttAssertNoDiagnostics(t, result.Diagnostics)
}

func TestCheckResolvesLocalAndImportedSymbolsByVisibility(t *testing.T) {
	target := nttModule("abc/shop.puff",
		nttFunction("publicPrice", true, nil, 1),
		nttFunction("privatePrice", false, nil, 2),
		&ast.GlobalAssignment{
			Public:   true,
			NodeBase: nttBase(3),
			Target:   nttVariable("tax", false, 3),
			Value:    nttFloat(0.1, 3),
		},
		&ast.GlobalAssignment{
			NodeBase: nttBase(4),
			Target:   nttVariable("secret", false, 4),
			Value:    nttInt(1, 4),
		},
	)
	importedCall := nttQualifiedCall([]string{"economy", "publicPrice"}, true, 7)
	importedVariable := nttImportedVariable("economy", "tax", 8)
	main := nttModule("main.puff",
		nttFunction("localPrivate", false, nil, 1),
		nttFunction("localPublic", true, nil, 2),
		nttEvent("load",
			nttExprStmt(nttCall("localPrivate", false, 5), 5),
			nttExprStmt(nttCall("localPublic", false, 6), 6),
			nttExprStmt(importedCall, 7),
			nttExprStmt(importedVariable, 8),
		),
	)
	main.Imports["economy"] = &Import{Path: "abc/shop", Prefix: "economy", Target: target}

	result := Check(nttProject(main, target))

	nttAssertNoDiagnostics(t, result.Diagnostics)
	if symbol := main.ResolvedCalls[importedCall]; symbol == nil ||
		symbol.Module != target || symbol.Name != "publicPrice" || !symbol.Public {
		t.Fatalf("expected alias-qualified public function resolution, got %#v", symbol)
	}
	if symbol := main.ResolvedVariables[importedVariable]; symbol == nil ||
		symbol.Module != target || symbol.Name != "tax" || !symbol.Public {
		t.Fatalf("expected alias-qualified public variable resolution, got %#v", symbol)
	}
}

func TestCheckDoesNotExposeImportedSymbolsUnqualifiedOrPrivately(t *testing.T) {
	tests := []struct {
		name string
		expr ast.Expression
		want diagnostic.Diagnostic
	}{
		{
			name: "unqualified public function",
			expr: nttCall("publicPrice", false, 10),
			want: diagnostic.Diagnostic{
				Code:    diagnostic.CodeUndefinedFunction,
				Message: "Undefined function: publicPrice",
				Hint:    "Declare fun publicPrice before using it, or import it from a module.",
				Span:    nttSpan(10),
			},
		},
		{
			name: "unqualified public variable",
			expr: nttVariable("tax", false, 11),
			want: diagnostic.Diagnostic{
				Code:    diagnostic.CodeUndefinedVariable,
				Message: "Undefined variable: $tax",
				Hint:    "Declare it before using it: $tax = 0",
				Span:    nttSpan(11),
			},
		},
		{
			name: "private imported function",
			expr: nttQualifiedCall([]string{"shop", "privatePrice"}, true, 12),
			want: diagnostic.Diagnostic{
				Code:    diagnostic.CodeUndefinedFunction,
				Message: "Undefined function: shop.privatePrice",
				Hint:    "Declare fun shop.privatePrice before using it, or import it from a module.",
				Span:    nttSpan(12),
			},
		},
		{
			name: "private imported variable",
			expr: nttImportedVariable("shop", "secret", 13),
			want: diagnostic.Diagnostic{
				Code:    diagnostic.CodeUndefinedVariable,
				Message: "Undefined variable: shop.$secret",
				Hint:    "Declare it before using it: shop.$secret = 0",
				Span:    nttSpan(13),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			target := nttModule("abc/shop.puff",
				nttFunction("publicPrice", true, nil, 1),
				nttFunction("privatePrice", false, nil, 2),
				&ast.GlobalAssignment{Public: true, Target: nttVariable("tax", false, 3), Value: nttInt(1, 3)},
				&ast.GlobalAssignment{Target: nttVariable("secret", false, 4), Value: nttInt(1, 4)},
			)
			main := nttModule("main.puff", nttEvent("load", nttExprStmt(test.expr, test.expr.Span().StartLine)))
			main.Imports["shop"] = &Import{Path: "abc/shop", Prefix: "shop", Target: target}
			test.want.Phase = diagnostic.PhaseSemantics
			test.want.Severity = diagnostic.SeverityError
			test.want.File = "main.puff"

			result := Check(nttProject(main, target))

			nttAssertDiagnostic(t, result.Diagnostics, test.want)
		})
	}
}

func TestCheckRejectsAssignmentToImportedPublicVariable(t *testing.T) {
	target := nttModule("abc/shop.puff", &ast.GlobalAssignment{
		Public: true,
		Target: nttVariable("tax", false, 1),
		Value:  nttFloat(0.1, 1),
	})
	imported := nttImportedVariable("shop", "tax", 9)
	main := nttModule("main.puff", nttEvent("load", &ast.AssignmentStmt{
		NodeBase: nttBase(9),
		Target:   imported,
		Value:    nttFloat(0.2, 9),
	}))
	main.Imports["shop"] = &Import{Path: "abc/shop", Prefix: "shop", Target: target}

	result := Check(nttProject(main, target))

	nttAssertDiagnostic(t, result.Diagnostics, diagnostic.Diagnostic{
		Code:     diagnostic.CodeAssignToImportedPublicVar,
		Phase:    diagnostic.PhaseSemantics,
		Severity: diagnostic.SeverityError,
		Message:  "Cannot assign to imported public variable: shop.$tax",
		Hint:     "Use a public function like shop.setTax(0.2).",
		File:     "main.puff",
		Span:     imported.Span(),
	})
}

func TestCheckRejectsPublicLocalVariableAST(t *testing.T) {
	local := nttVariable("price", true, 3)
	module := nttModule("main.puff", &ast.GlobalAssignment{
		NodeBase: nttBase(3),
		Public:   true,
		Target:   local,
		Value:    nttInt(50, 3),
	})

	result := Check(nttProject(module))

	nttAssertDiagnostic(t, result.Diagnostics, diagnostic.Diagnostic{
		Code:     diagnostic.CodeInvalidPublicLocalVariable,
		Phase:    diagnostic.PhaseSemantics,
		Severity: diagnostic.SeverityError,
		Message:  "Local variables cannot be public.",
		Hint:     "Only global variables can be exported.",
		File:     "main.puff",
		Span:     local.Span(),
	})
}

func TestCheckValidatesFunctionArguments(t *testing.T) {
	parameter := ast.Parameter{
		NodeBase: nttBase(1),
		Name:     nttIdentifier("amount", 1),
		Type:     nttType("int", 1),
	}
	tests := []struct {
		name string
		call *ast.CallExpr
		want diagnostic.Diagnostic
	}{
		{
			name: "missing parentheses and arguments",
			call: nttCall("reward", false, 5),
			want: diagnostic.Diagnostic{
				Code:    diagnostic.CodeMissingArguments,
				Message: "Missing arguments for function: reward",
				Hint:    "Call it with parentheses: reward(amount)",
				Span:    nttSpan(5),
			},
		},
		{
			name: "too many",
			call: nttCallWithArgs("reward", 6, nttInt(1, 6), nttInt(2, 6)),
			want: diagnostic.Diagnostic{
				Code:    diagnostic.CodeTooManyArguments,
				Message: "Too many arguments.",
				Span:    nttSpan(6),
			},
		},
		{
			name: "wrong type",
			call: nttCallWithArgs("reward", 7, nttString("many", 7)),
			want: diagnostic.Diagnostic{
				Code:    diagnostic.CodeInvalidArgumentType,
				Message: "Invalid argument type.",
				Span:    nttSpan(7),
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			module := nttModule("main.puff",
				nttFunction("reward", false, []ast.Parameter{parameter}, 1),
				nttEvent("load", nttExprStmt(test.call, test.call.Span().StartLine)),
			)
			test.want.Phase = diagnostic.PhaseSemantics
			test.want.Severity = diagnostic.SeverityError
			test.want.File = "main.puff"

			result := Check(nttProject(module))

			nttAssertDiagnostic(t, result.Diagnostics, test.want)
		})
	}
}

func TestCheckReportsExactAdditionTypeMismatch(t *testing.T) {
	expression := &ast.BinaryExpr{
		NodeBase: nttBase(4),
		Left:     nttInt(1, 4),
		Operator: token.Plus,
		Right:    nttString("one", 4),
	}
	module := nttModule("main.puff", &ast.GlobalAssignment{
		Target: nttVariable("total", false, 4),
		Value:  expression,
	})

	result := Check(nttProject(module))

	nttAssertDiagnostic(t, result.Diagnostics, diagnostic.Diagnostic{
		Code:     diagnostic.CodeTypeMismatch,
		Phase:    diagnostic.PhaseSemantics,
		Severity: diagnostic.SeverityError,
		Message:  "Type mismatch: cannot add int and string.",
		Hint:     "Convert one value or use compatible types.",
		File:     "main.puff",
		Span:     expression.Span(),
	})
}

func TestCheckAcceptsNumericCompatibility(t *testing.T) {
	expressions := []ast.Expression{
		&ast.BinaryExpr{NodeBase: nttBase(1), Left: nttInt(1, 1), Operator: token.Plus, Right: nttInt(2, 1)},
		&ast.BinaryExpr{NodeBase: nttBase(2), Left: nttInt(1, 2), Operator: token.Plus, Right: nttFloat(2.5, 2)},
		&ast.BinaryExpr{NodeBase: nttBase(3), Left: nttFloat(1.5, 3), Operator: token.Plus, Right: nttInt(2, 3)},
		&ast.BinaryExpr{NodeBase: nttBase(4), Left: nttFloat(1.5, 4), Operator: token.Plus, Right: nttFloat(2.5, 4)},
	}
	declarations := make([]ast.Declaration, 0, len(expressions))
	for index, expression := range expressions {
		declarations = append(declarations, &ast.GlobalAssignment{
			Target: nttVariable("number", false, index+1),
			Value:  expression,
		})
	}

	result := Check(nttProject(nttModule("main.puff", declarations...)))

	nttAssertNoDiagnostics(t, result.Diagnostics)
	module := result.Project.Modules[0]
	wantKinds := []TypeKind{TypeInt, TypeFloat, TypeFloat, TypeFloat}
	for index, expression := range expressions {
		if got := module.ExpressionTypes[expression].Kind; got != wantKinds[index] {
			t.Fatalf("expression %d: expected type %s, got %s", index, wantKinds[index], got)
		}
	}
}

func nttProject(modules ...*Module) *Project {
	return &Project{Root: "/project", Modules: modules}
}

func nttModule(relPath string, declarations ...ast.Declaration) *Module {
	return &Module{
		Source:  source.NewFile("/project/src/"+relPath, relPath, ""),
		Syntax:  &ast.File{Declarations: declarations},
		Imports: map[string]*Import{},
	}
}

func nttFunction(name string, public bool, parameters []ast.Parameter, line int) *ast.FunctionDecl {
	return &ast.FunctionDecl{
		NodeBase:   nttBase(line),
		Public:     public,
		Name:       nttIdentifier(name, line),
		Parameters: parameters,
	}
}

func nttEvent(name string, statements ...ast.Statement) *ast.EventDecl {
	return &ast.EventDecl{
		NodeBase: nttBase(2),
		Name:     []ast.Identifier{nttIdentifier(name, 2)},
		Body:     ast.Block{Statements: statements},
	}
}

func nttExprStmt(expression ast.Expression, line int) *ast.ExprStmt {
	return &ast.ExprStmt{NodeBase: nttBase(line), Expression: expression}
}

func nttVariable(name string, local bool, line int) *ast.VariableExpr {
	return &ast.VariableExpr{
		NodeBase: nttBase(line),
		Name:     nttIdentifier(name, line),
		Local:    local,
	}
}

func nttImportedVariable(prefix string, name string, line int) *ast.VariableExpr {
	qualifier := nttIdentifier(prefix, line)
	return &ast.VariableExpr{
		NodeBase:  nttBase(line),
		Qualifier: &qualifier,
		Name:      nttIdentifier(name, line),
	}
}

func nttCall(name string, explicit bool, line int) *ast.CallExpr {
	return nttQualifiedCall([]string{name}, explicit, line)
}

func nttCallWithArgs(name string, line int, arguments ...ast.Expression) *ast.CallExpr {
	call := nttCall(name, true, line)
	call.Arguments = arguments
	return call
}

func nttQualifiedCall(parts []string, explicit bool, line int) *ast.CallExpr {
	identifiers := make([]ast.Identifier, 0, len(parts))
	for _, part := range parts {
		identifiers = append(identifiers, nttIdentifier(part, line))
	}
	return &ast.CallExpr{
		NodeBase:       nttBase(line),
		Callee:         ast.QualifiedName{NodeBase: nttBase(line), Parts: identifiers},
		ExplicitParens: explicit,
	}
}

func nttType(name string, line int) *ast.TypeRef {
	return &ast.TypeRef{
		NodeBase: nttBase(line),
		Name:     nttIdentifier(name, line),
	}
}

func nttGenericType(name string, line int, arguments ...*ast.TypeRef) *ast.TypeRef {
	typeRef := nttType(name, line)
	typeRef.Arguments = arguments
	return typeRef
}

func nttInt(value int64, line int) *ast.IntLiteral {
	return &ast.IntLiteral{NodeBase: nttBase(line), Value: value}
}

func nttFloat(value float64, line int) *ast.FloatLiteral {
	return &ast.FloatLiteral{NodeBase: nttBase(line), Value: value}
}

func nttString(value string, line int) *ast.StringExpr {
	return &ast.StringExpr{
		NodeBase: nttBase(line),
		Quote:    '"',
		Parts: []ast.StringPart{
			&ast.StringText{NodeBase: nttBase(line), Raw: value, Value: value},
		},
	}
}

func nttIdentifier(name string, line int) ast.Identifier {
	return ast.Identifier{NodeBase: nttBase(line), Name: name}
}

func nttBase(line int) ast.NodeBase {
	return ast.NodeBase{SourceSpan: nttSpan(line)}
}

func nttSpan(line int) diagnostic.Span {
	return diagnostic.Span{
		StartLine:   line,
		StartColumn: 3,
		EndLine:     line,
		EndColumn:   12,
		StartOffset: line * 20,
		EndOffset:   line*20 + 9,
	}
}

func nttAssertNoDiagnostics(t *testing.T, diagnostics []diagnostic.Diagnostic) {
	t.Helper()
	if len(diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", diagnostics)
	}
}

func nttAssertDiagnostic(
	t *testing.T,
	diagnostics []diagnostic.Diagnostic,
	want diagnostic.Diagnostic,
) {
	t.Helper()
	if len(diagnostics) != 1 {
		t.Fatalf("expected exactly one diagnostic without cascades, got %#v", diagnostics)
	}
	if !reflect.DeepEqual(diagnostics[0], want) {
		t.Fatalf("unexpected diagnostic:\ngot  %#v\nwant %#v", diagnostics[0], want)
	}
}
