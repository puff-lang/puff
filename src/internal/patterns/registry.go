package patterns

import (
	"fmt"
	"strings"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/token"
)

type registeredDefinition struct {
	definition Definition
	template   []templatePart
}

type Builder struct {
	definitions map[Kind][]registeredDefinition
	ids         map[string]struct{}
}

func NewRegistry() *Builder {
	return &Builder{
		definitions: make(map[Kind][]registeredDefinition),
		ids:         make(map[string]struct{}),
	}
}

func NewCoreRegistry() Registry {
	registry := NewRegistry()
	mustRegister(registry.RegisterEffect(CoreSendEffectID, CoreSendEffectSyntax))
	mustRegister(registry.RegisterEvent(CoreLoadEventID, CoreLoadEventSyntax))
	mustRegister(registry.RegisterEvent(CoreTickEventID, CoreTickEventSyntax))
	return registry
}

func mustRegister(err error) {
	if err != nil {
		panic(err)
	}
}

func (registry *Builder) RegisterEffect(id, syntax string) error {
	return registry.register(Definition{ID: id, Kind: KindEffect, Syntax: syntax})
}

func (registry *Builder) RegisterExpression(id, syntax string) error {
	return registry.register(Definition{ID: id, Kind: KindExpression, Syntax: syntax})
}

func (registry *Builder) RegisterCondition(id, syntax string) error {
	return registry.register(Definition{ID: id, Kind: KindCondition, Syntax: syntax})
}

func (registry *Builder) RegisterEvent(id, syntax string) error {
	return registry.register(Definition{ID: id, Kind: KindEvent, Syntax: syntax})
}

func (registry *Builder) register(definition Definition) error {
	if definition.ID == "" {
		return fmt.Errorf("pattern ID cannot be empty")
	}
	if _, exists := registry.ids[definition.ID]; exists {
		return fmt.Errorf("pattern ID %q is already registered", definition.ID)
	}
	template, err := compileTemplate(definition.Syntax)
	if err != nil {
		return fmt.Errorf("compile pattern %q: %w", definition.ID, err)
	}

	registry.definitions[definition.Kind] = append(registry.definitions[definition.Kind], registeredDefinition{
		definition: definition,
		template:   template,
	})
	registry.ids[definition.ID] = struct{}{}
	return nil
}

func (registry *Builder) ResolveEffect(file string, node *ast.EffectStmt) (*ResolvedEffect, *diagnostic.Diagnostic) {
	definition, captures, issue := registry.resolve(KindEffect, node.Tokens, file, node.Span(), "")
	if issue != nil {
		return nil, issue
	}
	return &ResolvedEffect{Definition: definition, Captures: captures, Node: node}, nil
}

func (registry *Builder) ResolveExpression(file string, node *ast.PatternExpr) (*ResolvedExpression, *diagnostic.Diagnostic) {
	definition, captures, issue := registry.resolve(KindExpression, node.Tokens, file, node.Span(), "")
	if issue != nil {
		return nil, issue
	}
	return &ResolvedExpression{Definition: definition, Captures: captures, Node: node}, nil
}

func (registry *Builder) ResolveCondition(file string, node *ast.PatternExpr) (*ResolvedCondition, *diagnostic.Diagnostic) {
	definition, captures, issue := registry.resolve(KindCondition, node.Tokens, file, node.Span(), "")
	if issue != nil {
		return nil, issue
	}
	return &ResolvedCondition{Definition: definition, Captures: captures, Node: node}, nil
}

func (registry *Builder) ResolveEvent(file string, node *ast.EventDecl) (*ResolvedEvent, *diagnostic.Diagnostic) {
	tokens := eventTokens(node)
	span := node.Span()
	if len(node.Name) > 0 {
		span = ast.JoinSpans(node.Name[0].Span(), node.Name[len(node.Name)-1].Span())
	}
	definition, captures, issue := registry.resolve(KindEvent, tokens, file, span, eventName(node))
	if issue != nil {
		return nil, issue
	}
	return &ResolvedEvent{Definition: definition, Captures: captures, Node: node}, nil
}

func (registry *Builder) resolve(kind Kind, tokens []token.Token, file string, span diagnostic.Span, eventText string) (Definition, Captures, *diagnostic.Diagnostic) {
	type match struct {
		definition Definition
		captures   Captures
	}

	var matched *match
	for _, candidate := range registry.definitions[kind] {
		for _, captures := range matchTemplate(candidate.template, tokens) {
			if matched != nil {
				issue := patternDiagnostic(
					diagnostic.CodeAmbiguousPattern,
					"Ambiguous pattern.",
					"Make the statement more explicit or adjust pattern priorities.",
					file,
					span,
				)
				return Definition{}, nil, &issue
			}
			current := match{definition: candidate.definition, captures: captures}
			matched = &current
		}
	}

	if matched == nil {
		issue := unknownDiagnostic(kind, file, span, eventText)
		return Definition{}, nil, &issue
	}
	return matched.definition, matched.captures, nil
}

func unknownDiagnostic(kind Kind, file string, span diagnostic.Span, eventText string) diagnostic.Diagnostic {
	switch kind {
	case KindEffect:
		return patternDiagnostic(
			diagnostic.CodeUnknownEffectPattern,
			"Unknown effect pattern.",
			"Check the syntax or require a library that registers this effect.",
			file,
			span,
		)
	case KindExpression:
		return patternDiagnostic(
			diagnostic.CodeUnknownExpressionPattern,
			"Unknown expression pattern.",
			"Require the library that defines this expression.",
			file,
			span,
		)
	case KindCondition:
		return patternDiagnostic(
			diagnostic.CodeUnknownConditionPattern,
			"Unknown condition pattern.",
			"",
			file,
			span,
		)
	case KindEvent:
		return patternDiagnostic(
			diagnostic.CodeUnknownEventPattern,
			"Unknown event pattern: "+eventText,
			"Require the library that registers this event.",
			file,
			span,
		)
	default:
		panic(fmt.Sprintf("unknown pattern kind %q", kind))
	}
}

func patternDiagnostic(code diagnostic.Code, message, hint, file string, span diagnostic.Span) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Code:     code,
		Phase:    diagnostic.PhasePattern,
		Severity: diagnostic.SeverityError,
		Message:  message,
		Hint:     hint,
		File:     file,
		Span:     span,
	}
}

func eventTokens(node *ast.EventDecl) []token.Token {
	tokens := make([]token.Token, 0, len(node.Name))
	for _, identifier := range node.Name {
		span := identifier.Span()
		tokens = append(tokens, token.Token{
			Type:        token.Ident,
			Lexeme:      identifier.Name,
			Value:       identifier.Name,
			Line:        span.StartLine,
			Column:      span.StartColumn,
			StartOffset: span.StartOffset,
			EndOffset:   span.EndOffset,
		})
	}
	return tokens
}

func eventName(node *ast.EventDecl) string {
	names := make([]string, 0, len(node.Name))
	for _, identifier := range node.Name {
		names = append(names, identifier.Name)
	}
	return strings.Join(names, " ")
}
