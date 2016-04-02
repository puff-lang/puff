package parse

import (
	"fmt"
	"strings"
	"token"
	"unicode"
	"unicode/utf8"
)

const eof = -1

type LexFn func(*lexer) LexFn

type lexer struct {
	name   string
	input  string
	tokens chan token.Token
	state  LexFn

	start      int
	pos        int
	width      int
	lastPos    int
	parenDepth int
}

// next returns the next rune in the input.
func (l *lexer) next() rune {
	if int(l.pos) >= len(l.input) {
		l.width = 0
		return eof
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	// l.width = Pos(w)
	l.width = w
	l.pos += l.width
	return r
}

// peek returns but does not consume the next rune in the input.
func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// backup steps back one rune. Can only be called once per call of next.
func (l *lexer) backup() {
	l.pos -= l.width
}

// emit passes an item back to the client.
func (l *lexer) emit(t token.TokenType) {
	l.tokens <- token.NewToken(t, l.start, l.input[l.start:l.pos])
	l.start = l.pos
}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.start = l.pos
}

// accept consumes the next rune if it's from the valid set.
func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}


func (l *lexer) skipWhitespace() {
	r := l.peek()
	for r == ' ' || r == '\t' || r == '\n' || r == '\r' {
		l.next()
	}
}

// acceptRun consumes a run of runes from the valid set.
func (l *lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}
	l.backup()
}

// lineNumber reports which line we're on, based on the position of
// the previous item returned by nextItem. Doing it this way
// means we don't have to worry about peek double counting.
func (l *lexer) lineNumber() int {
	return 1 + strings.Count(l.input[:l.lastPos], "\n")
}

// errorf returns an error token and terminates the scan by passing
// back a nil pointer that will be the next state, terminating l.nextItem.
func (l *lexer) errorf(format string, args ...interface{}) LexFn {
	l.tokens <- token.NewToken(token.ILLEGAL, l.start, fmt.Sprintf(format, args...))
	return nil
}

// nextToken returns the next item from the input.
// Called by the parser, not in the lexing goroutine.
func (l *lexer) nextToken() token.Token {
	token := <-l.tokens
	l.lastPos = token.Pos()
	return token
}

// drain drains the output so the lexing goroutine will exit.
// Called by the parser, not in the lexing goroutine.
/*
func (l *lexer) drain() {
    for range l.tokens {
    }
}
*/

// lex creates a new scanner for the input string.
func lex(name, input, left, right string) *lexer {
	l := &lexer{
		name:   name,
		input:  input,
		state:  lexStatement,
		tokens: make(chan token.Token),
	}
	go l.run()
	return l
}

// run runs the state machine for the lexer.
func (l *lexer) run() {
	for l.state = lexStatement; l.state != nil; {
		l.state = l.state(l)
	}
	close(l.tokens)
}

func lexStatement(l *lexer) LexFn {
	// l.skipBlankLines()
	ch := l.next()
	switch {
	case isLetter(ch):
		l.backup()
		return lexIdentifier
	case isSpace(ch):
		l.backup()
		return lexSpace
	case ch == '=':
		n := l.next()
		if n == '>' {
			l.emit(token.ARROW)
			return lexStatement
		} else {
			l.backup()
			return lexEqual
		}
	case ch == '/':
		nextChar := l.peek()
		if nextChar == '/' {
			return lexLineComment
		}else if  nextChar == '*' {
			return lexBlockComment	
		} 
		fallthrough
	case ch == eof:
		l.emit(token.EOF)
		return nil
	default:
		l.backup()
		return lexExpr
	}
}

func lexLineComment(l *lexer) LexFn {
	r := l.next()
	Loop:
	for {
		r = l.peek()
	 	if r == '\n' {
	 		l.emit(token.COMMENT)
	 		break Loop
	 	} else {
	 		l.next()
	 	}
	}
	return lexStatement
}

func  lexBlockComment(l *lexer) LexFn {
	r := l.next()
	Loop:
	for {
	 	if r == '*' {
	 		if l.next() == '/' {
	 			l.emit(token.COMMENT)
	 			break Loop
	 		}	 		
	 	}
	 	fmt.Printf("%c", r)
	 	r = l.next()
	}
	return lexStatement
}

func lexIdentifier(l *lexer) LexFn {
Loop:
	for {
		switch r := l.next(); {
		case isAlphaNumeric(r):
			// absorb
		default:
			l.backup()
			word := l.input[l.start:l.pos]
			if !l.atTerminator() {
				return l.errorf("bad character %#U", r)
			}
			switch {
			case token.Keywords[word] > token.Keyword:
				l.emit(token.Keywords[word])
			case word == "True", word == "False":
				l.emit(token.BOOLEAN)
			default:
				l.emit(token.IDENT)
			}
			break Loop
		}
	}
	return lexStatement
}

func lexExpr(l *lexer) LexFn {
	r := l.next()
	switch {
	case ('0' <= r && r <= '9'):
		l.backup()
		return lexNumber
	case r == '(':
		l.emit(token.LPAREN)
		l.parenDepth++
	case r == ')':
		l.emit(token.RPAREN)
		l.parenDepth--
		if l.parenDepth < 0 {
			return l.errorf("unexpected right paren %#U", r)
		}
	case r == '{':
		l.emit(token.LBRACE)
		return lexStatement
	case r == '}':
		l.emit(token.RBRACE)
		return lexStatement
	case r == '+':
		l.emit(token.ADD)
		return lexStatement
	case r == ',':
		l.emit(token.COMMA)
		return lexStatement
	case r == '-':
		l.emit(token.SUB)
		return lexStatement
	case r == '"':
		return lexQuote
	case r == '\'':
		return lexChar
	}
	return lexStatement
}

// lexChar scans a character constant. The initial quote is already
// scanned. Syntax checking is done by the parser.
func lexChar(l *lexer) LexFn {
Loop:
	for {
		switch l.next() {
		case '\\':
			if r := l.next(); r != eof && r != '\n' {
				break
			}
			fallthrough
		case eof, '\n':
			return l.errorf("unterminated character constant")
		case '\'':
			break Loop
		}
	}
	l.emit(token.CHAR)
	return lexStatement
}

// lexQuote scans a quoted string.
func lexQuote(l *lexer) LexFn {
Loop:
	for {
		switch l.next() {
		case '\\':
			if r := l.next(); r != eof && r != '\n' {
				break
			}
			fallthrough
		case eof, '\n':
			return l.errorf("unterminated quoted string")
		case '"':
			break Loop
		}
	}
	l.emit(token.STRING)
	return lexStatement
}

func lexNumber(l *lexer) LexFn {
	seenDecimalPoint := false
	// Optional leading sign.
	// l.accept("+-")
	// Is it hex?
	digits := "0123456789"
	if l.accept("0") && l.accept("xX") {
		digits = "0123456789abcdefABCDEF"
	}
	l.acceptRun(digits)
	if l.accept(".") {
		seenDecimalPoint = true
		l.acceptRun(digits)
	}
	if l.accept("eE") {
		l.accept("+-")
		l.acceptRun("0123456789")
	}
	// Next thing mustn't be alphanumeric.
	if isAlphaNumeric(l.peek()) {
		l.next()
		return l.errorf("bad number syntax: %q", l.input[l.start:l.pos])
	} else {
		if seenDecimalPoint {
			l.emit(token.FLOAT)
		} else {
			l.emit(token.INT)
		}
	}

	return lexStatement
}

func lexSpace(l *lexer) LexFn {
Loop:
	for {
		switch r := l.next(); {
		case isSpace(r):
			fmt.Println(r)
			l.ignore()
		default:
			l.backup()
			break Loop
		}
	}
	return lexStatement
}

func lexEqual(l *lexer) LexFn {
	ch := l.next()

	if ch == '=' {
		l.emit(token.EQL)
	} else {
		l.backup()
		l.emit(token.ASSIGN)
	}

	return lexStatement
}

// isSpace reports whether r is a space character.
func isSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n'
}

// isEndOfLine reports whether r is an end-of-line character.
func isEndOfLine(r rune) bool {
	return r == '\r' || r == '\n'
}

// isLetter reports whether r is an alphabetic.
func isLetter(r rune) bool {
	return unicode.IsLetter(r)
}

// isAlphaNumeric reports whether r is an alphabetic, digit, or underscore.
func isAlphaNumeric(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// atTerminator reports whether the input is at valid termination character to
// appear after an identifier.
func (l *lexer) atTerminator() bool {
	r := l.peek()
	if isSpace(r) || isEndOfLine(r) {
		return true
	}
	switch r {
	case eof, '.', ',', '|', ':', ')', '(':
		return true
	}
	return false
}
