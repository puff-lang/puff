package sema

import (
	"fmt"
	"sort"
	"strings"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/diagnostic"
)

type checker struct {
	project             *Project
	diagnostics         []diagnostic.Diagnostic
	suppressDiagnostics bool
	unresolvedGlobal    *globalDependencyState
}

type globalDependencyState struct {
	unresolved bool
	reported   bool
}

func Check(project *Project) Result {
	if project == nil {
		return Result{Diagnostics: []diagnostic.Diagnostic{}}
	}

	checker := &checker{
		project:     project,
		diagnostics: make([]diagnostic.Diagnostic, 0),
	}
	checker.initializeModules()
	checker.indexDeclarations()
	checker.checkModules()

	return Result{
		Project:     project,
		Diagnostics: checker.diagnostics,
	}
}

func (checker *checker) initializeModules() {
	for _, module := range checker.project.Modules {
		if module == nil {
			continue
		}
		module.Symbols = newSymbolTable()
		module.ExpressionTypes = make(map[ast.Expression]Type)
		module.ResolvedCalls = make(map[*ast.CallExpr]*FunctionSymbol)
		module.ResolvedVariables = make(map[*ast.VariableExpr]*VariableSymbol)
	}
}

func (checker *checker) indexDeclarations() {
	for _, module := range checker.project.Modules {
		if module == nil || module.Syntax == nil {
			continue
		}
		for _, declaration := range module.Syntax.Declarations {
			switch declaration := declaration.(type) {
			case *ast.FunctionDecl:
				checker.indexFunction(module, declaration)
			case *ast.GlobalAssignment:
				checker.indexGlobal(module, declaration)
			}
		}
	}
}

func (checker *checker) indexFunction(module *Module, declaration *ast.FunctionDecl) {
	if declaration == nil {
		return
	}

	symbol := &FunctionSymbol{
		Name:        declaration.Name.Name,
		Declaration: declaration,
		Module:      module,
		ReturnType:  Type{Kind: TypeUnknown},
		Public:      declaration.Public,
	}
	for _, parameter := range declaration.Parameters {
		symbol.Parameters = append(symbol.Parameters, checker.resolveType(module, parameter.Type))
	}
	if declaration.ReturnType != nil {
		symbol.ReturnType = checker.resolveType(module, declaration.ReturnType)
	}
	if symbol.Name != "" {
		module.Symbols.Functions[symbol.Name] = symbol
	}
}

func (checker *checker) indexGlobal(module *Module, declaration *ast.GlobalAssignment) {
	if declaration == nil || declaration.Target == nil {
		return
	}

	target := declaration.Target
	if target.Local && declaration.Public {
		checker.report(module, target, diagnostic.CodeInvalidPublicLocalVariable,
			"Local variables cannot be public.",
			"Only global variables can be exported.")
		return
	}
	if target.Local || target.Name.Name == "" {
		return
	}

	path, depth := globalPath(target)
	module.Symbols.Globals[path] = &VariableSymbol{
		Name:        target.Name.Name,
		Declaration: declaration,
		Module:      module,
		Type:        Type{Kind: TypeUnknown},
		Public:      declaration.Public,
		AccessDepth: depth,
	}
}

func (checker *checker) resolveType(module *Module, ref *ast.TypeRef) Type {
	if ref == nil {
		return Type{Kind: TypeUnknown}
	}

	kind, ok := builtInTypes[ref.Name.Name]
	if !ok {
		checker.report(module, &ref.Name, diagnostic.CodeUndefinedType,
			fmt.Sprintf("Undefined type: %s", ref.Name.Name), "")
		for _, argument := range ref.Arguments {
			checker.resolveType(module, argument)
		}
		return Type{Kind: TypeUnknown}
	}

	typ := Type{Kind: kind, Name: ref.Name.Name}
	for _, argument := range ref.Arguments {
		typ.Arguments = append(typ.Arguments, checker.resolveType(module, argument))
	}
	return typ
}

func (checker *checker) checkModules() {
	for _, module := range checker.project.Modules {
		if module == nil || module.Syntax == nil {
			continue
		}
		checker.checkRequiredEvents(module)
	}
	checker.checkGlobalInitializersInDependencyOrder()

	for _, module := range checker.project.Modules {
		if module == nil || module.Syntax == nil {
			continue
		}
		for _, declaration := range module.Syntax.Declarations {
			switch declaration := declaration.(type) {
			case *ast.FunctionDecl:
				checker.checkFunction(module, declaration)
			case *ast.EventDecl:
				checker.checkEvent(module, declaration)
			}
		}
	}
}

func (checker *checker) checkGlobalInitializersInDependencyOrder() {
	modules := append([]*Module(nil), checker.project.Modules...)
	sort.SliceStable(modules, func(left, right int) bool {
		if modules[left] == nil {
			return false
		}
		if modules[right] == nil {
			return true
		}
		return modules[left].Source.RelPath < modules[right].Source.RelPath
	})

	globalCount := 0
	for _, module := range modules {
		if module == nil || module.Syntax == nil {
			continue
		}
		for _, declaration := range module.Syntax.Declarations {
			if _, ok := declaration.(*ast.GlobalAssignment); ok {
				globalCount++
			}
		}
	}

	checker.suppressDiagnostics = true
	for iteration := 0; iteration <= globalCount; iteration++ {
		checker.resetGlobalInitialization(modules)
		changed := false
		for _, module := range modules {
			changed = checker.checkGlobalInitializers(module, true) || changed
		}
		if !changed {
			break
		}
	}
	checker.suppressDiagnostics = false

	checker.resetGlobalInitialization(modules)
	checker.resetGlobalReports(modules)
	for _, module := range modules {
		checker.checkGlobalInitializers(module, true)
	}
}

func (checker *checker) resetGlobalInitialization(modules []*Module) {
	for _, module := range modules {
		if module == nil || module.Symbols == nil {
			continue
		}
		for _, symbol := range module.Symbols.Globals {
			symbol.initialized = false
		}
	}
}

func (checker *checker) resetGlobalReports(modules []*Module) {
	for _, module := range modules {
		if module == nil || module.Symbols == nil {
			continue
		}
		for _, symbol := range module.Symbols.Globals {
			symbol.reported = false
		}
	}
}

func (checker *checker) checkGlobalInitializers(module *Module, updateTypes bool) bool {
	if module == nil || module.Syntax == nil {
		return false
	}

	changed := false
	for _, declaration := range module.Syntax.Declarations {
		global, ok := declaration.(*ast.GlobalAssignment)
		if !ok || global == nil {
			continue
		}

		dependency := globalDependencyState{}
		checker.unresolvedGlobal = &dependency
		typ := checker.checkExpression(module, nil, global.Value)
		checker.unresolvedGlobal = nil
		if global.Target == nil || global.Target.Local {
			continue
		}
		if endsWithEmptyIndex(global.Target) && !typ.IsUnknown() && typ.Kind != TypeList {
			checker.typeMismatch(module, global.Value,
				fmt.Sprintf("Type mismatch: cannot assign %s to %s[].", typ.String(), variableName(global.Target)))
		}
		path, _ := globalPath(global.Target)
		if symbol, ok := module.Symbols.Globals[path]; ok {
			if updateTypes && !sameType(symbol.Type, typ) {
				symbol.Type = typ
				changed = true
			}
			resolution := globalResolved
			if dependency.unresolved {
				resolution = globalUnresolved
			}
			if symbol.resolution != resolution {
				symbol.resolution = resolution
				changed = true
			}
			if !checker.suppressDiagnostics {
				symbol.reported = dependency.reported
			}
			symbol.initialized = true
			module.ResolvedVariables[global.Target] = symbol
		}
		checker.checkVariableAccesses(module, nil, global.Target)
	}
	return changed
}

func sameType(left Type, right Type) bool {
	if left.Kind != right.Kind || left.Name != right.Name ||
		left.incompatible != right.incompatible || left.placeholder != right.placeholder {
		return false
	}
	if len(left.Arguments) != len(right.Arguments) {
		return false
	}
	for index := range left.Arguments {
		if !sameType(left.Arguments[index], right.Arguments[index]) {
			return false
		}
	}
	return true
}

func endsWithEmptyIndex(variable *ast.VariableExpr) bool {
	if variable == nil || len(variable.Accesses) == 0 {
		return false
	}
	_, ok := variable.Accesses[len(variable.Accesses)-1].(*ast.EmptyIndexAccess)
	return ok
}

func (checker *checker) checkFunction(module *Module, declaration *ast.FunctionDecl) {
	if declaration == nil {
		return
	}

	currentScope := checker.runtimeScope()
	for index := range declaration.Parameters {
		parameter := &declaration.Parameters[index]
		typ := Type{Kind: TypeUnknown}
		if symbol := module.Symbols.Functions[declaration.Name.Name]; symbol != nil && index < len(symbol.Parameters) {
			typ = symbol.Parameters[index]
		}
		currentScope.defineName(parameter.Name.Name, typ)
	}

	context := flowContext{
		function: declaration,
		returnType: func() Type {
			if symbol := module.Symbols.Functions[declaration.Name.Name]; symbol != nil {
				return symbol.ReturnType
			}
			return Type{Kind: TypeUnknown}
		}(),
	}
	fallsThrough := checker.checkBlock(module, currentScope, declaration.Body, context)
	if declaration.ReturnType != nil && fallsThrough {
		checker.report(module, &declaration.Name, diagnostic.CodeMissingReturn,
			fmt.Sprintf("Function %s must return %s in all paths.", declaration.Name.Name, context.returnType.String()),
			"Add an else branch or a final return.")
	}
}

func (checker *checker) checkEvent(module *Module, declaration *ast.EventDecl) {
	if declaration == nil {
		return
	}

	currentScope := checker.runtimeScope()
	if eventName(declaration) == "join" {
		currentScope.defineName("player", Type{Kind: TypeNamed, Name: "Player"})
	}
	checker.checkBlock(module, currentScope, declaration.Body, flowContext{})
}

func (checker *checker) runtimeScope() *scope {
	currentScope := newExecutionScope()
	currentScope.defineName("console", Type{Kind: TypeNamed, Name: "Command"})
	return currentScope
}

func (checker *checker) checkRequiredEvents(module *Module) {
	required := make(map[string]*ast.MetadataEntry)
	for index := range module.Syntax.Metadata {
		entry := &module.Syntax.Metadata[index]
		if entry.Key != "tags" {
			continue
		}
		for _, tag := range strings.Split(entry.Value, ",") {
			tag = strings.TrimSpace(tag)
			if tag == "load" || tag == "tick" {
				required[tag] = entry
			}
		}
	}
	if len(required) == 0 {
		return
	}

	declared := make(map[string]bool)
	for _, declaration := range module.Syntax.Declarations {
		event, ok := declaration.(*ast.EventDecl)
		if !ok || event == nil || len(event.Name) != 1 {
			continue
		}
		declared[event.Name[0].Name] = true
	}

	if entry := required["load"]; entry != nil && !declared["load"] {
		checker.report(module, entry, diagnostic.CodeMissingLoadEvent,
			"Missing required event: on load",
			"Add an on load block or remove the load tag.")
	}
	if entry := required["tick"]; entry != nil && !declared["tick"] {
		checker.report(module, entry, diagnostic.CodeMissingTickEvent,
			"Missing required event: on tick",
			"Add an on tick block or remove the tick tag.")
	}
}

func eventName(declaration *ast.EventDecl) string {
	if declaration == nil || len(declaration.Name) != 1 {
		return ""
	}
	return declaration.Name[0].Name
}

func (checker *checker) report(
	module *Module,
	node ast.Node,
	code diagnostic.Code,
	message string,
	hint string,
) {
	if checker.suppressDiagnostics {
		return
	}
	checker.diagnostics = append(checker.diagnostics, semanticDiagnostic(module, node, code, message, hint))
}
