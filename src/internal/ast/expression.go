package ast

import "github.com/puff-lang/puff/internal/token"

type NilLiteral struct {
	NodeBase
}

func (*NilLiteral) expressionNode() {}

type BoolLiteral struct {
	NodeBase
	Value bool
}

func (*BoolLiteral) expressionNode() {}

type IntLiteral struct {
	NodeBase
	Value int64
}

func (*IntLiteral) expressionNode() {}

type FloatLiteral struct {
	NodeBase
	Value float64
}

func (*FloatLiteral) expressionNode() {}

type UnaryExpr struct {
	NodeBase
	Operator token.Type
	Operand  Expression
}

func (*UnaryExpr) expressionNode() {}

type BinaryExpr struct {
	NodeBase
	Left     Expression
	Operator token.Type
	Right    Expression
}

func (*BinaryExpr) expressionNode() {}

type GroupExpr struct {
	NodeBase
	Expression Expression
}

func (*GroupExpr) expressionNode() {}

type QualifiedName struct {
	NodeBase
	Parts []Identifier
}

type CallExpr struct {
	NodeBase
	Callee         QualifiedName
	Arguments      []Expression
	ExplicitParens bool
}

func (*CallExpr) expressionNode() {}

type ListExpr struct {
	NodeBase
	Elements []Expression
}

func (*ListExpr) expressionNode() {}

type MapExpr struct {
	NodeBase
	Entries []MapEntry
}

func (*MapExpr) expressionNode() {}

type MapEntry struct {
	NodeBase
	Key   Expression
	Value Expression
}

type RangeExpr struct {
	NodeBase
	Start Expression
	End   Expression
}

func (*RangeExpr) expressionNode() {}

type StringExpr struct {
	NodeBase
	Quote byte
	Parts []StringPart
}

func (*StringExpr) expressionNode() {}

type StringText struct {
	NodeBase
	Raw   string
	Value string
}

func (*StringText) stringPartNode() {}

type StringInterpolation struct {
	NodeBase
	Expression Expression
}

func (*StringInterpolation) stringPartNode() {}

type VariableExpr struct {
	NodeBase
	Qualifier *Identifier
	Name      Identifier
	Local     bool
	Accesses  []VariableAccess
}

func (*VariableExpr) expressionNode() {}
func (*VariableExpr) assignableNode() {}

type FieldAccess struct {
	NodeBase
	Field Identifier
}

func (*FieldAccess) variableAccessNode() {}

type IndexAccess struct {
	NodeBase
	Index Expression
}

func (*IndexAccess) variableAccessNode() {}

type EmptyIndexAccess struct {
	NodeBase
}

func (*EmptyIndexAccess) variableAccessNode() {}

type PatternExpr struct {
	NodeBase
	Tokens []token.Token
}

func (*PatternExpr) expressionNode() {}

type AccessExpr struct {
	NodeBase
	Tokens []token.Token
}

func (*AccessExpr) expressionNode() {}
func (*AccessExpr) assignableNode() {}
