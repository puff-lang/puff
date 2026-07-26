package ast

import "github.com/puff-lang/puff/internal/token"

type AssignmentStmt struct {
	NodeBase
	Target *VariableExpr
	Value  Expression
}

func (*AssignmentStmt) statementNode() {}

type AddStmt struct {
	NodeBase
	Value  Expression
	Target Expression
}

func (*AddStmt) statementNode() {}

type IfStmt struct {
	NodeBase
	Condition Expression
	Then      Block
	ElseIf    []ElseIfClause
	Else      *Block
}

func (*IfStmt) statementNode() {}

type ElseIfClause struct {
	NodeBase
	Condition Expression
	Body      Block
}

type LoopTimesStmt struct {
	NodeBase
	Count Expression
	Body  Block
}

func (*LoopTimesStmt) statementNode() {}

type LoopRangeStmt struct {
	NodeBase
	Start Expression
	End   Expression
	Body  Block
}

func (*LoopRangeStmt) statementNode() {}

type LoopPlayersStmt struct {
	NodeBase
	Body Block
}

func (*LoopPlayersStmt) statementNode() {}

type LoopEntitiesStmt struct {
	NodeBase
	Radius Expression
	Around Expression
	Body   Block
}

func (*LoopEntitiesStmt) statementNode() {}

type ReturnStmt struct {
	NodeBase
	Value Expression
}

func (*ReturnStmt) statementNode() {}

type StopStmt struct {
	NodeBase
}

func (*StopStmt) statementNode() {}

type ExprStmt struct {
	NodeBase
	Expression Expression
}

func (*ExprStmt) statementNode() {}

type EffectStmt struct {
	NodeBase
	Tokens []token.Token
}

func (*EffectStmt) statementNode() {}
