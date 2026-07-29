package sema

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/lexer"
	"github.com/puff-lang/puff/internal/parser"
	"github.com/puff-lang/puff/internal/project"
	"github.com/puff-lang/puff/internal/source"
)

func TestResolveIntegrationSelectsLocalModules(t *testing.T) {
	tests := []struct {
		name       string
		fixture    string
		importPath string
		prefix     string
		targetPath string
	}{
		{name: "direct file", fixture: "direct-file", importPath: "abc/shop", prefix: "shop", targetPath: "abc/shop.puff"},
		{name: "directory module", fixture: "directory-module", importPath: "abc/shop", prefix: "shop", targetPath: "abc/shop/main.puff"},
		{name: "explicit alias", fixture: "alias", importPath: "abc/shop", prefix: "economy", targetPath: "abc/shop.puff"},
		{name: "alias repairs inferred prefix", fixture: "invalid-prefix-aliased", importPath: "github.com/123/123", prefix: "lib123", targetPath: "github.com/123/123.puff"},
		{name: "custom source directory", fixture: "custom-source", importPath: "abc/shop", prefix: "shop", targetPath: "abc/shop.puff"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			loaded, result := resolveFixture(t, test.fixture)
			if len(result.Diagnostics) != 0 {
				t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
			}
			assertModuleOrder(t, loaded, result.Project)

			main := requireResolvedModule(t, result.Project, "main.puff")
			resolved, ok := main.Import(test.prefix)
			if !ok {
				t.Fatalf("expected import prefix %q, got %#v", test.prefix, main.Imports)
			}
			if resolved.Path != test.importPath {
				t.Errorf("expected import path %q, got %q", test.importPath, resolved.Path)
			}
			if resolved.Prefix != test.prefix {
				t.Errorf("expected prefix %q, got %q", test.prefix, resolved.Prefix)
			}
			if resolved.Declaration == nil {
				t.Fatal("expected import declaration")
			}
			if resolved.Target == nil || resolved.Target.Source.RelPath != test.targetPath {
				t.Fatalf("expected target %q, got %#v", test.targetPath, resolved.Target)
			}
			indexedTarget := requireResolvedModule(t, result.Project, test.targetPath)
			if resolved.Target != indexedTarget {
				t.Error("expected import target to reference the indexed project module")
			}
		})
	}
}

func TestResolveIntegrationRejectsFailedImports(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		code    diagnostic.Code
		message string
		hint    string
	}{
		{
			name:    "ambiguous candidates",
			fixture: "ambiguous",
			code:    diagnostic.CodeAmbiguousImport,
			message: "Ambiguous import: both src/abc/shop.puff and src/abc/shop/main.puff exist.",
			hint:    "Remove one file or import a more specific path.",
		},
		{
			name:    "invalid inferred prefix",
			fixture: "invalid-prefix",
			code:    diagnostic.CodeInvalidImportPrefix,
			message: `Import path ends with "123", which is not a valid prefix.`,
			hint:    `Use an alias: require "github.com/123/123" as lib123`,
		},
		{
			name:    "missing candidates",
			fixture: "missing",
			code:    diagnostic.CodeImportNotFound,
			message: "Import not found: abc/shop",
			hint:    "Check the path or install the dependency.",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, result := resolveFixture(t, test.fixture)
			if len(result.Diagnostics) != 1 {
				t.Fatalf("expected one diagnostic, got %#v", result.Diagnostics)
			}

			main := requireResolvedModule(t, result.Project, "main.puff")
			if len(main.Imports) != 0 {
				t.Fatalf("expected no failed import bindings, got %#v", main.Imports)
			}
			assertImportDiagnostic(t, result.Diagnostics[0], test.code, test.message, test.hint, main)
		})
	}
}

func TestResolveIntegrationRejectsNonStaticAndEmptyPaths(t *testing.T) {
	_, result := resolveFixture(t, "invalid-import")
	if len(result.Diagnostics) != 2 {
		t.Fatalf("expected two diagnostics, got %#v", result.Diagnostics)
	}

	main := requireResolvedModule(t, result.Project, "main.puff")
	if len(main.Imports) != 0 {
		t.Fatalf("expected no invalid import bindings, got %#v", main.Imports)
	}
	for index, got := range result.Diagnostics {
		assertImportDiagnostic(t, got, diagnostic.CodeInvalidImport, "Invalid import path.", "", main)
		if got.Span != main.Syntax.Requirements[index].Path.Span() {
			t.Errorf("diagnostic %d: expected path span %#v, got %#v", index, main.Syntax.Requirements[index].Path.Span(), got.Span)
		}
	}
}

func TestResolveIntegrationKeepsImportedSymbolsNamespaced(t *testing.T) {
	_, result := resolveFixture(t, "prefix-required")
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", result.Diagnostics)
	}

	main := requireResolvedModule(t, result.Project, "main.puff")
	if len(main.Imports) != 1 {
		t.Fatalf("expected only the module prefix binding, got %#v", main.Imports)
	}
	if _, ok := main.Import("finalPrice"); ok {
		t.Error("imported function must not be injected into the importer")
	}
	if _, ok := main.Import("tax"); ok {
		t.Error("imported variable must not be injected into the importer")
	}

	callAssignment := requireGlobalAssignment(t, main.Syntax.Declarations[0])
	call, ok := callAssignment.Value.(*ast.CallExpr)
	if !ok {
		t.Fatalf("expected qualified call, got %T", callAssignment.Value)
	}
	gotCallee := make([]string, len(call.Callee.Parts))
	for index, part := range call.Callee.Parts {
		gotCallee[index] = part.Name
	}
	if want := []string{"shop", "finalPrice"}; !reflect.DeepEqual(gotCallee, want) {
		t.Errorf("expected qualified callee %v, got %v", want, gotCallee)
	}

	variableAssignment := requireGlobalAssignment(t, main.Syntax.Declarations[1])
	variable, ok := variableAssignment.Value.(*ast.VariableExpr)
	if !ok {
		t.Fatalf("expected qualified variable, got %T", variableAssignment.Value)
	}
	if variable.Qualifier == nil || variable.Qualifier.Name != "shop" || variable.Name.Name != "tax" {
		t.Errorf("expected shop.$tax, got %#v", variable)
	}

	imported, _ := main.Import("shop")
	if len(imported.Target.Syntax.Declarations) != 2 {
		t.Fatalf("expected imported declarations to remain on the target module, got %#v", imported.Target.Syntax.Declarations)
	}
}

func resolveFixture(t *testing.T, name string) (source.Project, Result) {
	t.Helper()

	root := filepath.Join("testdata", name)
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
		parsed := parser.Parse(file, lexer.Lex(file))
		if len(parsed.Diagnostics) != 0 {
			t.Fatalf("parse %s: %#v", file.RelPath, parsed.Diagnostics)
		}
		syntax[file.RelPath] = parsed.File
	}

	return loaded, Resolve(loaded, syntax)
}

func assertModuleOrder(t *testing.T, loaded source.Project, resolved *Project) {
	t.Helper()

	if resolved == nil {
		t.Fatal("expected resolved project")
	}
	if resolved.Root != loaded.Root {
		t.Errorf("expected root %q, got %q", loaded.Root, resolved.Root)
	}
	if len(resolved.Modules) != len(loaded.Files) {
		t.Fatalf("expected %d modules, got %d", len(loaded.Files), len(resolved.Modules))
	}
	for index, file := range loaded.Files {
		if resolved.Modules[index].Source.RelPath != file.RelPath {
			t.Errorf("module %d: expected %q, got %q", index, file.RelPath, resolved.Modules[index].Source.RelPath)
		}
	}
}

func requireResolvedModule(t *testing.T, resolved *Project, relPath string) *Module {
	t.Helper()

	module, ok := resolved.Module(relPath)
	if !ok {
		t.Fatalf("expected module %q", relPath)
	}
	return module
}

func assertImportDiagnostic(
	t *testing.T,
	got diagnostic.Diagnostic,
	code diagnostic.Code,
	message string,
	hint string,
	importer *Module,
) {
	t.Helper()

	if got.Code != code {
		t.Errorf("expected code %s, got %s", code, got.Code)
	}
	if got.Phase != diagnostic.PhaseSemantics {
		t.Errorf("expected semantics phase, got %s", got.Phase)
	}
	if got.Severity != diagnostic.SeverityError {
		t.Errorf("expected error severity, got %s", got.Severity)
	}
	if got.File != importer.Source.RelPath {
		t.Errorf("expected importer file %q, got %q", importer.Source.RelPath, got.File)
	}
	if got.Message != message {
		t.Errorf("expected message %q, got %q", message, got.Message)
	}
	if got.Hint != hint {
		t.Errorf("expected hint %q, got %q", hint, got.Hint)
	}
	if len(importer.Syntax.Requirements) == 1 && got.Span != importer.Syntax.Requirements[0].Path.Span() {
		t.Errorf("expected path span %#v, got %#v", importer.Syntax.Requirements[0].Path.Span(), got.Span)
	}
}

func requireGlobalAssignment(t *testing.T, declaration ast.Declaration) *ast.GlobalAssignment {
	t.Helper()

	assignment, ok := declaration.(*ast.GlobalAssignment)
	if !ok {
		t.Fatalf("expected global assignment, got %T", declaration)
	}
	return assignment
}
