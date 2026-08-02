package patterns

import (
	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/token"
)

type Kind string

const (
	KindEffect     Kind = "effect"
	KindExpression Kind = "expression"
	KindCondition  Kind = "condition"
	KindEvent      Kind = "event"
)

const (
	CoreSendEffectID     = "core.send"
	CoreSendEffectSyntax = "send %text% to %target%"
	CoreLoadEventID      = "core.load"
	CoreLoadEventSyntax  = "load"
	CoreTickEventID      = "core.tick"
	CoreTickEventSyntax  = "tick"
)

type Definition struct {
	ID     string
	Kind   Kind
	Syntax string
}

type Capture struct {
	Tokens []token.Token
	Span   diagnostic.Span
}

type Captures map[string]Capture

type ResolvedEffect struct {
	Definition Definition
	Captures   Captures
	Node       *ast.EffectStmt
}

type ResolvedExpression struct {
	Definition Definition
	Captures   Captures
	Node       *ast.PatternExpr
}

type ResolvedCondition struct {
	Definition Definition
	Captures   Captures
	Node       *ast.PatternExpr
}

type ResolvedEvent struct {
	Definition Definition
	Captures   Captures
	Node       *ast.EventDecl
}

type Registry interface {
	RegisterEffect(id, syntax string) error
	RegisterExpression(id, syntax string) error
	RegisterCondition(id, syntax string) error
	RegisterEvent(id, syntax string) error
	ResolveEffect(file string, node *ast.EffectStmt) (*ResolvedEffect, *diagnostic.Diagnostic)
	ResolveExpression(file string, node *ast.PatternExpr) (*ResolvedExpression, *diagnostic.Diagnostic)
	ResolveCondition(file string, node *ast.PatternExpr) (*ResolvedCondition, *diagnostic.Diagnostic)
	ResolveEvent(file string, node *ast.EventDecl) (*ResolvedEvent, *diagnostic.Diagnostic)
}
