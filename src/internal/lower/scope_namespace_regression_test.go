package lower

import (
	"testing"

	"github.com/puff-lang/puff/internal/ir"
)

func TestLowerInterpolationPrefersShadowingParameter(t *testing.T) {
	const input = `fun value -> string
   return "module function"
end

fun show(value: string)
   send "{value}" to console
end
`
	result := Lower(checkedProject(t, map[string]string{"main.puff": input}))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("lower shadowed interpolation: %#v", result.Diagnostics)
	}

	function := findUserFunction(t, result.Project, "show")
	effect, ok := function.Commands[0].(*ir.Effect)
	if !ok {
		t.Fatalf("command = %T, want *ir.Effect", function.Commands[0])
	}
	text := effectArgument(t, effect, "text").(*ir.Text)
	interpolation := text.Parts[0].(*ir.TextInterpolation)
	reference, ok := interpolation.Value.(*ir.Reference)
	if !ok {
		t.Fatalf("interpolation = %T, want shadowing *ir.Reference", interpolation.Value)
	}
	if reference.Name != "value" || reference.Type.Kind != ir.TypeString || reference.Symbol != (ir.SymbolID{}) {
		t.Errorf("reference = %#v, want local string parameter value with no symbol ID", reference)
	}
}

func TestLowerExplicitInterpolationCallRemainsCall(t *testing.T) {
	const input = `fun value -> string
   return "module function"
end

fun show
   send "{value()}" to console
end
`
	result := Lower(checkedProject(t, map[string]string{"main.puff": input}))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("lower explicit interpolation call: %#v", result.Diagnostics)
	}

	function := findUserFunction(t, result.Project, "show")
	effect := function.Commands[0].(*ir.Effect)
	text := effectArgument(t, effect, "text").(*ir.Text)
	interpolation := text.Parts[0].(*ir.TextInterpolation)
	call, ok := interpolation.Value.(*ir.Call)
	if !ok {
		t.Fatalf("interpolation = %T, want explicit *ir.Call", interpolation.Value)
	}
	if got, want := call.Function, (ir.SymbolID{Module: "main.puff", Name: "value"}); got != want {
		t.Errorf("call function = %#v, want %#v", got, want)
	}
}

func TestLowerPreservesCanonicalModulesAndNamespaces(t *testing.T) {
	sources := map[string]string{
		"bravo.puff": "# namespace: bravo\n\nfun noop\nend\n",
		"alpha.puff": "# namespace: alpha\n\nfun noop\nend\n",
	}
	result := Lower(checkedProject(t, sources, "bravo.puff", "alpha.puff"))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("lower namespaced modules: %#v", result.Diagnostics)
	}
	if len(result.Project.Modules) != 2 {
		t.Fatalf("modules = %#v, want two modules", result.Project.Modules)
	}
	want := []struct {
		path      string
		namespace string
	}{
		{path: "alpha.puff", namespace: "alpha"},
		{path: "bravo.puff", namespace: "bravo"},
	}
	for index, expected := range want {
		module := result.Project.Modules[index]
		if module.Path != expected.path || module.Namespace != expected.namespace || module.Source.File != expected.path {
			t.Errorf("module %d = %#v, want path=%q namespace=%q source=%q", index, module, expected.path, expected.namespace, expected.path)
		}
	}

	alpha := Lower(checkedProject(t, map[string]string{"main.puff": "# namespace: alpha\n\nfun noop\nend\n"}))
	bravo := Lower(checkedProject(t, map[string]string{"main.puff": "# namespace: bravo\n\nfun noop\nend\n"}))
	if got, other := renderIR(alpha.Project), renderIR(bravo.Project); got == other {
		t.Fatalf("different namespaces rendered identical IR:\n%s", got)
	}
}
