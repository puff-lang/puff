package sema

import "github.com/puff-lang/puff/internal/ast"

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
	initialized bool
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

type scope struct {
	parent *scope
	owner  *scope
	names  map[string]Type
	locals map[string]*VariableSymbol
}

func newExecutionScope() *scope {
	current := &scope{
		names:  make(map[string]Type),
		locals: make(map[string]*VariableSymbol),
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
