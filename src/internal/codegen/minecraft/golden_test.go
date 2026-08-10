package minecraft

import (
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/puff-lang/puff/internal/ir"
	"github.com/puff-lang/puff/internal/project"
)

func TestGenerateTargetGolden(t *testing.T) {
	result := Generate(targetProject(), targetConfig())
	if len(result.Diagnostics) != 0 {
		t.Fatalf("Generate() diagnostics = %#v, want none", result.Diagnostics)
	}

	want := readGoldenOutput(t, filepath.Join("testdata", "target"))
	if !reflect.DeepEqual(result.Output.Files, want) {
		t.Fatalf("Generate() files mismatch\n got: %#v\nwant: %#v", result.Output.Files, want)
	}
	assertSortedPaths(t, result.Output.Files)
}

func TestGenerateUsesOfficialLoadAndTickTags(t *testing.T) {
	program := &ir.Project{
		Modules: []ir.Module{{Path: "main.puff", Namespace: "example"}},
		Functions: []ir.Function{
			{
				ID:     ir.SymbolID{Module: "main.puff", Name: "event/load/1"},
				Kind:   ir.FunctionEvent,
				Result: ir.Type{Kind: ir.TypeNil},
			},
			{
				ID:     ir.SymbolID{Module: "main.puff", Name: "event/tick/1"},
				Kind:   ir.FunctionEvent,
				Result: ir.Type{Kind: ir.TypeNil},
			},
		},
		Tags: []ir.Tag{
			{Name: "load", Functions: []ir.SymbolID{{Module: "main.puff", Name: "event/load/1"}}},
			{Name: "tick", Functions: []ir.SymbolID{{Module: "main.puff", Name: "event/tick/1"}}},
		},
	}

	result := Generate(program, targetConfig())
	if len(result.Diagnostics) != 0 {
		t.Fatalf("Generate() diagnostics = %#v, want none", result.Diagnostics)
	}

	files := outputByPath(result.Output)
	assertFile(t, files, "data/minecraft/tags/function/load.json", "{\"values\":[\"example:__puff/load\"]}\n")
	assertFile(t, files, "data/minecraft/tags/function/tick.json", "{\"values\":[\"example:main/event/tick/1\"]}\n")
	assertFile(t, files, "data/example/function/__puff/load.mcfunction", "function example:main/event/load/1\n")
	assertFile(t, files, "data/example/function/main/event/load/1.mcfunction", "")
	assertFile(t, files, "data/example/function/main/event/tick/1.mcfunction", "")
}

func TestGenerateFallsBackToPackNamespace(t *testing.T) {
	program := &ir.Project{
		Modules: []ir.Module{{Path: "main.puff"}},
		Functions: []ir.Function{{
			ID:     ir.SymbolID{Module: "main.puff", Name: "event/tick/1"},
			Kind:   ir.FunctionEvent,
			Result: ir.Type{Kind: ir.TypeNil},
		}},
		Tags: []ir.Tag{{
			Name:      "tick",
			Functions: []ir.SymbolID{{Module: "main.puff", Name: "event/tick/1"}},
		}},
	}
	config := targetConfig()
	config.Pack.ID = "fallback"

	result := Generate(program, config)
	if len(result.Diagnostics) != 0 {
		t.Fatalf("Generate() diagnostics = %#v, want none", result.Diagnostics)
	}

	files := outputByPath(result.Output)
	assertFile(t, files, "data/fallback/function/main/event/tick/1.mcfunction", "")
	assertFile(t, files, "data/minecraft/tags/function/tick.json", "{\"values\":[\"fallback:main/event/tick/1\"]}\n")
}

func TestGenerateRejectsConsoleSend(t *testing.T) {
	tick := ir.SymbolID{Module: "main.puff", Name: "event/tick/1"}
	program := &ir.Project{
		Modules: []ir.Module{{Path: "main.puff", Namespace: "example"}},
		Functions: []ir.Function{{
			ID:     tick,
			Kind:   ir.FunctionEvent,
			Result: ir.Type{Kind: ir.TypeNil},
			Commands: []ir.Command{&ir.Effect{
				PatternID: "core.send",
				Arguments: []ir.Argument{
					{Name: "text", Value: &ir.Text{Parts: []ir.TextPart{&ir.TextLiteral{Value: "Tick"}}}},
					{Name: "target", Value: &ir.Reference{Name: "console", Type: ir.Type{Kind: ir.TypeNamed, Name: "Console"}}},
				},
			}},
		}},
		Tags: []ir.Tag{{Name: "tick", Functions: []ir.SymbolID{tick}}},
	}

	result := Generate(program, targetConfig())
	if len(result.Output.Files) != 0 {
		t.Fatalf("Generate() files = %#v, want none", result.Output.Files)
	}
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "CODEGEN_ERROR" {
		t.Fatalf("Generate() diagnostics = %#v, want CODEGEN_ERROR", result.Diagnostics)
	}
}

func targetProject() *ir.Project {
	serverName := ir.SymbolID{Module: "main.puff", Name: "serverName"}
	load := ir.SymbolID{Module: "main.puff", Name: "event/load/1"}

	returnText := &ir.Text{Parts: []ir.TextPart{
		&ir.TextLiteral{Value: "Lobby"},
	}}
	sendText := &ir.Text{Parts: []ir.TextPart{
		&ir.TextLiteral{Value: "Loaded: "},
		&ir.TextInterpolation{Value: &ir.Call{Function: serverName}},
	}}

	return &ir.Project{
		Modules: []ir.Module{{Path: "main.puff", Namespace: "example"}},
		Globals: []ir.Global{{
			ID:          ir.SymbolID{Module: "main.puff", Name: "coins"},
			Type:        ir.Type{Kind: ir.TypeInt},
			Initializer: &ir.Int{Value: 100},
		}},
		Functions: []ir.Function{
			{
				ID:       serverName,
				Kind:     ir.FunctionUser,
				Result:   ir.Type{Kind: ir.TypeString},
				Commands: []ir.Command{&ir.Return{Value: returnText}},
			},
			{
				ID:     load,
				Kind:   ir.FunctionEvent,
				Result: ir.Type{Kind: ir.TypeNil},
				Commands: []ir.Command{&ir.Effect{
					PatternID: "core.send",
					Arguments: []ir.Argument{
						{Name: "text", Value: sendText},
						{
							Name: "target",
							Value: &ir.Reference{
								Name: "player",
								Type: ir.Type{Kind: ir.TypeNamed, Name: "Player"},
							},
						},
					},
				}},
			},
		},
		Tags: []ir.Tag{{Name: "load", Functions: []ir.SymbolID{load}}},
	}
}

func targetConfig() project.Config {
	return project.Config{
		Pack: project.PackConfig{ID: "example", Name: "Example Pack"},
		Minecraft: project.MinecraftConfig{
			Versions: ">=1.21 <=1.21.6",
			Target:   "1.21.6",
		},
	}
}

func readGoldenOutput(t *testing.T, root string) []File {
	t.Helper()

	var files []File
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, File{Path: filepath.ToSlash(relative), Data: data})
		return nil
	})
	if err != nil {
		t.Fatalf("read golden output: %v", err)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files
}

func outputByPath(output Output) map[string][]byte {
	files := make(map[string][]byte, len(output.Files))
	for _, file := range output.Files {
		files[file.Path] = file.Data
	}
	return files
}

func assertFile(t *testing.T, files map[string][]byte, path, want string) {
	t.Helper()
	got, ok := files[path]
	if !ok {
		t.Fatalf("missing output file %q", path)
	}
	if string(got) != want {
		t.Errorf("%s = %q, want %q", path, got, want)
	}
}

func assertSortedPaths(t *testing.T, files []File) {
	t.Helper()
	for index := 1; index < len(files); index++ {
		if strings.Compare(files[index-1].Path, files[index].Path) >= 0 {
			t.Fatalf("output paths are not strictly sorted: %q before %q", files[index-1].Path, files[index].Path)
		}
	}
}
