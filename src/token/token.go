package token

import "fmt"

type Token struct {
	typ TokenType
	pos int
	val string
}

type TokenType int

// List of tokens
const (
	ILLEGAL TokenType = iota
	EOF
	COMMENT
	WHITESPACE

	IDENT   // main
	INT     // 12345
	BOOLEAN // True
	FLOAT   // 123.45
	CHAR    // 'a'
	STRING  // "abc"

	ASSIGN // =

	ADD // +
	SUB // -
	MUL // *
	QUO // /
	REM // %

	AND // &
	OR  // |
	XOR // ^
	SHL // <<
	SHR // >>

	LAND // &&
	LOR  // ||

	EQL // ==
	LSS // <
	GTR // >
	NOT // !

	LPAREN // (
	LBRACK // [
	LBRACE // {
	COMMA  // ,
	PERIOD // .

	RPAREN    // )
	RBRACK    // ]
	RBRACE    // }
	SEMICOLON // ;
	COLON     // :
	ARROW     // ->

	Keyword // Delimiter for keywords
	FN
	LET
	IN
	TYPE
	DATA

	IF 		//if
	THEN 	//then
	ELSE	//else
)

var Tokens = [...]string{
	ILLEGAL: "ILLEGAL",

	EOF:     "EOF",
	COMMENT: "COMMENT",

	IDENT:   "IDENT",
	INT:     "INT",
	BOOLEAN: "BOOL",
	FLOAT:   "FLOAT",
	CHAR:    "CHAR",
	STRING:  "STRING",

	ASSIGN: "=",

	ADD: "+",
	SUB: "-",
	MUL: "*",
	QUO: "/",
	REM: "%",

	AND: "&",
	OR:  "|",
	XOR: "^",
	SHL: "<<",
	SHR: ">>",

	LAND: "&&",
	LOR:  "||",

	EQL: "==",
	LSS: "<",
	GTR: ">",
	NOT: "!",

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

	FN:   "fn",
	LET:  "let",
	IN:   "in",
	TYPE: "type",
	DATA: "data",

	WHITESPACE: " \t\r\n",

	IF:		"if",
	THEN: 	"then",
	ELSE:	"else",
}

var Keywords = map[string]TokenType{
	"fn":   FN,
	"let":  LET,
	"in":   IN,
	"type": TYPE,
	"data": DATA,
	"if":	IF,
	"then":	THEN,
	"else":	ELSE,
}

func NewToken(typ TokenType, pos int, val string) Token {
	return Token{typ, pos, val}
}

func (t *Token) Pos() int {
	return t.pos
}

func (t *Token) Type() TokenType {
	return t.typ
}

func (t *Token) Val() string {
	return t.val
}

func (i TokenType) String() string {
	s := Tokens[i]
	if s == "" {
		return fmt.Sprintf("token%d", int(i))
	}
	return s
}