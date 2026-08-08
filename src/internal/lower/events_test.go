package lower

import (
	"testing"

	"github.com/puff-lang/puff/internal/ir"
	"github.com/puff-lang/puff/internal/patterns"
)

func TestLowerCoreTickCreatesTagAndHandler(t *testing.T) {
	const input = `# tags: tick

on tick
   send "Tick" to console
end
`
	result := Lower(checkedProject(t, map[string]string{"main.puff": input}))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("lower tick: %#v", result.Diagnostics)
	}
	if len(result.Project.Tags) != 1 {
		t.Fatalf("tags = %#v, want one tick tag", result.Project.Tags)
	}
	tag := result.Project.Tags[0]
	if tag.Name != "tick" || len(tag.Functions) != 1 {
		t.Fatalf("tick tag = %#v, want one handler", tag)
	}

	handler := findEventFunction(t, result.Project, "tick")
	if handler.ID != tag.Functions[0] {
		t.Errorf("tag handler = %#v, function ID = %#v", tag.Functions[0], handler.ID)
	}
	if handler.Kind != ir.FunctionEvent || handler.Result.Kind != ir.TypeNil {
		t.Errorf("tick handler kind/result = %s/%s, want event/nil", handler.Kind, handler.Result.Kind)
	}
	if len(handler.Commands) != 1 {
		t.Fatalf("tick commands = %#v, want one effect", handler.Commands)
	}
	effect, ok := handler.Commands[0].(*ir.Effect)
	if !ok || effect.PatternID != patterns.CoreSendEffectID {
		t.Fatalf("tick command = %#v, want %s effect", handler.Commands[0], patterns.CoreSendEffectID)
	}
	target, ok := effectArgument(t, effect, "target").(*ir.Reference)
	if !ok || target.Name != "console" {
		t.Errorf("tick target = %#v, want console reference", target)
	}
}
