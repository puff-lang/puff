package ast

type File struct {
	NodeBase
	Metadata     []MetadataEntry
	Requirements []*RequireDecl
	Declarations []Declaration
}

type MetadataEntry struct {
	NodeBase
	Key   string
	Value string
}

type RequireDecl struct {
	NodeBase
	Path  *StringExpr
	Alias *Identifier
}

func (*RequireDecl) declarationNode() {}

type FunctionDecl struct {
	NodeBase
	Public     bool
	Name       Identifier
	Parameters []Parameter
	ReturnType *TypeRef
	Body       Block
}

func (*FunctionDecl) declarationNode() {}

type Parameter struct {
	NodeBase
	Name Identifier
	Type *TypeRef
}

type TypeRef struct {
	NodeBase
	Name      Identifier
	Arguments []*TypeRef
}

type EventDecl struct {
	NodeBase
	Name []Identifier
	Body Block
}

func (*EventDecl) declarationNode() {}

type GlobalAssignment struct {
	NodeBase
	Public bool
	Target *VariableExpr
	Value  Expression
}

func (*GlobalAssignment) declarationNode() {}
