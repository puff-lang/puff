package lexer

import (
	"testing"

	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/token"
)

func TestLexSingleAndDoubleQuotedStrings(t *testing.T) {
	for _, sourceText := range []string{`"hello # world"`, `'hello # world'`} {
		t.Run(sourceText, func(t *testing.T) {
			result := Lex(testFile(sourceText))

			if len(result.Diagnostics) != 0 {
				t.Fatalf("expected no diagnostics, got %v", result.Diagnostics)
			}
			assertTokenTypes(t, result.Tokens, []token.Type{
				token.StringStart,
				token.StringText,
				token.StringEnd,
				token.Newline,
				token.EOF,
			})
			if result.Tokens[1].Lexeme != "hello # world" || result.Tokens[1].Value != "hello # world" {
				t.Fatalf("unexpected string text token: %#v", result.Tokens[1])
			}
		})
	}
}

func TestLexStringEscapes(t *testing.T) {
	for _, sourceText := range []string{`"A\n\t\\\"\'"`, `'A\n\t\\\"\''`} {
		t.Run(sourceText, func(t *testing.T) {
			result := Lex(testFile(sourceText))

			if len(result.Diagnostics) != 0 {
				t.Fatalf("expected no diagnostics, got %v", result.Diagnostics)
			}
			if result.Tokens[1].Value != "A\n\t\\\"'" {
				t.Fatalf("expected decoded escapes, got %q", result.Tokens[1].Value)
			}
		})
	}
}

func TestLexStringInterpolation(t *testing.T) {
	for _, sourceText := range []string{`"Coins: {$coins + 10}"`, `'Coins: {$coins + 10}'`} {
		t.Run(sourceText, func(t *testing.T) {
			result := Lex(testFile(sourceText))

			if len(result.Diagnostics) != 0 {
				t.Fatalf("expected no diagnostics, got %v", result.Diagnostics)
			}
			assertTokenTypes(t, result.Tokens, []token.Type{
				token.StringStart,
				token.StringText,
				token.InterpStart,
				token.Dollar,
				token.Ident,
				token.Plus,
				token.Int,
				token.InterpEnd,
				token.StringEnd,
				token.Newline,
				token.EOF,
			})
			if result.Tokens[1].Value != "Coins: " {
				t.Fatalf("expected interpolation prefix, got %q", result.Tokens[1].Value)
			}
		})
	}
}

func TestLexStringLiteralBraces(t *testing.T) {
	result := Lex(testFile(`"Use {{player}} and }} #"`))

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %v", result.Diagnostics)
	}
	if result.Tokens[1].Lexeme != "Use {{player}} and }} #" {
		t.Fatalf("expected raw brace lexeme, got %q", result.Tokens[1].Lexeme)
	}
	if result.Tokens[1].Value != "Use {player} and } #" {
		t.Fatalf("expected decoded literal braces, got %q", result.Tokens[1].Value)
	}
}

func TestLexStringErrors(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		code    diagnostic.Code
		message string
	}{
		{
			name:    "invalid escape",
			source:  `"hello\q"`,
			code:    diagnostic.CodeInvalidEscapeSequence,
			message: `Invalid escape sequence: \q`,
		},
		{
			name:    "unterminated at eof",
			source:  `"Hello`,
			code:    diagnostic.CodeUnterminatedString,
			message: "Unterminated string.",
		},
		{
			name:    "unterminated at newline",
			source:  "\"Hello\n",
			code:    diagnostic.CodeUnterminatedString,
			message: "Unterminated string.",
		},
		{
			name:    "unterminated interpolation",
			source:  `"Coins: {$coins"`,
			code:    diagnostic.CodeUnterminatedInterpolation,
			message: "Unterminated string interpolation.",
		},
		{
			name:    "empty interpolation",
			source:  `"Value: {}"`,
			code:    diagnostic.CodeEmptyInterpolation,
			message: "Empty string interpolation.",
		},
		{
			name:    "empty interpolation with spaces",
			source:  `"Value: {   }"`,
			code:    diagnostic.CodeEmptyInterpolation,
			message: "Empty string interpolation.",
		},
		{
			name:    "unescaped close brace",
			source:  `"Hello }"`,
			code:    diagnostic.CodeUnescapedCloseBrace,
			message: "Unescaped close brace in string.",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := Lex(testFile(test.source))

			assertDiagnosticCodes(t, result.Diagnostics, []diagnostic.Code{test.code})
			if result.Diagnostics[0].Message != test.message {
				t.Fatalf("expected message %q, got %q", test.message, result.Diagnostics[0].Message)
			}
		})
	}
}

func TestLexStringsInsideInterpolation(t *testing.T) {
	result := Lex(testFile(`"Result: {format("Value } here", $coins)}"`))

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %v", result.Diagnostics)
	}
	assertTokenTypes(t, result.Tokens, []token.Type{
		token.StringStart,
		token.StringText,
		token.InterpStart,
		token.Ident,
		token.LParen,
		token.StringStart,
		token.StringText,
		token.StringEnd,
		token.Comma,
		token.Dollar,
		token.Ident,
		token.RParen,
		token.InterpEnd,
		token.StringEnd,
		token.Newline,
		token.EOF,
	})
	if result.Tokens[6].Value != "Value } here" {
		t.Fatalf("expected inner string text, got %q", result.Tokens[6].Value)
	}
}

func TestLexListInsideInterpolation(t *testing.T) {
	result := Lex(testFile(`"Items: {["sword", "apple"]}"`))

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %v", result.Diagnostics)
	}
	assertTokenTypes(t, result.Tokens, []token.Type{
		token.StringStart,
		token.StringText,
		token.InterpStart,
		token.LBracket,
		token.StringStart,
		token.StringText,
		token.StringEnd,
		token.Comma,
		token.StringStart,
		token.StringText,
		token.StringEnd,
		token.RBracket,
		token.InterpEnd,
		token.StringEnd,
		token.Newline,
		token.EOF,
	})
}

func TestLexDoesNotNestInterpolationInInnerString(t *testing.T) {
	result := Lex(testFile(`"Result: {format("Coins: {$coins}")}"`))

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %v", result.Diagnostics)
	}

	interpolationCount := 0
	foundInnerText := false
	for _, tok := range result.Tokens {
		if tok.Type == token.InterpStart {
			interpolationCount++
		}
		if tok.Type == token.StringText && tok.Value == "Coins: {$coins}" {
			foundInnerText = true
		}
	}
	if interpolationCount != 1 {
		t.Fatalf("expected one interpolation, got %d", interpolationCount)
	}
	if !foundInnerText {
		t.Fatal("expected nested interpolation syntax to remain inner string text")
	}
}
