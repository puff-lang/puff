package lexer

import (
	"slices"
	"strings"
	"testing"

	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/source"
	"github.com/puff-lang/puff/internal/token"
)

func testFile(text string) source.File {
	return source.NewFile("project/src/main.puff", "main.puff", text)
}

func tokenTypes(tokens []token.Token) []token.Type {
	types := make([]token.Type, len(tokens))
	for i, tok := range tokens {
		types[i] = tok.Type
	}

	return types
}

func diagnosticCodes(diagnostics []diagnostic.Diagnostic) []diagnostic.Code {
	codes := make([]diagnostic.Code, len(diagnostics))
	for i, diag := range diagnostics {
		codes[i] = diag.Code
	}

	return codes
}

func assertTokenTypes(t *testing.T, got []token.Token, want []token.Type) {
	t.Helper()

	gotTypes := tokenTypes(got)
	if len(gotTypes) != len(want) {
		t.Fatalf("expected token types %v, got %v", want, gotTypes)
	}
	for i := range want {
		if gotTypes[i] != want[i] {
			t.Fatalf("expected token type %d to be %q, got %q; all tokens: %v", i, want[i], gotTypes[i], gotTypes)
		}
	}
}

func assertDiagnosticCodes(t *testing.T, got []diagnostic.Diagnostic, want []diagnostic.Code) {
	t.Helper()

	gotCodes := diagnosticCodes(got)
	if len(gotCodes) != len(want) {
		t.Fatalf("expected diagnostic codes %v, got %v", want, gotCodes)
	}
	for i := range want {
		if gotCodes[i] != want[i] {
			t.Fatalf("expected diagnostic code %d to be %q, got %q; all diagnostics: %v", i, want[i], gotCodes[i], gotCodes)
		}
	}
}

func TestLexCollectsFrontMatterMetadata(t *testing.T) {
	result := Lex(testFile("\n# namespace: example\n# tags: load, tick, load\n\non load\nend\n"))

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %v", result.Diagnostics)
	}
	if result.Metadata.Namespace != "example" {
		t.Fatalf("expected namespace %q, got %q", "example", result.Metadata.Namespace)
	}
	if len(result.Metadata.Tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(result.Metadata.Tags))
	}
	if result.Metadata.Tags[0] != "load" || result.Metadata.Tags[1] != "tick" {
		t.Fatalf("expected tags [load tick], got %v", result.Metadata.Tags)
	}

	assertTokenTypes(t, result.Tokens, []token.Type{
		token.On,
		token.Ident,
		token.Newline,
		token.End,
		token.Newline,
		token.EOF,
	})
}

func TestLexIgnoresCommonCommentsAndCollapsesNewlines(t *testing.T) {
	result := Lex(testFile("# ordinary comment\n$coins = 100 # inline\n\n\n$name = 200\n"))

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %v", result.Diagnostics)
	}

	assertTokenTypes(t, result.Tokens, []token.Type{
		token.Dollar,
		token.Ident,
		token.Equal,
		token.Int,
		token.Newline,
		token.Dollar,
		token.Ident,
		token.Equal,
		token.Int,
		token.Newline,
		token.EOF,
	})
}

func TestLexHandlesBOMAndCRLF(t *testing.T) {
	result := Lex(testFile("\uFEFF$coins = 100\r\n$name = 200\r\n"))

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %v", result.Diagnostics)
	}

	assertTokenTypes(t, result.Tokens, []token.Type{
		token.Dollar,
		token.Ident,
		token.Equal,
		token.Int,
		token.Newline,
		token.Dollar,
		token.Ident,
		token.Equal,
		token.Int,
		token.Newline,
		token.EOF,
	})
}

func TestLexKeywordsIdentifiersAndLiterals(t *testing.T) {
	result := Lex(testFile("fun Fun nil true false int and or not\n"))

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %v", result.Diagnostics)
	}

	assertTokenTypes(t, result.Tokens, []token.Type{
		token.Fun,
		token.Ident,
		token.Nil,
		token.True,
		token.False,
		token.Ident,
		token.And,
		token.Or,
		token.Not,
		token.Newline,
		token.EOF,
	})

	if result.Tokens[1].Lexeme != "Fun" {
		t.Fatalf("expected case-sensitive identifier %q, got %q", "Fun", result.Tokens[1].Lexeme)
	}
	if result.Tokens[3].Value != true {
		t.Fatalf("expected true literal value, got %v", result.Tokens[3].Value)
	}
	if result.Tokens[4].Value != false {
		t.Fatalf("expected false literal value, got %v", result.Tokens[4].Value)
	}
}

func TestLexNumbersAndRange(t *testing.T) {
	result := Lex(testFile("$a = 100\n$b = 10.5\n$c = 1..10\n"))

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %v", result.Diagnostics)
	}

	assertTokenTypes(t, result.Tokens, []token.Type{
		token.Dollar, token.Ident, token.Equal, token.Int, token.Newline,
		token.Dollar, token.Ident, token.Equal, token.Float, token.Newline,
		token.Dollar, token.Ident, token.Equal, token.Int, token.DotDot, token.Int, token.Newline,
		token.EOF,
	})

	if result.Tokens[3].Value != 100 {
		t.Fatalf("expected int value %v, got %v", 100, result.Tokens[3].Value)
	}
	if result.Tokens[8].Value != 10.5 {
		t.Fatalf("expected float value %v, got %v", 10.5, result.Tokens[8].Value)
	}
}

func TestLexSymbolsAndOperators(t *testing.T) {
	result := Lex(testFile("() [] {} , . .. : -> = == != > >= < <= + - * / % $_price\n"))

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %v", result.Diagnostics)
	}

	assertTokenTypes(t, result.Tokens, []token.Type{
		token.LParen,
		token.RParen,
		token.LBracket,
		token.RBracket,
		token.LBrace,
		token.RBrace,
		token.Comma,
		token.Dot,
		token.DotDot,
		token.Colon,
		token.Arrow,
		token.Equal,
		token.EqualEqual,
		token.BangEqual,
		token.Greater,
		token.GreaterEq,
		token.Less,
		token.LessEq,
		token.Plus,
		token.Minus,
		token.Star,
		token.Slash,
		token.Percent,
		token.Dollar,
		token.Underscore,
		token.Ident,
		token.Newline,
		token.EOF,
	})
}

func TestLexSuppressesNewlinesInsideGroups(t *testing.T) {
	result := Lex(testFile("$result = add(\n1,\n2\n)\n"))

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %v", result.Diagnostics)
	}

	assertTokenTypes(t, result.Tokens, []token.Type{
		token.Dollar,
		token.Ident,
		token.Equal,
		token.Add,
		token.LParen,
		token.Int,
		token.Comma,
		token.Int,
		token.RParen,
		token.Newline,
		token.EOF,
	})
}

func TestLexInsertsNewlineBeforeEOF(t *testing.T) {
	result := Lex(testFile("$coins = 100"))

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %v", result.Diagnostics)
	}

	assertTokenTypes(t, result.Tokens, []token.Type{
		token.Dollar,
		token.Ident,
		token.Equal,
		token.Int,
		token.Newline,
		token.EOF,
	})
}

func TestLexTracksTokenPositions(t *testing.T) {
	result := Lex(testFile("$coins = 100\n"))

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %v", result.Diagnostics)
	}

	intToken := result.Tokens[3]
	if intToken.Line != 1 {
		t.Fatalf("expected int line %d, got %d", 1, intToken.Line)
	}
	if intToken.Column != 10 {
		t.Fatalf("expected int column %d, got %d", 10, intToken.Column)
	}
	if intToken.StartOffset != 9 {
		t.Fatalf("expected int start offset %d, got %d", 9, intToken.StartOffset)
	}
	if intToken.EndOffset != 12 {
		t.Fatalf("expected int end offset %d, got %d", 12, intToken.EndOffset)
	}
}

func TestLexGoldenFunction(t *testing.T) {
	result := Lex(testFile("fun add(a: int, b: int) -> int\n   return a + b\nend\n"))

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %v", result.Diagnostics)
	}

	assertTokenTypes(t, result.Tokens, []token.Type{
		token.Fun, token.Add, token.LParen,
		token.Ident, token.Colon, token.Ident, token.Comma,
		token.Ident, token.Colon, token.Ident, token.RParen,
		token.Arrow, token.Ident, token.Newline,
		token.Return, token.Ident, token.Plus, token.Ident, token.Newline,
		token.End, token.Newline, token.EOF,
	})
}

func TestLexRecognizesEveryKeywordExactly(t *testing.T) {
	result := Lex(testFile("require as pub fun on end if else loop times numbers players entities from to in radius around return stop add and or not function endif loopPlayers int Player player send\n"))

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %v", result.Diagnostics)
	}

	assertTokenTypes(t, result.Tokens, []token.Type{
		token.Require, token.As, token.Pub, token.Fun, token.On, token.End,
		token.If, token.Else, token.Loop, token.Times, token.Numbers,
		token.Players, token.Entities, token.From, token.To, token.In,
		token.Radius, token.Around, token.Return, token.Stop, token.Add,
		token.And, token.Or, token.Not,
		token.Ident, token.Ident, token.Ident, token.Ident, token.Ident,
		token.Ident, token.Ident, token.Newline, token.EOF,
	})
}

func TestLexHandlesVariablesNegativeNumbersAndGroupedNewlines(t *testing.T) {
	result := Lex(testFile("\t$loop = -10\n$player.end = shop.$tax\n$list = [\n1,\n]\n$map = {\nkey: 1\n}\n"))

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %v", result.Diagnostics)
	}

	assertTokenTypes(t, result.Tokens, []token.Type{
		token.Dollar, token.Loop, token.Equal, token.Minus, token.Int, token.Newline,
		token.Dollar, token.Ident, token.Dot, token.End, token.Equal,
		token.Ident, token.Dot, token.Dollar, token.Ident, token.Newline,
		token.Dollar, token.Ident, token.Equal, token.LBracket,
		token.Int, token.Comma, token.RBracket, token.Newline,
		token.Dollar, token.Ident, token.Equal, token.LBrace,
		token.Ident, token.Colon, token.Int, token.RBrace, token.Newline,
		token.EOF,
	})
}

func TestLexKeepsMetadataAtTheTop(t *testing.T) {
	result := Lex(testFile("\n# namespace : example\n# tags:load, minecraft:tick\non load\n# tags: ignored\nend\n"))

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %v", result.Diagnostics)
	}
	if result.Metadata.Namespace != "example" {
		t.Fatalf("expected namespace %q, got %q", "example", result.Metadata.Namespace)
	}
	if len(result.Metadata.Tags) != 2 || result.Metadata.Tags[0] != "load" || result.Metadata.Tags[1] != "minecraft:tick" {
		t.Fatalf("expected tags [load minecraft:tick], got %v", result.Metadata.Tags)
	}
}

func TestLexAcceptsDocumentedMetadataCharacters(t *testing.T) {
	for _, namespace := range []string{"example", "my_pack", "my-pack", "my.pack", "123pack"} {
		t.Run("namespace_"+namespace, func(t *testing.T) {
			result := Lex(testFile("# namespace: " + namespace + "\non load\nend\n"))

			if len(result.Diagnostics) != 0 {
				t.Fatalf("expected no diagnostics, got %v", result.Diagnostics)
			}
			if result.Metadata.Namespace != namespace {
				t.Fatalf("expected namespace %q, got %q", namespace, result.Metadata.Namespace)
			}
		})
	}

	tags := []string{"load", "custom.event", "custom-tag", "folder/my_tag", "minecraft:load", "my_pack:folder/my_tag"}
	result := Lex(testFile("# tags: " + strings.Join(tags, ", ") + "\non load\nend\n"))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %v", result.Diagnostics)
	}
	if !slices.Equal(result.Metadata.Tags, tags) {
		t.Fatalf("expected tags %v, got %v", tags, result.Metadata.Tags)
	}
}

func TestLexTracksLiteralAndEOFMetadata(t *testing.T) {
	result := Lex(testFile("$a = 1.5"))

	if len(result.Diagnostics) != 0 {
		t.Fatalf("expected no diagnostics, got %v", result.Diagnostics)
	}

	floatToken := result.Tokens[3]
	if floatToken.Lexeme != "1.5" || floatToken.Value != 1.5 {
		t.Fatalf("expected float token 1.5, got lexeme %q and value %v", floatToken.Lexeme, floatToken.Value)
	}

	newlineToken := result.Tokens[4]
	if newlineToken.StartOffset != 8 || newlineToken.EndOffset != 8 {
		t.Fatalf("expected synthetic newline at offset 8, got %d..%d", newlineToken.StartOffset, newlineToken.EndOffset)
	}

	eofToken := result.Tokens[5]
	if eofToken.Line != 1 || eofToken.Column != 9 || eofToken.StartOffset != 8 || eofToken.EndOffset != 8 {
		t.Fatalf("expected EOF at 1:9 offset 8, got %d:%d offset %d..%d", eofToken.Line, eofToken.Column, eofToken.StartOffset, eofToken.EndOffset)
	}
}

func TestLexReportsCoreErrors(t *testing.T) {
	tests := []struct {
		name string
		text string
		code diagnostic.Code
	}{
		{
			name: "invalid utf8",
			text: string([]byte{0xff}),
			code: diagnostic.CodeInvalidUTF8,
		},
		{
			name: "invalid line ending",
			text: "$coins = 100\r$name = 200",
			code: diagnostic.CodeInvalidLineEnding,
		},
		{
			name: "invalid character",
			text: "$coins = 100 @ 20\n",
			code: diagnostic.CodeInvalidCharacter,
		},
		{
			name: "bom in middle",
			text: "$coins = \uFEFF100\n",
			code: diagnostic.CodeInvalidCharacter,
		},
		{
			name: "invalid trailing dot number",
			text: "$value = 1.\n",
			code: diagnostic.CodeInvalidNumber,
		},
		{
			name: "invalid leading dot number",
			text: "$value = .5\n",
			code: diagnostic.CodeInvalidNumber,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := Lex(testFile(test.text))

			assertDiagnosticCodes(t, result.Diagnostics, []diagnostic.Code{test.code})
		})
	}
}

func TestLexReportsUnsupportedNumberForms(t *testing.T) {
	for _, text := range []string{"1_000", "1e9", "0xFF"} {
		t.Run(text, func(t *testing.T) {
			result := Lex(testFile("$value = " + text + "\n"))

			assertDiagnosticCodes(t, result.Diagnostics, []diagnostic.Code{diagnostic.CodeInvalidNumber})
		})
	}
}

func TestLexRejectsUnicodeIdentifiers(t *testing.T) {
	result := Lex(testFile("fun calcularPreco\u00e7o\nend\n"))

	assertDiagnosticCodes(t, result.Diagnostics, []diagnostic.Code{diagnostic.CodeInvalidCharacter})
}

func TestLexRecoversAfterInvalidCharacter(t *testing.T) {
	result := Lex(testFile("$coins = 100 @ 20\n$name = 30\n"))

	assertDiagnosticCodes(t, result.Diagnostics, []diagnostic.Code{diagnostic.CodeInvalidCharacter})
	diag := result.Diagnostics[0]
	if diag.Phase != diagnostic.PhaseLexer || diag.Severity != diagnostic.SeverityError {
		t.Fatalf("expected lexer error diagnostic, got phase %q severity %q", diag.Phase, diag.Severity)
	}
	if diag.Message != "Invalid character: @" || diag.Hint != "Remove the character or replace it with valid Puff syntax." {
		t.Fatalf("unexpected diagnostic contract: %#v", diag)
	}
	if diag.File != "main.puff" || diag.Span.StartLine != 1 || diag.Span.StartColumn != 14 || diag.Span.StartOffset != 13 || diag.Span.EndOffset != 14 {
		t.Fatalf("unexpected diagnostic location: %#v", diag)
	}

	assertTokenTypes(t, result.Tokens, []token.Type{
		token.Dollar, token.Ident, token.Equal, token.Int, token.Int, token.Newline,
		token.Dollar, token.Ident, token.Equal, token.Int, token.Newline, token.EOF,
	})
}

func TestLexReportsMultipleDiagnostics(t *testing.T) {
	result := Lex(testFile("@ `\n"))

	assertDiagnosticCodes(t, result.Diagnostics, []diagnostic.Code{
		diagnostic.CodeInvalidCharacter,
		diagnostic.CodeInvalidCharacter,
	})
}

func TestLexReportsMetadataErrors(t *testing.T) {
	tests := []struct {
		name string
		text string
		code diagnostic.Code
	}{
		{
			name: "unknown key",
			text: "# author: Fabio\non load\nend\n",
			code: diagnostic.CodeUnknownMetadataKey,
		},
		{
			name: "duplicate key",
			text: "# namespace: example\n# namespace: other\non load\nend\n",
			code: diagnostic.CodeDuplicateMetadataKey,
		},
		{
			name: "invalid namespace",
			text: "# namespace: Example\non load\nend\n",
			code: diagnostic.CodeInvalidMetadataValue,
		},
		{
			name: "namespace with space",
			text: "# namespace: my pack\non load\nend\n",
			code: diagnostic.CodeInvalidMetadataValue,
		},
		{
			name: "namespace with slash",
			text: "# namespace: my/pack\non load\nend\n",
			code: diagnostic.CodeInvalidMetadataValue,
		},
		{
			name: "invalid tag",
			text: "# tags: Load\non load\nend\n",
			code: diagnostic.CodeInvalidMetadataValue,
		},
		{
			name: "tag with space",
			text: "# tags: my tag\non load\nend\n",
			code: diagnostic.CodeInvalidMetadataValue,
		},
		{
			name: "tag missing path",
			text: "# tags: my_pack:\non load\nend\n",
			code: diagnostic.CodeInvalidMetadataValue,
		},
		{
			name: "tag missing namespace",
			text: "# tags: :load\non load\nend\n",
			code: diagnostic.CodeInvalidMetadataValue,
		},
		{
			name: "empty tags",
			text: "# tags:\non load\nend\n",
			code: diagnostic.CodeInvalidMetadataValue,
		},
		{
			name: "empty tag item",
			text: "# tags: load,,tick\non load\nend\n",
			code: diagnostic.CodeInvalidMetadataValue,
		},
		{
			name: "duplicate tags key",
			text: "# tags: load\n# tags: tick\non load\nend\n",
			code: diagnostic.CodeDuplicateMetadataKey,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := Lex(testFile(test.text))

			assertDiagnosticCodes(t, result.Diagnostics, []diagnostic.Code{test.code})
		})
	}
}
