package lexer

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/source"
	"github.com/puff-lang/puff/internal/token"
)

var (
	namespacePattern = regexp.MustCompile(`^[a-z0-9._-]+$`)
	tagPattern       = regexp.MustCompile(`^([a-z0-9._-]+:)?[a-z0-9/._-]+$`)
)

type Metadata struct {
	Namespace string
	Tags      []string
}

type Result struct {
	Metadata    Metadata
	Tokens      []token.Token
	Diagnostics []diagnostic.Diagnostic
}

type lexer struct {
	file        source.File
	input       string
	start       int
	current     int
	tokens      []token.Token
	diagnostics []diagnostic.Diagnostic
	metadata    Metadata
	seenMeta    map[string]bool
	atTop       bool
	parenDepth  int
	brackDepth  int
	braceDepth  int
}

func Lex(file source.File) Result {
	lexer := &lexer{
		file:     file,
		input:    file.Text,
		seenMeta: map[string]bool{},
		atTop:    true,
	}

	lexer.scan()

	return Result{
		Metadata:    lexer.metadata,
		Tokens:      lexer.tokens,
		Diagnostics: lexer.diagnostics,
	}
}

func (lexer *lexer) scan() {
	if strings.HasPrefix(lexer.input, "\uFEFF") {
		lexer.current += len("\uFEFF")
	}

	for !lexer.isAtEnd() {
		lexer.start = lexer.current
		lexer.scanToken()
	}

	if len(lexer.tokens) > 0 && lexer.tokens[len(lexer.tokens)-1].Type != token.Newline {
		lexer.start = lexer.current
		lexer.addToken(token.Newline, "\n", nil, lexer.current)
	}

	lexer.start = lexer.current
	lexer.addToken(token.EOF, "", nil, lexer.current)
}

func (lexer *lexer) scanToken() {
	char, size, ok := lexer.advanceRune()
	if !ok {
		lexer.report(diagnostic.CodeInvalidUTF8, "File is not valid UTF-8.", "Save the file as UTF-8.", lexer.start, lexer.start+1)
		lexer.current = lexer.start + 1
		return
	}

	switch char {
	case ' ', '\t':
		return
	case '\n':
		lexer.emitNewline()
	case '\r':
		if lexer.match('\n') {
			lexer.emitNewline()
			return
		}
		lexer.report(diagnostic.CodeInvalidLineEnding, "Invalid line ending.", "Use LF or CRLF line endings.", lexer.start, lexer.current)
	case '#':
		lexer.scanComment()
	case '(':
		lexer.parenDepth++
		lexer.addToken(token.LParen, lexer.input[lexer.start:lexer.current], nil, lexer.current)
	case ')':
		lexer.addToken(token.RParen, lexer.input[lexer.start:lexer.current], nil, lexer.current)
		lexer.parenDepth = max(lexer.parenDepth-1, 0)
	case '[':
		lexer.brackDepth++
		lexer.addToken(token.LBracket, lexer.input[lexer.start:lexer.current], nil, lexer.current)
	case ']':
		lexer.addToken(token.RBracket, lexer.input[lexer.start:lexer.current], nil, lexer.current)
		lexer.brackDepth = max(lexer.brackDepth-1, 0)
	case '{':
		lexer.braceDepth++
		lexer.addToken(token.LBrace, lexer.input[lexer.start:lexer.current], nil, lexer.current)
	case '}':
		lexer.addToken(token.RBrace, lexer.input[lexer.start:lexer.current], nil, lexer.current)
		lexer.braceDepth = max(lexer.braceDepth-1, 0)
	case ',':
		lexer.addToken(token.Comma, lexer.input[lexer.start:lexer.current], nil, lexer.current)
	case '.':
		if lexer.match('.') {
			lexer.addToken(token.DotDot, lexer.input[lexer.start:lexer.current], nil, lexer.current)
			return
		}
		if isDigit(lexer.peek()) {
			lexer.consumeDigits()
			lexer.report(diagnostic.CodeInvalidNumber, "Invalid number literal.", "Use an integer like 100 or a float like 100.0.", lexer.start, lexer.current)
			return
		}
		lexer.addToken(token.Dot, lexer.input[lexer.start:lexer.current], nil, lexer.current)
	case ':':
		lexer.addToken(token.Colon, lexer.input[lexer.start:lexer.current], nil, lexer.current)
	case '-':
		if lexer.match('>') {
			lexer.addToken(token.Arrow, lexer.input[lexer.start:lexer.current], nil, lexer.current)
			return
		}
		lexer.addToken(token.Minus, lexer.input[lexer.start:lexer.current], nil, lexer.current)
	case '=':
		if lexer.match('=') {
			lexer.addToken(token.EqualEqual, lexer.input[lexer.start:lexer.current], nil, lexer.current)
			return
		}
		lexer.addToken(token.Equal, lexer.input[lexer.start:lexer.current], nil, lexer.current)
	case '!':
		if lexer.match('=') {
			lexer.addToken(token.BangEqual, lexer.input[lexer.start:lexer.current], nil, lexer.current)
			return
		}
		lexer.reportInvalidCharacter()
	case '>':
		if lexer.match('=') {
			lexer.addToken(token.GreaterEq, lexer.input[lexer.start:lexer.current], nil, lexer.current)
			return
		}
		lexer.addToken(token.Greater, lexer.input[lexer.start:lexer.current], nil, lexer.current)
	case '<':
		if lexer.match('=') {
			lexer.addToken(token.LessEq, lexer.input[lexer.start:lexer.current], nil, lexer.current)
			return
		}
		lexer.addToken(token.Less, lexer.input[lexer.start:lexer.current], nil, lexer.current)
	case '+':
		lexer.addToken(token.Plus, lexer.input[lexer.start:lexer.current], nil, lexer.current)
	case '*':
		lexer.addToken(token.Star, lexer.input[lexer.start:lexer.current], nil, lexer.current)
	case '/':
		lexer.addToken(token.Slash, lexer.input[lexer.start:lexer.current], nil, lexer.current)
	case '%':
		lexer.addToken(token.Percent, lexer.input[lexer.start:lexer.current], nil, lexer.current)
	case '$':
		lexer.addToken(token.Dollar, lexer.input[lexer.start:lexer.current], nil, lexer.current)
	case '_':
		lexer.addToken(token.Underscore, lexer.input[lexer.start:lexer.current], nil, lexer.current)
	case '\uFEFF':
		lexer.reportInvalidCharacter()
	case '"', '\'':
		lexer.reportInvalidCharacter()
	default:
		if isDigit(byte(char)) {
			lexer.scanNumber()
			return
		}
		if isLetter(char) {
			lexer.scanIdentifier(size)
			return
		}
		lexer.reportInvalidCharacter()
	}
}

func (lexer *lexer) scanComment() {
	lineEnd := lexer.current
	for lineEnd < len(lexer.input) && lexer.input[lineEnd] != '\n' && lexer.input[lineEnd] != '\r' {
		lineEnd++
	}

	comment := lexer.input[lexer.start:lineEnd]
	if lexer.atTop && strings.Contains(comment, ":") {
		lexer.scanMetadata(comment)
	}

	lexer.current = lineEnd
}

func (lexer *lexer) scanMetadata(comment string) {
	content := strings.TrimSpace(strings.TrimPrefix(comment, "#"))
	key, value, found := strings.Cut(content, ":")
	if !found {
		return
	}

	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)

	switch key {
	case "namespace", "tags":
	default:
		lexer.report(diagnostic.CodeUnknownMetadataKey, fmt.Sprintf("Unknown metadata key: %s", key), "Supported metadata keys are namespace and tags.", lexer.start, lexer.start+len(comment))
		return
	}

	if lexer.seenMeta[key] {
		lexer.report(diagnostic.CodeDuplicateMetadataKey, fmt.Sprintf("Duplicate metadata key: %s", key), fmt.Sprintf("Declare %s only once.", key), lexer.start, lexer.start+len(comment))
		return
	}
	lexer.seenMeta[key] = true

	if key == "namespace" {
		if !namespacePattern.MatchString(value) {
			lexer.report(diagnostic.CodeInvalidMetadataValue, "Invalid metadata value for namespace.", "Use only lowercase letters, numbers, underscore, dash and dot.", lexer.start, lexer.start+len(comment))
			return
		}

		lexer.metadata.Namespace = value
		return
	}

	lexer.metadata.Tags = parseTags(value)
	for _, tag := range lexer.metadata.Tags {
		if !tagPattern.MatchString(tag) {
			lexer.report(diagnostic.CodeInvalidMetadataValue, "Invalid metadata value for tags.", "Use valid Minecraft resource locations separated by commas.", lexer.start, lexer.start+len(comment))
			return
		}
	}
}

func (lexer *lexer) emitNewline() {
	if lexer.parenDepth > 0 || lexer.brackDepth > 0 || lexer.braceDepth > 0 {
		return
	}
	if len(lexer.tokens) == 0 || lexer.tokens[len(lexer.tokens)-1].Type == token.Newline {
		return
	}

	lexer.addToken(token.Newline, "\n", nil, lexer.current)
}

func (lexer *lexer) scanNumber() {
	lexer.consumeDigits()

	if lexer.peek() == '.' {
		if lexer.peekNext() == '.' {
			value, _ := strconv.Atoi(lexer.input[lexer.start:lexer.current])
			lexer.addToken(token.Int, lexer.input[lexer.start:lexer.current], value, lexer.current)
			return
		}

		lexer.current++
		if !isDigit(lexer.peek()) {
			lexer.report(diagnostic.CodeInvalidNumber, "Invalid number literal.", "Use an integer like 100 or a float like 100.0.", lexer.start, lexer.current)
			return
		}

		lexer.consumeDigits()
		value, _ := strconv.ParseFloat(lexer.input[lexer.start:lexer.current], 64)
		lexer.addToken(token.Float, lexer.input[lexer.start:lexer.current], value, lexer.current)
		return
	}

	value, _ := strconv.Atoi(lexer.input[lexer.start:lexer.current])
	lexer.addToken(token.Int, lexer.input[lexer.start:lexer.current], value, lexer.current)
}

func (lexer *lexer) scanIdentifier(firstSize int) {
	_ = firstSize
	for isLetter(rune(lexer.peek())) || isDigit(lexer.peek()) || lexer.peek() == '_' {
		lexer.current++
	}

	lexeme := lexer.input[lexer.start:lexer.current]
	switch lexeme {
	case "nil":
		lexer.addToken(token.Nil, lexeme, nil, lexer.current)
	case "true":
		lexer.addToken(token.True, lexeme, true, lexer.current)
	case "false":
		lexer.addToken(token.False, lexeme, false, lexer.current)
	default:
		if tokenType, ok := keywords[lexeme]; ok {
			lexer.addToken(tokenType, lexeme, nil, lexer.current)
			return
		}
		lexer.addToken(token.Ident, lexeme, nil, lexer.current)
	}
}

func (lexer *lexer) addToken(tokenType token.Type, lexeme string, value any, end int) {
	line, column, _ := lexer.file.Map.LineColumn(lexer.start)
	lexer.tokens = append(lexer.tokens, token.Token{
		Type:        tokenType,
		Lexeme:      lexeme,
		Value:       value,
		Line:        line,
		Column:      column,
		StartOffset: lexer.start,
		EndOffset:   end,
	})

	if tokenType != token.Newline && tokenType != token.EOF {
		lexer.atTop = false
	}
}

func (lexer *lexer) reportInvalidCharacter() {
	lexer.report(diagnostic.CodeInvalidCharacter, fmt.Sprintf("Invalid character: %s", lexer.input[lexer.start:lexer.current]), "Remove the character or replace it with valid Puff syntax.", lexer.start, lexer.current)
}

func (lexer *lexer) report(code diagnostic.Code, message string, hint string, start int, end int) {
	startLine, startColumn, _ := lexer.file.Map.LineColumn(start)
	endLine, endColumn, _ := lexer.file.Map.LineColumn(end)
	lexer.diagnostics = append(lexer.diagnostics, diagnostic.Diagnostic{
		Code:     code,
		Phase:    diagnostic.PhaseLexer,
		Severity: diagnostic.SeverityError,
		Message:  message,
		Hint:     hint,
		File:     lexer.file.RelPath,
		Span: diagnostic.Span{
			StartLine:   startLine,
			StartColumn: startColumn,
			EndLine:     endLine,
			EndColumn:   endColumn,
			StartOffset: start,
			EndOffset:   end,
		},
	})
}

func (lexer *lexer) advanceRune() (rune, int, bool) {
	char, size := utf8.DecodeRuneInString(lexer.input[lexer.current:])
	if char == utf8.RuneError && size == 1 {
		return 0, 0, false
	}

	lexer.current += size
	return char, size, true
}

func (lexer *lexer) match(expected byte) bool {
	if lexer.isAtEnd() || lexer.input[lexer.current] != expected {
		return false
	}

	lexer.current++
	return true
}

func (lexer *lexer) consumeDigits() {
	for isDigit(lexer.peek()) {
		lexer.current++
	}
}

func (lexer *lexer) peek() byte {
	if lexer.isAtEnd() {
		return 0
	}

	return lexer.input[lexer.current]
}

func (lexer *lexer) peekNext() byte {
	if lexer.current+1 >= len(lexer.input) {
		return 0
	}

	return lexer.input[lexer.current+1]
}

func (lexer *lexer) isAtEnd() bool {
	return lexer.current >= len(lexer.input)
}

func parseTags(value string) []string {
	parts := strings.Split(value, ",")
	tags := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		tag := strings.TrimSpace(part)
		if tag == "" || seen[tag] {
			continue
		}

		tags = append(tags, tag)
		seen[tag] = true
	}

	return tags
}

func isDigit(char byte) bool {
	return char >= '0' && char <= '9'
}

func isLetter(char rune) bool {
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z')
}

var keywords = map[string]token.Type{
	"require":  token.Require,
	"as":       token.As,
	"pub":      token.Pub,
	"fun":      token.Fun,
	"on":       token.On,
	"end":      token.End,
	"if":       token.If,
	"else":     token.Else,
	"loop":     token.Loop,
	"times":    token.Times,
	"numbers":  token.Numbers,
	"players":  token.Players,
	"entities": token.Entities,
	"from":     token.From,
	"to":       token.To,
	"in":       token.In,
	"radius":   token.Radius,
	"around":   token.Around,
	"return":   token.Return,
	"stop":     token.Stop,
	"add":      token.Add,
	"and":      token.And,
	"or":       token.Or,
	"not":      token.Not,
}
