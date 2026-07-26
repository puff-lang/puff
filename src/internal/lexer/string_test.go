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
	result := Lex(testFile(`"Result: {format("Value } }} {} here", $coins)}"`))

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
	if result.Tokens[6].Value != "Value } } {} here" {
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

	assertDiagnosticCodes(t, result.Diagnostics, []diagnostic.Code{diagnostic.CodeInvalidCharacter})
	if result.Diagnostics[0].Message != "Nested string interpolation is not allowed." {
		t.Fatalf("unexpected nested interpolation message: %q", result.Diagnostics[0].Message)
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

func TestLexStringTokenOffsets(t *testing.T) {
	result := Lex(testFile(`"é\n{{x}} {$a}!"`))

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %v", result.Diagnostics)
	}

	prefix := result.Tokens[1]
	if prefix.Lexeme != `é\n{{x}} ` || prefix.Value != "é\n{x} " {
		t.Fatalf("unexpected prefix token: %#v", prefix)
	}
	if prefix.StartOffset != 1 || prefix.EndOffset != 11 {
		t.Fatalf("expected prefix offsets 1..11, got %d..%d", prefix.StartOffset, prefix.EndOffset)
	}

	if result.Tokens[2].Type != token.InterpStart || result.Tokens[2].StartOffset != 11 || result.Tokens[2].EndOffset != 12 {
		t.Fatalf("unexpected interpolation start: %#v", result.Tokens[2])
	}
	if result.Tokens[5].Type != token.InterpEnd || result.Tokens[5].StartOffset != 14 || result.Tokens[5].EndOffset != 15 {
		t.Fatalf("unexpected interpolation end: %#v", result.Tokens[5])
	}

	suffix := result.Tokens[6]
	if suffix.Lexeme != "!" || suffix.Value != "!" || suffix.StartOffset != 15 || suffix.EndOffset != 16 {
		t.Fatalf("unexpected suffix token: %#v", suffix)
	}
	if result.Tokens[7].Type != token.StringEnd || result.Tokens[7].StartOffset != 16 || result.Tokens[7].EndOffset != 17 {
		t.Fatalf("unexpected string end: %#v", result.Tokens[7])
	}
}

func TestLexRecoversAfterUnterminatedString(t *testing.T) {
	result := Lex(testFile("\"bad\n$ok = 1\n"))

	assertDiagnosticCodes(t, result.Diagnostics, []diagnostic.Code{diagnostic.CodeUnterminatedString})
	assertTokenTypes(t, result.Tokens, []token.Type{
		token.StringStart,
		token.StringText,
		token.Newline,
		token.Dollar,
		token.Ident,
		token.Equal,
		token.Int,
		token.Newline,
		token.EOF,
	})
}

func TestLexRestoresStateAfterMalformedSameQuoteInterpolation(t *testing.T) {
	result := Lex(testFile("\"x: {$a + \" + \"next\"\n$ok = 1\n"))

	assertDiagnosticCodes(t, result.Diagnostics, []diagnostic.Code{diagnostic.CodeUnterminatedInterpolation})
	if len(result.Tokens) < 6 {
		t.Fatalf("expected recovery tokens, got %v", tokenTypes(result.Tokens))
	}

	foundNextLine := false
	for index, tok := range result.Tokens {
		if tok.Type == token.Dollar && index+1 < len(result.Tokens) && result.Tokens[index+1].Lexeme == "ok" {
			foundNextLine = true
			break
		}
	}
	if !foundNextLine {
		t.Fatalf("expected lexer to resume on the next line, got %v", tokenTypes(result.Tokens))
	}
}

func TestLexRecoversAfterUnterminatedInterpolation(t *testing.T) {
	result := Lex(testFile("\"bad: {$value\n$ok = 1\n"))

	assertDiagnosticCodes(t, result.Diagnostics, []diagnostic.Code{diagnostic.CodeUnterminatedInterpolation})
	assertTokenTypes(t, result.Tokens, []token.Type{
		token.StringStart,
		token.StringText,
		token.InterpStart,
		token.Dollar,
		token.Ident,
		token.Newline,
		token.Dollar,
		token.Ident,
		token.Equal,
		token.Int,
		token.Newline,
		token.EOF,
	})
}

func TestLexMultipleInterpolationsRestoreBraceDepth(t *testing.T) {
	result := Lex(testFile("\"{$a} {$b}\"\n$items = [\n1\n]\n"))

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %v", result.Diagnostics)
	}

	interpolationCount := 0
	for _, tok := range result.Tokens {
		if tok.Type == token.InterpStart {
			interpolationCount++
		}
	}
	if interpolationCount != 2 {
		t.Fatalf("expected two interpolations, got %d", interpolationCount)
	}

	assertTokenTypes(t, result.Tokens, []token.Type{
		token.StringStart,
		token.InterpStart,
		token.Dollar,
		token.Ident,
		token.InterpEnd,
		token.StringText,
		token.InterpStart,
		token.Dollar,
		token.Ident,
		token.InterpEnd,
		token.StringEnd,
		token.Newline,
		token.Dollar,
		token.Ident,
		token.Equal,
		token.LBracket,
		token.Int,
		token.RBracket,
		token.Newline,
		token.EOF,
	})
}
