package lower

import (
	"testing"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/ir"
)

func TestLowerRejectsResidualVariableAccess(t *testing.T) {
	const input = `$shop.name = "Puff"

fun read
   return $shop.name.extra
end
`
	project := checkedProject(t, map[string]string{"main.puff": input})
	module, ok := project.Module("main.puff")
	if !ok {
		t.Fatal("missing main.puff module")
	}
	function := module.Syntax.Declarations[1].(*ast.FunctionDecl)
	variable := function.Body.Statements[0].(*ast.ReturnStmt).Value.(*ast.VariableExpr)

	result := Lower(project)
	if len(result.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v, want exactly one UNSUPPORTED_AST_NODE", result.Diagnostics)
	}
	issue := result.Diagnostics[0]
	if issue.Code != diagnostic.CodeUnsupportedASTNode || issue.Phase != diagnostic.PhaseIR {
		t.Errorf("diagnostic = %#v, want IR diagnostic %q", issue, diagnostic.CodeUnsupportedASTNode)
	}
	if issue.File != "main.puff" || issue.Span != variable.Span() {
		t.Errorf("diagnostic location = %q %+v, want variable span %+v in main.puff", issue.File, issue.Span, variable.Span())
	}
	if len(result.Project.Globals) != 1 || result.Project.Globals[0].ID != (ir.SymbolID{Module: "main.puff", Name: "shop.name"}) {
		t.Fatalf("globals = %#v, want declared global main.puff::shop.name", result.Project.Globals)
	}
	if commands := findUserFunction(t, result.Project, "read").Commands; len(commands) != 0 {
		t.Errorf("commands = %#v, want no misleading return command", commands)
	}
}

func TestLowerParameterReference(t *testing.T) {
	const input = `fun identity(value: string) -> string
   return value
end
`
	result := Lower(checkedProject(t, map[string]string{"main.puff": input}))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("lower parameter reference: %#v", result.Diagnostics)
	}

	function := findUserFunction(t, result.Project, "identity")
	if len(function.Commands) != 1 {
		t.Fatalf("commands = %#v, want one return", function.Commands)
	}
	returned, ok := function.Commands[0].(*ir.Return)
	if !ok {
		t.Fatalf("command = %T, want *ir.Return", function.Commands[0])
	}
	reference, ok := returned.Value.(*ir.Reference)
	if !ok {
		t.Fatalf("return value = %T, want *ir.Reference", returned.Value)
	}
	if reference.Name != "value" || reference.Type.Kind != ir.TypeString || reference.Symbol != (ir.SymbolID{}) {
		t.Errorf("reference = %#v, want local string parameter named value with no symbol ID", reference)
	}
}

func TestLowerRejectsMalformedSendInterpolation(t *testing.T) {
	tests := []struct {
		name          string
		interpolation string
	}{
		{name: "missing dot", interpolation: "lib visible"},
		{name: "trailing dot", interpolation: "visible."},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := lowerImportedInterpolation(t, test.interpolation)
			assertInvalidIRWithoutEffect(t, result, "main.puff")
		})
	}
}

func TestLowerAcceptsExplicitEmptyInterpolationCall(t *testing.T) {
	result := lowerImportedInterpolation(t, "lib.visible()")
	if len(result.Diagnostics) != 0 {
		t.Fatalf("lower explicit interpolation call: %#v", result.Diagnostics)
	}
	if got := countEffects(result.Project); got != 1 {
		t.Fatalf("effects = %d, want one valid effect", got)
	}
}

func lowerImportedInterpolation(t *testing.T, interpolation string) Result {
	t.Helper()

	sources := map[string]string{
		"main.puff": `require "lib/functions" as lib

on load
   send "{` + interpolation + `}" to player
end
`,
		"lib/functions.puff": `pub fun visible -> string
   return "visible"
end
`,
	}
	return Lower(checkedProjectWithImport(t, sources, "lib"))
}
