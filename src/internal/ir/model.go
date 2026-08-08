package ir

import "github.com/puff-lang/puff/internal/diagnostic"

type Project struct {
	Globals   []Global
	Functions []Function
	Tags      []Tag
}

type SourceRef struct {
	File string
	Span diagnostic.Span
}

type SymbolID struct {
	Module string
	Name   string
}

type TypeKind string

const (
	TypeUnknown TypeKind = "unknown"
	TypeNil     TypeKind = "nil"
	TypeBool    TypeKind = "bool"
	TypeInt     TypeKind = "int"
	TypeFloat   TypeKind = "float"
	TypeString  TypeKind = "string"
	TypeList    TypeKind = "list"
	TypeMap     TypeKind = "map"
	TypeRange   TypeKind = "range"
	TypeNamed   TypeKind = "named"
)

type Type struct {
	Kind      TypeKind
	Name      string
	Arguments []Type
}

type Global struct {
	ID          SymbolID
	Public      bool
	Type        Type
	Initializer Value
	Source      SourceRef
}

type FunctionKind string

const (
	FunctionUser  FunctionKind = "user"
	FunctionEvent FunctionKind = "event"
)

type Parameter struct {
	Name   string
	Type   Type
	Source SourceRef
}

type Function struct {
	ID         SymbolID
	Kind       FunctionKind
	Public     bool
	Parameters []Parameter
	Result     Type
	Commands   []Command
	Source     SourceRef
}

type Tag struct {
	Name      string
	Functions []SymbolID
}
