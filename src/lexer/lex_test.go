package lexer

import (
	"fmt"
	"testing"
)

var tokenName = [...]string{
    ILLEGAL: "ILLEGAL",

    EOF:     "EOF",
    COMMENT: "COMMENT",

    IDENT:  "IDENT",
    INT:    "INT",
    BOOLEAN:"BOOL",
    FLOAT:  "FLOAT",
    CHAR:   "CHAR",
    STRING: "STRING",

    ASSIGN: "=",

    ADD: "+",
    SUB: "-",
    MUL: "*",
    QUO: "/",
    REM: "%",

    AND:     "&",
    OR:      "|",
    XOR:     "^",
    SHL:     "<<",
    SHR:     ">>",

    LAND:  "&&",
    LOR:   "||",

    EQL:    "==",
    LSS:    "<",
    GTR:    ">",
    NOT:    "!",

    LPAREN: "(",
    LBRACK: "[",
    LBRACE: "{",
    COMMA:  ",",
    PERIOD: ".",

    RPAREN:    ")",
    RBRACK:    "]",
    RBRACE:    "}",
    SEMICOLON: ";",
    COLON:     ":",

    FN: "fn",
    LET:  "let",
    TYPE: "type",
    DATA: "data",

    WHITESPACE: " \t\r\n",
}

func (i tokenType) String() string {
	s := tokenName[i]
	if s == "" {
		return fmt.Sprintf("token%d", int(i))
	}
	return s
}

type lexTest struct {
	name  string
	input string
	tokens []token
}

var (
	tEOF        = token{EOF, 0, ""}
	tLParen     = token{LPAREN, 0, "("}
	tRParen     = token{RPAREN, 0, ")"}
)

var lexTests = []lexTest{
	{"empty", "", []token{tEOF}},
	{"number", "2", []token{{INT, 0, "2"}, 	token{EOF, 0, ""}}},
	{"add", "2 + 2", []token{
		{INT, 0, "2"},
		{ADD, 0, "+"},
		{INT, 0, "2"},
		tEOF,
	}},
	{"expr in paren", "(2 + 2)", []token{
		tLParen,
		{INT, 0, "2"},
		{ADD, 0, "+"},
		{INT, 0, "2"},
		tRParen,
		tEOF,
	}},
	{"number assignment", "let x = 5", []token{
		{LET, 0, "let"},
		{IDENT, 0, "x"},
		{ASSIGN, 0, "="},
		{INT, 0, "5"},
		tEOF,
	}},
	{"string assignment", "let x = \"hello\"", []token{
		{LET, 0, "let"},
		{IDENT, 0, "x"},
		{ASSIGN, 0, "="},
		{STRING, 0, "\"hello\""},
		tEOF,
	}},
	{"function expression", "fn x -> x + 1", []token{
		{FN, 0, "fn"},
		{IDENT, 0, "x"},
		{ARROW, 0, "->"},
		{IDENT, 0, "x"},
		{ADD, 0, "+"},
		{INT, 0, "1"},
		tEOF,
	}},
	{"function expression assignment", "let yo = fn x -> x + 1", []token{
		{LET, 0, "let"},
		{IDENT, 0, "yo"},
		{ASSIGN, 0, "="},
		{FN, 0, "fn"},
		{IDENT, 0, "x"},
		{ARROW, 0, "->"},
		{IDENT, 0, "x"},
		{ADD, 0, "+"},
		{INT, 0, "1"},
		tEOF,
	}},
}

func collect(t *lexTest, left, right string) (tokens []token) {
	l := lex(t.name, t.input, left, right)

	for {
		token := l.nextToken()
		tokens = append(tokens, token)
		if token.typ == EOF || token.typ == ILLEGAL {
			break
		}
	}
	return
}

func equal(i1, i2 []token, checkPos bool) bool {

	if len(i1) != len(i2) {
		return false
	}
	for k := range i1 {
		if i1[k].typ != i2[k].typ {
			return false
		}
		if i1[k].val != i2[k].val {
			return false
		}
		if checkPos && i1[k].pos != i2[k].pos {
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