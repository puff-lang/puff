package compiler_test

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/puff-lang/puff/internal/codegen/minecraft"
	"github.com/puff-lang/puff/internal/compiler"
	"github.com/puff-lang/puff/internal/diagnostic"
)

func TestBundleWritesExpectedDatapack(t *testing.T) {
	root := bundleTestProject(t, bundleTestValidSource)

	result := compiler.Bundle(context.Background(), compiler.BundleOptions{StartDir: root})

	if !result.Diagnostics.OK {
		t.Fatalf("Bundle() diagnostics = %#v, want success", result.Diagnostics)
	}
	want := bundleTestExpectedDatapack()
	if got := bundleTestOutputTree(result.Output); !reflect.DeepEqual(got, want) {
		t.Errorf("Bundle() output = %#v, want %#v", got, want)
	}
	outputDir := filepath.Join(root, "build", "datapack")
	if got := bundleTestReadTree(t, outputDir); !reflect.DeepEqual(got, want) {
		t.Errorf("written datapack = %#v, want %#v", got, want)
	}
}

func TestBundleIsByteStableAndReplacesStaleOutput(t *testing.T) {
	root := bundleTestProject(t, bundleTestValidSource)
	options := compiler.BundleOptions{StartDir: root}
	outputDir := filepath.Join(root, "build", "datapack")

	first := compiler.Bundle(context.Background(), options)
	if !first.Diagnostics.OK {
		t.Fatalf("first Bundle() diagnostics = %#v, want success", first.Diagnostics)
	}
	firstVirtual := bundleTestOutputTree(first.Output)
	firstDisk := bundleTestReadTree(t, outputDir)

	stalePath := filepath.Join(outputDir, "data", "example", "function", "stale.mcfunction")
	bundleTestWriteFile(t, stalePath, "say stale\n")

	second := compiler.Bundle(context.Background(), options)
	if !second.Diagnostics.OK {
		t.Fatalf("second Bundle() diagnostics = %#v, want success", second.Diagnostics)
	}
	if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
		t.Fatalf("stale file survived rebuild: %v", err)
	}
	if got := bundleTestOutputTree(second.Output); !reflect.DeepEqual(got, firstVirtual) {
		t.Errorf("second virtual output = %#v, want byte-stable %#v", got, firstVirtual)
	}
	if got := bundleTestReadTree(t, outputDir); !reflect.DeepEqual(got, firstDisk) {
		t.Errorf("second written datapack = %#v, want byte-stable %#v", got, firstDisk)
	}
}

func TestBundlePreservesExistingOutputOnDiagnostics(t *testing.T) {
	tests := []struct {
		name      string
		source    string
		wantCode  diagnostic.Code
		wantPhase diagnostic.Phase
	}{
		{
			name:      "semantic error",
			source:    "# tags: load\n\non tick\nend\n",
			wantCode:  diagnostic.CodeMissingLoadEvent,
			wantPhase: diagnostic.PhaseSemantics,
		},
		{
			name:      "codegen error",
			source:    "# tags: load\n\non load\n   send \"server only\" to console\nend\n",
			wantCode:  diagnostic.CodeCodegenError,
			wantPhase: diagnostic.PhaseCodegen,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := bundleTestProject(t, test.source)
			outputDir := filepath.Join(root, "build", "datapack")
			bundleTestWriteFile(t, filepath.Join(outputDir, "sentinel.txt"), "keep me\n")

			result := compiler.Bundle(context.Background(), compiler.BundleOptions{StartDir: root})

			if result.Diagnostics.OK {
				t.Fatal("Bundle() diagnostics reported success, want failure")
			}
			if !bundleTestHasDiagnostic(result.Diagnostics, test.wantCode, test.wantPhase) {
				t.Errorf("Bundle() diagnostics = %#v, want %s in %s", result.Diagnostics, test.wantCode, test.wantPhase)
			}
			if len(result.Output.Files) != 0 {
				t.Errorf("Bundle() output files = %#v, want empty output", result.Output.Files)
			}
			want := map[string]string{"sentinel.txt": "keep me\n"}
			if got := bundleTestReadTree(t, outputDir); !reflect.DeepEqual(got, want) {
				t.Errorf("output after failed Bundle() = %#v, want unchanged %#v", got, want)
			}
		})
	}
}

const bundleTestConfig = `[pack]
id = "example"
name = "Example Pack"

[minecraft]
versions = ">=1.21 <=1.21.6"
target = "1.21.6"

[build]
source = "src"
output = "build/datapack"
`

const bundleTestValidSource = `# namespace: example
# tags: load

$coins = 100

fun serverName -> string
   return "Lobby"
end

on load
   send "Loaded: {serverName}" to player
end
`

func bundleTestProject(t *testing.T, source string) string {
	t.Helper()

	root := t.TempDir()
	bundleTestWriteFile(t, filepath.Join(root, "puff.toml"), bundleTestConfig)
	bundleTestWriteFile(t, filepath.Join(root, "src", "main.puff"), source)
	return root
}

func bundleTestWriteFile(t *testing.T, path, data string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create directory for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func bundleTestExpectedDatapack() map[string]string {
	return map[string]string{
		"data/example/function/__puff/load.mcfunction":       "execute unless data storage example:puff_runtime globals.\"example:main/coins\" run data modify storage example:puff_runtime globals.\"example:main/coins\" set value 100\nfunction example:main/event/load/1\n",
		"data/example/function/main/event/load/1.mcfunction": "function example:main/server_name\ntellraw @a [{\"text\":\"Loaded: \"},{\"nbt\":\"returns.\\\"example:main/server_name\\\"\",\"storage\":\"example:puff_runtime\",\"interpret\":false}]\n",
		"data/example/function/main/server_name.mcfunction":  "return run data modify storage example:puff_runtime returns.\"example:main/server_name\" set value \"Lobby\"\n",
		"data/minecraft/tags/function/load.json":             "{\"values\":[\"example:__puff/load\"]}\n",
		"pack.mcmeta":                                        "{\"pack\":{\"pack_format\":80,\"description\":\"Example Pack\"}}\n",
	}
}

func bundleTestOutputTree(output minecraft.Output) map[string]string {
	files := make(map[string]string, len(output.Files))
	for _, file := range output.Files {
		files[file.Path] = string(file.Data)
	}
	return files
}

func bundleTestReadTree(t *testing.T, root string) map[string]string {
	t.Helper()

	files := make(map[string]string)
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
		files[filepath.ToSlash(relative)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatalf("read tree %s: %v", root, err)
	}
	return files
}

func bundleTestHasDiagnostic(result diagnostic.Result, code diagnostic.Code, phase diagnostic.Phase) bool {
	for _, issue := range result.Errors {
		if issue.Code == code && issue.Phase == phase {
			return true
		}
	}
	return false
}
