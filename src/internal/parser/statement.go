package parser

import (
	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/token"
)

func (parser *parser) parseBlock(allowElse bool) ast.Block {
	start := parser.peek().StartOffset
	var statements []ast.Statement
	parser.skipNewlines()
	for !parser.atEnd() && !parser.check(token.End) {
		if parser.check(token.Else) {
			if allowElse {
				break
			}
			parser.reportUnexpected(parser.peek(), "else can only appear inside an if block.")
			parser.synchronizeLine()
			parser.match(token.Newline)
			continue
		}
		statement := parser.parseStatement()
		if statement != nil {
			statements = append(statements, statement)
		}
		parser.skipNewlines()
	}
	return ast.Block{
		NodeBase:   parser.base(start, parser.peek().StartOffset),
		Statements: statements,
	}
}

func (parser *parser) parseStatement() ast.Statement {
	switch parser.peek().Type {
	case token.Dollar:
		return parser.parseVariableStatement(nil)
	case token.Add:
		return parser.parseAddStatement()
	case token.If:
		return parser.parseIfStatement()
	case token.Loop:
		return parser.parseLoopStatement()
	case token.Return:
		return parser.parseReturnStatement()
	case token.Stop:
		return parser.parseStopStatement()
	}

	if parser.checkName() && parser.peekAt(1).Type == token.Dot && parser.peekAt(2).Type == token.Dollar {
		qualifier := parser.parseIdentifier()
		parser.advance()
		return parser.parseVariableStatement(qualifier)
	}
	return parser.parseExpressionOrEffectStatement()
}

func (parser *parser) parseVariableStatement(qualifier *ast.Identifier) ast.Statement {
	start := parser.peek().StartOffset
	expression := parser.parseVariable(qualifier)
	variable, _ := expression.(*ast.VariableExpr)
	if !parser.match(token.Equal) {
		parser.requireLineEnd()
		return &ast.ExprStmt{
			NodeBase:   parser.base(start, variable.Span().EndOffset),
			Expression: variable,
		}
	}

	value := parser.parseExpressionUntil(token.Newline)
	end := parser.statementEnd(start, value)
	parser.requireLineEnd()
	return &ast.AssignmentStmt{
		NodeBase: parser.base(start, end),
		Target:   variable,
		Value:    value,
	}
}

func (parser *parser) parseAddStatement() ast.Statement {
	start := parser.advance().StartOffset
	value := parser.parseExpressionUntil(token.To)
	if !parser.match(token.To) {
		parser.reportExpected("to", "")
	}

	var target ast.Assignable
	if parser.check(token.Dollar) {
		target, _ = parser.parseVariable(nil).(ast.Assignable)
	} else if parser.checkName() {
		target = parser.parseAccessExpression()
	} else {
		parser.reportExpected("assignable target", "")
		parser.synchronizeLine()
	}
	end := start
	if target != nil {
		end = target.Span().EndOffset
	} else if value != nil {
		end = value.Span().EndOffset
	}
	parser.requireLineEnd()
	return &ast.AddStmt{
		NodeBase: parser.base(start, end),
		Value:    value,
		Target:   target,
	}
}

func (parser *parser) parseAccessExpression() ast.Assignable {
	start := parser.current
	for !parser.check(token.Newline) && !parser.atEnd() {
		parser.advance()
	}
	tokens := append([]token.Token(nil), parser.tokens[start:parser.current]...)
	if len(tokens) == 0 {
		return nil
	}
	return &ast.AccessExpr{
		NodeBase: parser.base(tokens[0].StartOffset, tokens[len(tokens)-1].EndOffset),
		Tokens:   tokens,
	}
}

func (parser *parser) parseIfStatement() ast.Statement {
	start := parser.advance().StartOffset
	condition := parser.parseExpressionUntil(token.Newline)
	parser.requireLineEnd()
	thenBlock := parser.parseBlock(true)

	var elseIf []ast.ElseIfClause
	var elseBlock *ast.Block
	for parser.match(token.Else) {
		clauseStart := parser.previous().StartOffset
		if parser.match(token.If) {
			clauseCondition := parser.parseExpressionUntil(token.Newline)
			parser.requireLineEnd()
			body := parser.parseBlock(true)
			elseIf = append(elseIf, ast.ElseIfClause{
				NodeBase:  parser.base(clauseStart, body.Span().EndOffset),
				Condition: clauseCondition,
				Body:      body,
			})
			continue
		}

		parser.requireLineEnd()
		body := parser.parseBlock(false)
		elseBlock = &body
		break
	}

	end := parser.consumeBlockEnd(start)
	return &ast.IfStmt{
		NodeBase:  parser.base(start, end),
		Condition: condition,
		Then:      thenBlock,
		ElseIf:    elseIf,
		Else:      elseBlock,
	}
}

func (parser *parser) parseLoopStatement() ast.Statement {
	start := parser.advance().StartOffset
	switch {
	case parser.match(token.Players):
		parser.requireLineEnd()
		body := parser.parseBlock(false)
		end := parser.consumeBlockEnd(start)
		return &ast.LoopPlayersStmt{NodeBase: parser.base(start, end), Body: body}
	case parser.match(token.Numbers):
		if !parser.match(token.From) {
			parser.reportExpected("from", "")
		}
		rangeStart := parser.parseExpressionUntil(token.To, token.Newline)
		hasTo := parser.match(token.To)
		if !hasTo {
			if rangeStart != nil {
				parser.reportExpected("to", "")
			}
			parser.synchronizeLine()
		}
		var rangeEnd ast.Expression
		if hasTo {
			rangeEnd = parser.parseExpressionUntil(token.Newline)
		}
		parser.requireLineEnd()
		body := parser.parseBlock(false)
		end := parser.consumeBlockEnd(start)
		return &ast.LoopRangeStmt{
			NodeBase: parser.base(start, end),
			Start:    rangeStart,
			End:      rangeEnd,
			Body:     body,
		}
	case parser.match(token.Entities):
		if !parser.match(token.In) {
			parser.reportExpected("in", "")
		}
		if !parser.match(token.Radius) {
			parser.reportExpected("radius", "")
		}
		radius := parser.parseExpressionUntil(token.Around, token.Newline)
		hasAround := parser.match(token.Around)
		if !hasAround {
			if radius != nil {
				parser.reportExpected("around", "")
			}
			parser.synchronizeLine()
		}
		var around ast.Expression
		if hasAround {
			around = parser.parseExpressionUntil(token.Newline)
		}
		parser.requireLineEnd()
		body := parser.parseBlock(false)
		end := parser.consumeBlockEnd(start)
		return &ast.LoopEntitiesStmt{
			NodeBase: parser.base(start, end),
			Radius:   radius,
			Around:   around,
			Body:     body,
		}
	default:
		count := parser.parseExpressionUntil(token.Times, token.Newline)
		if !parser.match(token.Times) {
			if count != nil {
				parser.reportExpected("times", "")
			}
			parser.synchronizeLine()
		}
		parser.requireLineEnd()
		body := parser.parseBlock(false)
		end := parser.consumeBlockEnd(start)
		return &ast.LoopTimesStmt{
			NodeBase: parser.base(start, end),
			Count:    count,
			Body:     body,
		}
	}
}

func (parser *parser) parseReturnStatement() ast.Statement {
	startToken := parser.advance()
	var value ast.Expression
	if !parser.check(token.Newline) && !parser.atEnd() {
		value = parser.parseExpressionUntil(token.Newline)
	}
	end := startToken.EndOffset
	if value != nil {
		end = value.Span().EndOffset
	}
	parser.requireLineEnd()
	return &ast.ReturnStmt{
		NodeBase: parser.base(startToken.StartOffset, end),
		Value:    value,
	}
}

func (parser *parser) parseStopStatement() ast.Statement {
	tok := parser.advance()
	parser.requireLineEnd()
	return &ast.StopStmt{NodeBase: parser.base(tok.StartOffset, tok.EndOffset)}
}

func (parser *parser) parseExpressionOrEffectStatement() ast.Statement {
	startIndex := parser.current
	expression := parser.parseExpressionUntil(token.Newline)
	if pattern, ok := expression.(*ast.PatternExpr); ok {
		parser.synchronizeLine()
		tokens := append([]token.Token(nil), parser.tokens[startIndex:parser.current]...)
		end := pattern.Span().EndOffset
		if len(tokens) > 0 {
			end = tokens[len(tokens)-1].EndOffset
		}
		parser.requireLineEnd()
		return &ast.EffectStmt{
			NodeBase: parser.base(parser.tokens[startIndex].StartOffset, end),
			Tokens:   tokens,
		}
	}

	end := parser.statementEnd(parser.tokens[startIndex].StartOffset, expression)
	parser.requireLineEnd()
	return &ast.ExprStmt{
		NodeBase:   parser.base(parser.tokens[startIndex].StartOffset, end),
		Expression: expression,
	}
}

func (parser *parser) consumeBlockEnd(openingStart int) int {
	if parser.match(token.End) {
		end := parser.previous().EndOffset
		parser.requireLineEnd()
		return end
	}
	eof := parser.peek()
	parser.report(
		diagnostic.CodeExpectedEnd,
		`Expected "end" before end of file.`,
		"Add end to close the block.",
		eof.StartOffset,
		eof.EndOffset,
	)
	return eof.EndOffset
}

func (parser *parser) statementEnd(start int, expression ast.Expression) int {
	if expression != nil {
		return expression.Span().EndOffset
	}
	if parser.current > 0 {
		return parser.previous().EndOffset
	}
	return start
}
