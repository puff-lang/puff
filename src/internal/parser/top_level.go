package parser

import (
	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/token"
)

func (parser *parser) parseRequire() *ast.RequireDecl {
	start := parser.advance().StartOffset
	path := parser.parseString()

	var alias *ast.Identifier
	if parser.match(token.As) {
		if parser.checkName() {
			alias = parser.parseIdentifier()
		} else {
			parser.reportExpected("identifier", "")
		}
	}

	end := parser.lineEndOffset(start)
	parser.requireLineEnd()
	return &ast.RequireDecl{
		NodeBase: parser.base(start, end),
		Path:     path,
		Alias:    alias,
	}
}

func (parser *parser) parseFunction(public bool) *ast.FunctionDecl {
	start := parser.peek().StartOffset
	if public {
		parser.advance()
	}
	parser.advance()

	name := ast.Identifier{}
	if parser.checkName() {
		name = *parser.parseIdentifier()
	} else {
		parser.reportExpected("function name", "")
	}

	var parameters []ast.Parameter
	if parser.match(token.LParen) {
		parameters = parser.parseParameters()
	}

	var returnType *ast.TypeRef
	if parser.match(token.Arrow) {
		returnType = parser.parseType()
	}

	parser.requireLineEnd()
	body := parser.parseBlock(false)
	end := parser.consumeBlockEnd(start)
	return &ast.FunctionDecl{
		NodeBase:   parser.base(start, end),
		Public:     public,
		Name:       name,
		Parameters: parameters,
		ReturnType: returnType,
		Body:       body,
	}
}

func (parser *parser) parseParameters() []ast.Parameter {
	var parameters []ast.Parameter
	for !parser.check(token.RParen) && !parser.check(token.Arrow) && !parser.check(token.Newline) && !parser.atEnd() {
		start := parser.peek().StartOffset
		if !parser.checkName() {
			parser.reportExpected("parameter name", "")
			parser.synchronizeUntil(token.Comma, token.RParen, token.Newline)
		} else {
			name := parser.parseIdentifier()
			var parameterType *ast.TypeRef
			if parser.match(token.Colon) {
				parameterType = parser.parseType()
			}
			end := parser.previous().EndOffset
			parameters = append(parameters, ast.Parameter{
				NodeBase: parser.base(start, end),
				Name:     *name,
				Type:     parameterType,
			})
		}

		if !parser.match(token.Comma) {
			break
		}
		if parser.check(token.RParen) {
			parser.reportExpected("parameter name", "")
			break
		}
	}

	if !parser.match(token.RParen) {
		parser.reportExpected(")", "Close the parameter list before the return type.")
	}
	return parameters
}

func (parser *parser) parseType() *ast.TypeRef {
	start := parser.peek().StartOffset
	if !parser.checkName() {
		parser.reportExpected("type", "")
		return nil
	}

	name := parser.parseIdentifier()
	typeRef := &ast.TypeRef{Name: *name}
	if parser.match(token.Less) {
		if parser.check(token.Greater) {
			parser.reportExpected("type", "")
		}
		for !parser.check(token.Greater) && !parser.check(token.Newline) && !parser.atEnd() {
			argument := parser.parseType()
			if argument != nil {
				typeRef.Arguments = append(typeRef.Arguments, argument)
			}
			if !parser.match(token.Comma) {
				break
			}
			if parser.check(token.Greater) {
				parser.reportExpected("type", "")
				break
			}
		}
		if !parser.match(token.Greater) {
			parser.reportExpected(">", "")
		}
	}

	typeRef.NodeBase = parser.base(start, parser.previous().EndOffset)
	return typeRef
}

func (parser *parser) parseEvent() *ast.EventDecl {
	start := parser.advance().StartOffset
	var name []ast.Identifier
	for parser.checkName() {
		name = append(name, *parser.parseIdentifier())
	}
	if len(name) == 0 {
		parser.reportExpected("event name", "")
	}

	parser.requireLineEnd()
	body := parser.parseBlock(false)
	end := parser.consumeBlockEnd(start)
	return &ast.EventDecl{
		NodeBase: parser.base(start, end),
		Name:     name,
		Body:     body,
	}
}

func (parser *parser) parseGlobal(public bool) *ast.GlobalAssignment {
	start := parser.peek().StartOffset
	if public {
		parser.advance()
	}
	target, _ := parser.parseVariable(nil).(*ast.VariableExpr)
	if target != nil && target.Local && target.Name.Name != "" && !public {
		parser.report(
			diagnostic.CodeInvalidTopLevelStatement,
			"Executable statements are not allowed at the top level.",
			"Move this statement into an event or function.",
			target.Span().StartOffset,
			target.Span().EndOffset,
		)
	}
	if !parser.match(token.Equal) {
		parser.reportExpected("=", "")
	}
	value := parser.parseExpressionUntil(token.Newline)
	end := parser.statementEnd(start, value)
	parser.requireLineEnd()

	return &ast.GlobalAssignment{
		NodeBase: parser.base(start, end),
		Public:   public,
		Target:   target,
		Value:    value,
	}
}

func (parser *parser) parseString() *ast.StringExpr {
	if !parser.check(token.StringStart) {
		parser.reportExpected("string", "")
		return nil
	}
	expression, _ := parser.parseStringExpression().(*ast.StringExpr)
	return expression
}

func (parser *parser) parseIdentifier() *ast.Identifier {
	tok := parser.advance()
	return &ast.Identifier{
		NodeBase: parser.base(tok.StartOffset, tok.EndOffset),
		Name:     tok.Lexeme,
	}
}

func (parser *parser) checkName() bool {
	switch parser.peek().Type {
	case token.Ident,
		token.Require,
		token.As,
		token.Pub,
		token.Fun,
		token.On,
		token.End,
		token.If,
		token.Else,
		token.Loop,
		token.Times,
		token.Numbers,
		token.Players,
		token.Entities,
		token.From,
		token.To,
		token.In,
		token.Radius,
		token.Around,
		token.Return,
		token.Stop,
		token.Add,
		token.And,
		token.Or,
		token.Not:
		return true
	default:
		return false
	}
}

func (parser *parser) requireLineEnd() {
	if parser.match(token.Newline) || parser.atEnd() {
		return
	}
	if parser.current > 0 && parser.peek().Line > parser.previous().Line {
		return
	}
	tok := parser.peek()
	parser.report(
		diagnostic.CodeExpectedNewline,
		"Expected newline.",
		"",
		tok.StartOffset,
		tok.EndOffset,
	)
	parser.synchronizeLine()
	parser.match(token.Newline)
}

func (parser *parser) lineEndOffset(fallback int) int {
	if parser.atEnd() || parser.check(token.Newline) {
		if parser.current == 0 {
			return fallback
		}
		return parser.previous().EndOffset
	}
	index := parser.current
	for index < len(parser.tokens) && parser.tokens[index].Type != token.Newline && parser.tokens[index].Type != token.EOF {
		index++
	}
	if index == parser.current {
		return fallback
	}
	return parser.tokens[index-1].EndOffset
}

func (parser *parser) synchronizeUntil(types ...token.Type) {
	for !parser.atEnd() {
		for _, tokenType := range types {
			if parser.check(tokenType) {
				return
			}
		}
		parser.advance()
	}
}
