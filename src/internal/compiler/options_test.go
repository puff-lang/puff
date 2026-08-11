package compiler_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/puff-lang/puff/internal/compiler"
	"github.com/puff-lang/puff/internal/diagnostic"
)

func TestBundleOutputOverrides(t *testing.T) {
	tests := []struct {
		name       string
		override   func(t *testing.T, root string) string
		outputPath func(root, override string) string
	}{
		{
			name: "relative to project root",
			override: func(_ *testing.T, _ string) string {
				return filepath.Join("custom", "datapack")
			},
			outputPath: func(root, override string) string {
				return filepath.Join(root, override)
			},
		},
		{
			name: "absolute",
			override: func(t *testing.T, _ string) string {
				return filepath.Join(t.TempDir(), "datapack")
			},
			outputPath: func(_ string, override string) string {
				return override
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := optionsTestProject(t, true)
			override := test.override(t, root)

			result := compiler.Bundle(context.Background(), compiler.BundleOptions{
				StartDir: root,
				Output:   override,
			})

			if !result.Diagnostics.OK {
				t.Fatalf("Bundle() diagnostics = %#v, want success", result.Diagnostics)
			}
			if len(result.Output.Files) == 0 {
				t.Fatal("Bundle() output is empty")
			}
			output := test.outputPath(root, override)
			if _, err := os.Stat(filepath.Join(output, "pack.mcmeta")); err != nil {
				t.Fatalf("stat overridden pack.mcmeta: %v", err)
			}
			if _, err := os.Stat(filepath.Join(root, "configured-dist")); !os.IsNotExist(err) {
				t.Fatalf("configured output was written despite override: %v", err)
			}
		})
	}
}

func TestBundleTargetOverrideChangesPackFormat(t *testing.T) {
	root := optionsTestProject(t, true)

	result := compiler.Bundle(context.Background(), compiler.BundleOptions{
		StartDir: root,
		Target:   "1.21.5",
	})

	if !result.Diagnostics.OK {
		t.Fatalf("Bundle() diagnostics = %#v, want success", result.Diagnostics)
	}
	data, err := os.ReadFile(filepath.Join(root, "configured-dist", "pack.mcmeta"))
	if err != nil {
		t.Fatalf("read pack.mcmeta: %v", err)
	}
	var metadata struct {
		Pack struct {
			Format int `json:"pack_format"`
		} `json:"pack"`
	}
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("decode pack.mcmeta: %v", err)
	}
	if metadata.Pack.Format != 71 {
		t.Fatalf("pack_format = %d, want 71 for target 1.21.5", metadata.Pack.Format)
	}
}

func TestCheckMapsProjectLoadingFailuresToInvalidConfig(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) string
	}{
		{
			name: "malformed config",
			setup: func(t *testing.T) string {
				root := t.TempDir()
				optionsTestWriteFile(t, filepath.Join(root, "puff.toml"), "[pack\n")
				return root
			},
		},
		{
			name: "missing source directory",
			setup: func(t *testing.T) string {
				return optionsTestProject(t, false)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := compiler.Check(context.Background(), compiler.CheckOptions{StartDir: test.setup(t)})

			optionsTestAssertErrorCode(t, result.Diagnostics, diagnostic.CodeInvalidConfig)
		})
	}
}

func TestBundleCanceledContextDoesNotWriteOutput(t *testing.T) {
	root := optionsTestProject(t, true)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := compiler.Bundle(ctx, compiler.BundleOptions{StartDir: root})

	optionsTestAssertErrorCode(t, result.Diagnostics, diagnostic.CodeCommandFailed)
	if len(result.Output.Files) != 0 {
		t.Fatalf("Bundle() output = %#v, want empty output", result.Output.Files)
	}
	if _, err := os.Stat(filepath.Join(root, "configured-dist")); !os.IsNotExist(err) {
		t.Fatalf("canceled Bundle() wrote output: %v", err)
	}
}

func TestBundleRejectsOutputOverlappingProjectFiles(t *testing.T) {
	tests := []struct {
		name     string
		override string
	}{
		{name: "project root", override: "."},
		{name: "source directory", override: "src"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := optionsTestProject(t, true)
			configPath := filepath.Join(root, "puff.toml")
			sourcePath := filepath.Join(root, "src", "main.puff")

			result := compiler.Bundle(context.Background(), compiler.BundleOptions{
				StartDir: root,
				Output:   test.override,
			})

			optionsTestAssertErrorCode(t, result.Diagnostics, diagnostic.CodeCodegenError)
			if len(result.Output.Files) != 0 {
				t.Fatalf("Bundle() output = %#v, want empty output", result.Output.Files)
			}
			if _, err := os.Stat(configPath); err != nil {
				t.Fatalf("project config was modified: %v", err)
			}
			if _, err := os.Stat(sourcePath); err != nil {
				t.Fatalf("project source was modified: %v", err)
			}
		})
	}
}

func optionsTestProject(t *testing.T, withSource bool) string {
	t.Helper()

	root := t.TempDir()
	optionsTestWriteFile(t, filepath.Join(root, "puff.toml"), `[pack]
id = "example"
name = "Example Pack"

[minecraft]
versions = ">=1.21 <=1.21.6"
target = "1.21.6"

[build]
source = "src"
output = "configured-dist"
`)
	if withSource {
		optionsTestWriteFile(t, filepath.Join(root, "src", "main.puff"), "# tags: load\n\non load\nend\n")
	}

	return root
}

func optionsTestWriteFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("create parent directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func optionsTestAssertErrorCode(t *testing.T, result diagnostic.Result, want diagnostic.Code) {
	t.Helper()

	if result.OK {
		t.Fatalf("diagnostics = %#v, want failure", result)
	}
	if len(result.Errors) != 1 || result.Errors[0].Code != want {
		t.Fatalf("diagnostic errors = %#v, want one %s", result.Errors, want)
	}
}
