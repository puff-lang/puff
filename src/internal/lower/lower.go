package lower

import (
	"fmt"
	"sort"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/ir"
	"github.com/puff-lang/puff/internal/patterns"
	"github.com/puff-lang/puff/internal/sema"
	"github.com/puff-lang/puff/internal/token"
)

type Result struct {
	Project     *ir.Project
	Diagnostics []diagnostic.Diagnostic
}

type lowerer struct {
	registry    patterns.Registry
	project     *ir.Project
	diagnostics []diagnostic.Diagnostic
	tags        map[string]int
}

func Lower(validated *sema.Project) Result {
	current := &lowerer{
		registry: patterns.NewCoreRegistry(),
		project:  &ir.Project{},
		tags:     make(map[string]int),
	}
	current.lowerProject(validated)
	return Result{Project: current.project, Diagnostics: current.diagnostics}
}

func (current *lowerer) lowerProject(validated *sema.Project) {
	if validated == nil {
		return
	}

	modules := append([]*sema.Module(nil), validated.Modules...)
	sort.SliceStable(modules, func(left, right int) bool {
		if modules[left] == nil {
			return false
		}
		if modules[right] == nil {
			return true
		}
		return modules[left].Source.RelPath < modules[right].Source.RelPath
	})
	for _, module := range modules {
		if module != nil && module.Syntax != nil {
			current.lowerModule(module)
		}
	}
}

func (current *lowerer) lowerModule(module *sema.Module) {
	eventOccurrences := make(map[string]int)
	for _, declaration := range module.Syntax.Declarations {
		switch node := declaration.(type) {
		case *ast.GlobalAssignment:
			current.lowerGlobal(module, node)
		case *ast.FunctionDecl:
			current.lowerFunction(module, node)
		case *ast.EventDecl:
			current.lowerEvent(module, node, eventOccurrences)
		default:
			current.unsupported(module, declaration)
		}
	}
}

func (current *lowerer) lowerGlobal(module *sema.Module, node *ast.GlobalAssignment) {
	value, ok := current.lowerExpression(module, node.Value)
	if !ok || node.Target == nil {
		return
	}

	symbol := module.ResolvedVariables[node.Target]
	id, ok := globalSymbolID(symbol)
	if !ok {
		current.invalid(module, node.Target, "unresolved global variable identity")
		return
	}
	current.project.Globals = append(current.project.Globals, ir.Global{
		ID:          id,
		Public:      node.Public,
		Type:        lowerType(symbol.Type),
		Initializer: value,
		Source:      sourceRef(module, node.Span()),
	})
}

func (current *lowerer) lowerFunction(module *sema.Module, node *ast.FunctionDecl) {
	function := ir.Function{
		ID:     symbolID(module, node.Name.Name),
		Kind:   ir.FunctionUser,
		Public: node.Public,
		Result: ir.Type{Kind: ir.TypeUnknown},
		Source: sourceRef(module, node.Span()),
	}

	var symbol *sema.FunctionSymbol
	if module.Symbols != nil {
		symbol = module.Symbols.Functions[node.Name.Name]
	}
	if symbol != nil {
		function.Result = lowerType(symbol.ReturnType)
	}
	for index := range node.Parameters {
		parameter := &node.Parameters[index]
		typ := sema.Type{Kind: sema.TypeUnknown}
		if symbol != nil && index < len(symbol.Parameters) {
			typ = symbol.Parameters[index]
		}
		function.Parameters = append(function.Parameters, ir.Parameter{
			Name:   parameter.Name.Name,
			Type:   lowerType(typ),
			Source: sourceRef(module, parameter.Span()),
		})
	}
	function.Commands = current.lowerBlock(module, node.Body)
	current.project.Functions = append(current.project.Functions, function)
}

func (current *lowerer) lowerEvent(module *sema.Module, node *ast.EventDecl, occurrences map[string]int) {
	resolved, issue := current.registry.ResolveEvent(module.Source.RelPath, node)
	if issue != nil {
		current.diagnostics = append(current.diagnostics, *issue)
		return
	}

	name := ""
	switch resolved.Definition.ID {
	case patterns.CoreLoadEventID:
		name = "load"
	case patterns.CoreTickEventID:
		name = "tick"
	default:
		current.invalid(module, node, "unsupported event pattern "+resolved.Definition.ID)
		return
	}

	occurrences[name]++
	id := symbolID(module, fmt.Sprintf("event/%s/%d", name, occurrences[name]))
	current.project.Functions = append(current.project.Functions, ir.Function{
		ID:       id,
		Kind:     ir.FunctionEvent,
		Result:   ir.Type{Kind: ir.TypeNil},
		Commands: current.lowerBlock(module, node.Body),
		Source:   sourceRef(module, node.Span()),
	})
	current.addTag(name, id)
}

func (current *lowerer) addTag(name string, function ir.SymbolID) {
	if index, ok := current.tags[name]; ok {
		current.project.Tags[index].Functions = append(current.project.Tags[index].Functions, function)
		return
	}
	current.tags[name] = len(current.project.Tags)
	current.project.Tags = append(current.project.Tags, ir.Tag{Name: name, Functions: []ir.SymbolID{function}})
}

func (current *lowerer) lowerBlock(module *sema.Module, block ast.Block) []ir.Command {
	commands := make([]ir.Command, 0, len(block.Statements))
	for _, statement := range block.Statements {
		switch node := statement.(type) {
		case *ast.ReturnStmt:
			var value ir.Value = &ir.Nil{Source: sourceRef(module, node.Span())}
			ok := true
			if node.Value != nil {
				value, ok = current.lowerExpression(module, node.Value)
			}
			if ok {
				commands = append(commands, &ir.Return{Value: value, Source: sourceRef(module, node.Span())})
			}
		case *ast.EffectStmt:
			if effect, ok := current.lowerEffect(module, node); ok {
				commands = append(commands, effect)
			}
		default:
			current.unsupported(module, statement)
		}
	}
	return commands
}

func (current *lowerer) lowerEffect(module *sema.Module, node *ast.EffectStmt) (*ir.Effect, bool) {
	resolved, issue := current.registry.ResolveEffect(module.Source.RelPath, node)
	if issue != nil {
		current.diagnostics = append(current.diagnostics, *issue)
		return nil, false
	}
	if resolved.Definition.ID != patterns.CoreSendEffectID {
		current.invalid(module, node, "unsupported effect pattern "+resolved.Definition.ID)
		return nil, false
	}

	type capture struct {
		name  string
		value patterns.Capture
	}
	captures := make([]capture, 0, len(resolved.Captures))
	for name, value := range resolved.Captures {
		captures = append(captures, capture{name: name, value: value})
	}
	sort.SliceStable(captures, func(left, right int) bool {
		return captures[left].value.Span.StartOffset < captures[right].value.Span.StartOffset
	})

	effect := &ir.Effect{PatternID: resolved.Definition.ID, Source: sourceRef(module, node.Span())}
	for _, captured := range captures {
		value, ok := current.lowerCapture(module, captured.name, captured.value)
		if !ok {
			return nil, false
		}
		effect.Arguments = append(effect.Arguments, ir.Argument{Name: captured.name, Value: value})
	}
	return effect, true
}

func (current *lowerer) lowerCapture(module *sema.Module, name string, captured patterns.Capture) (ir.Value, bool) {
	switch name {
	case "text":
		return current.lowerCapturedText(module, captured)
	case "target":
		if len(captured.Tokens) == 1 && captured.Tokens[0].Type == token.Ident {
			name, _ := captured.Tokens[0].Value.(string)
			if name == "" {
				name = captured.Tokens[0].Lexeme
			}
			typeName := ""
			switch name {
			case "player":
				typeName = "Player"
			case "console":
				typeName = "Console"
			default:
				current.invalidSpan(module, captured.Span, "unsupported send target")
				return nil, false
			}
			return &ir.Reference{
				Name:   name,
				Type:   ir.Type{Kind: ir.TypeNamed, Name: typeName},
				Source: sourceRef(module, captured.Span),
			}, true
		}
	}
	current.invalidSpan(module, captured.Span, "unsupported "+name+" capture")
	return nil, false
}

func (current *lowerer) lowerCapturedText(module *sema.Module, captured patterns.Capture) (ir.Value, bool) {
	tokens := captured.Tokens
	if len(tokens) < 2 || tokens[0].Type != token.StringStart || tokens[len(tokens)-1].Type != token.StringEnd {
		current.invalidSpan(module, captured.Span, "send text is not a string")
		return nil, false
	}

	text := &ir.Text{Source: sourceRef(module, captured.Span)}
	for index := 1; index < len(tokens)-1; index++ {
		currentToken := tokens[index]
		switch currentToken.Type {
		case token.StringText:
			value, _ := currentToken.Value.(string)
			text.Parts = append(text.Parts, &ir.TextLiteral{
				Value:  value,
				Source: sourceRef(module, tokenSpan(module, currentToken)),
			})
		case token.InterpStart:
			end := index + 1
			for end < len(tokens)-1 && tokens[end].Type != token.InterpEnd {
				end++
			}
			if end == len(tokens)-1 {
				current.invalidSpan(module, captured.Span, "unterminated text interpolation")
				return nil, false
			}
			value, ok := current.lowerInterpolation(module, tokens[index+1:end])
			if !ok {
				return nil, false
			}
			span := joinTokenSpans(module, currentToken, tokens[end])
			text.Parts = append(text.Parts, &ir.TextInterpolation{Value: value, Source: sourceRef(module, span)})
			index = end
		default:
			current.invalidSpan(module, tokenSpan(module, currentToken), "unsupported token in text capture")
			return nil, false
		}
	}
	return text, true
}

func (current *lowerer) lowerInterpolation(module *sema.Module, tokens []token.Token) (ir.Value, bool) {
	if len(tokens) == 0 {
		return nil, false
	}

	parts := make([]string, 0, len(tokens))
	for index := 0; index < len(tokens); index++ {
		if tokens[index].Type != token.Ident {
			if tokens[index].Type == token.LParen && index+1 == len(tokens)-1 && tokens[index+1].Type == token.RParen {
				break
			}
			current.invalidSpan(module, joinTokenSpans(module, tokens[0], tokens[len(tokens)-1]), "unsupported text interpolation")
			return nil, false
		}
		parts = append(parts, tokens[index].Lexeme)
		if index+1 < len(tokens) && tokens[index+1].Type == token.Dot {
			index++
		}
	}

	var function *sema.FunctionSymbol
	imported := false
	if len(parts) == 1 && module.Symbols != nil {
		function = module.Symbols.Functions[parts[0]]
	} else if len(parts) == 2 {
		imported = true
		if imported := module.Imports[parts[0]]; imported != nil && imported.Target != nil && imported.Target.Symbols != nil {
			function = imported.Target.Symbols.Functions[parts[1]]
		}
	}
	span := joinTokenSpans(module, tokens[0], tokens[len(tokens)-1])
	if function == nil || function.Module == nil {
		current.invalidSpan(module, span, "unresolved function in text interpolation")
		return nil, false
	}
	if imported && !function.Public {
		current.invalidSpan(module, span, "private imported function in text interpolation")
		return nil, false
	}
	if len(function.Parameters) != 0 {
		current.invalidSpan(module, span, "function with parameters in text interpolation")
		return nil, false
	}
	return &ir.Call{
		Function: ir.SymbolID{Module: function.Module.Source.RelPath, Name: function.Name},
		Source:   sourceRef(module, span),
	}, true
}

func (current *lowerer) lowerExpression(module *sema.Module, expression ast.Expression) (ir.Value, bool) {
	if expression == nil {
		current.invalidSpan(module, diagnostic.Span{}, "missing expression")
		return nil, false
	}
	ref := sourceRef(module, expression.Span())
	switch node := expression.(type) {
	case *ast.NilLiteral:
		return &ir.Nil{Source: ref}, true
	case *ast.BoolLiteral:
		return &ir.Bool{Value: node.Value, Source: ref}, true
	case *ast.IntLiteral:
		return &ir.Int{Value: node.Value, Source: ref}, true
	case *ast.FloatLiteral:
		return &ir.Float{Value: node.Value, Source: ref}, true
	case *ast.StringExpr:
		text := &ir.Text{Source: ref}
		for _, part := range node.Parts {
			switch value := part.(type) {
			case *ast.StringText:
				text.Parts = append(text.Parts, &ir.TextLiteral{Value: value.Value, Source: sourceRef(module, value.Span())})
			case *ast.StringInterpolation:
				lowered, ok := current.lowerExpression(module, value.Expression)
				if !ok {
					return nil, false
				}
				text.Parts = append(text.Parts, &ir.TextInterpolation{Value: lowered, Source: sourceRef(module, value.Span())})
			default:
				current.unsupported(module, part)
				return nil, false
			}
		}
		return text, true
	case *ast.CallExpr:
		symbol := module.ResolvedCalls[node]
		if symbol == nil || symbol.Module == nil {
			current.invalid(module, node, "unresolved function call")
			return nil, false
		}
		call := &ir.Call{Function: ir.SymbolID{Module: symbol.Module.Source.RelPath, Name: symbol.Name}, Source: ref}
		for _, argument := range node.Arguments {
			value, ok := current.lowerExpression(module, argument)
			if !ok {
				return nil, false
			}
			call.Arguments = append(call.Arguments, value)
		}
		return call, true
	case *ast.VariableExpr:
		symbol := module.ResolvedVariables[node]
		if symbol == nil {
			current.invalid(module, node, "unresolved variable reference")
			return nil, false
		}
		reference := &ir.Reference{Name: symbol.Name, Type: lowerType(symbol.Type), Source: ref}
		if !symbol.Local {
			id, ok := globalSymbolID(symbol)
			if !ok {
				current.invalid(module, node, "unresolved global variable identity")
				return nil, false
			}
			reference.Symbol = id
		}
		return reference, true
	case *ast.GroupExpr:
		return current.lowerExpression(module, node.Expression)
	default:
		current.unsupported(module, expression)
		return nil, false
	}
}

func lowerType(typ sema.Type) ir.Type {
	kind := ir.TypeUnknown
	switch typ.Kind {
	case sema.TypeNil:
		kind = ir.TypeNil
	case sema.TypeBool:
		kind = ir.TypeBool
	case sema.TypeInt:
		kind = ir.TypeInt
	case sema.TypeFloat:
		kind = ir.TypeFloat
	case sema.TypeString:
		kind = ir.TypeString
	case sema.TypeList:
		kind = ir.TypeList
	case sema.TypeMap:
		kind = ir.TypeMap
	case sema.TypeRange:
		kind = ir.TypeRange
	case sema.TypeNamed:
		kind = ir.TypeNamed
	}
	lowered := ir.Type{Kind: kind, Name: typ.Name}
	for _, argument := range typ.Arguments {
		lowered.Arguments = append(lowered.Arguments, lowerType(argument))
	}
	return lowered
}

func (current *lowerer) unsupported(module *sema.Module, node ast.Node) {
	current.diagnostics = append(current.diagnostics, diagnostic.Diagnostic{
		Code:     diagnostic.CodeUnsupportedASTNode,
		Phase:    diagnostic.PhaseIR,
		Severity: diagnostic.SeverityError,
		Message:  fmt.Sprintf("Unsupported AST node: %T.", node),
		File:     module.Source.RelPath,
		Span:     node.Span(),
	})
}

func (current *lowerer) invalid(module *sema.Module, node ast.Node, message string) {
	current.invalidSpan(module, node.Span(), message)
}

func (current *lowerer) invalidSpan(module *sema.Module, span diagnostic.Span, message string) {
	current.diagnostics = append(current.diagnostics, diagnostic.Diagnostic{
		Code:     diagnostic.CodeInvalidIRNode,
		Phase:    diagnostic.PhaseIR,
		Severity: diagnostic.SeverityError,
		Message:  "Cannot lower to IR: " + message + ".",
		File:     module.Source.RelPath,
		Span:     span,
	})
}

func symbolID(module *sema.Module, name string) ir.SymbolID {
	return ir.SymbolID{Module: module.Source.RelPath, Name: name}
}

func globalSymbolID(symbol *sema.VariableSymbol) (ir.SymbolID, bool) {
	if symbol == nil || symbol.Module == nil || symbol.Module.Symbols == nil {
		return ir.SymbolID{}, false
	}

	paths := make([]string, 0, 1)
	for path, candidate := range symbol.Module.Symbols.Globals {
		if candidate == symbol {
			paths = append(paths, path)
		}
	}
	if len(paths) == 0 {
		return ir.SymbolID{}, false
	}
	sort.Strings(paths)
	return ir.SymbolID{Module: symbol.Module.Source.RelPath, Name: paths[0]}, true
}

func sourceRef(module *sema.Module, span diagnostic.Span) ir.SourceRef {
	return ir.SourceRef{File: module.Source.RelPath, Span: span}
}

func tokenSpan(module *sema.Module, current token.Token) diagnostic.Span {
	startLine, startColumn, _ := module.Source.Map.LineColumn(current.StartOffset)
	endLine, endColumn, _ := module.Source.Map.LineColumn(current.EndOffset)
	return diagnostic.Span{
		StartLine:   startLine,
		StartColumn: startColumn,
		EndLine:     endLine,
		EndColumn:   endColumn,
		StartOffset: current.StartOffset,
		EndOffset:   current.EndOffset,
	}
}

func joinTokenSpans(module *sema.Module, first, last token.Token) diagnostic.Span {
	start := tokenSpan(module, first)
	end := tokenSpan(module, last)
	return diagnostic.Span{
		StartLine:   start.StartLine,
		StartColumn: start.StartColumn,
		EndLine:     end.EndLine,
		EndColumn:   end.EndColumn,
		StartOffset: start.StartOffset,
		EndOffset:   end.EndOffset,
	}
}
