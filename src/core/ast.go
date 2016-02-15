package core

type Name string

type Expr interface {
	isExpr()
}

type EVar Name
func (e EVar) isExpr() {}

type ENum struct {
	IsInt   bool    // Number has an integral value.
	IsUint  bool    // Number has an unsigned integral value.
	IsFloat bool    // Number has a floating-point value.
	Int64   int64   // The signed integer value.
	Uint64  uint64  // The unsigned integer value.
	Float64 float64 // The floating-point value.
	Text    string  // The original textual representation from the input.
}
func (e ENum) isExpr() {}

type EConstr struct {
	Tag   int
	Arity int
}
func (e EConstr) isExpr() {}

type EAp struct {
	Left Expr
	Body Expr
}
func (e EAp) isExpr() {}

type ELet struct {
	IsRec bool
	Defns []Defn
	Body  Expr
}
func (e ELet) isExpr() {}

type ELam struct {
	Params []Name
	Body   Expr
}
func (e ELam) isExpr() {}

type Alter struct {
	Num  int
	Vars []Name
	Expr Expr
}

type Defn struct {
	Var  Name
	Expr Expr
}

type ScDefn struct {
	Name Name
	Args []Name
	Expr Expr
}

type Program []ScDefn

func BindersOf(defns []Defn) []Name {
	var names []Name 
	for _, defn := range defns {
		names = append(names, defn.Var)
	}

	return names
}

func RhssOf(defns []Defn) []Expr {
	var rhss []Expr 
	for _, defn := range defns {
		rhss = append(rhss, defn.Expr)
	}

	return rhss
}

func IsAtomicExpr(expr interface{}) bool {
	switch expr.(type) {
	case EVar:
		return true
	case ENum:
		return true
	default:
		return false
	}
}

