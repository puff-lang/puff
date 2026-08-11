package compiler_test

import (
	"context"
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

	result := compiler.Check(context.Background(), compiler.CheckOptions{StartDir: nested})
	if !result.Diagnostics.OK {
		t.Fatalf("Check() diagnostics = %#v, want success", result.Diagnostics)
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
		name   string
		source string
		code   diagnostic.Code
		phase  diagnostic.Phase
	}{
		{
			name:   "lexer",
			source: "$coins = 100 @ 20\n",
			code:   diagnostic.CodeInvalidCharacter,
			phase:  diagnostic.PhaseLexer,
		},
		{
			name:   "invalid UTF-8",
			source: string([]byte{0xff}),
			code:   diagnostic.CodeInvalidUTF8,
			phase:  diagnostic.PhaseLexer,
		},
		{
			name: "parser",
			source: `on load
   send "Loaded" to player
`,
			code:  diagnostic.CodeExpectedEnd,
			phase: diagnostic.PhaseParser,
		},
		{
			name: "semantics",
			source: `# tags: load

on tick
end
`,
			code:  diagnostic.CodeMissingLoadEvent,
			phase: diagnostic.PhaseSemantics,
		},
		{
			name: "pattern",
			source: `on load
   explode player
end
`,
			code:  diagnostic.CodeUnknownEffectPattern,
			phase: diagnostic.PhasePattern,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := checkTestWriteProject(t, map[string]string{"src/main.puff": test.source})
			result := compiler.Check(context.Background(), compiler.CheckOptions{StartDir: root})
			if result.Diagnostics.OK {
				t.Fatalf("Check() succeeded, want %s diagnostic", test.code)
			}
			checkTestRequireDiagnostic(t, result.Diagnostics.Errors, test.code, test.phase)

			output := filepath.Join(root, "build", "datapack")
			if _, err := os.Stat(output); !os.IsNotExist(err) {
				t.Fatalf("Check() created output at %q; stat error = %v", output, err)
			}
		})
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

func checkTestRequireDiagnostic(
	t *testing.T,
	diagnostics []diagnostic.Diagnostic,
	code diagnostic.Code,
	phase diagnostic.Phase,
) {
	t.Helper()

	for _, issue := range diagnostics {
		if issue.Code == code && issue.Phase == phase {
			return
		}
	}
	t.Fatalf("diagnostics = %#v, want code %q in phase %q", diagnostics, code, phase)
}
