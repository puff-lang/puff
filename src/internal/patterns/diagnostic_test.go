package patterns

import (
	"testing"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/token"
)

func TestUnknownPatternDiagnostics(t *testing.T) {
	const file = "packs/demo/main.puff"

	effectSpan := diagnostic.Span{StartLine: 2, StartColumn: 3, EndLine: 2, EndColumn: 10, StartOffset: 12, EndOffset: 19}
	expressionSpan := diagnostic.Span{StartLine: 4, StartColumn: 6, EndLine: 4, EndColumn: 13, StartOffset: 31, EndOffset: 38}
	conditionSpan := diagnostic.Span{StartLine: 6, StartColumn: 4, EndLine: 6, EndColumn: 11, StartOffset: 52, EndOffset: 59}
	eventNameSpan := diagnostic.Span{StartLine: 8, StartColumn: 4, EndLine: 8, EndColumn: 16, StartOffset: 74, EndOffset: 86}

	effect := &ast.EffectStmt{
		NodeBase: ast.NodeBase{SourceSpan: effectSpan},
		Tokens:   []token.Token{manualPatternToken(token.Ident, "unknown", effectSpan)},
	}
	expression := &ast.PatternExpr{
		NodeBase: ast.NodeBase{SourceSpan: expressionSpan},
		Tokens:   []token.Token{manualPatternToken(token.Ident, "mystery", expressionSpan)},
	}
	condition := &ast.PatternExpr{
		NodeBase: ast.NodeBase{SourceSpan: conditionSpan},
		Tokens:   []token.Token{manualPatternToken(token.Ident, "unclear", conditionSpan)},
	}
	event := &ast.EventDecl{
		NodeBase: ast.NodeBase{SourceSpan: diagnostic.Span{
			StartLine: 8, StartColumn: 1, EndLine: 10, EndColumn: 4, StartOffset: 71, EndOffset: 104,
		}},
		Name: []ast.Identifier{
			{
				NodeBase: ast.NodeBase{SourceSpan: diagnostic.Span{
					StartLine: 8, StartColumn: 4, EndLine: 8, EndColumn: 10, StartOffset: 74, EndOffset: 80,
				}},
				Name: "custom",
			},
			{
				NodeBase: ast.NodeBase{SourceSpan: diagnostic.Span{
					StartLine: 8, StartColumn: 11, EndLine: 8, EndColumn: 16, StartOffset: 81, EndOffset: 86,
				}},
				Name: "event",
			},
		},
	}

	tests := []struct {
		name    string
		resolve func(Registry) (bool, *diagnostic.Diagnostic)
		want    diagnostic.Diagnostic
	}{
		{
			name: "effect",
			resolve: func(registry Registry) (bool, *diagnostic.Diagnostic) {
				resolved, issue := registry.ResolveEffect(file, effect)
				return resolved == nil, issue
			},
			want: diagnostic.Diagnostic{
				Code:     diagnostic.CodeUnknownEffectPattern,
				Phase:    diagnostic.PhasePattern,
				Severity: diagnostic.SeverityError,
				Message:  "Unknown effect pattern.",
				Hint:     "Check the syntax or require a library that registers this effect.",
				File:     file,
				Span:     effectSpan,
			},
		},
		{
			name: "expression",
			resolve: func(registry Registry) (bool, *diagnostic.Diagnostic) {
				resolved, issue := registry.ResolveExpression(file, expression)
				return resolved == nil, issue
			},
			want: diagnostic.Diagnostic{
				Code:     diagnostic.CodeUnknownExpressionPattern,
				Phase:    diagnostic.PhasePattern,
				Severity: diagnostic.SeverityError,
				Message:  "Unknown expression pattern.",
				Hint:     "Require the library that defines this expression.",
				File:     file,
				Span:     expressionSpan,
			},
		},
		{
			name: "condition",
			resolve: func(registry Registry) (bool, *diagnostic.Diagnostic) {
				resolved, issue := registry.ResolveCondition(file, condition)
				return resolved == nil, issue
			},
			want: diagnostic.Diagnostic{
				Code:     diagnostic.CodeUnknownConditionPattern,
				Phase:    diagnostic.PhasePattern,
				Severity: diagnostic.SeverityError,
				Message:  "Unknown condition pattern.",
				Hint:     "",
				File:     file,
				Span:     conditionSpan,
			},
		},
		{
			name: "event",
			resolve: func(registry Registry) (bool, *diagnostic.Diagnostic) {
				resolved, issue := registry.ResolveEvent(file, event)
				return resolved == nil, issue
			},
			want: diagnostic.Diagnostic{
				Code:     diagnostic.CodeUnknownEventPattern,
				Phase:    diagnostic.PhasePattern,
				Severity: diagnostic.SeverityError,
				Message:  "Unknown event pattern: custom event",
				Hint:     "Require the library that registers this event.",
				File:     file,
				Span:     eventNameSpan,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resultIsNil, issue := test.resolve(NewRegistry())
			if !resultIsNil {
				t.Fatal("expected resolved result to be nil")
			}
			assertPatternDiagnostic(t, issue, test.want)
		})
	}
}

func TestAmbiguousPatternDiagnostic(t *testing.T) {
	const file = "packs/demo/main.puff"
	span := diagnostic.Span{StartLine: 3, StartColumn: 2, EndLine: 3, EndColumn: 10, StartOffset: 18, EndOffset: 26}
	node := &ast.EffectStmt{
		NodeBase: ast.NodeBase{SourceSpan: span},
		Tokens: []token.Token{
			manualPatternToken(token.Ident, "do", diagnostic.Span{
				StartLine: 3, StartColumn: 2, EndLine: 3, EndColumn: 4, StartOffset: 18, EndOffset: 20,
			}),
			manualPatternToken(token.Ident, "thing", diagnostic.Span{
				StartLine: 3, StartColumn: 5, EndLine: 3, EndColumn: 10, StartOffset: 21, EndOffset: 26,
			}),
		},
	}

	registry := NewRegistry()
	if err := registry.RegisterEffect("test.first", "do %value%"); err != nil {
		t.Fatalf("register first effect: %v", err)
	}
	if err := registry.RegisterEffect("test.second", "do %subject%"); err != nil {
		t.Fatalf("register second effect: %v", err)
	}

	resolved, issue := registry.ResolveEffect(file, node)
	if resolved != nil {
		t.Fatalf("expected resolved result to be nil, got %#v", resolved)
	}
	assertPatternDiagnostic(t, issue, diagnostic.Diagnostic{
		Code:     diagnostic.CodeAmbiguousPattern,
		Phase:    diagnostic.PhasePattern,
		Severity: diagnostic.SeverityError,
		Message:  "Ambiguous pattern.",
		Hint:     "Make the statement more explicit or adjust pattern priorities.",
		File:     file,
		Span:     span,
	})
}

func manualPatternToken(kind token.Type, lexeme string, span diagnostic.Span) token.Token {
	return token.Token{
		Type:        kind,
		Lexeme:      lexeme,
		Value:       lexeme,
		Line:        span.StartLine,
		Column:      span.StartColumn,
		StartOffset: span.StartOffset,
		EndOffset:   span.EndOffset,
	}
}

func assertPatternDiagnostic(t *testing.T, got *diagnostic.Diagnostic, want diagnostic.Diagnostic) {
	t.Helper()
	if got == nil {
		t.Fatal("expected diagnostic, got nil")
	}
	if got.Code != want.Code {
		t.Errorf("Code = %q, want %q", got.Code, want.Code)
	}
	if got.Phase != want.Phase {
		t.Errorf("Phase = %q, want %q", got.Phase, want.Phase)
	}
	if got.Severity != want.Severity {
		t.Errorf("Severity = %q, want %q", got.Severity, want.Severity)
	}
	if got.Message != want.Message {
		t.Errorf("Message = %q, want %q", got.Message, want.Message)
	}
	if got.Hint != want.Hint {
		t.Errorf("Hint = %q, want %q", got.Hint, want.Hint)
	}
	if got.File != want.File {
		t.Errorf("File = %q, want %q", got.File, want.File)
	}
	if got.Span != want.Span {
		t.Errorf("Span = %+v, want %+v", got.Span, want.Span)
	}
}
