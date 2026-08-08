package lower

import (
	"testing"

	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/ir"
	"github.com/puff-lang/puff/internal/sema"
)

func TestLowerPreservesNestedGlobalIdentity(t *testing.T) {
	const input = `$shop.name = "Puff"
$shop.owner = "Fabio"

fun shopName -> string
   return $shop.name
end
`
	result := Lower(checkedProject(t, map[string]string{"main.puff": input}))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("lower nested globals: %#v", result.Diagnostics)
	}

	wantGlobals := map[ir.SymbolID]bool{
		{Module: "main.puff", Name: "shop.name"}:  false,
		{Module: "main.puff", Name: "shop.owner"}: false,
	}
	if len(result.Project.Globals) != len(wantGlobals) {
		t.Fatalf("globals = %#v, want two distinct nested globals", result.Project.Globals)
	}
	for _, global := range result.Project.Globals {
		if _, ok := wantGlobals[global.ID]; !ok {
			t.Errorf("unexpected global ID %#v", global.ID)
			continue
		}
		if wantGlobals[global.ID] {
			t.Errorf("duplicate global ID %#v", global.ID)
		}
		wantGlobals[global.ID] = true
	}
	for id, found := range wantGlobals {
		if !found {
			t.Errorf("missing global ID %#v", id)
		}
	}

	function := findUserFunction(t, result.Project, "shopName")
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
	if got, want := reference.Symbol, (ir.SymbolID{Module: "main.puff", Name: "shop.name"}); got != want {
		t.Errorf("reference symbol = %#v, want %#v", got, want)
	}
}

func TestLowerRejectsParameterizedFunctionInterpolation(t *testing.T) {
	const input = `fun greet(name: string) -> string
   return "hello"
end

on load
   send "{greet}" to player
end
`
	result := Lower(checkedProject(t, map[string]string{"main.puff": input}))
	assertInvalidIRWithoutEffect(t, result, "main.puff")
}

func TestLowerRejectsPrivateImportedFunctionInterpolation(t *testing.T) {
	sources := map[string]string{
		"main.puff": `require "lib/functions" as lib

on load
   send "{lib.hidden}" to player
end
`,
		"lib/functions.puff": `fun hidden -> string
   return "hidden"
end
`,
	}
	project := checkedProjectWithImport(t, sources, "lib")
	main, _ := project.Module("main.puff")
	hidden := main.Imports["lib"].Target.Symbols.Functions["hidden"]
	if hidden == nil || hidden.Public {
		t.Fatalf("test setup did not resolve a private imported function: %#v", hidden)
	}
	result := Lower(project)
	assertInvalidIRWithoutEffect(t, result, "main.puff")
}

func TestLowerAcceptsPublicZeroArgumentImportedFunctionInterpolation(t *testing.T) {
	sources := map[string]string{
		"main.puff": `require "lib/functions" as lib

on load
   send "{lib.visible}" to player
end
`,
		"lib/functions.puff": `pub fun visible -> string
   return "visible"
end
`,
	}
	project := checkedProjectWithImport(t, sources, "lib")
	main, _ := project.Module("main.puff")
	visible := main.Imports["lib"].Target.Symbols.Functions["visible"]
	if visible == nil || !visible.Public || len(visible.Parameters) != 0 {
		t.Fatalf("test setup did not resolve a public zero-argument function: %#v", visible)
	}
	result := Lower(project)
	if len(result.Diagnostics) != 0 {
		t.Fatalf("lower public imported interpolation: %#v", result.Diagnostics)
	}
	if got := countEffects(result.Project); got != 1 {
		t.Fatalf("effects = %d, want one valid effect", got)
	}

	handler := findEventFunction(t, result.Project, "load")
	effect, ok := handler.Commands[0].(*ir.Effect)
	if !ok {
		t.Fatalf("event command = %T, want *ir.Effect", handler.Commands[0])
	}
	text, ok := effectArgument(t, effect, "text").(*ir.Text)
	if !ok || len(text.Parts) != 1 {
		t.Fatalf("text argument = %#v, want one interpolation", text)
	}
	interpolation, ok := text.Parts[0].(*ir.TextInterpolation)
	if !ok {
		t.Fatalf("text part = %T, want *ir.TextInterpolation", text.Parts[0])
	}
	call, ok := interpolation.Value.(*ir.Call)
	if !ok {
		t.Fatalf("interpolation = %T, want *ir.Call", interpolation.Value)
	}
	want := ir.SymbolID{Module: "lib/functions.puff", Name: "visible"}
	if call.Function != want || len(call.Arguments) != 0 {
		t.Errorf("call = %#v, want zero-argument call to %#v", call, want)
	}
}

func TestLowerBareReturnAsNil(t *testing.T) {
	const input = `fun noop
   return
end
`
	result := Lower(checkedProject(t, map[string]string{"main.puff": input}))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("lower bare return: %#v", result.Diagnostics)
	}

	function := findUserFunction(t, result.Project, "noop")
	if len(function.Commands) != 1 {
		t.Fatalf("commands = %#v, want one return", function.Commands)
	}
	returned, ok := function.Commands[0].(*ir.Return)
	if !ok {
		t.Fatalf("command = %T, want *ir.Return", function.Commands[0])
	}
	if _, ok := returned.Value.(*ir.Nil); !ok {
		t.Errorf("bare return value = %T, want *ir.Nil", returned.Value)
	}
}

func findUserFunction(t *testing.T, project *ir.Project, name string) *ir.Function {
	t.Helper()

	for index := range project.Functions {
		function := &project.Functions[index]
		if function.Kind == ir.FunctionUser && function.ID.Name == name {
			return function
		}
	}
	t.Fatalf("missing user function %q in %#v", name, project.Functions)
	return nil
}

func checkedProjectWithImport(t *testing.T, sources map[string]string, prefix string) *sema.Project {
	t.Helper()

	project := checkedProject(t, sources)
	main, ok := project.Module("main.puff")
	if !ok {
		t.Fatal("missing main.puff module")
	}
	library, ok := project.Module("lib/functions.puff")
	if !ok {
		t.Fatal("missing lib/functions.puff module")
	}
	main.Imports[prefix] = &sema.Import{
		Path:   "lib/functions",
		Prefix: prefix,
		Target: library,
	}
	checked := sema.Check(project)
	if len(checked.Diagnostics) != 0 {
		t.Fatalf("check imported project: %#v", checked.Diagnostics)
	}
	return checked.Project
}

func assertInvalidIRWithoutEffect(t *testing.T, result Result, file string) {
	t.Helper()

	if len(result.Diagnostics) != 1 {
		t.Fatalf("diagnostics = %#v, want exactly one INVALID_IR_NODE", result.Diagnostics)
	}
	issue := result.Diagnostics[0]
	if issue.Code != diagnostic.CodeInvalidIRNode || issue.Phase != diagnostic.PhaseIR || issue.Severity != diagnostic.SeverityError {
		t.Errorf("diagnostic = %#v, want IR error %q", issue, diagnostic.CodeInvalidIRNode)
	}
	if issue.File != file || issue.Span.StartOffset >= issue.Span.EndOffset {
		t.Errorf("diagnostic location = %q %+v, want non-empty span in %q", issue.File, issue.Span, file)
	}
	if got := countEffects(result.Project); got != 0 {
		t.Errorf("effects = %d, want no invalid effect output", got)
	}
}
