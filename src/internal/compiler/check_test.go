package compiler_test

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/puff-lang/puff/internal/compiler"
	"github.com/puff-lang/puff/internal/diagnostic"
)

const checkTestConfig = `[pack]
id = "compiler-check"
name = "Compiler Check"

[minecraft]
versions = ">=1.21 <=1.21.6"
target = "1.21.6"

[build]
source = "src"
output = "build/datapack"
`

func TestCheckAcceptsValidMultiModuleProjectFromNestedDirectory(t *testing.T) {
	root := checkTestWriteProject(t, map[string]string{
		"src/main.puff": `# tags: load
require "lib/shop"

on load
end
`,
		"src/lib/shop.puff": `pub fun price -> int
   return 5
end
`,
	})
	nested := filepath.Join(root, "nested", "working", "directory")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("create nested start directory: %v", err)
	}

	before := checkTestTree(t, root)
	result := compiler.Check(context.Background(), compiler.CheckOptions{StartDir: nested})
	if !result.Diagnostics.OK {
		t.Fatalf("Check() diagnostics = %#v, want success", result.Diagnostics)
	}
	if after := checkTestTree(t, root); !reflect.DeepEqual(after, before) {
		t.Fatalf("project tree after Check() = %#v, want unchanged %#v", after, before)
	}

	wantPaths := []string{"lib/shop.puff", "main.puff"}
	if got := checkTestRelPaths(result); !reflect.DeepEqual(got, wantPaths) {
		t.Errorf("Check() files = %#v, want %#v", got, wantPaths)
	}

	second := compiler.Check(context.Background(), compiler.CheckOptions{StartDir: nested})
	if got := checkTestRelPaths(second); !reflect.DeepEqual(got, wantPaths) {
		t.Errorf("second Check() files = %#v, want deterministic order %#v", got, wantPaths)
	}

	output := filepath.Join(root, "build", "datapack")
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("Check() created output at %q; stat error = %v", output, err)
	}
}

func TestCheckReportsMissingProjectConfig(t *testing.T) {
	result := compiler.Check(context.Background(), compiler.CheckOptions{StartDir: t.TempDir()})
	if result.Diagnostics.OK {
		t.Fatal("Check() succeeded without puff.toml")
	}
	if len(result.Diagnostics.Errors) != 1 {
		t.Fatalf("Check() errors = %#v, want exactly one", result.Diagnostics.Errors)
	}

	issue := result.Diagnostics.Errors[0]
	if issue.Code != diagnostic.CodeMissingPuffTOML {
		t.Errorf("diagnostic code = %q, want %q", issue.Code, diagnostic.CodeMissingPuffTOML)
	}
	if issue.Phase != diagnostic.PhaseProject {
		t.Errorf("diagnostic phase = %q, want %q", issue.Phase, diagnostic.PhaseProject)
	}
	if issue.Message != "Missing puff.toml." {
		t.Errorf("diagnostic message = %q, want %q", issue.Message, "Missing puff.toml.")
	}
	if issue.Hint != "Run puff init to create a new project." {
		t.Errorf("diagnostic hint = %q, want %q", issue.Hint, "Run puff init to create a new project.")
	}
}

func TestCheckReportsDiagnosticsAcrossAnalysisPhases(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		errorCount int
		want       diagnostic.Diagnostic
		additional []diagnostic.Diagnostic
	}{
		{
			name:       "lexer",
			source:     "$coins = 100 @ 20\n",
			errorCount: 2,
			want: checkTestDiagnostic(
				diagnostic.CodeInvalidCharacter,
				diagnostic.PhaseLexer,
				"Invalid character: @",
				"Remove the character or replace it with valid Puff syntax.",
				diagnostic.Span{StartLine: 1, StartColumn: 14, EndLine: 1, EndColumn: 15, StartOffset: 13, EndOffset: 14},
			),
			additional: []diagnostic.Diagnostic{checkTestDiagnostic(
				diagnostic.CodeExpectedNewline,
				diagnostic.PhaseParser,
				"Expected newline.",
				"",
				diagnostic.Span{StartLine: 1, StartColumn: 16, EndLine: 1, EndColumn: 18, StartOffset: 15, EndOffset: 17},
			)},
		},
		{
			name:       "invalid UTF-8",
			source:     string([]byte{0xff}),
			errorCount: 1,
			want: checkTestDiagnostic(
				diagnostic.CodeInvalidUTF8,
				diagnostic.PhaseLexer,
				"File is not valid UTF-8.",
				"Save the file as UTF-8.",
				diagnostic.Span{StartLine: 1, StartColumn: 1, EndLine: 1, EndColumn: 2, StartOffset: 0, EndOffset: 1},
			),
		},
		{
			name: "parser",
			source: `on load
   send "Loaded" to player
`,
			errorCount: 1,
			want: checkTestDiagnostic(
				diagnostic.CodeExpectedEnd,
				diagnostic.PhaseParser,
				`Expected "end" before end of file.`,
				"Add end to close the block.",
				diagnostic.Span{StartLine: 3, StartColumn: 1, EndLine: 3, EndColumn: 1, StartOffset: 35, EndOffset: 35},
			),
		},
		{
			name: "semantics",
			source: `# tags: load

on tick
end
`,
			errorCount: 1,
			want: checkTestDiagnostic(
				diagnostic.CodeMissingLoadEvent,
				diagnostic.PhaseSemantics,
				"Missing required event: on load",
				"Add an on load block or remove the load tag.",
				diagnostic.Span{StartLine: 1, StartColumn: 1, EndLine: 1, EndColumn: 13, StartOffset: 0, EndOffset: 12},
			),
		},
		{
			name: "pattern",
			source: `on load
   explode player
end
`,
			errorCount: 1,
			want: checkTestDiagnostic(
				diagnostic.CodeUnknownEffectPattern,
				diagnostic.PhasePattern,
				"Unknown effect pattern.",
				"Check the syntax or require a library that registers this effect.",
				diagnostic.Span{StartLine: 2, StartColumn: 4, EndLine: 2, EndColumn: 18, StartOffset: 11, EndOffset: 25},
			),
		},
		{
			name: "resolver",
			source: `require "missing"

on load
end
`,
			errorCount: 1,
			want: checkTestDiagnostic(
				diagnostic.CodeImportNotFound,
				diagnostic.PhaseSemantics,
				"Import not found: missing",
				"Check the path or install the dependency.",
				diagnostic.Span{StartLine: 1, StartColumn: 9, EndLine: 1, EndColumn: 18, StartOffset: 8, EndOffset: 17},
			),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := checkTestWriteProject(t, map[string]string{"src/main.puff": test.source})
			result := compiler.Check(context.Background(), compiler.CheckOptions{StartDir: root})
			if result.Diagnostics.OK {
				t.Fatalf("Check() succeeded, want %s diagnostic", test.want.Code)
			}
			checkTestRequireDiagnostics(t, result.Diagnostics, test.errorCount, test.want, test.additional)

			output := filepath.Join(root, "build", "datapack")
			if _, err := os.Stat(output); !os.IsNotExist(err) {
				t.Fatalf("Check() created output at %q; stat error = %v", output, err)
			}
		})
	}
}

func TestCheckDefaultsStartDirToWorkingDirectory(t *testing.T) {
	root := checkTestWriteProject(t, map[string]string{
		"src/main.puff": "on load\nend\n",
	})
	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("change working directory: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})

	result := compiler.Check(context.Background(), compiler.CheckOptions{})
	if !result.Diagnostics.OK {
		t.Fatalf("Check() diagnostics = %#v, want success", result.Diagnostics)
	}
}

func TestCheckStopsBeforeCodegen(t *testing.T) {
	root := checkTestWriteProject(t, map[string]string{
		"src/main.puff": "# tags: load\n\non load\n   send \"server only\" to console\nend\n",
	})

	result := compiler.Check(context.Background(), compiler.CheckOptions{StartDir: root})

	if !result.Diagnostics.OK {
		t.Fatalf("Check() diagnostics = %#v, want success before codegen", result.Diagnostics)
	}
	output := filepath.Join(root, "build", "datapack")
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("Check() created output at %q; stat error = %v", output, err)
	}
}

func checkTestWriteProject(t *testing.T, files map[string]string) string {
	t.Helper()

	root := t.TempDir()
	checkTestWriteFile(t, root, "puff.toml", checkTestConfig)
	for path, content := range files {
		checkTestWriteFile(t, root, path, content)
	}
	return root
}

func checkTestWriteFile(t *testing.T, root string, path string, content string) {
	t.Helper()

	absolute := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(absolute), 0o755); err != nil {
		t.Fatalf("create parent for %q: %v", path, err)
	}
	if err := os.WriteFile(absolute, []byte(content), 0o644); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}

func checkTestRelPaths(result compiler.CheckResult) []string {
	paths := make([]string, len(result.Files))
	for index, file := range result.Files {
		paths[index] = file.RelPath
	}
	return paths
}

func checkTestTree(t *testing.T, root string) map[string]string {
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
		t.Fatalf("read project tree: %v", err)
	}
	return files
}

func checkTestRequireDiagnostics(t *testing.T, result diagnostic.Result, count int, want diagnostic.Diagnostic, additional []diagnostic.Diagnostic) {
	t.Helper()

	wantErrors := append([]diagnostic.Diagnostic{want}, additional...)
	if len(wantErrors) != count {
		t.Fatalf("invalid test expectation: %d diagnostics, want count %d", len(wantErrors), count)
	}
	if !reflect.DeepEqual(result.Errors, wantErrors) {
		t.Fatalf("diagnostic errors = %#v, want %#v", result.Errors, wantErrors)
	}
	if len(result.Warnings) != 0 {
		t.Fatalf("diagnostic warnings = %#v, want none", result.Warnings)
	}
}

func checkTestDiagnostic(code diagnostic.Code, phase diagnostic.Phase, message string, hint string, span diagnostic.Span) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Code:     code,
		Phase:    phase,
		Severity: diagnostic.SeverityError,
		Message:  message,
		Hint:     hint,
		File:     "main.puff",
		Span:     span,
	}
}
