package parser

import (
	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/token"
)

func (parser *parser) parseExpressionUntil(stops ...token.Type) ast.Expression {
	previousStops := parser.expressionStops
	parser.expressionStops = make(map[token.Type]bool, len(stops))
	for _, stop := range stops {
		parser.expressionStops[stop] = true
	}
	expression := parser.parseRange()
	parser.expressionStops = previousStops
	return expression
}

func (parser *parser) parseRange() ast.Expression {
	left := parser.parseOr()
	if left == nil || !parser.match(token.DotDot) {
		return left
	}

	right := parser.parseOr()
	if right == nil {
		parser.reportExpected("expression", "")
		return left
	}
	expression := &ast.RangeExpr{
		NodeBase: parser.base(left.Span().StartOffset, right.Span().EndOffset),
		Start:    left,
		End:      right,
	}
	if parser.check(token.DotDot) {
		parser.reportUnexpected(parser.peek(), "Ranges cannot be chained.")
	}
	return expression
}

func (parser *parser) parseOr() ast.Expression {
	return parser.parseBinary(parser.parseAnd, token.Or)
}

func (parser *parser) parseAnd() ast.Expression {
	return parser.parseBinary(parser.parseEquality, token.And)
}

func (parser *parser) parseEquality() ast.Expression {
	return parser.parseBinary(parser.parseComparison, token.EqualEqual, token.BangEqual)
}

func (parser *parser) parseComparison() ast.Expression {
	return parser.parseBinary(parser.parseTerm, token.Greater, token.GreaterEq, token.Less, token.LessEq)
}

func (parser *parser) parseTerm() ast.Expression {
	return parser.parseBinary(parser.parseFactor, token.Plus, token.Minus)
}

func (parser *parser) parseFactor() ast.Expression {
	return parser.parseBinary(parser.parseUnary, token.Star, token.Slash, token.Percent)
}

func (parser *parser) parseBinary(operand func() ast.Expression, operators ...token.Type) ast.Expression {
	expression := operand()
	for expression != nil && parser.match(operators...) {
		operator := parser.previous()
		right := operand()
		if right == nil {
			parser.reportExpected("expression", "")
			return expression
		}
		expression = &ast.BinaryExpr{
			NodeBase: parser.base(expression.Span().StartOffset, right.Span().EndOffset),
			Left:     expression,
			Operator: operator.Type,
			Right:    right,
		}
	}
	return expression
}

func (parser *parser) parseUnary() ast.Expression {
	if parser.match(token.Not, token.Minus) {
		operator := parser.previous()
		operand := parser.parseUnary()
		if operand == nil {
			parser.reportExpected("expression", "")
			return nil
		}
		return &ast.UnaryExpr{
			NodeBase: parser.base(operator.StartOffset, operand.Span().EndOffset),
			Operator: operator.Type,
			Operand:  operand,
		}
	}
	return parser.parsePrimary()
}

func (parser *parser) parsePrimary() ast.Expression {
	if parser.atExpressionStop() || parser.atEnd() {
		return nil
	}

	switch parser.peek().Type {
	case token.Nil:
		tok := parser.advance()
		return &ast.NilLiteral{NodeBase: parser.base(tok.StartOffset, tok.EndOffset)}
	case token.True, token.False:
		tok := parser.advance()
		return &ast.BoolLiteral{NodeBase: parser.base(tok.StartOffset, tok.EndOffset), Value: tok.Type == token.True}
	case token.Int:
		tok := parser.advance()
		value, _ := tok.Value.(int)
		return &ast.IntLiteral{NodeBase: parser.base(tok.StartOffset, tok.EndOffset), Value: int64(value)}
	case token.Float:
		tok := parser.advance()
		value, _ := tok.Value.(float64)
		return &ast.FloatLiteral{NodeBase: parser.base(tok.StartOffset, tok.EndOffset), Value: value}
	case token.StringStart:
		return parser.parseStringExpression()
	case token.LParen:
		return parser.parseGroup()
	case token.LBracket:
		return parser.parseList()
	case token.LBrace:
		return parser.parseMap()
	case token.Dollar:
		return parser.parseVariable(nil)
	}

	if parser.checkName() {
		return parser.parseNameExpression()
	}

	parser.reportExpected("expression", "")
	parser.advance()
	return nil
}

func (parser *parser) parseGroup() ast.Expression {
	start := parser.advance().StartOffset
	expression := parser.parseExpressionUntil(token.RParen)
	if expression == nil {
		parser.reportExpected("expression", "")
	}
	if !parser.match(token.RParen) {
		parser.reportExpected(")", "")
		end := start
		if expression != nil {
			end = expression.Span().EndOffset
		}
		return &ast.GroupExpr{NodeBase: parser.base(start, end), Expression: expression}
	}
	return &ast.GroupExpr{
		NodeBase:   parser.base(start, parser.previous().EndOffset),
		Expression: expression,
	}
}

func (parser *parser) parseList() ast.Expression {
	start := parser.advance().StartOffset
	var elements []ast.Expression
	for !parser.check(token.RBracket) && !parser.atEnd() {
		element := parser.parseExpressionUntil(token.Comma, token.RBracket)
		if element != nil {
			elements = append(elements, element)
		} else {
			parser.reportExpected("expression", "")
		}
		if parser.match(token.Comma) {
			if parser.check(token.RBracket) {
				break
			}
			continue
		}
		if !parser.check(token.RBracket) {
			parser.reportExpected(`"," or "]"`, "")
			parser.synchronizeUntil(token.Comma, token.RBracket, token.Newline)
			parser.match(token.Comma)
		}
	}
	end := parser.collectionEnd(start, token.RBracket, "]")
	return &ast.ListExpr{NodeBase: parser.base(start, end), Elements: elements}
}

func (parser *parser) parseMap() ast.Expression {
	start := parser.advance().StartOffset
	var entries []ast.MapEntry
	for !parser.check(token.RBrace) && !parser.atEnd() {
		entryStart := parser.peek().StartOffset
		key := parser.parseExpressionUntil(token.Colon)
		if key == nil {
			parser.reportExpected("expression", "")
		}
		hasColon := parser.match(token.Colon)
		if !hasColon {
			parser.reportExpected(":", "")
			parser.synchronizeUntil(token.Comma, token.RBrace, token.Newline)
		}
		var value ast.Expression
		if hasColon {
			value = parser.parseExpressionUntil(token.Comma, token.RBrace)
		}
		if hasColon && value == nil {
			parser.reportExpected("expression", "")
		}
		entryEnd := entryStart
		if value != nil {
			entryEnd = value.Span().EndOffset
		} else if key != nil {
			entryEnd = key.Span().EndOffset
		}
		entries = append(entries, ast.MapEntry{
			NodeBase: parser.base(entryStart, entryEnd),
			Key:      key,
			Value:    value,
		})
		if parser.match(token.Comma) {
			if parser.check(token.RBrace) {
				break
			}
			continue
		}
		if !parser.check(token.RBrace) {
			parser.reportExpected(`"," or "}"`, "")
			parser.synchronizeUntil(token.Comma, token.RBrace, token.Newline)
			parser.match(token.Comma)
		}
	}
	end := parser.collectionEnd(start, token.RBrace, "}")
	return &ast.MapExpr{NodeBase: parser.base(start, end), Entries: entries}
}

func (parser *parser) collectionEnd(start int, closing token.Type, spelling string) int {
	if parser.match(closing) {
		return parser.previous().EndOffset
	}
	parser.reportExpected(spelling, "")
	if parser.current > 0 {
		return parser.previous().EndOffset
	}
	return start
}

func (parser *parser) parseStringExpression() ast.Expression {
	startToken := parser.advance()
	expression := &ast.StringExpr{Quote: startToken.Lexeme[0]}
	for !parser.check(token.StringEnd) && !parser.check(token.Newline) && !parser.atEnd() {
		switch parser.peek().Type {
		case token.StringText:
			tok := parser.advance()
			value, _ := tok.Value.(string)
			expression.Parts = append(expression.Parts, &ast.StringText{
				NodeBase: parser.base(tok.StartOffset, tok.EndOffset),
				Raw:      tok.Lexeme,
				Value:    value,
			})
		case token.InterpStart:
			partStart := parser.advance().StartOffset
			value := parser.parseExpressionUntil(token.InterpEnd)
			end := partStart
			if parser.match(token.InterpEnd) {
				end = parser.previous().EndOffset
			} else {
				parser.reportExpected("}", "")
				parser.synchronizeUntil(token.InterpEnd, token.StringEnd, token.Newline)
				if parser.match(token.InterpEnd) {
					end = parser.previous().EndOffset
				}
				if value != nil {
					if end == partStart {
						end = value.Span().EndOffset
					}
				}
			}
			expression.Parts = append(expression.Parts, &ast.StringInterpolation{
				NodeBase:   parser.base(partStart, end),
				Expression: value,
			})
		default:
			parser.reportUnexpected(parser.peek(), "")
			parser.advance()
		}
	}

	end := startToken.EndOffset
	if parser.match(token.StringEnd) {
		end = parser.previous().EndOffset
	} else if len(expression.Parts) > 0 {
		end = expression.Parts[len(expression.Parts)-1].Span().EndOffset
	}
	expression.NodeBase = parser.base(startToken.StartOffset, end)
	return expression
}

func (parser *parser) parseNameExpression() ast.Expression {
	startIndex := parser.current
	if parser.peekAt(1).Type == token.Dot && parser.peekAt(2).Type == token.Dollar {
		qualifier := parser.parseIdentifier()
		parser.advance()
		return parser.parseVariable(qualifier)
	}

	parts := []ast.Identifier{*parser.parseIdentifier()}
	for parser.match(token.Dot) {
		if !parser.checkName() {
			parser.reportExpected("identifier", "")
			break
		}
		parts = append(parts, *parser.parseIdentifier())
	}

	if parser.match(token.LParen) {
		return parser.finishCall(parts, true, parser.tokens[startIndex].StartOffset)
	}

	if parser.canContinuePattern() {
		parser.current = startIndex
		return parser.parsePatternPrimary()
	}

	end := parts[len(parts)-1].Span().EndOffset
	return &ast.CallExpr{
		NodeBase: parser.base(parser.tokens[startIndex].StartOffset, end),
		Callee: ast.QualifiedName{
			NodeBase: parser.base(parser.tokens[startIndex].StartOffset, end),
			Parts:    parts,
		},
	}
}

func (parser *parser) finishCall(parts []ast.Identifier, explicit bool, start int) ast.Expression {
	var arguments []ast.Expression
	for !parser.check(token.RParen) && !parser.atEnd() {
		argument := parser.parseExpressionUntil(token.Comma, token.RParen)
		if argument != nil {
			arguments = append(arguments, argument)
		} else {
			parser.reportExpected("expression", "")
		}
		if parser.match(token.Comma) {
			if parser.check(token.RParen) {
				parser.reportExpected("expression", "")
				break
			}
			continue
		}
		if !parser.check(token.RParen) {
			parser.reportExpected(`"," or ")"`, "")
			parser.synchronizeUntil(token.Comma, token.RParen, token.Newline)
			parser.match(token.Comma)
		}
	}
	end := start
	if parser.match(token.RParen) {
		end = parser.previous().EndOffset
	} else {
		parser.reportExpected(")", "")
		if len(arguments) > 0 {
			end = arguments[len(arguments)-1].Span().EndOffset
		}
	}
	calleeEnd := parts[len(parts)-1].Span().EndOffset
	return &ast.CallExpr{
		NodeBase: parser.base(start, end),
		Callee: ast.QualifiedName{
			NodeBase: parser.base(start, calleeEnd),
			Parts:    parts,
		},
		Arguments:      arguments,
		ExplicitParens: explicit,
	}
}

func (parser *parser) parsePatternPrimary() ast.Expression {
	start := parser.current
	for !parser.atEnd() && !parser.atExpressionStop() && !isBinaryOperator(parser.peek().Type) {
		parser.advance()
	}
	tokens := append([]token.Token(nil), parser.tokens[start:parser.current]...)
	if len(tokens) == 0 {
		return nil
	}
	return &ast.PatternExpr{
		NodeBase: parser.base(tokens[0].StartOffset, tokens[len(tokens)-1].EndOffset),
		Tokens:   tokens,
	}
}

func (parser *parser) parseVariable(qualifier *ast.Identifier) ast.Expression {
	start := parser.peek().StartOffset
	if qualifier != nil {
		start = qualifier.Span().StartOffset
	}
	parser.advance()
	local := parser.match(token.Underscore)
	if !parser.checkName() {
		parser.reportExpected("variable name", "")
		return &ast.VariableExpr{NodeBase: parser.base(start, parser.previous().EndOffset), Qualifier: qualifier, Local: local}
	}
	name := parser.parseIdentifier()
	if qualifier != nil && local {
		parser.reportExpected("global variable name", "")
	}

	var accesses []ast.VariableAccess
	for {
		switch {
		case parser.match(token.Dot):
			accessStart := parser.previous().StartOffset
			if !parser.checkName() {
				parser.reportExpected("field name", "")
				continue
			}
			field := parser.parseIdentifier()
			accesses = append(accesses, &ast.FieldAccess{
				NodeBase: parser.base(accessStart, field.Span().EndOffset),
				Field:    *field,
			})
		case parser.match(token.LBracket):
			accessStart := parser.previous().StartOffset
			if parser.match(token.RBracket) {
				accesses = append(accesses, &ast.EmptyIndexAccess{NodeBase: parser.base(accessStart, parser.previous().EndOffset)})
				continue
			}
			index := parser.parseExpressionUntil(token.RBracket)
			end := accessStart
			if parser.match(token.RBracket) {
				end = parser.previous().EndOffset
			} else {
				parser.reportExpected("]", "")
				if index != nil {
					end = index.Span().EndOffset
				}
			}
			accesses = append(accesses, &ast.IndexAccess{
				NodeBase: parser.base(accessStart, end),
				Index:    index,
			})
		default:
			end := name.Span().EndOffset
			if len(accesses) > 0 {
				end = accesses[len(accesses)-1].Span().EndOffset
			}
			return &ast.VariableExpr{
				NodeBase:  parser.base(start, end),
				Qualifier: qualifier,
				Name:      *name,
				Local:     local,
				Accesses:  accesses,
			}
		}
	}
}

func (parser *parser) canContinuePattern() bool {
	return !parser.atEnd() &&
		!parser.atExpressionStop() &&
		!isBinaryOperator(parser.peek().Type) &&
		parser.peek().Type != token.Newline
}

func (parser *parser) atExpressionStop() bool {
	return parser.expressionStops != nil && parser.expressionStops[parser.peek().Type]
}

func isBinaryOperator(tokenType token.Type) bool {
	switch tokenType {
	case token.DotDot,
		token.Or,
		token.And,
		token.EqualEqual,
		token.BangEqual,
		token.Greater,
		token.GreaterEq,
		token.Less,
		token.LessEq,
		token.Plus,
		token.Minus,
		token.Star,
		token.Slash,
		token.Percent:
		return true
	default:
		return false
	}
}
