package sema

import (
	"strings"
)

type TypeKind string

const (
	TypeUnknown  TypeKind = "unknown"
	TypeNil      TypeKind = "nil"
	TypeBool     TypeKind = "bool"
	TypeInt      TypeKind = "int"
	TypeFloat    TypeKind = "float"
	TypeString   TypeKind = "string"
	TypeList     TypeKind = "list"
	TypeMap      TypeKind = "map"
	TypeRange    TypeKind = "range"
	TypeFunction TypeKind = "function"
	TypeNamed    TypeKind = "named"
)

type Type struct {
	Kind      TypeKind
	Name      string
	Arguments []Type
}

func (typ Type) String() string {
	name := typ.Name
	if name == "" {
		name = string(typ.Kind)
	}
	if len(typ.Arguments) == 0 {
		return name
	}

	arguments := make([]string, 0, len(typ.Arguments))
	for _, argument := range typ.Arguments {
		arguments = append(arguments, argument.String())
	}
	return name + "<" + strings.Join(arguments, ", ") + ">"
}

func (typ Type) IsUnknown() bool {
	return typ.Kind == TypeUnknown
}

var builtInTypes = map[string]TypeKind{
	"nil":          TypeNil,
	"bool":         TypeBool,
	"int":          TypeInt,
	"float":        TypeFloat,
	"string":       TypeString,
	"list":         TypeList,
	"map":          TypeMap,
	"range":        TypeRange,
	"function":     TypeFunction,
	"Player":       TypeNamed,
	"Entity":       TypeNamed,
	"Mob":          TypeNamed,
	"Item":         TypeNamed,
	"Block":        TypeNamed,
	"Location":     TypeNamed,
	"Vector":       TypeNamed,
	"NBT":          TypeNamed,
	"Identifier":   TypeNamed,
	"Score":        TypeNamed,
	"Objective":    TypeNamed,
	"Tag":          TypeNamed,
	"Command":      TypeNamed,
	"Predicate":    TypeNamed,
	"Error":        TypeNamed,
	"TypeError":    TypeNamed,
	"NameError":    TypeNamed,
	"SyntaxError":  TypeNamed,
	"RuntimeError": TypeNamed,
	"IndexError":   TypeNamed,
	"KeyError":     TypeNamed,
	"ValueError":   TypeNamed,
}

func compatible(expected Type, actual Type) bool {
	if expected.IsUnknown() || actual.IsUnknown() {
		return true
	}
	if expected.Kind == TypeFloat && actual.Kind == TypeInt {
		return true
	}
	if expected.Kind != actual.Kind {
		return false
	}
	if expected.Kind == TypeNamed && expected.Name != actual.Name {
		return false
	}
	if len(expected.Arguments) == 0 || len(actual.Arguments) == 0 {
		return true
	}
	if len(expected.Arguments) != len(actual.Arguments) {
		return false
	}
	for index := range expected.Arguments {
		if !compatible(expected.Arguments[index], actual.Arguments[index]) {
			return false
		}
	}
	return true
}

func numericType(left Type, right Type) Type {
	if left.IsUnknown() || right.IsUnknown() {
		return Type{Kind: TypeUnknown}
	}
	if left.Kind != TypeInt && left.Kind != TypeFloat {
		return Type{Kind: TypeUnknown}
	}
	if right.Kind != TypeInt && right.Kind != TypeFloat {
		return Type{Kind: TypeUnknown}
	}
	if left.Kind == TypeFloat || right.Kind == TypeFloat {
		return Type{Kind: TypeFloat}
	}
	return Type{Kind: TypeInt}
}
