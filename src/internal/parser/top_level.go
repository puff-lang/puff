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
	body, end := parser.scanBlock(start)
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
		for !parser.check(token.Greater) && !parser.check(token.Newline) && !parser.atEnd() {
			argument := parser.parseType()
			if argument != nil {
				typeRef.Arguments = append(typeRef.Arguments, argument)
			}
			if !parser.match(token.Comma) {
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
	body, end := parser.scanBlock(start)
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
	target := parser.parseGlobalVariable()
	if !parser.match(token.Equal) {
		parser.reportExpected("=", "")
	}

	expressionStart := parser.current
	end := parser.lineEndOffset(start)
	for !parser.check(token.Newline) && !parser.atEnd() {
		parser.advance()
	}
	var value ast.Expression
	if expressionStart < parser.current {
		value = parser.parseDeferredExpression(parser.tokens[expressionStart:parser.current])
	}
	parser.requireLineEnd()

	return &ast.GlobalAssignment{
		NodeBase: parser.base(start, end),
		Public:   public,
		Target:   target,
		Value:    value,
	}
}

func (parser *parser) parseGlobalVariable() *ast.VariableExpr {
	start := parser.peek().StartOffset
	if !parser.match(token.Dollar) {
		parser.reportExpected("$", "")
		return nil
	}
	if !parser.checkName() {
		parser.reportExpected("global variable name", "")
		return nil
	}

	name := parser.parseIdentifier()
	var accesses []ast.VariableAccess
	for {
		switch {
		case parser.match(token.Dot):
			if !parser.checkName() {
				parser.reportExpected("field name", "")
				return &ast.VariableExpr{NodeBase: parser.base(start, parser.previous().EndOffset), Name: *name, Accesses: accesses}
			}
			field := parser.parseIdentifier()
			accesses = append(accesses, &ast.FieldAccess{
				NodeBase: parser.base(parser.tokens[parser.current-2].StartOffset, field.Span().EndOffset),
				Field:    *field,
			})
		case parser.match(token.LBracket):
			accessStart := parser.previous().StartOffset
			if parser.match(token.RBracket) {
				accesses = append(accesses, &ast.EmptyIndexAccess{NodeBase: parser.base(accessStart, parser.previous().EndOffset)})
				continue
			}
			indexStart := parser.current
			for !parser.check(token.RBracket) && !parser.check(token.Newline) && !parser.atEnd() {
				parser.advance()
			}
			var index ast.Expression
			if indexStart < parser.current {
				index = parser.parseDeferredExpression(parser.tokens[indexStart:parser.current])
			}
			if !parser.match(token.RBracket) {
				parser.reportExpected("]", "")
			}
			accesses = append(accesses, &ast.IndexAccess{
				NodeBase: parser.base(accessStart, parser.previous().EndOffset),
				Index:    index,
			})
		default:
			end := name.Span().EndOffset
			if len(accesses) > 0 {
				end = accesses[len(accesses)-1].Span().EndOffset
			}
			return &ast.VariableExpr{
				NodeBase: parser.base(start, end),
				Name:     *name,
				Accesses: accesses,
			}
		}
	}
}

func (parser *parser) parseDeferredExpression(tokens []token.Token) ast.Expression {
	start := tokens[0].StartOffset
	end := tokens[len(tokens)-1].EndOffset
	base := parser.base(start, end)

	if len(tokens) == 1 {
		switch tokens[0].Type {
		case token.Nil:
			return &ast.NilLiteral{NodeBase: base}
		case token.True:
			return &ast.BoolLiteral{NodeBase: base, Value: true}
		case token.False:
			return &ast.BoolLiteral{NodeBase: base, Value: false}
		case token.Int:
			value, _ := tokens[0].Value.(int)
			return &ast.IntLiteral{NodeBase: base, Value: int64(value)}
		case token.Float:
			value, _ := tokens[0].Value.(float64)
			return &ast.FloatLiteral{NodeBase: base, Value: value}
		}
	}
	if tokens[0].Type == token.StringStart && tokens[len(tokens)-1].Type == token.StringEnd {
		return parser.stringFromTokens(tokens)
	}
	if len(tokens) == 2 && tokens[0].Type == token.LBracket && tokens[1].Type == token.RBracket {
		return &ast.ListExpr{NodeBase: base}
	}

	return &ast.PatternExpr{NodeBase: base, Tokens: append([]token.Token(nil), tokens...)}
}

func (parser *parser) parseString() *ast.StringExpr {
	if !parser.check(token.StringStart) {
		parser.reportExpected("string", "")
		return nil
	}

	start := parser.current
	parser.advance()
	depth := 0
	for !parser.atEnd() {
		switch parser.peek().Type {
		case token.InterpStart:
			depth++
		case token.InterpEnd:
			depth--
		case token.StringEnd:
			if depth == 0 {
				parser.advance()
				return parser.stringFromTokens(parser.tokens[start:parser.current])
			}
		case token.Newline:
			parser.reportExpected("closing quote", "")
			return parser.stringFromTokens(parser.tokens[start:parser.current])
		}
		parser.advance()
	}
	parser.reportExpected("closing quote", "")
	return parser.stringFromTokens(parser.tokens[start:parser.current])
}

func (parser *parser) stringFromTokens(tokens []token.Token) *ast.StringExpr {
	if len(tokens) == 0 {
		return nil
	}

	expression := &ast.StringExpr{
		NodeBase: parser.base(tokens[0].StartOffset, tokens[len(tokens)-1].EndOffset),
	}
	if len(tokens[0].Lexeme) > 0 {
		expression.Quote = tokens[0].Lexeme[0]
	}

	for index := 1; index < len(tokens)-1; index++ {
		tok := tokens[index]
		switch tok.Type {
		case token.StringText:
			value, _ := tok.Value.(string)
			expression.Parts = append(expression.Parts, &ast.StringText{
				NodeBase: parser.base(tok.StartOffset, tok.EndOffset),
				Raw:      tok.Lexeme,
				Value:    value,
			})
		case token.InterpStart:
			interpolationStart := index + 1
			for index+1 < len(tokens)-1 && tokens[index+1].Type != token.InterpEnd {
				index++
			}
			parts := tokens[interpolationStart : index+1]
			var value ast.Expression
			if len(parts) > 0 {
				value = parser.parseDeferredExpression(parts)
			}
			end := tok.EndOffset
			if index+1 < len(tokens) && tokens[index+1].Type == token.InterpEnd {
				index++
				end = tokens[index].EndOffset
			}
			expression.Parts = append(expression.Parts, &ast.StringInterpolation{
				NodeBase:   parser.base(tok.StartOffset, end),
				Expression: value,
			})
		}
	}

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

func (parser *parser) scanBlock(openingStart int) (ast.Block, int) {
	bodyStart := parser.peek().StartOffset
	depth := 0
	lineStart := true
	for !parser.atEnd() {
		tok := parser.peek()
		if lineStart {
			switch tok.Type {
			case token.If, token.Loop:
				depth++
			case token.Else:
				if depth == 0 {
					parser.reportUnexpected(tok, "else can only appear inside an if block.")
				}
			case token.End:
				if depth == 0 {
					endToken := parser.advance()
					parser.requireLineEnd()
					return ast.Block{NodeBase: parser.base(bodyStart, endToken.StartOffset)}, endToken.EndOffset
				}
				depth--
			}
			lineStart = false
		}

		if parser.match(token.Newline) {
			lineStart = true
			continue
		}
		parser.advance()
	}

	eof := parser.peek()
	parser.report(
		diagnostic.CodeExpectedEnd,
		`Expected "end" before end of file.`,
		"Add end to close the block.",
		eof.StartOffset,
		eof.EndOffset,
	)
	return ast.Block{NodeBase: parser.base(bodyStart, eof.StartOffset)}, eof.EndOffset
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
