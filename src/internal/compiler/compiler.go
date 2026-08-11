package compiler

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/codegen/minecraft"
	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/ir"
	"github.com/puff-lang/puff/internal/lexer"
	"github.com/puff-lang/puff/internal/lower"
	"github.com/puff-lang/puff/internal/parser"
	"github.com/puff-lang/puff/internal/project"
	"github.com/puff-lang/puff/internal/sema"
	"github.com/puff-lang/puff/internal/source"
)

type CheckOptions struct {
	StartDir string
}

type CheckResult struct {
	Diagnostics diagnostic.Result
	Files       []source.File
}

type BundleOptions struct {
	StartDir string
	Target   string
	Output   string
}

type BundleResult struct {
	Diagnostics diagnostic.Result
	Output      minecraft.Output
}

type analysis struct {
	config      *project.Config
	root        string
	files       []source.File
	program     *ir.Project
	diagnostics []diagnostic.Diagnostic
}

func Check(ctx context.Context, opts CheckOptions) CheckResult {
	result := analyze(ctx, opts.StartDir)

	return CheckResult{
		Diagnostics: diagnostic.FromDiagnostics(result.diagnostics...),
		Files:       result.files,
	}
}

func Bundle(ctx context.Context, opts BundleOptions) BundleResult {
	result := analyze(ctx, opts.StartDir)
	if hasErrors(result.diagnostics) {
		return bundleFailure(result.diagnostics)
	}

	config := *result.config
	if opts.Target != "" {
		config.Minecraft.Target = opts.Target
	}

	if issue := contextIssue(ctx); issue != nil {
		result.diagnostics = append(result.diagnostics, *issue)
		return bundleFailure(result.diagnostics)
	}

	generated := minecraft.Generate(result.program, config)
	result.diagnostics = append(result.diagnostics, generated.Diagnostics...)
	if hasErrors(result.diagnostics) {
		return bundleFailure(result.diagnostics)
	}

	if issue := contextIssue(ctx); issue != nil {
		result.diagnostics = append(result.diagnostics, *issue)
		return bundleFailure(result.diagnostics)
	}

	destination, err := outputDir(result.root, config, opts.Output)
	if err != nil {
		result.diagnostics = append(result.diagnostics, codegenFailure(err))
		return bundleFailure(result.diagnostics)
	}
	if err := minecraft.Write(generated.Output, destination); err != nil {
		result.diagnostics = append(result.diagnostics, codegenFailure(err))
		return bundleFailure(result.diagnostics)
	}

	return BundleResult{
		Diagnostics: diagnostic.FromDiagnostics(result.diagnostics...),
		Output:      generated.Output,
	}
}

func analyze(ctx context.Context, startDir string) analysis {
	var result analysis
	if issue := contextIssue(ctx); issue != nil {
		result.diagnostics = append(result.diagnostics, *issue)
		return result
	}

	if startDir == "" {
		startDir = "."
	}

	config, root, err := project.LoadNearestConfig(startDir)
	if err != nil {
		result.diagnostics = append(result.diagnostics, projectFailure(err))
		return result
	}
	result.config = config
	result.root = root

	sources, err := source.LoadProject(root, *config)
	if err != nil {
		result.diagnostics = append(result.diagnostics, projectFailure(err))
		return result
	}
	result.files = sources.Files

	syntax := make(map[string]*ast.File, len(sources.Files))
	for _, file := range sources.Files {
		if issue := contextIssue(ctx); issue != nil {
			result.diagnostics = append(result.diagnostics, *issue)
			return result
		}

		lexed := lexer.Lex(file)
		if containsCode(lexed.Diagnostics, diagnostic.CodeInvalidUTF8) {
			result.diagnostics = append(result.diagnostics, lexed.Diagnostics...)
			continue
		}

		parsed := parser.Parse(file, lexed)
		result.diagnostics = append(result.diagnostics, parsed.Diagnostics...)
		if !hasErrors(parsed.Diagnostics) {
			syntax[file.RelPath] = parsed.File
		}
	}
	if hasErrors(result.diagnostics) {
		return result
	}

	if issue := contextIssue(ctx); issue != nil {
		result.diagnostics = append(result.diagnostics, *issue)
		return result
	}
	resolved := sema.Resolve(sources, syntax)
	result.diagnostics = append(result.diagnostics, resolved.Diagnostics...)
	if hasErrors(result.diagnostics) {
		return result
	}

	if issue := contextIssue(ctx); issue != nil {
		result.diagnostics = append(result.diagnostics, *issue)
		return result
	}
	checked := sema.Check(resolved.Project)
	result.diagnostics = append(result.diagnostics, checked.Diagnostics...)
	if hasErrors(result.diagnostics) {
		return result
	}

	if issue := contextIssue(ctx); issue != nil {
		result.diagnostics = append(result.diagnostics, *issue)
		return result
	}
	lowered := lower.Lower(checked.Project)
	result.diagnostics = append(result.diagnostics, lowered.Diagnostics...)
	if hasErrors(result.diagnostics) {
		return result
	}
	result.program = lowered.Project

	return result
}

func hasErrors(diagnostics []diagnostic.Diagnostic) bool {
	for _, issue := range diagnostics {
		if issue.Severity == diagnostic.SeverityError {
			return true
		}
	}

	return false
}

func containsCode(diagnostics []diagnostic.Diagnostic, code diagnostic.Code) bool {
	for _, issue := range diagnostics {
		if issue.Code == code {
			return true
		}
	}

	return false
}

func outputDir(root string, config project.Config, override string) (string, error) {
	output := override
	if output == "" {
		output = config.Build.Output
	}
	if !filepath.IsAbs(output) {
		output = filepath.Join(root, output)
	}

	destination, err := canonicalPath(output)
	if err != nil {
		return "", fmt.Errorf("resolve output directory: %w", err)
	}
	projectRoot, err := canonicalPath(root)
	if err != nil {
		return "", fmt.Errorf("resolve project root: %w", err)
	}
	sourceDir := config.Build.Source
	if !filepath.IsAbs(sourceDir) {
		sourceDir = filepath.Join(root, sourceDir)
	}
	sourceDir, err = canonicalPath(sourceDir)
	if err != nil {
		return "", fmt.Errorf("resolve source directory: %w", err)
	}

	if containsPath(destination, projectRoot) {
		return "", errors.New("output directory cannot contain the project root")
	}
	if containsPath(destination, sourceDir) || containsPath(sourceDir, destination) {
		return "", errors.New("output directory cannot overlap the source directory")
	}

	return destination, nil
}

func canonicalPath(name string) (string, error) {
	absolute, err := filepath.Abs(name)
	if err != nil {
		return "", err
	}

	current := absolute
	var missing []string
	for {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			for index := len(missing) - 1; index >= 0; index-- {
				resolved = filepath.Join(resolved, missing[index])
			}
			return filepath.Clean(resolved), nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}

		parent := filepath.Dir(current)
		if parent == current {
			return filepath.Clean(absolute), nil
		}
		missing = append(missing, filepath.Base(current))
		current = parent
	}
}

func containsPath(parent string, child string) bool {
	relative, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}

	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func projectFailure(err error) diagnostic.Diagnostic {
	if errors.Is(err, project.ErrConfigNotFound) {
		return diagnostic.Diagnostic{
			Code:     diagnostic.CodeMissingPuffTOML,
			Phase:    diagnostic.PhaseProject,
			Severity: diagnostic.SeverityError,
			Message:  "Missing puff.toml.",
			Hint:     "Run puff init to create a new project.",
			Span:     defaultSpan(),
		}
	}

	return diagnostic.Diagnostic{
		Code:     diagnostic.CodeInvalidConfig,
		Phase:    diagnostic.PhaseProject,
		Severity: diagnostic.SeverityError,
		Message:  "Invalid puff.toml.",
		Hint:     err.Error(),
		File:     project.ConfigFileName,
		Span:     defaultSpan(),
	}
}

func codegenFailure(err error) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Code:     diagnostic.CodeCodegenError,
		Phase:    diagnostic.PhaseCodegen,
		Severity: diagnostic.SeverityError,
		Message:  "Failed to generate datapack.",
		Hint:     err.Error(),
		Span:     defaultSpan(),
	}
}

func contextIssue(ctx context.Context) *diagnostic.Diagnostic {
	if err := ctx.Err(); err != nil {
		return &diagnostic.Diagnostic{
			Code:     diagnostic.CodeCommandFailed,
			Phase:    diagnostic.PhaseCLI,
			Severity: diagnostic.SeverityError,
			Message:  "Command failed.",
			Hint:     err.Error(),
			Span:     defaultSpan(),
		}
	}

	return nil
}

func bundleFailure(diagnostics []diagnostic.Diagnostic) BundleResult {
	return BundleResult{Diagnostics: diagnostic.FromDiagnostics(diagnostics...)}
}

func defaultSpan() diagnostic.Span {
	return diagnostic.Span{
		StartLine:   1,
		StartColumn: 1,
		EndLine:     1,
		EndColumn:   1,
	}
}
