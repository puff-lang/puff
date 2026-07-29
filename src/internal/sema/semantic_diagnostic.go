package sema

import (
	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
)

func semanticDiagnostic(
	module *Module,
	node ast.Node,
	code diagnostic.Code,
	message string,
	hint string,
) diagnostic.Diagnostic {
	var span diagnostic.Span
	if node != nil {
		span = node.Span()
	}

	file := ""
	if module != nil {
		file = module.Source.RelPath
	}

	return diagnostic.Diagnostic{
		Code:     code,
		Phase:    diagnostic.PhaseSemantics,
		Severity: diagnostic.SeverityError,
		Message:  message,
		Hint:     hint,
		File:     file,
		Span:     span,
	}
}
