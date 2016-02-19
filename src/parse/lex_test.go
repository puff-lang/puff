package parse

import (
	"token"
	"testing"
)

var tokenName = [...]string{
	token.ILLEGAL: "ILLEGAL",

	token.EOF:     "EOF",
	token.COMMENT: "COMMENT",

	token.IDENT:   "IDENT",
	token.INT:     "INT",
	token.BOOLEAN: "BOOL",
	token.FLOAT:   "FLOAT",
	token.CHAR:    "CHAR",
	token.STRING:  "STRING",

	token.ASSIGN: "=",

	token.ADD: "+",
	token.SUB: "-",
	token.MUL: "*",
	token.QUO: "/",
	token.REM: "%",

	token.AND: "&",
	token.OR:  "|",
	token.XOR: "^",
	token.SHL: "<<",
	token.SHR: ">>",

	token.LAND: "&&",
	token.LOR:  "||",

	token.EQL: "==",
	token.LSS: "<",
	token.GTR: ">",
	token.NOT: "!",

	token.LPAREN: "(",
	token.LBRACK: "[",
	token.LBRACE: "{",
	token.COMMA:  ",",
	token.PERIOD: ".",

	token.RPAREN:    ")",
	token.RBRACK:    "]",
	token.RBRACE:    "}",
	token.SEMICOLON: ";",
	token.COLON:     ":",

	token.FN:   "fn",
	token.LET:  "let",
	token.TYPE: "type",
	token.DATA: "data",

	token.IF:	"if",
	token.THEN: "then",
	token.ELSE:	"else",

	token.WHITESPACE: " \t\r\n",
}

/*func (i token.TokenType) String() string {
	s := tokenName[i]
	if s == "" {
		return fmt.Sprintf("token%d", int(i))
	}
	return s
}*/

type lexTest struct {
	name   string
	input  string
	tokens []token.Token
}

var (
	tEOF    = token.NewToken(token.EOF, 0, "")
	tLParen = token.NewToken(token.LPAREN, 0, "(")
	tRParen = token.NewToken(token.RPAREN, 0, ")")
)

var lexTests = []lexTest{
	{"empty", "", []token.Token{tEOF}},
	{"number", "2", []token.Token{
		token.NewToken(token.INT, 0, "2"),
		token.NewToken(token.EOF, 0, ""),
	}},
	{"add", "2 + 2", []token.Token{
		token.NewToken(token.INT, 0, "2"),
		token.NewToken(token.ADD, 0, "+"),
		token.NewToken(token.INT, 0, "2"),
		tEOF,
	}},
	{"expr in paren", "(2 + 2)", []token.Token{
		tLParen,
		token.NewToken(token.INT, 0, "2"),
		token.NewToken(token.ADD, 0, "+"),
		token.NewToken(token.INT, 0, "2"),
		tRParen,
		tEOF,
	}},
	{"number assignment", "let x = 5", []token.Token{
		token.NewToken(token.LET, 0, "let"),
		token.NewToken(token.IDENT, 0, "x"),
		token.NewToken(token.ASSIGN, 0, "="),
		token.NewToken(token.INT, 0, "5"),
		tEOF,
	}},
	{"string assignment", "let x = \"hello\"", []token.Token{
		token.NewToken(token.LET, 0, "let"),
		token.NewToken(token.IDENT, 0, "x"),
		token.NewToken(token.ASSIGN, 0, "="),
		token.NewToken(token.STRING, 0, "\"hello\""),
		tEOF,
	}},
	{"line comment use", "//here our first comment goes.\n ", []token.Token{
		token.NewToken(token.COMMENT, 0, "//here our first comment goes."),
		tEOF,
	}},
	{"block comment use", "/* here our first \n* comment\n* goes.\n */ \n let x = \"hello\"\n ", []token.Token{
		token.NewToken(token.COMMENT, 0, "/* here our first \n* comment\n* goes.\n */"),
		token.NewToken(token.LET, 0, "let"),
		token.NewToken(token.IDENT, 0, "x"),
		token.NewToken(token.ASSIGN, 0, "="),
		token.NewToken(token.STRING, 0, "\"hello\""),
		tEOF,
	}},
	{"function expression", "fn x -> x + 1", []token.Token{
		token.NewToken(token.FN, 0, "fn"),
		token.NewToken(token.IDENT, 0, "x"),
		token.NewToken(token.ARROW, 0, "->"),
		token.NewToken(token.IDENT, 0, "x"),
		token.NewToken(token.ADD, 0, "+"),
		token.NewToken(token.INT, 0, "1"),
		tEOF,
	}},
	{"if then else Block", "if x = 5 then 0 else 1", []token.Token{
		token.NewToken(token.IF, 0, "if"),
		token.NewToken(token.IDENT, 0, "x"),
		token.NewToken(token.ASSIGN, 0, "="),
		token.NewToken(token.INT, 0, "5"),
		token.NewToken(token.THEN, 0, "then"),
		token.NewToken(token.INT, 0, "0"),
		token.NewToken(token.ELSE, 0, "else"),
		token.NewToken(token.INT, 0, "1"),
		tEOF,
	}},
	{"function expression assignment", "\nlet yo = fn x -> x + 1", []token.Token{
		token.NewToken(token.LET, 0, "let"),
		token.NewToken(token.IDENT, 0, "yo"),
		token.NewToken(token.ASSIGN, 0, "="),
		token.NewToken(token.FN, 0, "fn"),
		token.NewToken(token.IDENT, 0, "x"),
		token.NewToken(token.ARROW, 0, "->"),
		token.NewToken(token.IDENT, 0, "x"),
		token.NewToken(token.ADD, 0, "+"),
		token.NewToken(token.INT, 0, "1"),
		tEOF,
	}},
}

func collect(t *lexTest, left, right string) (tokens []token.Token) {
	l := lex(t.name, t.input, left, right)

	for {
		tok := l.nextToken()
		tokens = append(tokens, tok)
		if tok.Type() == token.EOF || tok.Type() == token.ILLEGAL {
			break
		}
	}
	return
}

func equal(i1, i2 []token.Token, checkPos bool) bool {

	if len(i1) != len(i2) {
		return false
	}
	for k := range i1 {
		if i1[k].Type() != i2[k].Type() {
			return false
		}
		if i1[k].Val() != i2[k].Val() {
			return false
		}
		if checkPos && i1[k].Pos() != i2[k].Pos() {
			return false
		}
	}
	return true
}

func TestLex(t *testing.T) {
	for _, test := range lexTests {
		tokens := collect(&test, "", "")
		if !equal(tokens, test.tokens, false) {
			t.Errorf("%s: got\n\t%+v\nexpected\n\t%v", test.name, tokens, test.tokens)
		}
	}
}
