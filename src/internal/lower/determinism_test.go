package lower

import (
	"testing"

	"github.com/puff-lang/puff/internal/ir"
)

func TestLowerCanonicalizesModulesAndPreservesCommandOrder(t *testing.T) {
	sources := map[string]string{
		"a.puff": `$a = 1

fun alpha -> string
   return "A"
end

on load
   send "first" to console
   send "second" to console
end
`,
		"z.puff": `$z = 2

fun omega -> string
   return "Z"
end
`,
	}

	forward := Lower(checkedProject(t, sources, "a.puff", "z.puff"))
	reversed := Lower(checkedProject(t, sources, "z.puff", "a.puff"))
	if len(forward.Diagnostics) != 0 || len(reversed.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: forward=%#v reversed=%#v", forward.Diagnostics, reversed.Diagnostics)
	}
	if got, want := renderIR(reversed.Project), renderIR(forward.Project); got != want {
		t.Fatalf("module input order changed IR\nforward:\n%s\nreversed:\n%s", want, got)
	}

	wantGlobals := []ir.SymbolID{
		{Module: "a.puff", Name: "a"},
		{Module: "z.puff", Name: "z"},
	}
	if len(forward.Project.Globals) != len(wantGlobals) {
		t.Fatalf("globals = %#v, want %#v", forward.Project.Globals, wantGlobals)
	}
	for index, want := range wantGlobals {
		if got := forward.Project.Globals[index].ID; got != want {
			t.Errorf("global %d ID = %#v, want %#v", index, got, want)
		}
	}

	wantFunctions := []ir.SymbolID{
		{Module: "a.puff", Name: "alpha"},
		{Module: "a.puff", Name: "event/load/1"},
		{Module: "z.puff", Name: "omega"},
	}
	if len(forward.Project.Functions) != len(wantFunctions) {
		t.Fatalf("functions = %#v, want IDs %#v", forward.Project.Functions, wantFunctions)
	}
	for index, want := range wantFunctions {
		if got := forward.Project.Functions[index].ID; got != want {
			t.Errorf("function %d ID = %#v, want %#v", index, got, want)
		}
	}

	handler := findEventFunction(t, forward.Project, "load")
	if len(handler.Commands) != 2 {
		t.Fatalf("load commands = %#v, want two effects", handler.Commands)
	}
	for index, wantText := range []string{"first", "second"} {
		effect, ok := handler.Commands[index].(*ir.Effect)
		if !ok {
			t.Fatalf("command %d = %T, want *ir.Effect", index, handler.Commands[index])
		}
		text, ok := effectArgument(t, effect, "text").(*ir.Text)
		if !ok || len(text.Parts) != 1 {
			t.Fatalf("command %d text = %#v, want one literal", index, text)
		}
		literal, ok := text.Parts[0].(*ir.TextLiteral)
		if !ok || literal.Value != wantText {
			t.Errorf("command %d text = %#v, want %q", index, text.Parts[0], wantText)
		}
	}
}
