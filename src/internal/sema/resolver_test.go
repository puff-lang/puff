package sema

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/source"
)

func TestResolveLocalModuleCandidates(t *testing.T) {
	tests := []struct {
		name       string
		targetPath string
	}{
		{name: "direct file", targetPath: "abc/shop.puff"},
		{name: "directory main", targetPath: "abc/shop/main.puff"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requirement := staticRequire("abc/shop", "")
			input := testSourceProject("main.puff", test.targetPath)
			syntax := testSyntax(input, map[string][]*ast.RequireDecl{
				"main.puff": {requirement},
			})

			result := Resolve(input, syntax)

			assertNoDiagnostics(t, result.Diagnostics)
			importer := requireTestModule(t, result.Project, "main.puff")
			target := requireTestModule(t, result.Project, test.targetPath)
			resolved, ok := importer.Import("shop")
			if !ok {
				t.Fatal("expected import to be bound under default prefix shop")
			}
			if resolved.Declaration != requirement || resolved.Path != "abc/shop" ||
				resolved.Prefix != "shop" || resolved.Target != target {
				t.Fatalf("unexpected resolved import: %#v", resolved)
			}
			if len(importer.Imports) != 1 {
				t.Fatalf("expected one prefix binding, got %#v", importer.Imports)
			}
		})
	}
}

func TestResolveReportsAmbiguousAndMissingImports(t *testing.T) {
	tests := []struct {
		name    string
		files   []string
		code    diagnostic.Code
		message string
		hint    string
	}{
		{
			name:    "ambiguous",
			files:   []string{"main.puff", "abc/shop.puff", "abc/shop/main.puff"},
			code:    diagnostic.CodeAmbiguousImport,
			message: "Ambiguous import: both src/abc/shop.puff and src/abc/shop/main.puff exist.",
			hint:    "Remove one file or import a more specific path.",
		},
		{
			name:    "missing",
			files:   []string{"main.puff"},
			code:    diagnostic.CodeImportNotFound,
			message: "Import not found: abc/shop",
			hint:    "Check the path or install the dependency.",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requirement := staticRequire("abc/shop", "")
			input := testSourceProject(test.files...)
			syntax := testSyntax(input, map[string][]*ast.RequireDecl{
				"main.puff": {requirement},
			})

			result := Resolve(input, syntax)

			assertSingleDiagnostic(t, result.Diagnostics, diagnostic.Diagnostic{
				Code:     test.code,
				Phase:    diagnostic.PhaseSemantics,
				Severity: diagnostic.SeverityError,
				Message:  test.message,
				Hint:     test.hint,
				File:     "main.puff",
				Span:     requirement.Path.Span(),
			})
			importer := requireTestModule(t, result.Project, "main.puff")
			if len(importer.Imports) != 0 {
				t.Fatalf("failed import must not create a binding: %#v", importer.Imports)
			}
		})
	}
}

func TestResolveImportPrefixesAndAliases(t *testing.T) {
	t.Run("explicit alias overrides default prefix", func(t *testing.T) {
		requirement := staticRequire("abc/shop", "economy")
		input := testSourceProject("main.puff", "abc/shop.puff")
		result := Resolve(input, testSyntax(input, map[string][]*ast.RequireDecl{
			"main.puff": {requirement},
		}))

		assertNoDiagnostics(t, result.Diagnostics)
		importer := requireTestModule(t, result.Project, "main.puff")
		resolved, ok := importer.Import("economy")
		if !ok || resolved.Prefix != "economy" {
			t.Fatalf("expected economy alias, got %#v", resolved)
		}
		if _, ok := importer.Import("shop"); ok {
			t.Fatal("default prefix must not remain bound when an alias is present")
		}
		if len(importer.Imports) != 1 {
			t.Fatalf("expected only the alias binding, got %#v", importer.Imports)
		}
	})

	t.Run("invalid inferred prefix", func(t *testing.T) {
		requirement := staticRequire("abc/123", "")
		input := testSourceProject("main.puff", "abc/123.puff")
		result := Resolve(input, testSyntax(input, map[string][]*ast.RequireDecl{
			"main.puff": {requirement},
		}))

		assertSingleDiagnostic(t, result.Diagnostics, diagnostic.Diagnostic{
			Code:     diagnostic.CodeInvalidImportPrefix,
			Phase:    diagnostic.PhaseSemantics,
			Severity: diagnostic.SeverityError,
			Message:  `Import path ends with "123", which is not a valid prefix.`,
			Hint:     `Use an alias: require "abc/123" as lib123`,
			File:     "main.puff",
			Span:     requirement.Path.Span(),
		})
		importer := requireTestModule(t, result.Project, "main.puff")
		if len(importer.Imports) != 0 {
			t.Fatalf("invalid prefix must not create a binding: %#v", importer.Imports)
		}
	})

	t.Run("alias repairs numeric final segment", func(t *testing.T) {
		requirement := staticRequire("abc/123", "lib123")
		input := testSourceProject("main.puff", "abc/123.puff")
		result := Resolve(input, testSyntax(input, map[string][]*ast.RequireDecl{
			"main.puff": {requirement},
		}))

		assertNoDiagnostics(t, result.Diagnostics)
		importer := requireTestModule(t, result.Project, "main.puff")
		resolved, ok := importer.Import("lib123")
		if !ok || resolved.Path != "abc/123" || resolved.Prefix != "lib123" {
			t.Fatalf("expected repaired prefix binding, got %#v", resolved)
		}
		if _, ok := importer.Import("123"); ok {
			t.Fatal("numeric default prefix must not be bound")
		}
	})
}

func TestResolveRejectsInvalidImportPaths(t *testing.T) {
	tests := []struct {
		name        string
		requirement *ast.RequireDecl
	}{
		{name: "empty", requirement: staticRequire("", "")},
		{name: "interpolated", requirement: interpolatedRequire()},
		{name: "non-canonical", requirement: staticRequire("./abc/shop", "")},
		{name: "escaping source root", requirement: staticRequire("../shop", "")},
		{name: "absolute", requirement: staticRequire("/abc/shop", "")},
		{name: "windows drive absolute", requirement: staticRequire("C:/abc/shop", "")},
		{name: "backslash", requirement: staticRequire(`abc\shop`, "")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := testSourceProject("main.puff")
			result := Resolve(input, testSyntax(input, map[string][]*ast.RequireDecl{
				"main.puff": {test.requirement},
			}))

			assertSingleDiagnostic(t, result.Diagnostics, diagnostic.Diagnostic{
				Code:     diagnostic.CodeInvalidImport,
				Phase:    diagnostic.PhaseSemantics,
				Severity: diagnostic.SeverityError,
				Message:  "Invalid import path.",
				File:     "main.puff",
				Span:     test.requirement.Path.Span(),
			})
			importer := requireTestModule(t, result.Project, "main.puff")
			if len(importer.Imports) != 0 {
				t.Fatalf("invalid path must not create a binding: %#v", importer.Imports)
			}
		})
	}
}

func TestResolveFormatsAmbiguityFromRelativeProjectRoot(t *testing.T) {
	requirement := staticRequire("abc/shop", "")
	input := testSourceProject("main.puff", "abc/shop.puff", "abc/shop/main.puff")
	input.Root = ""
	for index := range input.Files {
		input.Files[index].Path = filepath.FromSlash("src/" + input.Files[index].RelPath)
	}

	result := Resolve(input, testSyntax(input, map[string][]*ast.RequireDecl{
		"main.puff": {requirement},
	}))

	assertSingleDiagnostic(t, result.Diagnostics, diagnostic.Diagnostic{
		Code:     diagnostic.CodeAmbiguousImport,
		Phase:    diagnostic.PhaseSemantics,
		Severity: diagnostic.SeverityError,
		Message:  "Ambiguous import: both src/abc/shop.puff and src/abc/shop/main.puff exist.",
		Hint:     "Remove one file or import a more specific path.",
		File:     "main.puff",
		Span:     requirement.Path.Span(),
	})
}

func TestResolvePreservesFileOrderAndModuleLookup(t *testing.T) {
	input := testSourceProject("z.puff", "nested/main.puff", "a.puff")
	result := Resolve(input, testSyntax(input, nil))

	assertNoDiagnostics(t, result.Diagnostics)
	got := make([]string, 0, len(result.Project.Modules))
	for _, module := range result.Project.Modules {
		got = append(got, module.Source.RelPath)
	}
	want := []string{"z.puff", "nested/main.puff", "a.puff"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("module order changed: got %v, want %v", got, want)
	}
	if result.Project.Root != input.Root {
		t.Fatalf("project root changed: got %q, want %q", result.Project.Root, input.Root)
	}
	for _, relPath := range want {
		module, ok := result.Project.Module(relPath)
		if !ok || module.Source.RelPath != relPath {
			t.Fatalf("module lookup failed for %q: %#v", relPath, module)
		}
	}
	if module, ok := result.Project.Module("missing.puff"); ok || module != nil {
		t.Fatalf("missing module lookup returned %#v, %v", module, ok)
	}
}

func TestResolveKeepsImportedSymbolsNamespaced(t *testing.T) {
	requirement := staticRequire("abc/shop", "")
	input := testSourceProject("main.puff", "abc/shop.puff")
	syntax := testSyntax(input, map[string][]*ast.RequireDecl{
		"main.puff": {requirement},
	})
	syntax["abc/shop.puff"].Declarations = []ast.Declaration{
		&ast.FunctionDecl{Public: true, Name: ast.Identifier{Name: "finalPrice"}},
		&ast.GlobalAssignment{
			Public: true,
			Target: &ast.VariableExpr{Name: ast.Identifier{Name: "tax"}},
		},
	}

	result := Resolve(input, syntax)

	assertNoDiagnostics(t, result.Diagnostics)
	importer := requireTestModule(t, result.Project, "main.puff")
	if len(importer.Imports) != 1 {
		t.Fatalf("expected one namespace binding, got %#v", importer.Imports)
	}
	if _, ok := importer.Import("shop"); !ok {
		t.Fatal("expected imported symbols to be reachable through shop")
	}
	if _, ok := importer.Import("finalPrice"); ok {
		t.Fatal("imported function must not be injected as an unqualified binding")
	}
	if _, ok := importer.Import("tax"); ok {
		t.Fatal("imported variable must not be injected as an unqualified binding")
	}
	if len(importer.Syntax.Declarations) != 0 {
		t.Fatalf("resolver injected declarations into importer syntax: %#v", importer.Syntax.Declarations)
	}
}

func testSourceProject(relPaths ...string) source.Project {
	root := filepath.Join(string(filepath.Separator), "project")
	files := make([]source.File, 0, len(relPaths))
	for _, relPath := range relPaths {
		files = append(files, source.NewFile(
			filepath.Join(root, "src", filepath.FromSlash(relPath)),
			relPath,
			"",
		))
	}
	return source.Project{Root: root, Files: files}
}

func testSyntax(project source.Project, requirements map[string][]*ast.RequireDecl) map[string]*ast.File {
	syntax := make(map[string]*ast.File, len(project.Files))
	for _, file := range project.Files {
		syntax[file.RelPath] = &ast.File{
			Requirements: requirements[file.RelPath],
		}
	}
	return syntax
}

func staticRequire(importPath string, alias string) *ast.RequireDecl {
	pathSpan := diagnostic.Span{
		StartLine:   2,
		StartColumn: 9,
		EndLine:     2,
		EndColumn:   9 + len(importPath) + 2,
		StartOffset: 20,
		EndOffset:   20 + len(importPath) + 2,
	}
	path := &ast.StringExpr{
		NodeBase: ast.NodeBase{SourceSpan: pathSpan},
		Quote:    '"',
		Parts: []ast.StringPart{
			&ast.StringText{
				NodeBase: ast.NodeBase{SourceSpan: pathSpan},
				Raw:      importPath,
				Value:    importPath,
			},
		},
	}
	requirement := &ast.RequireDecl{
		NodeBase: ast.NodeBase{SourceSpan: diagnostic.Span{
			StartLine:   2,
			StartColumn: 1,
			EndLine:     2,
			EndColumn:   pathSpan.EndColumn,
			StartOffset: 12,
			EndOffset:   pathSpan.EndOffset,
		}},
		Path: path,
	}
	if alias != "" {
		requirement.Alias = &ast.Identifier{
			NodeBase: ast.NodeBase{SourceSpan: diagnostic.Span{
				StartLine:   2,
				StartColumn: pathSpan.EndColumn + 4,
				EndLine:     2,
				EndColumn:   pathSpan.EndColumn + 4 + len(alias),
				StartOffset: pathSpan.EndOffset + 4,
				EndOffset:   pathSpan.EndOffset + 4 + len(alias),
			}},
			Name: alias,
		}
	}
	return requirement
}

func interpolatedRequire() *ast.RequireDecl {
	requirement := staticRequire("abc/1", "")
	requirement.Path.Parts = []ast.StringPart{
		&ast.StringText{Raw: "abc/", Value: "abc/"},
		&ast.StringInterpolation{
			Expression: &ast.IntLiteral{Value: 1},
		},
	}
	return requirement
}

func requireTestModule(t *testing.T, project *Project, relPath string) *Module {
	t.Helper()
	if project == nil {
		t.Fatal("expected resolved project")
	}
	module, ok := project.Module(relPath)
	if !ok {
		t.Fatalf("expected module %q", relPath)
	}
	return module
}

func assertNoDiagnostics(t *testing.T, diagnostics []diagnostic.Diagnostic) {
	t.Helper()
	if len(diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %#v", diagnostics)
	}
}

func assertSingleDiagnostic(
	t *testing.T,
	diagnostics []diagnostic.Diagnostic,
	want diagnostic.Diagnostic,
) {
	t.Helper()
	if len(diagnostics) != 1 {
		t.Fatalf("expected one diagnostic, got %#v", diagnostics)
	}
	if !reflect.DeepEqual(diagnostics[0], want) {
		t.Fatalf("unexpected diagnostic:\ngot  %#v\nwant %#v", diagnostics[0], want)
	}
}
