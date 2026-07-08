package token

import "testing"

func TestTokenCarriesMetadata(t *testing.T) {
	token := Token{
		Type:        Int,
		Lexeme:      "100",
		Value:       100,
		Line:        3,
		Column:      10,
		StartOffset: 24,
		EndOffset:   27,
	}

	if token.Type != Int {
		t.Fatalf("expected token type %q, got %q", Int, token.Type)
	}
	if token.Lexeme != "100" {
		t.Fatalf("expected lexeme %q, got %q", "100", token.Lexeme)
	}
	if token.Value != 100 {
		t.Fatalf("expected value %v, got %v", 100, token.Value)
	}
	if token.Line != 3 {
		t.Fatalf("expected line %d, got %d", 3, token.Line)
	}
	if token.Column != 10 {
		t.Fatalf("expected column %d, got %d", 10, token.Column)
	}
	if token.StartOffset != 24 {
		t.Fatalf("expected start offset %d, got %d", 24, token.StartOffset)
	}
	if token.EndOffset != 27 {
		t.Fatalf("expected end offset %d, got %d", 27, token.EndOffset)
	}
}

func TestOfficialTokenTypesUseDocumentedValues(t *testing.T) {
	tests := []struct {
		name string
		got  Type
		want string
	}{
		{name: "eof", got: EOF, want: "EOF"},
		{name: "newline", got: Newline, want: "NEWLINE"},
		{name: "ident", got: Ident, want: "IDENT"},
		{name: "int", got: Int, want: "INT"},
		{name: "float", got: Float, want: "FLOAT"},
		{name: "string start", got: StringStart, want: "STRING_START"},
		{name: "string text", got: StringText, want: "STRING_TEXT"},
		{name: "string end", got: StringEnd, want: "STRING_END"},
		{name: "interp start", got: InterpStart, want: "INTERP_START"},
		{name: "interp end", got: InterpEnd, want: "INTERP_END"},
		{name: "nil", got: Nil, want: "NIL"},
		{name: "true", got: True, want: "TRUE"},
		{name: "false", got: False, want: "FALSE"},
		{name: "require", got: Require, want: "REQUIRE"},
		{name: "as", got: As, want: "AS"},
		{name: "pub", got: Pub, want: "PUB"},
		{name: "fun", got: Fun, want: "FUN"},
		{name: "on", got: On, want: "ON"},
		{name: "end", got: End, want: "END"},
		{name: "if", got: If, want: "IF"},
		{name: "else", got: Else, want: "ELSE"},
		{name: "loop", got: Loop, want: "LOOP"},
		{name: "times", got: Times, want: "TIMES"},
		{name: "numbers", got: Numbers, want: "NUMBERS"},
		{name: "players", got: Players, want: "PLAYERS"},
		{name: "entities", got: Entities, want: "ENTITIES"},
		{name: "from", got: From, want: "FROM"},
		{name: "to", got: To, want: "TO"},
		{name: "in", got: In, want: "IN"},
		{name: "radius", got: Radius, want: "RADIUS"},
		{name: "around", got: Around, want: "AROUND"},
		{name: "return", got: Return, want: "RETURN"},
		{name: "stop", got: Stop, want: "STOP"},
		{name: "add", got: Add, want: "ADD"},
		{name: "and", got: And, want: "AND"},
		{name: "or", got: Or, want: "OR"},
		{name: "not", got: Not, want: "NOT"},
		{name: "lparen", got: LParen, want: "LPAREN"},
		{name: "rparen", got: RParen, want: "RPAREN"},
		{name: "lbracket", got: LBracket, want: "LBRACKET"},
		{name: "rbracket", got: RBracket, want: "RBRACKET"},
		{name: "lbrace", got: LBrace, want: "LBRACE"},
		{name: "rbrace", got: RBrace, want: "RBRACE"},
		{name: "comma", got: Comma, want: "COMMA"},
		{name: "dot", got: Dot, want: "DOT"},
		{name: "dot dot", got: DotDot, want: "DOT_DOT"},
		{name: "colon", got: Colon, want: "COLON"},
		{name: "arrow", got: Arrow, want: "ARROW"},
		{name: "equal", got: Equal, want: "EQUAL"},
		{name: "equal equal", got: EqualEqual, want: "EQ_EQ"},
		{name: "bang equal", got: BangEqual, want: "BANG_EQ"},
		{name: "greater", got: Greater, want: "GT"},
		{name: "greater equal", got: GreaterEq, want: "GT_EQ"},
		{name: "less", got: Less, want: "LT"},
		{name: "less equal", got: LessEq, want: "LT_EQ"},
		{name: "plus", got: Plus, want: "PLUS"},
		{name: "minus", got: Minus, want: "MINUS"},
		{name: "star", got: Star, want: "STAR"},
		{name: "slash", got: Slash, want: "SLASH"},
		{name: "percent", got: Percent, want: "PERCENT"},
		{name: "dollar", got: Dollar, want: "DOLLAR"},
		{name: "underscore", got: Underscore, want: "UNDERSCORE"},
	}

	seen := map[Type]bool{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if string(test.got) != test.want {
				t.Fatalf("expected token type %q, got %q", test.want, test.got)
			}
			if seen[test.got] {
				t.Fatalf("expected token type %q to be unique", test.got)
			}
			seen[test.got] = true
		})
	}
}
