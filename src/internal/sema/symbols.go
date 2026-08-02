package sema

import (
	"strconv"
	"strings"

	"github.com/puff-lang/puff/internal/ast"
	"github.com/puff-lang/puff/internal/token"
)

type globalResolution uint8

const (
	globalUnresolved globalResolution = iota
	globalResolved
)

type FunctionSymbol struct {
	Name        string
	Declaration *ast.FunctionDecl
	Module      *Module
	Parameters  []Type
	ReturnType  Type
	Public      bool
}

type VariableSymbol struct {
	Name        string
	Declaration ast.Node
	Module      *Module
	Type        Type
	Public      bool
	Local       bool
	AccessDepth int
	initialized bool
	resolution  globalResolution
	reported    bool
}

type SymbolTable struct {
	Functions map[string]*FunctionSymbol
	Globals   map[string]*VariableSymbol
}

func newSymbolTable() *SymbolTable {
	return &SymbolTable{
		Functions: make(map[string]*FunctionSymbol),
		Globals:   make(map[string]*VariableSymbol),
	}
}

func globalPath(variable *ast.VariableExpr) (string, int) {
	if variable == nil || variable.Name.Name == "" {
		return "", 0
	}

	path := variable.Name.Name
	depth := 0
	for _, access := range variable.Accesses {
		part, ok := staticGlobalAccess(access)
		if !ok {
			break
		}
		path += part
		depth++
	}
	return path, depth
}

func (symbols *SymbolTable) lookupGlobal(variable *ast.VariableExpr) *VariableSymbol {
	if symbols == nil || variable == nil {
		return nil
	}

	path := variable.Name.Name
	symbol := symbols.Globals[path]
	for _, access := range variable.Accesses {
		part, ok := staticGlobalAccess(access)
		if !ok {
			break
		}
		path += part
		if candidate := symbols.Globals[path]; candidate != nil {
			symbol = candidate
		}
	}
	return symbol
}

func staticGlobalAccess(access ast.VariableAccess) (string, bool) {
	switch access := access.(type) {
	case *ast.FieldAccess:
		return "." + access.Field.Name, true
	case *ast.IndexAccess:
		value, ok := staticIndexValue(access.Index)
		if !ok {
			return "", false
		}
		return "[" + value + "]", true
	default:
		return "", false
	}
}

func staticIndexValue(expression ast.Expression) (string, bool) {
	switch expression := expression.(type) {
	case *ast.StringExpr:
		var value strings.Builder
		for _, part := range expression.Parts {
			text, ok := part.(*ast.StringText)
			if !ok {
				return "", false
			}
			value.WriteString(text.Value)
		}
		return "string:" + strconv.Quote(value.String()), true
	case *ast.IntLiteral:
		return "int:" + strconv.FormatInt(expression.Value, 10), true
	case *ast.FloatLiteral:
		return "float:" + strconv.FormatFloat(expression.Value, 'g', -1, 64), true
	case *ast.UnaryExpr:
		if expression.Operator != token.Minus {
			return "", false
		}
		switch operand := expression.Operand.(type) {
		case *ast.IntLiteral:
			return "int:" + strconv.FormatInt(-operand.Value, 10), true
		case *ast.FloatLiteral:
			return "float:" + strconv.FormatFloat(-operand.Value, 'g', -1, 64), true
		default:
			return "", false
		}
	case *ast.BoolLiteral:
		return "bool:" + strconv.FormatBool(expression.Value), true
	case *ast.NilLiteral:
		return "nil", true
	default:
		return "", false
	}
}

type scope struct {
	parent         *scope
	owner          *scope
	names          map[string]Type
	locals         map[string]*VariableSymbol
	runtimeGlobals map[string]*VariableSymbol
}

func newExecutionScope() *scope {
	current := &scope{
		names:          make(map[string]Type),
		locals:         make(map[string]*VariableSymbol),
		runtimeGlobals: make(map[string]*VariableSymbol),
	}
	current.owner = current
	return current
}

func newInjectedScope(parent *scope) *scope {
	current := &scope{
		parent: parent,
		names:  make(map[string]Type),
	}
	if parent != nil {
		current.owner = parent.owner
	}
	return current
}

func (current *scope) defineName(name string, typ Type) {
	if current != nil {
		current.names[name] = typ
	}
}

func (current *scope) lookupName(name string) (Type, bool) {
	for candidate := current; candidate != nil; candidate = candidate.parent {
		if typ, ok := candidate.names[name]; ok {
			return typ, true
		}
	}
	return Type{}, false
}

func (current *scope) defineLocal(symbol *VariableSymbol) {
	if current == nil || symbol == nil {
		return
	}
	owner := current.owner
	if owner == nil {
		owner = current
	}
	owner.locals[symbol.Name] = symbol
}

func (current *scope) lookupLocal(name string) (*VariableSymbol, bool) {
	if current == nil {
		return nil, false
	}
	owner := current.owner
	if owner == nil {
		owner = current
	}
	symbol, ok := owner.locals[name]
	return symbol, ok
}

func (current *scope) defineRuntimeGlobal(path string, symbol *VariableSymbol) {
	if current == nil || path == "" || symbol == nil {
		return
	}
	owner := current.owner
	if owner == nil {
		owner = current
	}
	owner.runtimeGlobals[path] = symbol
}

func (current *scope) lookupRuntimeGlobal(variable *ast.VariableExpr) (*VariableSymbol, bool) {
	if current == nil || variable == nil {
		return nil, false
	}
	owner := current.owner
	if owner == nil {
		owner = current
	}

	path := variable.Name.Name
	symbol := owner.runtimeGlobals[path]
	for _, access := range variable.Accesses {
		part, ok := staticGlobalAccess(access)
		if !ok {
			break
		}
		path += part
		if candidate := owner.runtimeGlobals[path]; candidate != nil {
			symbol = candidate
		}
	}
	return symbol, symbol != nil
}
