package sema

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/source"
)

func Resolve(sourceProject source.Project, syntax map[string]*ast.File) Result {
	project := &Project{
		Root:    sourceProject.Root,
		Modules: make([]*Module, 0, len(sourceProject.Files)),
	}
	modules := make(map[string]*Module, len(sourceProject.Files))

	for _, file := range sourceProject.Files {
		module := &Module{
			Source:  file,
			Syntax:  syntax[file.RelPath],
			Imports: make(map[string]*Import),
		}
		project.Modules = append(project.Modules, module)
		modules[file.RelPath] = module
	}

	var diagnostics []diagnostic.Diagnostic
	for _, module := range project.Modules {
		if module.Syntax == nil {
			continue
		}

		for _, declaration := range module.Syntax.Requirements {
			importPath, ok := staticImportPath(declaration)
			if ok {
				importPath, ok = canonicalImportPath(importPath)
			}
			if !ok {
				diagnostics = append(diagnostics, importDiagnostic(
					module,
					declaration,
					diagnostic.CodeInvalidImport,
					"Invalid import path.",
					"",
				))
				continue
			}

			prefix := path.Base(importPath)
			if declaration.Alias != nil {
				prefix = declaration.Alias.Name
			} else if !validPrefix(prefix) {
				diagnostics = append(diagnostics, importDiagnostic(
					module,
					declaration,
					diagnostic.CodeInvalidImportPrefix,
					fmt.Sprintf("Import path ends with %q, which is not a valid prefix.", prefix),
					fmt.Sprintf("Use an alias: require %q as %s", importPath, suggestedAlias(prefix)),
				))
				continue
			}

			fileCandidate := importPath + ".puff"
			directoryCandidate := path.Join(importPath, "main.puff")
			fileModule, hasFile := modules[fileCandidate]
			directoryModule, hasDirectory := modules[directoryCandidate]

			if hasFile && hasDirectory {
				diagnostics = append(diagnostics, importDiagnostic(
					module,
					declaration,
					diagnostic.CodeAmbiguousImport,
					fmt.Sprintf(
						"Ambiguous import: both %s and %s exist.",
						displayPath(sourceProject.Root, fileModule.Source),
						displayPath(sourceProject.Root, directoryModule.Source),
					),
					"Remove one file or import a more specific path.",
				))
				continue
			}

			target := fileModule
			if !hasFile {
				target = directoryModule
			}
			if target == nil {
				diagnostics = append(diagnostics, importDiagnostic(
					module,
					declaration,
					diagnostic.CodeImportNotFound,
					fmt.Sprintf("Import not found: %s", importPath),
					"Check the path or install the dependency.",
				))
				continue
			}

			module.Imports[prefix] = &Import{
				Declaration: declaration,
				Path:        importPath,
				Prefix:      prefix,
				Target:      target,
			}
		}
	}

	return Result{
		Project:     project,
		Diagnostics: diagnostics,
	}
}

func staticImportPath(declaration *ast.RequireDecl) (string, bool) {
	if declaration == nil || declaration.Path == nil {
		return "", false
	}

	var builder strings.Builder
	for _, part := range declaration.Path.Parts {
		text, ok := part.(*ast.StringText)
		if !ok {
			return "", false
		}
		builder.WriteString(text.Value)
	}

	importPath := builder.String()
	return importPath, importPath != ""
}

func canonicalImportPath(importPath string) (string, bool) {
	canonical := path.Clean(importPath)
	if canonical != importPath ||
		path.IsAbs(canonical) ||
		windowsDriveAbsolute(canonical) ||
		canonical == "." ||
		canonical == ".." ||
		strings.HasPrefix(canonical, "../") ||
		strings.Contains(canonical, `\`) {
		return "", false
	}
	return canonical, true
}

func windowsDriveAbsolute(importPath string) bool {
	return len(importPath) >= 3 &&
		asciiLetter(importPath[0]) &&
		importPath[1] == ':' &&
		importPath[2] == '/'
}

func validPrefix(prefix string) bool {
	for index, char := range []byte(prefix) {
		if index == 0 {
			if !asciiLetter(char) {
				return false
			}
			continue
		}
		if !asciiLetter(char) && (char < '0' || char > '9') && char != '_' {
			return false
		}
	}
	return prefix != ""
}

func asciiLetter(char byte) bool {
	return char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z'
}

func suggestedAlias(segment string) string {
	var builder strings.Builder
	builder.WriteString("lib")
	for _, char := range []byte(segment) {
		if asciiLetter(char) || char >= '0' && char <= '9' || char == '_' {
			builder.WriteByte(char)
		}
	}
	return builder.String()
}

func importDiagnostic(
	module *Module,
	declaration *ast.RequireDecl,
	code diagnostic.Code,
	message string,
	hint string,
) diagnostic.Diagnostic {
	var span diagnostic.Span
	if declaration != nil {
		span = declaration.Span()
		if declaration.Path != nil {
			span = declaration.Path.Span()
		}
	}

	return diagnostic.Diagnostic{
		Code:     code,
		Phase:    diagnostic.PhaseSemantics,
		Severity: diagnostic.SeverityError,
		Message:  message,
		Hint:     hint,
		File:     module.Source.RelPath,
		Span:     span,
	}
}

func displayPath(root string, file source.File) string {
	if root != "" && file.Path != "" {
		relative, err := filepath.Rel(root, file.Path)
		if err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return filepath.ToSlash(relative)
		}
	}
	if root == "" && file.Path != "" && !filepath.IsAbs(file.Path) {
		return filepath.ToSlash(file.Path)
	}
	return file.RelPath
}
