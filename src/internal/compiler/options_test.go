package compiler_test

import (
	"bytes"
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
		{name: "source descendant", override: filepath.Join("src", "generated")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := optionsTestProject(t, true)
			configPath := filepath.Join(root, "puff.toml")
			sourcePath := filepath.Join(root, "src", "main.puff")
			configBefore := optionsTestReadFile(t, configPath)
			sourceBefore := optionsTestReadFile(t, sourcePath)

			result := compiler.Bundle(context.Background(), compiler.BundleOptions{
				StartDir: root,
				Output:   test.override,
			})

			optionsTestAssertErrorCode(t, result.Diagnostics, diagnostic.CodeCodegenError)
			if len(result.Output.Files) != 0 {
				t.Fatalf("Bundle() output = %#v, want empty output", result.Output.Files)
			}
			if got := optionsTestReadFile(t, configPath); !bytes.Equal(got, configBefore) {
				t.Fatalf("project config = %q, want unchanged %q", got, configBefore)
			}
			if got := optionsTestReadFile(t, sourcePath); !bytes.Equal(got, sourceBefore) {
				t.Fatalf("project source = %q, want unchanged %q", got, sourceBefore)
			}
		})
	}
}

func TestBundleRejectsOutputContainingProjectRoot(t *testing.T) {
	sandbox := t.TempDir()
	root := filepath.Join(sandbox, "project")
	optionsTestPopulateProject(t, root, true)
	markerPath := filepath.Join(sandbox, "marker.txt")
	optionsTestWriteFile(t, markerPath, "keep me\n")
	markerBefore := optionsTestReadFile(t, markerPath)
	sourcePath := filepath.Join(root, "src", "main.puff")
	sourceBefore := optionsTestReadFile(t, sourcePath)

	result := compiler.Bundle(context.Background(), compiler.BundleOptions{
		StartDir: root,
		Output:   "..",
	})

	optionsTestAssertErrorCode(t, result.Diagnostics, diagnostic.CodeCodegenError)
	if got := optionsTestReadFile(t, markerPath); !bytes.Equal(got, markerBefore) {
		t.Fatalf("ancestor marker = %q, want unchanged %q", got, markerBefore)
	}
	if got := optionsTestReadFile(t, sourcePath); !bytes.Equal(got, sourceBefore) {
		t.Fatalf("project source = %q, want unchanged %q", got, sourceBefore)
	}
}

func TestBundleRejectsSymlinkOutputWithoutTouchingTarget(t *testing.T) {
	root := optionsTestProject(t, true)
	target := t.TempDir()
	markerPath := filepath.Join(target, "marker.txt")
	optionsTestWriteFile(t, markerPath, "keep me\n")
	link := filepath.Join(root, "linked-output")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("create output symlink: %v", err)
	}

	result := compiler.Bundle(context.Background(), compiler.BundleOptions{
		StartDir: root,
		Output:   link,
	})

	optionsTestAssertErrorCode(t, result.Diagnostics, diagnostic.CodeCodegenError)
	if got := string(optionsTestReadFile(t, markerPath)); got != "keep me\n" {
		t.Fatalf("symlink target marker = %q, want unchanged", got)
	}
}

func TestBundleReportsPublicationFailureWithoutReplacingFile(t *testing.T) {
	root := optionsTestProject(t, true)
	output := filepath.Join(root, "occupied")
	optionsTestWriteFile(t, output, "keep me\n")

	result := compiler.Bundle(context.Background(), compiler.BundleOptions{
		StartDir: root,
		Output:   output,
	})

	optionsTestAssertErrorCode(t, result.Diagnostics, diagnostic.CodeCodegenError)
	if len(result.Output.Files) != 0 {
		t.Fatalf("Bundle() output = %#v, want empty output", result.Output.Files)
	}
	if got := string(optionsTestReadFile(t, output)); got != "keep me\n" {
		t.Fatalf("occupied output = %q, want unchanged", got)
	}
}

func optionsTestProject(t *testing.T, withSource bool) string {
	t.Helper()

	root := t.TempDir()
	optionsTestPopulateProject(t, root, withSource)
	return root
}

func optionsTestPopulateProject(t *testing.T, root string, withSource bool) {
	t.Helper()

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

func optionsTestReadFile(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return data
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
