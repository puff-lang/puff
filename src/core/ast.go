package core

type Name string
//---------------Start of CoreExpr's---------------------------------------
type CoreExpr interface {
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

type EChar int
func (e EChar) isExpr() {}

type EConstrName Name
func (e EConstrName) isExpr() {}

type EConstr struct {
	Tag   int
	Arity int
}
func (e EConstr) isExpr() {}

type EAp struct {
	Left CoreExpr
	Body CoreExpr
}
func (e EAp) isExpr() {}

type ELet struct {
	IsRec bool
	Defns []Defn
	Body  CoreExpr
}
func (e ELet) isExpr() {}

type ELam struct {
	Params []Name
	Body   CoreExpr
}
func (e ELam) isExpr() {}

type ECaseSimple struct{
	Body CoreExpr
	Alt []Alter 
}
func (e ECaseSimple) isExpr() {}


type ECaseConstr struct{
	Body CoreExpr
	Alt []Alter
}
func (e ECaseConstr) isExpr() {}

type EError string
func (e EError) isExpr() {}

type ESelect struct{
	I int
	R int
	Name string
}
func (e ESelect) isExpr() {}

type Defn struct {
	Var  Name
	Expr CoreExpr
}
func (e Defn) isExpr() {}

type ScDefn struct {
	Name Name
	Args []Name
	Expr CoreExpr
}
func (e ScDefn) isExpr() {}
//----------------------------End Of CoreExpr's--------------------------------------


type Alter struct {
	Num  int
	Vars []Name
	Expr CoreExpr
}
type Program []ScDefn



func BindersOf(defns []Defn) []Name {
	var names []Name 
	for _, defn := range defns {
		names = append(names, defn.Var)
	}

	return names
}

func RhssOf(defns []Defn) []CoreExpr {
	var rhss []CoreExpr 
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

//-----------------------------Definition of Pattern-----------------------------------
type Pattern interface {
	isPattern()
}

type PNum int
func (p PNum) isPattern() {}

type PVar string
func (p PVar) isPattern() {}

type PChar int
func (p PChar) isPattern() {}

type PConstrName struct{
	Name string
	Patt []Pattern
}
func (p PConstrName) isPattern() {}

type PConstr struct{
	I int
	R int
	Patt []Pattern
}
func (p PConstr) isPattern() {}

type PDefault struct{}
func (p PDefault) isPattern() {}
//---------------------------------------------------------------------------------------

//-----------------------------Abstract Data Types----------------------------------

type Tag int

var trueTag Tag = Tag(1)
var falseTag Tag = Tag(0)
var consTag Tag = Tag(3)
var nilTag Tag = Tag(2)
var initialTag Tag = Tag(4)
var undefinedTag Tag = Tag(-1)