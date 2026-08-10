package minecraft_test

import (
	"encoding/json"
	"testing"

	"github.com/puff-lang/puff/internal/codegen/minecraft"
	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/ir"
	"github.com/puff-lang/puff/internal/project"
)

func TestGenerateRejectsInvalidNamespace(t *testing.T) {
	program := baseProgram()
	program.Modules[0].Namespace = "Invalid Namespace"

	assertGenerateFailure(t, program, baseConfig(), diagnostic.CodeInvalidNamespace)
}

func TestGenerateRejectsMalformedTarget(t *testing.T) {
	config := baseConfig()
	config.Minecraft.Target = "1.21.x"

	assertGenerateFailureInPhase(t, baseProgram(), config, diagnostic.CodeInvalidMinecraftVersion, diagnostic.PhaseProject)
}

func TestGenerateRejectsUnsupportedTarget(t *testing.T) {
	config := baseConfig()
	config.Minecraft.Target = "1.20.6"

	assertGenerateFailure(t, baseProgram(), config, diagnostic.CodeUnsupportedMinecraftVersion)
}

func TestGenerateAcceptsMatchingExplicitPackFormat(t *testing.T) {
	config := baseConfig()
	config.Minecraft.PackFormat = 80

	result := minecraft.Generate(baseProgram(), config)
	if len(result.Diagnostics) != 0 {
		t.Fatalf("Generate() diagnostics = %#v, want none", result.Diagnostics)
	}

	data := generatedFile(t, result.Output, "pack.mcmeta")
	var metadata struct {
		Pack struct {
			Format int `json:"pack_format"`
		} `json:"pack"`
	}
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("decode pack.mcmeta: %v\n%s", err, data)
	}
	if metadata.Pack.Format != 80 {
		t.Errorf("pack_format = %d, want explicit value 80", metadata.Pack.Format)
	}
}

func TestGenerateRejectsPackFormatIncompatibleWithTarget(t *testing.T) {
	config := baseConfig()
	config.Minecraft.PackFormat = 48

	assertGenerateFailureInPhase(t, baseProgram(), config, diagnostic.CodeInvalidMinecraftVersion, diagnostic.PhaseProject)
}

func TestGenerateInfersHighestTargetInRange(t *testing.T) {
	config := baseConfig()
	config.Minecraft.Target = ""

	result := minecraft.Generate(baseProgram(), config)
	if len(result.Diagnostics) != 0 {
		t.Fatalf("Generate() diagnostics = %#v, want none", result.Diagnostics)
	}

	data := generatedFile(t, result.Output, "pack.mcmeta")
	var metadata struct {
		Pack struct {
			Format int `json:"pack_format"`
		} `json:"pack"`
	}
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("decode pack.mcmeta: %v\n%s", err, data)
	}
	if metadata.Pack.Format != 80 {
		t.Errorf("inferred pack_format = %d, want 80", metadata.Pack.Format)
	}
}

func TestGenerateRejectsResourceNormalizationCollision(t *testing.T) {
	program := baseProgram()
	program.Functions = []ir.Function{
		userFunction("serverName"),
		userFunction("server_name"),
	}

	assertGenerateFailure(t, program, baseConfig(), diagnostic.CodeInvalidMinecraftResource)
}

func TestGenerateRejectsUnsafeModulePaths(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "traversal", path: "../escape.puff"},
		{name: "backslash", path: `dir\escape.puff`},
		{name: "windows absolute", path: `C:/escape.puff`},
		{name: "posix absolute", path: "/escape.puff"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			program := baseProgram()
			program.Modules[0].Path = test.path

			assertGenerateFailure(t, program, baseConfig(), diagnostic.CodeInvalidMinecraftResource)
		})
	}
}

func TestGenerateRejectsNonPortableGeneratedResource(t *testing.T) {
	program := &ir.Project{
		Modules:   []ir.Module{{Path: "con.puff", Namespace: "example"}},
		Functions: []ir.Function{userFunctionForModule("con.puff", "value")},
	}

	assertGenerateFailure(t, program, baseConfig(), diagnostic.CodeInvalidMinecraftResource)
}

func TestGenerateRejectsUnsupportedEffect(t *testing.T) {
	program := baseProgram()
	program.Functions = []ir.Function{{
		ID:     ir.SymbolID{Module: "main.puff", Name: "event/load/1"},
		Kind:   ir.FunctionEvent,
		Result: ir.Type{Kind: ir.TypeNil},
		Commands: []ir.Command{
			&ir.Effect{PatternID: "test.unsupported"},
		},
	}}

	assertGenerateFailure(t, program, baseConfig(), diagnostic.CodeCodegenError)
}

func TestGenerateRejectsControlEscapesBeforeMinecraft1215(t *testing.T) {
	program := programReturningText("line one\nline two")
	config := baseConfig()
	config.Minecraft.Versions = "1.21"
	config.Minecraft.Target = "1.21"

	assertGenerateFailure(t, program, config, diagnostic.CodeCodegenError)
}

func TestGenerateUsesControlEscapesFromMinecraft1215(t *testing.T) {
	program := programReturningText("line one\nline two")
	config := baseConfig()
	config.Minecraft.Versions = "1.21.5"
	config.Minecraft.Target = "1.21.5"

	result := minecraft.Generate(program, config)
	if len(result.Diagnostics) != 0 {
		t.Fatalf("Generate() diagnostics = %#v, want none", result.Diagnostics)
	}
	want := "return run data modify storage example:puff_runtime returns.\"example:main/message\" set value \"line one\\nline two\"\n"
	if got := string(generatedFile(t, result.Output, "data/example/function/main/message.mcfunction")); got != want {
		t.Fatalf("generated function = %q, want %q", got, want)
	}
}

func programReturningText(value string) *ir.Project {
	program := baseProgram()
	program.Functions = []ir.Function{{
		ID:     ir.SymbolID{Module: "main.puff", Name: "message"},
		Kind:   ir.FunctionUser,
		Result: ir.Type{Kind: ir.TypeString},
		Commands: []ir.Command{&ir.Return{Value: &ir.Text{Parts: []ir.TextPart{
			&ir.TextLiteral{Value: value},
		}}}},
	}}
	return program
}

func baseProgram() *ir.Project {
	return &ir.Project{
		Modules: []ir.Module{{Path: "main.puff", Namespace: "example"}},
	}
}

func baseConfig() project.Config {
	return project.Config{
		Pack: project.PackConfig{ID: "example", Name: "Example Datapack"},
		Minecraft: project.MinecraftConfig{
			Versions: ">=1.21 <=1.21.6",
			Target:   "1.21.6",
		},
	}
}

func userFunction(name string) ir.Function {
	return userFunctionForModule("main.puff", name)
}

func userFunctionForModule(module, name string) ir.Function {
	return ir.Function{
		ID:     ir.SymbolID{Module: module, Name: name},
		Kind:   ir.FunctionUser,
		Result: ir.Type{Kind: ir.TypeNil},
	}
}

func assertGenerateFailure(t *testing.T, program *ir.Project, config project.Config, code diagnostic.Code) {
	t.Helper()
	assertGenerateFailureInPhase(t, program, config, code, diagnostic.PhaseCodegen)
}

func assertGenerateFailureInPhase(t *testing.T, program *ir.Project, config project.Config, code diagnostic.Code, phase diagnostic.Phase) {
	t.Helper()

	result := minecraft.Generate(program, config)
	if len(result.Output.Files) != 0 {
		t.Fatalf("Generate() emitted files on failure: %#v", result.Output.Files)
	}
	if len(result.Diagnostics) != 1 {
		t.Fatalf("Generate() diagnostics = %#v, want exactly one %s", result.Diagnostics, code)
	}
	issue := result.Diagnostics[0]
	if issue.Code != code {
		t.Errorf("diagnostic code = %q, want %q", issue.Code, code)
	}
	if issue.Phase != phase {
		t.Errorf("diagnostic phase = %q, want %q", issue.Phase, phase)
	}
	if issue.Severity != diagnostic.SeverityError {
		t.Errorf("diagnostic severity = %q, want %q", issue.Severity, diagnostic.SeverityError)
	}
}

func generatedFile(t *testing.T, output minecraft.Output, path string) []byte {
	t.Helper()

	for _, file := range output.Files {
		if file.Path == path {
			return file.Data
		}
	}
	t.Fatalf("generated file %q not found in %#v", path, output.Files)
	return nil
}
