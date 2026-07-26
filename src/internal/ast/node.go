package ast

import "github.com/puff-lang/puff/internal/diagnostic"

type Node interface {
	Span() diagnostic.Span
	node()
}

type Declaration interface {
	Node
	declarationNode()
}

type Statement interface {
	Node
	statementNode()
}

type Expression interface {
	Node
	expressionNode()
}

type Assignable interface {
	Expression
	assignableNode()
}

type StringPart interface {
	Node
	stringPartNode()
}

type VariableAccess interface {
	Node
	variableAccessNode()
}

type NodeBase struct {
	SourceSpan diagnostic.Span
}

func (base NodeBase) Span() diagnostic.Span {
	return base.SourceSpan
}

func (NodeBase) node() {}

func JoinSpans(first, last diagnostic.Span) diagnostic.Span {
	return diagnostic.Span{
		StartLine:   first.StartLine,
		StartColumn: first.StartColumn,
		EndLine:     last.EndLine,
		EndColumn:   last.EndColumn,
		StartOffset: first.StartOffset,
		EndOffset:   last.EndOffset,
	}
}

func SpanBetween(first, last Node) diagnostic.Span {
	return JoinSpans(first.Span(), last.Span())
}

type Identifier struct {
	NodeBase
	Name string
}

type Block struct {
	NodeBase
	Statements []Statement
}
