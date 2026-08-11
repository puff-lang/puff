package minecraft_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/puff-lang/puff/internal/codegen/minecraft"
)

func TestWriteMaterializesExactOutput(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "datapack")
	output := minecraft.Output{Files: []minecraft.File{
		{Path: "pack.mcmeta", Data: []byte("metadata\n")},
		{Path: "data/example/function/main.mcfunction", Data: []byte("say hello\n")},
		{Path: "data/minecraft/tags/function/load.json", Data: []byte("{\"values\":[]}\n")},
	}}

	if err := minecraft.Write(output, destination); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	want := map[string]string{
		"pack.mcmeta":                            "metadata\n",
		"data/example/function/main.mcfunction":  "say hello\n",
		"data/minecraft/tags/function/load.json": "{\"values\":[]}\n",
	}
	got := readTree(t, destination)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("written tree = %#v, want %#v", got, want)
	}
}

func TestWriteReplacesPreviousOutputTree(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "datapack")
	stale := filepath.Join(destination, "data", "minecraft", "tags", "function", "tick.json")
	if err := os.MkdirAll(filepath.Dir(stale), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	output := minecraft.Output{Files: []minecraft.File{{Path: "pack.mcmeta", Data: []byte("new\n")}}}
	if err := minecraft.Write(output, destination); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("stale file survived replacement: %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(destination, "pack.mcmeta")); err != nil || string(got) != "new\n" {
		t.Fatalf("new output = %q, %v", got, err)
	}
}

func TestWriteContextRestoresPreviousOutputWhenCanceledBeforePublish(t *testing.T) {
	destination := filepath.Join(t.TempDir(), "datapack")
	stale := filepath.Join(destination, "sentinel.txt")
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("keep me\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := &cancelAfterContext{cancelAt: 5}
	err := minecraft.WriteContext(ctx, minecraft.Output{
		Files: []minecraft.File{{Path: "pack.mcmeta", Data: []byte("new\n")}},
	}, destination)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("WriteContext() error = %v, want context canceled", err)
	}
	want := map[string]string{"sentinel.txt": "keep me\n"}
	if got := readTree(t, destination); !reflect.DeepEqual(got, want) {
		t.Fatalf("output after cancellation = %#v, want %#v", got, want)
	}
}

func TestWriteRejectsUnsafeAndDuplicatePathsBeforeCreatingDestination(t *testing.T) {
	root := t.TempDir()
	absPath := filepath.Join(root, "absolute.mcfunction")
	tests := []struct {
		name    string
		files   []minecraft.File
		outside string
	}{
		{
			name: "traversal",
			files: []minecraft.File{
				{Path: "pack.mcmeta", Data: []byte("partial")},
				{Path: "../escape.mcfunction", Data: []byte("escaped")},
			},
			outside: filepath.Join(root, "escape.mcfunction"),
		},
		{
			name: "absolute",
			files: []minecraft.File{
				{Path: "pack.mcmeta", Data: []byte("partial")},
				{Path: absPath, Data: []byte("escaped")},
			},
			outside: absPath,
		},
		{
			name: "backslash",
			files: []minecraft.File{
				{Path: "pack.mcmeta", Data: []byte("partial")},
				{Path: `data\escape.mcfunction`, Data: []byte("escaped")},
			},
		},
		{
			name: "duplicate",
			files: []minecraft.File{
				{Path: "pack.mcmeta", Data: []byte("first")},
				{Path: "pack.mcmeta", Data: []byte("second")},
			},
		},
		{
			name: "portable case collision",
			files: []minecraft.File{
				{Path: "data/example/file", Data: []byte("first")},
				{Path: "Data/example/file", Data: []byte("second")},
			},
		},
		{
			name:  "windows reserved name",
			files: []minecraft.File{{Path: "data/con/file", Data: []byte("reserved")}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			destination := filepath.Join(root, "dest-"+test.name)
			err := minecraft.Write(minecraft.Output{Files: test.files}, destination)
			if err == nil {
				t.Fatal("Write() error = nil, want invalid output path error")
			}
			if _, statErr := os.Stat(destination); !os.IsNotExist(statErr) {
				t.Fatalf("destination was created before validation: %v", statErr)
			}
			if test.outside != "" {
				if _, statErr := os.Stat(test.outside); !os.IsNotExist(statErr) {
					t.Fatalf("outside file was created: %v", statErr)
				}
			}
		})
	}
}

func readTree(t *testing.T, root string) map[string]string {
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
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatalf("read written tree: %v", err)
	}
	return files
}

type cancelAfterContext struct {
	calls    int
	cancelAt int
}

func (ctx *cancelAfterContext) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

func (ctx *cancelAfterContext) Done() <-chan struct{} {
	return nil
}

func (ctx *cancelAfterContext) Err() error {
	ctx.calls++
	if ctx.calls >= ctx.cancelAt {
		return context.Canceled
	}
	return nil
}

func (ctx *cancelAfterContext) Value(any) any {
	return nil
}
