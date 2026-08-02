package patterns

import (
	"reflect"
	"testing"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/token"
)

func TestRegistryRegistersAndResolvesEachKind(t *testing.T) {
	registry := NewRegistry()
	registrations := []struct {
		name     string
		id       string
		kind     Kind
		register func(string, string) error
		resolve  func() (Definition, Captures, *diagnostic.Diagnostic)
	}{
		{
			name: "effect", id: "test.effect", kind: KindEffect,
			register: registry.RegisterEffect,
			resolve: func() (Definition, Captures, *diagnostic.Diagnostic) {
				resolved, issue := registry.ResolveEffect("kinds.puff", effectNode(testTokens("ready")))
				if resolved == nil {
					return Definition{}, nil, issue
				}
				return resolved.Definition, resolved.Captures, issue
			},
		},
		{
			name: "expression", id: "test.expression", kind: KindExpression,
			register: registry.RegisterExpression,
			resolve: func() (Definition, Captures, *diagnostic.Diagnostic) {
				resolved, issue := registry.ResolveExpression("kinds.puff", patternNode(testTokens("ready")))
				if resolved == nil {
					return Definition{}, nil, issue
				}
				return resolved.Definition, resolved.Captures, issue
			},
		},
		{
			name: "condition", id: "test.condition", kind: KindCondition,
			register: registry.RegisterCondition,
			resolve: func() (Definition, Captures, *diagnostic.Diagnostic) {
				resolved, issue := registry.ResolveCondition("kinds.puff", patternNode(testTokens("ready")))
				if resolved == nil {
					return Definition{}, nil, issue
				}
				return resolved.Definition, resolved.Captures, issue
			},
		},
		{
			name: "event", id: "test.event", kind: KindEvent,
			register: registry.RegisterEvent,
			resolve: func() (Definition, Captures, *diagnostic.Diagnostic) {
				resolved, issue := registry.ResolveEvent("kinds.puff", eventNode("ready"))
				if resolved == nil {
					return Definition{}, nil, issue
				}
				return resolved.Definition, resolved.Captures, issue
			},
		},
	}

	for _, test := range registrations {
		if err := test.register(test.id, "ready"); err != nil {
			t.Fatalf("register %s: %v", test.name, err)
		}
	}

	for _, test := range registrations {
		t.Run(test.name, func(t *testing.T) {
			definition, captures, issue := test.resolve()
			if issue != nil {
				t.Fatalf("resolve %s: %+v", test.name, *issue)
			}
			want := Definition{ID: test.id, Kind: test.kind, Syntax: "ready"}
			if definition != want {
				t.Fatalf("definition = %#v, want %#v", definition, want)
			}
			if len(captures) != 0 {
				t.Fatalf("captures = %#v, want none", captures)
			}
		})
	}
}

func TestRegistryInstancesAreIsolated(t *testing.T) {
	registered := NewRegistry()
	empty := NewRegistry()
	if err := registered.RegisterExpression("test.answer", "the answer"); err != nil {
		t.Fatalf("register expression: %v", err)
	}
	node := patternNode(testTokens("the", "answer"))

	resolved, issue := registered.ResolveExpression("isolation.puff", node)
	if issue != nil || resolved == nil || resolved.Definition.ID != "test.answer" {
		t.Fatalf("registered instance result = (%#v, %#v), want test.answer without diagnostic", resolved, issue)
	}

	resolved, issue = empty.ResolveExpression("isolation.puff", node)
	if resolved != nil {
		t.Fatalf("empty instance resolved %#v", resolved)
	}
	assertDiagnostic(t, issue, diagnostic.Diagnostic{
		Code:     diagnostic.CodeUnknownExpressionPattern,
		Phase:    diagnostic.PhasePattern,
		Severity: diagnostic.SeverityError,
		Message:  "Unknown expression pattern.",
		Hint:     "Require the library that defines this expression.",
		File:     "isolation.puff",
		Span:     node.Span(),
	})
}

func TestRegistryPreservesNamedMultiTokenCapturesAndSpans(t *testing.T) {
	registry := NewRegistry()
	if err := registry.RegisterEffect("test.notify", "notify %message% to %recipient%"); err != nil {
		t.Fatalf("register effect: %v", err)
	}
	tokens := positionedTokens(7, 4, 20,
		tokenSpec{token.Ident, "notify"},
		tokenSpec{token.StringStart, "\""},
		tokenSpec{token.StringText, "hello"},
		tokenSpec{token.StringEnd, "\""},
		tokenSpec{token.To, "to"},
		tokenSpec{token.Ident, "player"},
	)
	node := effectNode(tokens)

	resolved, issue := registry.ResolveEffect("captures.puff", node)
	if issue != nil {
		t.Fatalf("resolve effect: %+v", *issue)
	}
	if resolved == nil {
		t.Fatal("resolve effect returned nil without diagnostic")
	}
	if resolved.Definition.ID != "test.notify" || resolved.Definition.Kind != KindEffect {
		t.Fatalf("definition = %#v, want test.notify effect", resolved.Definition)
	}
	if len(resolved.Captures) != 2 {
		t.Fatalf("capture count = %d, want 2", len(resolved.Captures))
	}

	assertCapture(t, resolved.Captures, "message", tokens[1:4], diagnostic.Span{
		StartLine: 7, StartColumn: 11, EndLine: 7, EndColumn: 20,
		StartOffset: 27, EndOffset: 36,
	})
	assertCapture(t, resolved.Captures, "recipient", tokens[5:6], diagnostic.Span{
		StartLine: 7, StartColumn: 24, EndLine: 7, EndColumn: 30,
		StartOffset: 40, EndOffset: 46,
	})
}

func TestRegistryRejectsDuplicateIDsAndInvalidDefinitions(t *testing.T) {
	t.Run("duplicate ID across kinds", func(t *testing.T) {
		registry := NewRegistry()
		if err := registry.RegisterEffect("shared", "alpha"); err != nil {
			t.Fatalf("register initial ID: %v", err)
		}
		if err := registry.RegisterEvent("shared", "beta"); err == nil || err.Error() != `pattern ID "shared" is already registered` {
			t.Fatalf("duplicate error = %v", err)
		}

		node := eventNode("beta")
		resolved, issue := registry.ResolveEvent("duplicate.puff", node)
		if resolved != nil {
			t.Fatalf("failed registration leaked result %#v", resolved)
		}
		assertDiagnostic(t, issue, diagnostic.Diagnostic{
			Code:     diagnostic.CodeUnknownEventPattern,
			Phase:    diagnostic.PhasePattern,
			Severity: diagnostic.SeverityError,
			Message:  "Unknown event pattern: beta",
			Hint:     "Require the library that registers this event.",
			File:     "duplicate.puff",
			Span:     node.Name[0].Span(),
		})
	})

	tests := []struct {
		name    string
		id      string
		syntax  string
		message string
	}{
		{name: "empty ID", id: "", syntax: "valid", message: "pattern ID cannot be empty"},
		{name: "empty syntax", id: "bad.empty", syntax: "", message: `compile pattern "bad.empty": pattern syntax cannot be empty`},
		{name: "unclosed placeholder", id: "bad.placeholder", syntax: "send %target", message: `compile pattern "bad.placeholder": invalid placeholder "%target"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			registry := NewRegistry()
			err := registry.RegisterEffect(test.id, test.syntax)
			if err == nil || err.Error() != test.message {
				t.Fatalf("registration error = %v, want %q", err, test.message)
			}
			if test.id != "" {
				if err := registry.RegisterEffect(test.id, "valid"); err != nil {
					t.Fatalf("failed definition reserved ID: %v", err)
				}
			}
		})
	}
}

func TestRegistryReportsAmbiguityIndependentOfRegistrationOrder(t *testing.T) {
	orders := [][]Definition{
		{
			{ID: "test.value", Syntax: "say %value%"},
			{ID: "test.message", Syntax: "say %message%"},
		},
		{
			{ID: "test.message", Syntax: "say %message%"},
			{ID: "test.value", Syntax: "say %value%"},
		},
	}
	for index, order := range orders {
		t.Run([]string{"forward", "reverse"}[index], func(t *testing.T) {
			registry := NewRegistry()
			for _, definition := range order {
				if err := registry.RegisterEffect(definition.ID, definition.Syntax); err != nil {
					t.Fatalf("register %s: %v", definition.ID, err)
				}
			}
			node := effectNode(positionedTokens(9, 2, 100,
				tokenSpec{token.Ident, "say"},
				tokenSpec{token.Ident, "hello"},
			))

			resolved, issue := registry.ResolveEffect("ambiguous.puff", node)
			if resolved != nil {
				t.Fatalf("ambiguous input resolved to %#v", resolved.Definition)
			}
			assertDiagnostic(t, issue, diagnostic.Diagnostic{
				Code:     diagnostic.CodeAmbiguousPattern,
				Phase:    diagnostic.PhasePattern,
				Severity: diagnostic.SeverityError,
				Message:  "Ambiguous pattern.",
				Hint:     "Make the statement more explicit or adjust pattern priorities.",
				File:     "ambiguous.puff",
				Span:     node.Span(),
			})
		})
	}
}

type tokenSpec struct {
	type_  token.Type
	lexeme string
}

func testTokens(lexemes ...string) []token.Token {
	specs := make([]tokenSpec, len(lexemes))
	for index, lexeme := range lexemes {
		specs[index] = tokenSpec{type_: token.Ident, lexeme: lexeme}
	}
	return positionedTokens(1, 1, 0, specs...)
}

func positionedTokens(line, column, offset int, specs ...tokenSpec) []token.Token {
	tokens := make([]token.Token, len(specs))
	for index, spec := range specs {
		tokens[index] = token.Token{
			Type:        spec.type_,
			Lexeme:      spec.lexeme,
			Value:       spec.lexeme,
			Line:        line,
			Column:      column,
			StartOffset: offset,
			EndOffset:   offset + len(spec.lexeme),
		}
		column += len(spec.lexeme) + 1
		offset += len(spec.lexeme) + 1
	}
	return tokens
}

func effectNode(tokens []token.Token) *ast.EffectStmt {
	return &ast.EffectStmt{NodeBase: ast.NodeBase{SourceSpan: tokensSpan(tokens)}, Tokens: tokens}
}

func patternNode(tokens []token.Token) *ast.PatternExpr {
	return &ast.PatternExpr{NodeBase: ast.NodeBase{SourceSpan: tokensSpan(tokens)}, Tokens: tokens}
}

func eventNode(names ...string) *ast.EventDecl {
	tokens := testTokens(names...)
	identifiers := make([]ast.Identifier, len(tokens))
	for index, current := range tokens {
		identifiers[index] = ast.Identifier{
			NodeBase: ast.NodeBase{SourceSpan: tokenSpan(current)},
			Name:     current.Lexeme,
		}
	}
	return &ast.EventDecl{NodeBase: ast.NodeBase{SourceSpan: tokensSpan(tokens)}, Name: identifiers}
}

func tokensSpan(tokens []token.Token) diagnostic.Span {
	return diagnostic.Span{
		StartLine:   tokens[0].Line,
		StartColumn: tokens[0].Column,
		EndLine:     tokens[len(tokens)-1].Line,
		EndColumn:   tokens[len(tokens)-1].Column + len(tokens[len(tokens)-1].Lexeme),
		StartOffset: tokens[0].StartOffset,
		EndOffset:   tokens[len(tokens)-1].EndOffset,
	}
}

func tokenSpan(current token.Token) diagnostic.Span {
	return diagnostic.Span{
		StartLine: current.Line, StartColumn: current.Column,
		EndLine: current.Line, EndColumn: current.Column + len(current.Lexeme),
		StartOffset: current.StartOffset, EndOffset: current.EndOffset,
	}
}

func assertCapture(t *testing.T, captures Captures, name string, wantTokens []token.Token, wantSpan diagnostic.Span) {
	t.Helper()
	capture, ok := captures[name]
	if !ok {
		t.Fatalf("missing capture %q in %#v", name, captures)
	}
	if !reflect.DeepEqual(capture.Tokens, wantTokens) {
		t.Fatalf("capture %q tokens = %#v, want %#v", name, capture.Tokens, wantTokens)
	}
	if capture.Span != wantSpan {
		t.Fatalf("capture %q span = %#v, want %#v", name, capture.Span, wantSpan)
	}
}

func assertDiagnostic(t *testing.T, got *diagnostic.Diagnostic, want diagnostic.Diagnostic) {
	t.Helper()
	if got == nil {
		t.Fatalf("diagnostic = nil, want %#v", want)
	}
	if !reflect.DeepEqual(*got, want) {
		t.Fatalf("diagnostic = %#v, want %#v", *got, want)
	}
}
