package patterns

import (
	"testing"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/lexer"
	"github.com/puff-lang/puff/internal/parser"
	"github.com/puff-lang/puff/internal/source"
	"github.com/puff-lang/puff/internal/token"
)

func TestCoreRegistryResolvesSendEffects(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantText   string
		wantTarget string
		internalTo bool
	}{
		{
			name:       "player target",
			input:      "on load\nsend \"Hello\" to player\nend\n",
			wantText:   `"Hello"`,
			wantTarget: "player",
		},
		{
			name:       "console target with internal delimiter",
			input:      "on tick\nsend \"Route {origin to destination}\" to console\nend\n",
			wantText:   `"Route {origin to destination}"`,
			wantTarget: "console",
			internalTo: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event := parseCoreEvent(t, test.input)
			if len(event.Body.Statements) != 1 {
				t.Fatalf("expected one statement, got %#v", event.Body.Statements)
			}
			effect, ok := event.Body.Statements[0].(*ast.EffectStmt)
			if !ok {
				t.Fatalf("expected effect statement, got %T", event.Body.Statements[0])
			}

			resolved, issue := NewCoreRegistry().ResolveEffect("core.puff", effect)
			if issue != nil {
				t.Fatalf("expected send to resolve, got %#v", issue)
			}
			wantDefinition := Definition{ID: CoreSendEffectID, Kind: KindEffect, Syntax: CoreSendEffectSyntax}
			if resolved.Definition != wantDefinition {
				t.Fatalf("unexpected definition: want %#v, got %#v", wantDefinition, resolved.Definition)
			}
			if resolved.Node != effect {
				t.Fatalf("expected resolved effect to retain original node")
			}
			if len(resolved.Captures) != 2 {
				t.Fatalf("expected exactly text and target captures, got %#v", resolved.Captures)
			}
			assertCoreCapture(t, test.input, resolved.Captures, "text", test.wantText)
			assertCoreCapture(t, test.input, resolved.Captures, "target", test.wantTarget)
			if test.internalTo && !captureContainsToken(resolved.Captures["text"], token.To) {
				t.Fatal("expected text capture to retain the internal to token")
			}
		})
	}
}

func TestCoreRegistryResolvesLoadAndTickEvents(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantID     string
		wantSyntax string
	}{
		{name: "load", input: "on load\nend\n", wantID: CoreLoadEventID, wantSyntax: CoreLoadEventSyntax},
		{name: "tick", input: "on tick\nend\n", wantID: CoreTickEventID, wantSyntax: CoreTickEventSyntax},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			event := parseCoreEvent(t, test.input)
			resolved, issue := NewCoreRegistry().ResolveEvent("core.puff", event)
			if issue != nil {
				t.Fatalf("expected event to resolve, got %#v", issue)
			}
			wantDefinition := Definition{ID: test.wantID, Kind: KindEvent, Syntax: test.wantSyntax}
			if resolved.Definition != wantDefinition {
				t.Fatalf("unexpected definition: want %#v, got %#v", wantDefinition, resolved.Definition)
			}
			if resolved.Node != event {
				t.Fatalf("expected resolved event to retain original node")
			}
			if len(resolved.Captures) != 0 {
				t.Fatalf("expected no event captures, got %#v", resolved.Captures)
			}
		})
	}
}

func TestCoreRegistryRejectsInvalidSendShapes(t *testing.T) {
	tests := []struct {
		name      string
		statement string
	}{
		{name: "missing target", statement: `send "Hello"`},
		{name: "wrong delimiter", statement: `send "Hello" toward player`},
		{name: "case sensitive", statement: `Send "Hello" to player`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := "on load\n" + test.statement + "\nend\n"
			event := parseCoreEvent(t, input)
			effect, ok := event.Body.Statements[0].(*ast.EffectStmt)
			if !ok {
				t.Fatalf("expected effect statement, got %T", event.Body.Statements[0])
			}

			resolved, issue := NewCoreRegistry().ResolveEffect("core.puff", effect)
			if resolved != nil {
				t.Fatalf("expected no resolution, got %#v", resolved)
			}
			assertCoreDiagnostic(t, issue, diagnostic.CodeUnknownEffectPattern)
		})
	}
}

func TestCoreRegistryRejectsNonCoreEvents(t *testing.T) {
	for _, name := range []string{"join", "Load"} {
		t.Run(name, func(t *testing.T) {
			event := parseCoreEvent(t, "on "+name+"\nend\n")
			resolved, issue := NewCoreRegistry().ResolveEvent("core.puff", event)
			if resolved != nil {
				t.Fatalf("expected no resolution, got %#v", resolved)
			}
			assertCoreDiagnostic(t, issue, diagnostic.CodeUnknownEventPattern)
		})
	}
}

func parseCoreEvent(t *testing.T, text string) *ast.EventDecl {
	t.Helper()
	file := source.NewFile("core.puff", "core.puff", text)
	result := parser.Parse(file, lexer.Lex(file))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected source to parse without diagnostics, got %#v", result.Diagnostics)
	}
	if len(result.File.Declarations) != 1 {
		t.Fatalf("expected one event declaration, got %#v", result.File.Declarations)
	}
	event, ok := result.File.Declarations[0].(*ast.EventDecl)
	if !ok {
		t.Fatalf("expected event declaration, got %T", result.File.Declarations[0])
	}
	return event
}

func assertCoreCapture(t *testing.T, input string, captures Captures, name string, want string) {
	t.Helper()
	capture, ok := captures[name]
	if !ok {
		t.Fatalf("missing %q capture in %#v", name, captures)
	}
	if len(capture.Tokens) == 0 {
		t.Fatalf("expected %q capture to retain source tokens", name)
	}
	if capture.Tokens[0].StartOffset != capture.Span.StartOffset || capture.Tokens[len(capture.Tokens)-1].EndOffset != capture.Span.EndOffset {
		t.Fatalf("expected %q capture tokens to cover its span", name)
	}
	if got := input[capture.Span.StartOffset:capture.Span.EndOffset]; got != want {
		t.Fatalf("unexpected %q capture span: want %q, got %q", name, want, got)
	}
}

func captureContainsToken(capture Capture, kind token.Type) bool {
	for _, current := range capture.Tokens {
		if current.Type == kind {
			return true
		}
	}
	return false
}

func assertCoreDiagnostic(t *testing.T, issue *diagnostic.Diagnostic, code diagnostic.Code) {
	t.Helper()
	if issue == nil {
		t.Fatalf("expected %s diagnostic", code)
	}
	if issue.Code != code || issue.Phase != diagnostic.PhasePattern || issue.Severity != diagnostic.SeverityError {
		t.Fatalf("unexpected diagnostic: %#v", issue)
	}
}
