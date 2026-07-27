package parser

import (
	"fmt"
	"strings"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/lexer"
	"github.com/puff-lang/puff/internal/source"
	"github.com/puff-lang/puff/internal/token"
)

type Result struct {
	File        *ast.File
	Diagnostics []diagnostic.Diagnostic
}

type parser struct {
	file        source.File
	tokens      []token.Token
	current     int
	diagnostics []diagnostic.Diagnostic
}

func Parse(file source.File, lexed lexer.Result) Result {
	parser := &parser{
		file:        file,
		tokens:      lexed.Tokens,
		diagnostics: append([]diagnostic.Diagnostic(nil), lexed.Diagnostics...),
	}

	return Result{
		File:        parser.parseFile(lexed.Metadata),
		Diagnostics: parser.diagnostics,
	}
}

func (parser *parser) parseFile(metadata lexer.Metadata) *ast.File {
	file := &ast.File{
		NodeBase: ast.NodeBase{SourceSpan: parser.spanFromOffsets(0, len(parser.file.Text))},
		Metadata: metadataEntries(metadata),
	}

	parser.skipNewlines()
	seenDeclaration := false
	for !parser.atEnd() {
		switch {
		case parser.check(token.Require) && !seenDeclaration:
			file.Requirements = append(file.Requirements, parser.parseRequire())
		case parser.check(token.Require):
			parser.reportUnexpected(parser.peek(), "")
			parser.synchronizeLine()
		case parser.check(token.Fun):
			seenDeclaration = true
			file.Declarations = append(file.Declarations, parser.parseFunction(false))
		case parser.check(token.On):
			seenDeclaration = true
			file.Declarations = append(file.Declarations, parser.parseEvent())
		case parser.check(token.Dollar):
			seenDeclaration = true
			file.Declarations = append(file.Declarations, parser.parseGlobal(false))
		case parser.check(token.Pub):
			seenDeclaration = true
			if parser.peekNext().Type == token.Fun {
				file.Declarations = append(file.Declarations, parser.parseFunction(true))
			} else if parser.peekNext().Type == token.Dollar {
				file.Declarations = append(file.Declarations, parser.parseGlobal(true))
			} else {
				parser.reportUnexpected(parser.peek(), "")
				parser.synchronizeLine()
			}
		case parser.check(token.Else), parser.check(token.End):
			hint := ""
			if parser.check(token.Else) {
				hint = "else can only appear inside an if block."
			}
			parser.reportUnexpected(parser.peek(), hint)
			parser.synchronizeLine()
		default:
			parser.reportInvalidTopLevel()
			parser.synchronizeLine()
		}
		parser.skipNewlines()
	}

	return file
}

func metadataEntries(metadata lexer.Metadata) []ast.MetadataEntry {
	entries := make([]ast.MetadataEntry, 0, 2)
	if metadata.Namespace != "" {
		entries = append(entries, ast.MetadataEntry{Key: "namespace", Value: metadata.Namespace})
	}
	if len(metadata.Tags) > 0 {
		entries = append(entries, ast.MetadataEntry{Key: "tags", Value: strings.Join(metadata.Tags, ", ")})
	}
	return entries
}

func (parser *parser) reportExpected(expected string, hint string) {
	tok := parser.peek()
	parser.report(
		diagnostic.CodeExpectedToken,
		fmt.Sprintf("Expected %q.", expected),
		hint,
		tok.StartOffset,
		tok.EndOffset,
	)
}

func (parser *parser) reportUnexpected(tok token.Token, hint string) {
	parser.report(
		diagnostic.CodeUnexpectedToken,
		fmt.Sprintf("Unexpected token: %s", tok.Lexeme),
		hint,
		tok.StartOffset,
		tok.EndOffset,
	)
}

func (parser *parser) reportInvalidTopLevel() {
	start := parser.peek().StartOffset
	end := start
	for !parser.check(token.Newline) && !parser.atEnd() {
		end = parser.advance().EndOffset
	}
	parser.report(
		diagnostic.CodeInvalidTopLevelStatement,
		"Executable statements are not allowed at the top level.",
		"Move this statement into an event or function.",
		start,
		end,
	)
}

func (parser *parser) report(code diagnostic.Code, message string, hint string, start int, end int) {
	parser.diagnostics = append(parser.diagnostics, diagnostic.Diagnostic{
		Code:     code,
		Phase:    diagnostic.PhaseParser,
		Severity: diagnostic.SeverityError,
		Message:  message,
		Hint:     hint,
		File:     parser.file.RelPath,
		Span:     parser.spanFromOffsets(start, end),
	})
}

func (parser *parser) spanFromOffsets(start int, end int) diagnostic.Span {
	startLine, startColumn, _ := parser.file.Map.LineColumn(start)
	endLine, endColumn, _ := parser.file.Map.LineColumn(end)
	return diagnostic.Span{
		StartLine:   startLine,
		StartColumn: startColumn,
		EndLine:     endLine,
		EndColumn:   endColumn,
		StartOffset: start,
		EndOffset:   end,
	}
}

func (parser *parser) base(start int, end int) ast.NodeBase {
	return ast.NodeBase{SourceSpan: parser.spanFromOffsets(start, end)}
}

func (parser *parser) skipNewlines() {
	for parser.match(token.Newline) {
	}
}

func (parser *parser) synchronizeLine() {
	for !parser.check(token.Newline) && !parser.atEnd() {
		parser.advance()
	}
}

func (parser *parser) match(types ...token.Type) bool {
	for _, tokenType := range types {
		if parser.check(tokenType) {
			parser.advance()
			return true
		}
	}
	return false
}

func (parser *parser) check(tokenType token.Type) bool {
	return parser.peek().Type == tokenType
}

func (parser *parser) advance() token.Token {
	if !parser.atEnd() {
		parser.current++
	}
	return parser.tokens[parser.current-1]
}

func (parser *parser) peek() token.Token {
	if len(parser.tokens) == 0 {
		return token.Token{Type: token.EOF}
	}
	return parser.tokens[parser.current]
}

func (parser *parser) peekNext() token.Token {
	if parser.current+1 >= len(parser.tokens) {
		return parser.peek()
	}
	return parser.tokens[parser.current+1]
}

func (parser *parser) previous() token.Token {
	return parser.tokens[parser.current-1]
}

func (parser *parser) atEnd() bool {
	return parser.peek().Type == token.EOF
}
