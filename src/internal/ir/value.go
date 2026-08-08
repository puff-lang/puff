package ir

type Value interface {
	valueNode()
}

type Nil struct {
	Source SourceRef
}

func (*Nil) valueNode() {}

type Bool struct {
	Value  bool
	Source SourceRef
}

func (*Bool) valueNode() {}

type Int struct {
	Value  int64
	Source SourceRef
}

func (*Int) valueNode() {}

type Float struct {
	Value  float64
	Source SourceRef
}

func (*Float) valueNode() {}

type Text struct {
	Parts  []TextPart
	Source SourceRef
}

func (*Text) valueNode() {}

type TextPart interface {
	textPartNode()
}

type TextLiteral struct {
	Value  string
	Source SourceRef
}

func (*TextLiteral) textPartNode() {}

type TextInterpolation struct {
	Value  Value
	Source SourceRef
}

func (*TextInterpolation) textPartNode() {}

type Call struct {
	Function  SymbolID
	Arguments []Value
	Source    SourceRef
}

func (*Call) valueNode() {}

type Reference struct {
	Symbol SymbolID
	Name   string
	Type   Type
	Source SourceRef
}

func (*Reference) valueNode() {}
