package sema

import (
	"fmt"
	"strings"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
	"github.com/puff-lang/puff/internal/token"
)

func (checker *checker) checkExpression(module *Module, currentScope *scope, expression ast.Expression) Type {
	if expression == nil {
		return Type{Kind: TypeUnknown}
	}

	var typ Type
	switch expression := expression.(type) {
	case *ast.NilLiteral:
		typ = Type{Kind: TypeNil}
	case *ast.BoolLiteral:
		typ = Type{Kind: TypeBool}
	case *ast.IntLiteral:
		typ = Type{Kind: TypeInt}
	case *ast.FloatLiteral:
		typ = Type{Kind: TypeFloat}
	case *ast.StringExpr:
		typ = checker.checkString(module, currentScope, expression)
	case *ast.UnaryExpr:
		if expression == nil {
			return Type{Kind: TypeUnknown}
		}
		typ = checker.checkUnary(module, currentScope, expression)
	case *ast.BinaryExpr:
		if expression == nil {
			return Type{Kind: TypeUnknown}
		}
		typ = checker.checkBinary(module, currentScope, expression)
	case *ast.GroupExpr:
		if expression == nil {
			return Type{Kind: TypeUnknown}
		}
		typ = checker.checkExpression(module, currentScope, expression.Expression)
	case *ast.CallExpr:
		if expression == nil {
			return Type{Kind: TypeUnknown}
		}
		typ = checker.checkCall(module, currentScope, expression)
	case *ast.VariableExpr:
		typ = checker.checkVariable(module, currentScope, expression)
	case *ast.ListExpr:
		if expression == nil {
			return Type{Kind: TypeUnknown}
		}
		typ = checker.checkList(module, currentScope, expression)
	case *ast.MapExpr:
		if expression == nil {
			return Type{Kind: TypeUnknown}
		}
		typ = checker.checkMap(module, currentScope, expression)
	case *ast.RangeExpr:
		if expression == nil {
			return Type{Kind: TypeUnknown}
		}
		typ = checker.checkRange(module, currentScope, expression)
	case *ast.PatternExpr, *ast.AccessExpr:
		typ = Type{Kind: TypeUnknown}
	default:
		typ = Type{Kind: TypeUnknown}
	}

	if module != nil && module.ExpressionTypes != nil {
		module.ExpressionTypes[expression] = typ
	}
	return typ
}

func (checker *checker) checkString(module *Module, currentScope *scope, expression *ast.StringExpr) Type {
	if expression != nil {
		for _, part := range expression.Parts {
			if interpolation, ok := part.(*ast.StringInterpolation); ok {
				checker.checkExpression(module, currentScope, interpolation.Expression)
			}
		}
	}
	return Type{Kind: TypeString}
}

func (checker *checker) checkUnary(module *Module, currentScope *scope, expression *ast.UnaryExpr) Type {
	operand := checker.checkExpression(module, currentScope, expression.Operand)
	switch expression.Operator {
	case token.Not:
		if !operand.IsUnknown() && operand.Kind != TypeBool {
			checker.typeMismatch(module, expression,
				fmt.Sprintf("Type mismatch: cannot use not with %s.", operand.String()))
		}
		return Type{Kind: TypeBool}
	case token.Minus:
		if operand.IsUnknown() {
			return operand
		}
		if operand.Kind != TypeInt && operand.Kind != TypeFloat {
			checker.typeMismatch(module, expression,
				fmt.Sprintf("Type mismatch: cannot negate %s.", operand.String()))
			return Type{Kind: TypeUnknown}
		}
		return operand
	default:
		return Type{Kind: TypeUnknown}
	}
}

func (checker *checker) checkBinary(module *Module, currentScope *scope, expression *ast.BinaryExpr) Type {
	left := checker.checkExpression(module, currentScope, expression.Left)
	right := checker.checkExpression(module, currentScope, expression.Right)

	switch expression.Operator {
	case token.Plus, token.Minus, token.Star, token.Slash, token.Percent:
		if expression.Operator == token.Plus && left.Kind == TypeString && right.Kind == TypeString {
			return Type{Kind: TypeString}
		}
		result := numericType(left, right)
		if !left.IsUnknown() && !right.IsUnknown() && result.IsUnknown() {
			checker.typeMismatch(module, expression, arithmeticMismatch(expression.Operator, left, right))
		}
		return result
	case token.And, token.Or:
		if (!left.IsUnknown() && left.Kind != TypeBool) || (!right.IsUnknown() && right.Kind != TypeBool) {
			checker.typeMismatch(module, expression,
				fmt.Sprintf("Type mismatch: cannot use %s with %s and %s.",
					operatorText(expression.Operator), left.String(), right.String()))
		}
		return Type{Kind: TypeBool}
	case token.EqualEqual, token.BangEqual:
		if !left.IsUnknown() && !right.IsUnknown() &&
			!compatible(left, right) && !compatible(right, left) {
			checker.typeMismatch(module, expression,
				fmt.Sprintf("Type mismatch: cannot compare %s and %s.", left.String(), right.String()))
		}
		return Type{Kind: TypeBool}
	case token.Greater, token.GreaterEq, token.Less, token.LessEq:
		if !left.IsUnknown() && !right.IsUnknown() && numericType(left, right).IsUnknown() {
			checker.typeMismatch(module, expression,
				fmt.Sprintf("Type mismatch: cannot compare %s and %s.", left.String(), right.String()))
		}
		return Type{Kind: TypeBool}
	default:
		return Type{Kind: TypeUnknown}
	}
}

func (checker *checker) checkCall(module *Module, currentScope *scope, call *ast.CallExpr) Type {
	for _, argument := range call.Arguments {
		checker.checkExpression(module, currentScope, argument)
	}

	name := qualifiedName(call.Callee)
	if !call.ExplicitParens {
		if typ, ok := currentScope.lookupName(name); ok {
			return typ
		}
	}

	function := checker.resolveFunction(module, call)
	if function == nil {
		if isContextualName(name) {
			hint := "Declare the name before using it."
			if name == "player" && currentScope != nil {
				hint = `The name "player" is only available inside events that inject a player.`
			}
			checker.report(module, &call.Callee, diagnostic.CodeUndefinedName,
				fmt.Sprintf("Undefined name: %s", name),
				hint)
		} else {
			checker.report(module, &call.Callee, diagnostic.CodeUndefinedFunction,
				fmt.Sprintf("Undefined function: %s", name),
				fmt.Sprintf("Declare fun %s before using it, or import it from a module.", name))
		}
		return Type{Kind: TypeUnknown}
	}

	module.ResolvedCalls[call] = function
	checker.checkArguments(module, call, function)
	return function.ReturnType
}

func (checker *checker) resolveFunction(module *Module, call *ast.CallExpr) *FunctionSymbol {
	if module == nil || module.Symbols == nil || call == nil {
		return nil
	}

	parts := call.Callee.Parts
	if len(parts) == 1 {
		return module.Symbols.Functions[parts[0].Name]
	}
	if len(parts) != 2 {
		return nil
	}

	imported, ok := module.Import(parts[0].Name)
	if !ok || imported == nil || imported.Target == nil || imported.Target.Symbols == nil {
		return nil
	}
	function := imported.Target.Symbols.Functions[parts[1].Name]
	if function == nil || !function.Public {
		return nil
	}
	return function
}

func (checker *checker) checkArguments(module *Module, call *ast.CallExpr, function *FunctionSymbol) {
	expected := len(function.Parameters)
	actual := len(call.Arguments)
	if expected > 0 && actual < expected {
		checker.report(module, call, diagnostic.CodeMissingArguments,
			fmt.Sprintf("Missing arguments for function: %s", qualifiedName(call.Callee)),
			fmt.Sprintf("Call it with parentheses: %s(%s)",
				qualifiedName(call.Callee), strings.Join(parameterNames(function.Declaration), ", ")))
	}
	if actual > expected {
		checker.report(module, call, diagnostic.CodeTooManyArguments, "Too many arguments.", "")
	}

	limit := actual
	if expected < limit {
		limit = expected
	}
	for index := 0; index < limit; index++ {
		actualType := module.ExpressionTypes[call.Arguments[index]]
		if !compatible(function.Parameters[index], actualType) {
			checker.report(module, call.Arguments[index], diagnostic.CodeInvalidArgumentType,
				"Invalid argument type.", "")
		}
	}
}

func parameterNames(declaration *ast.FunctionDecl) []string {
	if declaration == nil {
		return nil
	}
	names := make([]string, 0, len(declaration.Parameters))
	for _, parameter := range declaration.Parameters {
		names = append(names, parameter.Name.Name)
	}
	return names
}

func qualifiedName(name ast.QualifiedName) string {
	parts := make([]string, 0, len(name.Parts))
	for _, part := range name.Parts {
		parts = append(parts, part.Name)
	}
	return strings.Join(parts, ".")
}

func isContextualName(name string) bool {
	switch name {
	case "player", "console", "loop.index", "loop.value", "loop.player", "loop.entity":
		return true
	default:
		return false
	}
}

func (checker *checker) checkVariable(module *Module, currentScope *scope, variable *ast.VariableExpr) Type {
	if variable == nil {
		return Type{Kind: TypeUnknown}
	}
	checker.checkVariableAccesses(module, currentScope, variable)

	var symbol *VariableSymbol
	if variable.Qualifier != nil {
		imported, ok := module.Import(variable.Qualifier.Name)
		if ok && imported != nil && imported.Target != nil && imported.Target.Symbols != nil {
			symbol = imported.Target.Symbols.lookupGlobal(variable)
			if symbol != nil && !symbol.Public {
				symbol = nil
			}
		}
	} else if variable.Local {
		symbol, _ = currentScope.lookupLocal(variable.Name.Name)
	} else if typ, ok := currentScope.lookupName(variable.Name.Name); ok {
		return checker.typeAfterAccesses(typ, variable.Accesses)
	} else if module != nil && module.Symbols != nil {
		symbol = module.Symbols.lookupGlobal(variable)
	}

	if symbol == nil {
		checker.report(module, variable, diagnostic.CodeUndefinedVariable,
			fmt.Sprintf("Undefined variable: %s", variableName(variable)),
			fmt.Sprintf("Declare it before using it: %s = 0", variableName(variable)))
		return Type{Kind: TypeUnknown}
	}
	if _, isGlobalDeclaration := symbol.Declaration.(*ast.GlobalAssignment); isGlobalDeclaration &&
		symbol.Module == module && !symbol.initialized {
		checker.report(module, variable, diagnostic.CodeUndefinedVariable,
			fmt.Sprintf("Undefined variable: %s", variableName(variable)),
			fmt.Sprintf("Declare it before using it: %s = 0", variableName(variable)))
		return Type{Kind: TypeUnknown}
	}

	module.ResolvedVariables[variable] = symbol
	return checker.typeAfterAccesses(symbol.Type, variable.Accesses[symbol.AccessDepth:])
}

func (checker *checker) checkVariableAccesses(module *Module, currentScope *scope, variable *ast.VariableExpr) {
	if variable == nil {
		return
	}
	for _, access := range variable.Accesses {
		if index, ok := access.(*ast.IndexAccess); ok {
			checker.checkExpression(module, currentScope, index.Index)
		}
	}
}

func (checker *checker) typeAfterAccesses(typ Type, accesses []ast.VariableAccess) Type {
	for _, access := range accesses {
		switch access.(type) {
		case *ast.FieldAccess:
			typ = Type{Kind: TypeUnknown}
		case *ast.IndexAccess:
			if len(typ.Arguments) > 0 && (typ.Kind == TypeList || typ.Kind == TypeRange) {
				typ = typ.Arguments[0]
			} else if len(typ.Arguments) > 1 && typ.Kind == TypeMap {
				typ = typ.Arguments[1]
			} else {
				typ = Type{Kind: TypeUnknown}
			}
		case *ast.EmptyIndexAccess:
			// Empty brackets identify the collection itself in Puff.
		}
	}
	return typ
}

func variableName(variable *ast.VariableExpr) string {
	if variable == nil {
		return "$"
	}
	name := "$"
	if variable.Local {
		name += "_"
	}
	name += variable.Name.Name
	if variable.Qualifier != nil {
		name = variable.Qualifier.Name + "." + name
	}
	for _, access := range variable.Accesses {
		field, ok := access.(*ast.FieldAccess)
		if !ok {
			break
		}
		name += "." + field.Field.Name
	}
	return name
}

func (checker *checker) checkList(module *Module, currentScope *scope, expression *ast.ListExpr) Type {
	elementType := Type{Kind: TypeUnknown}
	for index, element := range expression.Elements {
		current := checker.checkExpression(module, currentScope, element)
		if index == 0 {
			elementType = current
		} else {
			elementType = mergeInferredTypes(elementType, current)
		}
	}
	return Type{Kind: TypeList, Arguments: []Type{elementType}}
}

func (checker *checker) checkMap(module *Module, currentScope *scope, expression *ast.MapExpr) Type {
	keyType := Type{Kind: TypeUnknown}
	valueType := Type{Kind: TypeUnknown}
	for index, entry := range expression.Entries {
		key := checker.checkExpression(module, currentScope, entry.Key)
		value := checker.checkExpression(module, currentScope, entry.Value)
		if index == 0 {
			keyType = key
			valueType = value
			continue
		}
		keyType = mergeInferredTypes(keyType, key)
		valueType = mergeInferredTypes(valueType, value)
	}
	return Type{Kind: TypeMap, Arguments: []Type{keyType, valueType}}
}

func mergeInferredTypes(left Type, right Type) Type {
	if left.IsUnknown() && !left.incompatible || right.IsUnknown() && !right.incompatible {
		return Type{Kind: TypeUnknown}
	}
	if left.incompatible || right.incompatible {
		return Type{Kind: TypeUnknown, incompatible: true}
	}
	if numeric := numericType(left, right); !numeric.IsUnknown() {
		return numeric
	}
	if left.Kind != right.Kind || left.Kind == TypeNamed && left.Name != right.Name {
		return Type{Kind: TypeUnknown, incompatible: true}
	}
	if len(left.Arguments) == 0 {
		return right
	}
	if len(right.Arguments) == 0 {
		return left
	}
	if len(left.Arguments) != len(right.Arguments) {
		return Type{Kind: TypeUnknown, incompatible: true}
	}

	merged := left
	merged.Arguments = make([]Type, len(left.Arguments))
	for index := range left.Arguments {
		merged.Arguments[index] = mergeInferredTypes(left.Arguments[index], right.Arguments[index])
	}
	return merged
}

func (checker *checker) checkRange(module *Module, currentScope *scope, expression *ast.RangeExpr) Type {
	start := checker.checkExpression(module, currentScope, expression.Start)
	end := checker.checkExpression(module, currentScope, expression.End)
	element := numericType(start, end)
	if !start.IsUnknown() && !end.IsUnknown() && element.IsUnknown() {
		checker.typeMismatch(module, expression,
			fmt.Sprintf("Type mismatch: range bounds must be numeric, got %s and %s.",
				start.String(), end.String()))
	}
	return Type{Kind: TypeRange, Arguments: []Type{element}}
}

func (checker *checker) typeMismatch(module *Module, node ast.Node, message string) {
	checker.report(module, node, diagnostic.CodeTypeMismatch, message,
		"Convert one value or use compatible types.")
}

func arithmeticMismatch(operator token.Type, left Type, right Type) string {
	verb := map[token.Type]string{
		token.Plus:    "add",
		token.Minus:   "subtract",
		token.Star:    "multiply",
		token.Slash:   "divide",
		token.Percent: "apply modulo to",
	}[operator]
	return fmt.Sprintf("Type mismatch: cannot %s %s and %s.", verb, left.String(), right.String())
}

func operatorText(operator token.Type) string {
	switch operator {
	case token.And:
		return "and"
	case token.Or:
		return "or"
	default:
		return string(operator)
	}
}
