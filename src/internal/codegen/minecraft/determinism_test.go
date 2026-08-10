package minecraft

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/puff-lang/puff/internal/ir"
)

func TestGenerateIsStableAcrossReversedInputSlices(t *testing.T) {
	program := deterministicProject()
	reversed := reverseProject(program)

	first := Generate(program, targetConfig())
	second := Generate(reversed, targetConfig())
	if len(first.Diagnostics) != 0 {
		t.Fatalf("first Generate() diagnostics = %#v, want none", first.Diagnostics)
	}
	if len(second.Diagnostics) != 0 {
		t.Fatalf("second Generate() diagnostics = %#v, want none", second.Diagnostics)
	}
	if !reflect.DeepEqual(first.Output.Files, second.Output.Files) {
		t.Fatalf("Generate() output changes with input order\nfirst:  %#v\nsecond: %#v", first.Output.Files, second.Output.Files)
	}
	assertSortedPaths(t, first.Output.Files)
}

func TestGenerateUsesLFAndExactlyOneFinalNewline(t *testing.T) {
	result := Generate(targetProject(), targetConfig())
	if len(result.Diagnostics) != 0 {
		t.Fatalf("Generate() diagnostics = %#v, want none", result.Diagnostics)
	}

	for _, file := range result.Output.Files {
		if len(file.Data) == 0 {
			continue
		}
		if bytes.Contains(file.Data, []byte{'\r'}) {
			t.Errorf("%s contains a carriage return", file.Path)
		}
		if !bytes.HasSuffix(file.Data, []byte{'\n'}) {
			t.Errorf("%s does not end with LF", file.Path)
		}
		if bytes.HasSuffix(file.Data, []byte("\n\n")) {
			t.Errorf("%s ends with more than one LF", file.Path)
		}
	}
}

func deterministicProject() *ir.Project {
	program := targetProject()
	program.Modules = append(program.Modules, ir.Module{Path: "extra.puff", Namespace: "extra"})
	program.Globals = append(program.Globals, ir.Global{
		ID:          ir.SymbolID{Module: "extra.puff", Name: "enabled"},
		Type:        ir.Type{Kind: ir.TypeBool},
		Initializer: &ir.Bool{Value: true},
	})
	extraLoad := ir.SymbolID{Module: "extra.puff", Name: "event/load/1"}
	tick := ir.SymbolID{Module: "extra.puff", Name: "event/tick/1"}
	program.Functions = append(program.Functions,
		ir.Function{ID: extraLoad, Kind: ir.FunctionEvent, Result: ir.Type{Kind: ir.TypeNil}},
		ir.Function{ID: tick, Kind: ir.FunctionEvent, Result: ir.Type{Kind: ir.TypeNil}},
	)
	program.Tags[0].Functions = append(program.Tags[0].Functions, extraLoad)
	program.Tags = append(program.Tags, ir.Tag{Name: "tick", Functions: []ir.SymbolID{tick}})
	return program
}

func reverseProject(source *ir.Project) *ir.Project {
	project := *source
	project.Modules = reversedCopy(source.Modules)
	project.Globals = reversedCopy(source.Globals)
	project.Functions = reversedCopy(source.Functions)
	project.Tags = reversedCopy(source.Tags)
	for index := range project.Tags {
		project.Tags[index].Functions = reversedCopy(project.Tags[index].Functions)
	}
	return &project
}

func reversedCopy[T any](source []T) []T {
	reversed := append([]T(nil), source...)
	for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
		reversed[left], reversed[right] = reversed[right], reversed[left]
	}
	return reversed
}

func TestGoldenOutputHasNoWindowsPathSeparators(t *testing.T) {
	result := Generate(targetProject(), targetConfig())
	if len(result.Diagnostics) != 0 {
		t.Fatalf("Generate() diagnostics = %#v, want none", result.Diagnostics)
	}
	for _, file := range result.Output.Files {
		if strings.Contains(file.Path, `\`) {
			t.Errorf("output path %q contains a Windows separator", file.Path)
		}
	}
}

func TestGenerateUsesInjectiveReturnStorageKeys(t *testing.T) {
	program := &ir.Project{
		Modules: []ir.Module{
			{Path: "a/b.puff", Namespace: "example"},
			{Path: "a.b.puff", Namespace: "example"},
		},
		Functions: []ir.Function{
			stringFunction(ir.SymbolID{Module: "a/b.puff", Name: "value"}),
			stringFunction(ir.SymbolID{Module: "a.b.puff", Name: "value"}),
		},
	}

	result := Generate(program, targetConfig())
	if len(result.Diagnostics) != 0 {
		t.Fatalf("Generate() diagnostics = %#v, want none", result.Diagnostics)
	}
	files := outputByPath(result.Output)
	assertFile(t, files, "data/example/function/a/b/value.mcfunction", "return run data modify storage example:puff_runtime returns.\"example:a/b/value\" set value \"ok\"\n")
	assertFile(t, files, "data/example/function/a.b/value.mcfunction", "return run data modify storage example:puff_runtime returns.\"example:a.b/value\" set value \"ok\"\n")
}

func stringFunction(id ir.SymbolID) ir.Function {
	return ir.Function{
		ID:     id,
		Kind:   ir.FunctionUser,
		Result: ir.Type{Kind: ir.TypeString},
		Commands: []ir.Command{&ir.Return{Value: &ir.Text{Parts: []ir.TextPart{
			&ir.TextLiteral{Value: "ok"},
		}}}},
	}
}
