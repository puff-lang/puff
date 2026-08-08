package ir

type Command interface {
	commandNode()
}

type Return struct {
	Value  Value
	Source SourceRef
}

func (*Return) commandNode() {}

type Effect struct {
	PatternID string
	Arguments []Argument
	Source    SourceRef
}

func (*Effect) commandNode() {}

type Argument struct {
	Name  string
	Value Value
}
