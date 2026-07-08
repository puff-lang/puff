package token

type Type string

const (
	EOF     Type = "EOF"
	Newline Type = "NEWLINE"

	Ident       Type = "IDENT"
	Int         Type = "INT"
	Float       Type = "FLOAT"
	StringStart Type = "STRING_START"
	StringText  Type = "STRING_TEXT"
	StringEnd   Type = "STRING_END"
	InterpStart Type = "INTERP_START"
	InterpEnd   Type = "INTERP_END"
	Nil         Type = "NIL"
	True        Type = "TRUE"
	False       Type = "FALSE"

	Require  Type = "REQUIRE"
	As       Type = "AS"
	Pub      Type = "PUB"
	Fun      Type = "FUN"
	On       Type = "ON"
	End      Type = "END"
	If       Type = "IF"
	Else     Type = "ELSE"
	Loop     Type = "LOOP"
	Times    Type = "TIMES"
	Numbers  Type = "NUMBERS"
	Players  Type = "PLAYERS"
	Entities Type = "ENTITIES"
	From     Type = "FROM"
	To       Type = "TO"
	In       Type = "IN"
	Radius   Type = "RADIUS"
	Around   Type = "AROUND"
	Return   Type = "RETURN"
	Stop     Type = "STOP"
	Add      Type = "ADD"
	And      Type = "AND"
	Or       Type = "OR"
	Not      Type = "NOT"

	LParen     Type = "LPAREN"
	RParen     Type = "RPAREN"
	LBracket   Type = "LBRACKET"
	RBracket   Type = "RBRACKET"
	LBrace     Type = "LBRACE"
	RBrace     Type = "RBRACE"
	Comma      Type = "COMMA"
	Dot        Type = "DOT"
	DotDot     Type = "DOT_DOT"
	Colon      Type = "COLON"
	Arrow      Type = "ARROW"
	Equal      Type = "EQUAL"
	EqualEqual Type = "EQ_EQ"
	BangEqual  Type = "BANG_EQ"
	Greater    Type = "GT"
	GreaterEq  Type = "GT_EQ"
	Less       Type = "LT"
	LessEq     Type = "LT_EQ"
	Plus       Type = "PLUS"
	Minus      Type = "MINUS"
	Star       Type = "STAR"
	Slash      Type = "SLASH"
	Percent    Type = "PERCENT"
	Dollar     Type = "DOLLAR"
	Underscore Type = "UNDERSCORE"
)

type Token struct {
	Type        Type
	Lexeme      string
	Value       any
	Line        int
	Column      int
	StartOffset int
	EndOffset   int
}
