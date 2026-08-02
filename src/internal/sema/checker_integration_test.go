package sema

import (
	"path/filepath"
	"testing"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/lexer"
	"github.com/puff-lang/puff/internal/parser"
	"github.com/puff-lang/puff/internal/project"
	"github.com/puff-lang/puff/internal/source"
)

func TestCheckIntegrationAcceptsValidMultiModuleProject(t *testing.T) {
	resolved, checked := checkFixture(t, "valid")

	if checked.Project != resolved {
		t.Error("expected Check to preserve the resolved project")
	}
	if len(checked.Diagnostics) != 0 {
		t.Fatalf("expected no semantic diagnostics, got %#v", checked.Diagnostics)
	}

	main := requireResolvedModule(t, checked.Project, "main.puff")
	if imported, ok := main.Import("shop"); !ok || imported.Target.Source.RelPath != "lib/shop.puff" {
		t.Fatalf("expected shop to resolve to lib/shop.puff, got %#v", main.Imports)
	}
}

func TestCheckIntegrationReportsDocumentedDiagnosticsWithoutCascades(t *testing.T) {
	tests := []struct {
		name     string
		fixture  string
		expected []expectedSemanticDiagnostic
	}{
		{
			name:    "required metadata events",
			fixture: "required-events",
			expected: []expectedSemanticDiagnostic{
				{
					code:    diagnostic.CodeMissingLoadEvent,
					file:    "main.puff",
					line:    1,
					message: "Missing required event: on load",
					hint:    "Add an on load block or remove the load tag.",
				},
				{
					code:    diagnostic.CodeMissingTickEvent,
					file:    "main.puff",
					line:    1,
					message: "Missing required event: on tick",
					hint:    "Add an on tick block or remove the tick tag.",
				},
			},
		},
		{
			name:    "names types and calls",
			fixture: "names-types-calls",
			expected: []expectedSemanticDiagnostic{
				{
					code:    diagnostic.CodeUndefinedType,
					file:    "a_types.puff",
					line:    1,
					message: "Undefined type: MissingType",
				},
				{
					code:    diagnostic.CodeUndefinedVariable,
					file:    "b_names.puff",
					line:    4,
					message: "Undefined variable: $missing",
					hint:    "Declare it before using it: $missing = 0",
				},
				{
					code:    diagnostic.CodeUndefinedFunction,
					file:    "b_names.puff",
					line:    5,
					message: "Undefined function: missingFunction",
					hint:    "Declare fun missingFunction before using it, or import it from a module.",
				},
				{
					code:    diagnostic.CodeMissingArguments,
					file:    "b_names.puff",
					line:    6,
					message: "Missing arguments for function: add",
					hint:    "Call it with parentheses: add(a, b)",
				},
				{
					code:    diagnostic.CodeTooManyArguments,
					file:    "b_names.puff",
					line:    7,
					message: "Too many arguments.",
				},
				{
					code:    diagnostic.CodeInvalidArgumentType,
					file:    "b_names.puff",
					line:    8,
					message: "Invalid argument type.",
				},
				{
					code:    diagnostic.CodeUndefinedName,
					file:    "b_names.puff",
					line:    11,
					message: "Undefined name: player",
					hint:    "The name \"player\" is only available inside events that inject a player.",
				},
			},
		},
		{
			name:    "imported variable assignment",
			fixture: "imported-assignment",
			expected: []expectedSemanticDiagnostic{
				{
					code:    diagnostic.CodeAssignToImportedPublicVar,
					file:    "main.puff",
					line:    4,
					message: "Cannot assign to imported public variable: shop.$tax",
					hint:    "Use a public function like shop.setTax(0.2).",
				},
			},
		},
		{
			name:    "public local variable",
			fixture: "public-local",
			expected: []expectedSemanticDiagnostic{
				{
					code:    diagnostic.CodeInvalidPublicLocalVariable,
					file:    "main.puff",
					line:    1,
					message: "Local variables cannot be public.",
					hint:    "Only global variables can be exported.",
				},
			},
		},
		{
			name:    "return and stop distinctions",
			fixture: "returns",
			expected: []expectedSemanticDiagnostic{
				{
					code:    diagnostic.CodeInvalidReturnOutsideFunction,
					file:    "main.puff",
					line:    2,
					message: "return can only be used inside functions.",
					hint:    "Use stop to stop an event or execution block.",
				},
				{
					code:    diagnostic.CodeMissingReturnValue,
					file:    "main.puff",
					line:    14,
					message: "Missing return value.",
					hint:    "Return a value compatible with int.",
				},
				{
					code:    diagnostic.CodeMissingReturn,
					file:    "main.puff",
					line:    17,
					message: "Function missingPath must return int in all paths.",
					hint:    "Add an else branch or a final return.",
				},
				{
					code:    diagnostic.CodeInvalidStopInReturningFunc,
					file:    "main.puff",
					line:    24,
					message: "stop cannot replace a return value.",
					hint:    "Return a value compatible with int.",
				},
			},
		},
		{
			name:    "semantic regressions",
			fixture: "semantic-regressions",
			expected: []expectedSemanticDiagnostic{
				{
					code:    diagnostic.CodeUndefinedVariable,
					file:    "b_visibility.puff",
					line:    4,
					message: "Undefined variable: visibility.$config.secret",
					hint:    "Declare it before using it: visibility.$config.secret = 0",
				},
				{
					code:    diagnostic.CodeTypeMismatch,
					file:    "c_top_level_collection.puff",
					line:    1,
					message: "Type mismatch: cannot assign int to $players[].",
					hint:    "Convert one value or use compatible types.",
				},
				{
					code:    diagnostic.CodeTypeMismatch,
					file:    "a_import_order.puff",
					line:    4,
					message: "Type mismatch: cannot return int as string.",
					hint:    "Return a value compatible with string.",
				},
				{
					code:    diagnostic.CodeTypeMismatch,
					file:    "d_heterogeneous_collection.puff",
					line:    2,
					message: "Type mismatch: cannot return list<unknown> as list<int>.",
					hint:    "Return a value compatible with list<int>.",
				},
				{
					code:    diagnostic.CodeUndefinedVariable,
					file:    "e_branch_local.puff",
					line:    5,
					message: "Undefined variable: $_value",
					hint:    "Declare it before using it: $_value = 0",
				},
				{
					code:    diagnostic.CodeTypeMismatch,
					file:    "f_add_type.puff",
					line:    3,
					message: "Type mismatch: cannot add string to int.",
					hint:    "Convert one value or use compatible types.",
				},
			},
		},
		{
			name:    "final semantic edges",
			fixture: "final-edges",
			expected: []expectedSemanticDiagnostic{
				{
					code:    diagnostic.CodeUndefinedVariable,
					file:    "b_index_private_last_use.puff",
					line:    4,
					message: "Undefined variable: private_last.$stats",
					hint:    "Declare it before using it: private_last.$stats = 0",
				},
				{
					code:    diagnostic.CodeUndefinedVariable,
					file:    "d_index_public_last_use.puff",
					line:    4,
					message: "Undefined variable: public_last.$stats",
					hint:    "Declare it before using it: public_last.$stats = 0",
				},
				{
					code:    diagnostic.CodeTypeMismatch,
					file:    "f_cycle_copy.puff",
					line:    6,
					message: "Type mismatch: cannot return int as string.",
					hint:    "Return a value compatible with string.",
				},
				{
					code:    diagnostic.CodeTypeMismatch,
					file:    "h_add_scalar_list_target.puff",
					line:    3,
					message: "Type mismatch: cannot add int to int[].",
					hint:    "Convert one value or use compatible types.",
				},
				{
					code:    diagnostic.CodeTypeMismatch,
					file:    "i_add_merged_type.puff",
					line:    7,
					message: "Type mismatch: cannot add int to unknown.",
					hint:    "Convert one value or use compatible types.",
				},
			},
		},
		{
			name:    "closure semantic edges",
			fixture: "closure-edges",
			expected: []expectedSemanticDiagnostic{
				{
					code:    diagnostic.CodeUndefinedVariable,
					file:    "a_cycle_no_seed.puff",
					line:    3,
					message: "Undefined variable: cycle_b.$y",
					hint:    "Declare it before using it: cycle_b.$y = 0",
				},
				{
					code:    diagnostic.CodeUndefinedVariable,
					file:    "d_negative_private_last_use.puff",
					line:    4,
					message: "Undefined variable: private_last.$stats",
					hint:    "Declare it before using it: private_last.$stats = 0",
				},
				{
					code:    diagnostic.CodeUndefinedVariable,
					file:    "f_negative_public_last_use.puff",
					line:    4,
					message: "Undefined variable: public_last.$stats",
					hint:    "Declare it before using it: public_last.$stats = 0",
				},
				{
					code:    diagnostic.CodeTypeMismatch,
					file:    "g_nested_collection.puff",
					line:    2,
					message: "Type mismatch: cannot return list<list<int>> as list<list<string>>.",
					hint:    "Return a value compatible with list<list<string>>.",
				},
				{
					code:    diagnostic.CodeTypeMismatch,
					file:    "h_top_level_reassignment.puff",
					line:    6,
					message: "Type mismatch: cannot return string as int.",
					hint:    "Return a value compatible with int.",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, checked := checkFixture(t, test.fixture)
			assertSemanticDiagnostics(t, checked.Diagnostics, test.expected)
		})
	}
}

type expectedSemanticDiagnostic struct {
	code    diagnostic.Code
	file    string
	line    int
	message string
	hint    string
}

func checkFixture(t *testing.T, name string) (*Project, Result) {
	t.Helper()

	root := filepath.Join("testdata", "checker", name)
	config, err := project.LoadConfigFromDir(root)
	if err != nil {
		t.Fatalf("load fixture config: %v", err)
	}
	loaded, err := source.LoadProject(root, *config)
	if err != nil {
		t.Fatalf("load fixture sources: %v", err)
	}

	syntax := make(map[string]*ast.File, len(loaded.Files))
	for _, file := range loaded.Files {
		lexed := lexer.Lex(file)
		if len(lexed.Diagnostics) != 0 {
			t.Fatalf("lex %s: %#v", file.RelPath, lexed.Diagnostics)
		}
		parsed := parser.Parse(file, lexed)
		if len(parsed.Diagnostics) != 0 {
			t.Fatalf("parse %s: %#v", file.RelPath, parsed.Diagnostics)
		}
		syntax[file.RelPath] = parsed.File
	}

	resolved := Resolve(loaded, syntax)
	if len(resolved.Diagnostics) != 0 {
		t.Fatalf("resolve fixture %s: %#v", name, resolved.Diagnostics)
	}

	return resolved.Project, Check(resolved.Project)
}

func assertSemanticDiagnostics(
	t *testing.T,
	got []diagnostic.Diagnostic,
	expected []expectedSemanticDiagnostic,
) {
	t.Helper()

	if len(got) != len(expected) {
		t.Fatalf("expected %d diagnostics, got %#v", len(expected), got)
	}
	for index, want := range expected {
		actual := got[index]
		if actual.Code != want.code {
			t.Errorf("diagnostic %d: expected code %s, got %s", index, want.code, actual.Code)
		}
		if actual.Phase != diagnostic.PhaseSemantics {
			t.Errorf("diagnostic %d: expected semantics phase, got %s", index, actual.Phase)
		}
		if actual.Severity != diagnostic.SeverityError {
			t.Errorf("diagnostic %d: expected error severity, got %s", index, actual.Severity)
		}
		if actual.File != want.file {
			t.Errorf("diagnostic %d: expected file %q, got %q", index, want.file, actual.File)
		}
		if actual.Span.StartLine != want.line {
			t.Errorf("diagnostic %d: expected start line %d, got %#v", index, want.line, actual.Span)
		}
		if actual.Span.EndOffset <= actual.Span.StartOffset {
			t.Errorf("diagnostic %d: expected a non-empty span, got %#v", index, actual.Span)
		}
		if actual.Message != want.message {
			t.Errorf("diagnostic %d: expected message %q, got %q", index, want.message, actual.Message)
		}
		if actual.Hint != want.hint {
			t.Errorf("diagnostic %d: expected hint %q, got %q", index, want.hint, actual.Hint)
		}
	}
}
