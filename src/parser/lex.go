package parse

import (
    "fmt"
    "strings"
    "unicode"
    "unicode/utf8"
)

type token struct {
    typ tokenType
    pos int
    val string
}

type tokenType int

// List of tokens
const (
	ILLEGAL tokenType = iota
	EOF
	COMMENT
    WHITESPACE

	IDENT  // main
    INT    // 12345
    BOOLEAN // True
    FLOAT  // 123.45
    CHAR   // 'a'
    STRING // "abc" 

    ASSIGN // =

    ADD // +
    SUB // -
    MUL // *
    QUO // /
    REM // %

    AND     // &
    OR      // |
    XOR     // ^
    SHL     // <<
    SHR     // >>

    LAND  // &&
    LOR   // ||

    EQL    // ==
    LSS    // <
    GTR    // >
    NOT    // !

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

    Keywords  // Delimiter for keywords
    FN
    LET
    IN
    TYPE
    DATA
)


var tokens = [...]string{
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

    FN:   "fn",
    LET:  "let",
    IN:   "in",
    TYPE: "type",
    DATA: "data",

    WHITESPACE: " \t\r\n",
}

var keywords = map[string]tokenType {
    "fn": FN,
    "let": LET,
    "in": IN,
    "type": TYPE,
    "data": DATA,
}

const eof = -1

type LexFn func(*lexer) LexFn

type lexer struct {
    name       string
    input      string
    tokens     chan token
    state      LexFn

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
func (l *lexer) emit(t tokenType) {
    l.tokens <- token{t, l.start, l.input[l.start:l.pos]}
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
    l.tokens <- token{ILLEGAL, l.start, fmt.Sprintf(format, args...)}
    return nil
}

// nextToken returns the next item from the input.
// Called by the parser, not in the lexing goroutine.
func (l *lexer) nextToken() token {
    token := <-l.tokens
    l.lastPos = token.pos
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
        tokens: make(chan token),
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
    ch := l.next();
    switch {
    case isLetter(ch):
        l.backup()
        return lexIdentifier;
    case isSpace(ch):
        l.backup()
        return lexSpace
    case ch == '=':
        return lexEqual
    case ch == eof:
        l.emit(EOF)
        return nil
    default:
        l.backup()
        return lexExpr
    }
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
            case keywords[word] > Keywords:
                l.emit(keywords[word])
            case word == "True", word == "False":
                l.emit(BOOLEAN)
            default:
                l.emit(IDENT)
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
        l.emit(LPAREN)
        l.parenDepth++
    case r == ')':
        l.emit(RPAREN)
        l.parenDepth--
        if l.parenDepth < 0 {
            return l.errorf("unexpected right paren %#U", r)
        }
    case r == '{':
        l.emit(LBRACE)
        return lexStatement
    case r == '}':
        l.emit(RBRACE)
        return lexStatement
    case r == '+':
        l.emit(ADD)
        return lexStatement
    case r == '-':
        n := l.next()
        if n == '>' {
            l.emit(ARROW)
        } else {
            l.backup()
            l.emit(SUB)
        }
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
    l.emit(CHAR)
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
    l.emit(STRING)
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
        if (seenDecimalPoint) {
            l.emit(FLOAT)
        } else {
            l.emit(INT)
        }
    }
    
    return lexStatement
}

func lexSpace(l *lexer) LexFn {
Loop:
    for {
        switch r := l.next(); {
        case isSpace(r):
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
        l.emit(EQL)
    } else {
        l.backup()
        l.emit(ASSIGN)
    }

    return lexStatement
}

// isSpace reports whether r is a space character.
func isSpace(r rune) bool {
    return r == ' ' || r == '\t'
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