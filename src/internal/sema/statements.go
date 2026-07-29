package sema

import (
	"fmt"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
)

type flowContext struct {
	function   *ast.FunctionDecl
	returnType Type
}

func (checker *checker) checkBlock(
	module *Module,
	currentScope *scope,
	block ast.Block,
	context flowContext,
) bool {
	fallsThrough := true
	for _, statement := range block.Statements {
		statementFallsThrough := checker.checkStatement(module, currentScope, statement, context)
		if fallsThrough {
			fallsThrough = statementFallsThrough
		}
	}
	return fallsThrough
}

func (checker *checker) checkStatement(
	module *Module,
	currentScope *scope,
	statement ast.Statement,
	context flowContext,
) bool {
	switch statement := statement.(type) {
	case *ast.AssignmentStmt:
		checker.checkAssignment(module, currentScope, statement)
	case *ast.AddStmt:
		checker.checkAdd(module, currentScope, statement)
	case *ast.IfStmt:
		return checker.checkIf(module, currentScope, statement, context)
	case *ast.LoopTimesStmt:
		if statement == nil {
			return true
		}
		checker.checkLoopTimes(module, currentScope, statement, context)
	case *ast.LoopRangeStmt:
		if statement == nil {
			return true
		}
		checker.checkLoopRange(module, currentScope, statement, context)
	case *ast.LoopPlayersStmt:
		if statement == nil {
			return true
		}
		checker.checkLoopPlayers(module, currentScope, statement, context)
	case *ast.LoopEntitiesStmt:
		if statement == nil {
			return true
		}
		checker.checkLoopEntities(module, currentScope, statement, context)
	case *ast.ReturnStmt:
		if statement == nil {
			return true
		}
		checker.checkReturn(module, currentScope, statement, context)
		return false
	case *ast.StopStmt:
		if statement == nil {
			return true
		}
		checker.checkStop(module, statement, context)
		return false
	case *ast.ExprStmt:
		if statement == nil {
			return true
		}
		checker.checkExpression(module, currentScope, statement.Expression)
	case *ast.EffectStmt:
		// Effect internals are raw pattern tokens until T12.
	}
	return true
}

func (checker *checker) checkAssignment(
	module *Module,
	currentScope *scope,
	statement *ast.AssignmentStmt,
) {
	if statement == nil {
		return
	}
	valueType := checker.checkExpression(module, currentScope, statement.Value)
	target := statement.Target
	if target == nil {
		return
	}
	checker.checkVariableAccesses(module, currentScope, target)

	if target.Qualifier != nil {
		checker.checkImportedAssignment(module, target)
		return
	}

	if target.Local {
		symbol := &VariableSymbol{
			Name:        target.Name.Name,
			Declaration: statement,
			Module:      module,
			Type:        valueType,
			Local:       true,
		}
		currentScope.defineLocal(symbol)
		module.ResolvedVariables[target] = symbol
		return
	}

	if _, contextual := currentScope.lookupName(target.Name.Name); contextual {
		return
	}

	symbol := module.Symbols.Globals[target.Name.Name]
	if symbol == nil {
		symbol = &VariableSymbol{
			Name:        target.Name.Name,
			Declaration: statement,
			Module:      module,
		}
		module.Symbols.Globals[target.Name.Name] = symbol
	}
	if len(target.Accesses) == 0 {
		symbol.Type = valueType
	}
	module.ResolvedVariables[target] = symbol
}

func (checker *checker) checkImportedAssignment(module *Module, target *ast.VariableExpr) {
	imported, ok := module.Import(target.Qualifier.Name)
	if !ok || imported == nil || imported.Target == nil || imported.Target.Symbols == nil {
		checker.undefinedVariable(module, target)
		return
	}

	symbol := imported.Target.Symbols.Globals[target.Name.Name]
	if symbol == nil || !symbol.Public {
		checker.undefinedVariable(module, target)
		return
	}

	module.ResolvedVariables[target] = symbol
	checker.report(module, target, diagnostic.CodeAssignToImportedPublicVar,
		fmt.Sprintf("Cannot assign to imported public variable: %s", variableName(target)),
		fmt.Sprintf("Use a public function like %s.setTax(0.2).", target.Qualifier.Name))
}

func (checker *checker) undefinedVariable(module *Module, variable *ast.VariableExpr) {
	checker.report(module, variable, diagnostic.CodeUndefinedVariable,
		fmt.Sprintf("Undefined variable: %s", variableName(variable)),
		fmt.Sprintf("Declare it before using it: %s = 0", variableName(variable)))
}

func (checker *checker) checkAdd(module *Module, currentScope *scope, statement *ast.AddStmt) {
	if statement == nil {
		return
	}
	checker.checkExpression(module, currentScope, statement.Value)
	if target, ok := statement.Target.(*ast.VariableExpr); ok {
		checker.checkVariable(module, currentScope, target)
	}
	// AccessExpr is deliberately deferred to T12.
}

func (checker *checker) checkIf(
	module *Module,
	currentScope *scope,
	statement *ast.IfStmt,
	context flowContext,
) bool {
	if statement == nil {
		return true
	}
	checker.requireBool(module, statement.Condition,
		checker.checkExpression(module, currentScope, statement.Condition))
	fallsThrough := checker.checkBlock(module, currentScope, statement.Then, context)

	for _, clause := range statement.ElseIf {
		checker.requireBool(module, clause.Condition,
			checker.checkExpression(module, currentScope, clause.Condition))
		if checker.checkBlock(module, currentScope, clause.Body, context) {
			fallsThrough = true
		}
	}
	if statement.Else == nil {
		return true
	}
	if checker.checkBlock(module, currentScope, *statement.Else, context) {
		fallsThrough = true
	}
	return fallsThrough
}

func (checker *checker) checkLoopTimes(
	module *Module,
	currentScope *scope,
	statement *ast.LoopTimesStmt,
	context flowContext,
) {
	count := checker.checkExpression(module, currentScope, statement.Count)
	checker.requireNumeric(module, statement.Count, count)
	loopScope := newInjectedScope(currentScope)
	loopScope.defineName("loop.index", Type{Kind: TypeInt})
	checker.checkBlock(module, loopScope, statement.Body, context)
}

func (checker *checker) checkLoopRange(
	module *Module,
	currentScope *scope,
	statement *ast.LoopRangeStmt,
	context flowContext,
) {
	start := checker.checkExpression(module, currentScope, statement.Start)
	end := checker.checkExpression(module, currentScope, statement.End)
	checker.requireNumeric(module, statement.Start, start)
	checker.requireNumeric(module, statement.End, end)

	valueType := numericType(start, end)
	loopScope := newInjectedScope(currentScope)
	loopScope.defineName("loop.index", Type{Kind: TypeInt})
	loopScope.defineName("loop.value", valueType)
	checker.checkBlock(module, loopScope, statement.Body, context)
}

func (checker *checker) checkLoopPlayers(
	module *Module,
	currentScope *scope,
	statement *ast.LoopPlayersStmt,
	context flowContext,
) {
	loopScope := newInjectedScope(currentScope)
	loopScope.defineName("loop.index", Type{Kind: TypeInt})
	loopScope.defineName("loop.player", Type{Kind: TypeNamed, Name: "Player"})
	checker.checkBlock(module, loopScope, statement.Body, context)
}

func (checker *checker) checkLoopEntities(
	module *Module,
	currentScope *scope,
	statement *ast.LoopEntitiesStmt,
	context flowContext,
) {
	radius := checker.checkExpression(module, currentScope, statement.Radius)
	checker.requireNumeric(module, statement.Radius, radius)
	checker.checkExpression(module, currentScope, statement.Around)

	loopScope := newInjectedScope(currentScope)
	loopScope.defineName("loop.index", Type{Kind: TypeInt})
	loopScope.defineName("loop.entity", Type{Kind: TypeNamed, Name: "Entity"})
	checker.checkBlock(module, loopScope, statement.Body, context)
}

func (checker *checker) checkReturn(
	module *Module,
	currentScope *scope,
	statement *ast.ReturnStmt,
	context flowContext,
) {
	if context.function == nil {
		if statement.Value != nil {
			checker.checkExpression(module, currentScope, statement.Value)
		}
		checker.report(module, statement, diagnostic.CodeInvalidReturnOutsideFunction,
			"return can only be used inside functions.",
			"Use stop to stop an event or execution block.")
		return
	}

	if statement.Value == nil {
		if context.function.ReturnType != nil && !context.returnType.IsUnknown() {
			checker.report(module, statement, diagnostic.CodeMissingReturnValue,
				"Missing return value.",
				fmt.Sprintf("Return a value compatible with %s.", context.returnType.String()))
		}
		return
	}

	actual := checker.checkExpression(module, currentScope, statement.Value)
	if context.function.ReturnType != nil && !compatible(context.returnType, actual) {
		checker.report(module, statement.Value, diagnostic.CodeTypeMismatch,
			fmt.Sprintf("Type mismatch: cannot return %s as %s.", actual.String(), context.returnType.String()),
			fmt.Sprintf("Return a value compatible with %s.", context.returnType.String()))
	}
}

func (checker *checker) checkStop(module *Module, statement *ast.StopStmt, context flowContext) {
	if context.function == nil || context.function.ReturnType == nil || context.returnType.IsUnknown() {
		return
	}
	checker.report(module, statement, diagnostic.CodeInvalidStopInReturningFunc,
		"stop cannot replace a return value.",
		fmt.Sprintf("Return a value compatible with %s.", context.returnType.String()))
}

func (checker *checker) requireBool(module *Module, node ast.Node, typ Type) {
	if typ.IsUnknown() || typ.Kind == TypeBool {
		return
	}
	checker.typeMismatch(module, node,
		fmt.Sprintf("Type mismatch: condition must be bool, got %s.", typ.String()))
}

func (checker *checker) requireNumeric(module *Module, node ast.Node, typ Type) {
	if typ.IsUnknown() || typ.Kind == TypeInt || typ.Kind == TypeFloat {
		return
	}
	checker.typeMismatch(module, node,
		fmt.Sprintf("Type mismatch: expected a number, got %s.", typ.String()))
}
