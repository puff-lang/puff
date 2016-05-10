package core

import (
	"fmt"
	"strconv"
)

type GmState struct {
	gmo      GmOutput
	gmc      GmCode  //Current instruction stream
	gms      GmStack //Current stack
	gmd      GmDump  //current Dump
	gmvstack GmVStack
	gmh      GmHeap    //Heap of Nodes
	gmg      GmGlobals //Global Addresses in heap
	gmst     GmStats   //Statitics
}

type GmOutput []string

type GmVStack []int

// type GmCode and is simply a list of instructions.
type Instruction interface {
	isInstruction()
}

type Unwind struct{}

func (e Unwind) isInstruction() {}

type Pushglobal string

func (e Pushglobal) isInstruction() {}

type Pushint int

func (e Pushint) isInstruction() {}

type Pushchar int

func (e Pushchar) isInstruction() {}

type Push int

func (e Push) isInstruction() {}

type Mkap struct{}

func (e Mkap) isInstruction() {}

type Eval struct{}

func (e Eval) isInstruction() {}

type Update int

func (e Update) isInstruction() {}

type Pop int

func (e Pop) isInstruction() {}

type Alloc int

func (e Alloc) isInstruction() {}

type Slide int

func (e Slide) isInstruction() {}

type Add struct{}

func (e Add) isInstruction() {}

type Sub struct{}

func (e Sub) isInstruction() {}

type Mul struct{}

func (e Mul) isInstruction() {}

type Div struct{}

func (e Div) isInstruction() {}

type Mod struct{}

func (e Mod) isInstruction() {}

type Neg struct{}

func (e Neg) isInstruction() {}

type Eq struct{}

func (e Eq) isInstruction() {}

type Ne struct{}

func (e Ne) isInstruction() {}

type Lt struct{}

func (e Lt) isInstruction() {}

type Le struct{}

func (e Le) isInstruction() {}

type Gt struct{}

func (e Gt) isInstruction() {}

type Ge struct{}

func (e Ge) isInstruction() {}

type Cond struct {
	gm1 GmCode
	gm2 GmCode
}

func (e Cond) isInstruction() {}

type Error string

func (e Error) isInstruction() {}

type MkBool struct{}

func (e MkBool) isInstruction() {}

type MkInt struct{}

func (e MkInt) isInstruction() {}

type Select struct {
	I int
	R int
}

func (e Select) isInstruction() {}

type Pushbasic int

func (e Pushbasic) isInstruction() {}

type Get struct{}

func (e Get) isInstruction() {}

type Pack struct {
	Tag   int
	Arity int
}

func (e Pack) isInstruction() {}

type CasejumpObj struct {
	Int int
	gmC GmCode
}

type CasejumpSimple []CasejumpObj

func (e CasejumpSimple) isInstruction() {}

type CasejumpConstr []CasejumpObj

func (e CasejumpConstr) isInstruction() {}

type Pushconstr struct {
	Tag   int
	Arity int
}

func (e Pushconstr) isInstruction() {}

type GmCode []Instruction

func GetCode(gState GmState) GmCode {
	return gState.gmc
}

func putCode(gmc GmCode, gState GmState) GmState {
	gState.gmc = gmc
	return gState
}

//GmStack Implementation required for the GmState
const STACK_SIZE int = 10

type Addr int
type GmStack struct {
	Addrs []Addr
}

func getStack(gState GmState) GmStack {
	return gState.gms
}
func putStack(gms GmStack, gState GmState) GmState {
	gState.gms = gms
	return gState
}

func InitStack() GmStack {
	// return GmStack{[STACK_SIZE]Addr{}, -1}
	Addrs := []Addr{}

	return GmStack{Addrs}
}

func InitStackWithAddr(addr Addr) GmStack {
	// return GmStack{[STACK_SIZE]Addr{addr}, 0}
	Addrs := []Addr{addr}

	return GmStack{Addrs}
}

func (s *GmStack) PushStack(addr Addr) {
	// s.Index = s.Index + 1
	// s.Addrs[s.Index] = addr
	if len(s.Addrs) == 999 {
		panic(fmt.Errorf("Stack overflow"))
	}
	s.Addrs = append([]Addr{addr}, s.Addrs...)
}

func (s *GmStack) PopStack() Addr {
	// s.Index = s.Index -1
	// return s.Addrs[s.Index + 1]
	top := s.Addrs[0]
	s.Addrs = s.Addrs[1:]
	return top
}

func (s *GmStack) AddrsByIndexOf(n int) Addr {
	// if n > s.Index {
	// 	return -1
	// }
	// return s.Addrs[n]

	if n < 0 {
		return -1
	}

	return s.Addrs[n]
}

func (s *GmStack) StackLookup(addr Addr) bool {
	for _, as := range s.Addrs {
		if as == addr {
			return true
		}
	}
	return false
}

func (s *GmStack) TopOfStack() Addr { //Done
	// if s.Index < 0 {
	// 	return -1
	// }
	// return s.Addrs[s.Index]
	if len(s.Addrs) < 1 {
		return -1
	}

	return s.Addrs[0]
}

func (s *GmStack) BottomOfStack() Addr { //Done
	return s.Addrs[len(s.Addrs)-1]
}

func (s *GmStack) TailStack() GmStack {
	// if s.Index < 0 {
	// 	return GmStack{}
	// }
	// var argaddrs [STACK_SIZE]Addr
	// for i, addr := range s.Addrs {
	// 	if i != 0 {
	// 		argaddrs[i-1] = addr
	// 	}
	// }
	// return GmStack{argaddrs, s.Index-1}

	if len(s.Addrs) < 1 {
		return GmStack{}
	}

	return GmStack{s.Addrs[1:]}
}

func (s *GmStack) TakeNStack(n int) GmStack { //Done
	// if s.Index < n {
	// 	return *s
	// }
	// 	var argaddrs [STACK_SIZE]Addr
	// 	for i := s.Index; i > (s.Index - n - 1); i-- {
	// 		argaddrs[i] = s.Addrs[i]
	// 	}
	// return GmStack{argaddrs, n - 1}

	if len(s.Addrs) < 1 {
		return *s
	}

	return GmStack{s.Addrs[0:n]}
}

// From which side stack should be removed from top or bottom(Here, Assuming top)
func (s *GmStack) DropStack(n int) {
	// s.Index = s.Index -n
	s.Addrs = s.Addrs[n:]
}

//GmDump Implementation required for the GmState
type GmDumpItem struct {
	gmc      GmCode
	gms      GmStack
	gmvstack GmVStack
}

type GmDump []GmDumpItem

func getDump(gState GmState) GmDump {
	return gState.gmd
}

func putDump(gmd GmDump, gState GmState) GmState {
	gState.gmd = gmd
	return gState
}

//--------------------------------------------------------------------------
//GmHeap Implementation required for the GmState
//minimal G-machine have only three types of nodes
type Node interface {
	isNode()
}
type NNum int          //Numbers
func (e NNum) isNode() {}

type NChar int          //Character
func (e NChar) isNode() {}

type NAp struct {
	Left Addr
	Body Addr
}

func (e NAp) isNode() {}

type NGlobal struct { //Globals(contain no of arg that global expects & the code sequence to be exec when the global has enough argms)
	Nargs int
	GmC   GmCode
}

func (e NGlobal) isNode() {}

type NInd Addr

func (e NInd) isNode() {}

type NConstr struct {
	Tag   int
	Arity []Addr
}

func (e NConstr) isNode() {}

type NMarked struct {
	node Node
}

func (e NMarked) isNode() {}

const HEAP_SIZE int = 100

type GmHeap struct {
	hNode [HEAP_SIZE]Node
	nargs int
	instn []Instruction
	index Addr
}

func HInitial() GmHeap {
	var h GmHeap
	h.index = -1
	return h
}

func (h *GmHeap) HNull() Addr {
	h.index = 0
	return h.index
}

func (h *GmHeap) HAlloc(node Node) Addr {
	h.index = h.index + 1

	if h.index > 99 {
		panic("No space left on heap")
	}

	h.hNode[h.index] = node
	h.instn = []Instruction{}
	return h.index
}

func (h *GmHeap) HLookup(addr Addr) Node {
	if addr < 0 {
		return nil
	}
	if h.index >= addr {
		return h.hNode[addr]
	}
	return nil
}

func (h *GmHeap) HUpdate(addr Addr, node Node) {
	if addr == -1 {
		fmt.Println("Not found in Heap")
		return
	}
	if h.index >= addr {
		h.hNode[addr] = node
	}
}

func GetHeap(gState GmState) GmHeap {
	return gState.gmh
}

func putHeap(gmh GmHeap, gState GmState) GmState {
	gState.gmh = gmh
	return gState
}

//{[        {0 [4 test {} 1 {}]} {1 [0 2 {}]}   ] 0 [] 1}

//Implementation of GmGlobals for the GmState func PopStack()
type Object struct {
	Name Name
	Addr Addr
}
type GmGlobals []Object

func GetGlobals(gState GmState) GmGlobals {
	return gState.gmg
}

func GlobalsLookup(gmg GmGlobals, name Name) Addr {
	for _, obj := range gmg {
		if obj.Name == name {
			return obj.Addr
		}
	}
	fmt.Println("Error: Undeclared global identifier ", name)
	return Addr(-1)
}

//Implementation of GmStats for GmState
type GmStats int

func getStats(gState GmState) GmStats {
	return gState.gmst
}

func putStats(gmst GmStats, gState GmState) GmState {
	gState.gmst = gmst
	return gState
}

func initialDump() GmDump {
	return GmDump{GmDumpItem{GmCode{}, InitStack(), GmVStack{}}}
}

//Part of GmState implementation is over.
//--------------------------------------------------------------------------

var binaryOperators []string = []string{"+", "-", "*", "/", "==", "<", ">", "%"} //,  "!=", "<=", ">="}
var unaryOperators []string = []string{"negate"}

func createBinaryOp(name string) ScDefn {
	return ScDefn{Name(name), []Name{"x", "y"}, CoreExpr(EAp{EAp{(EVar(name)), (EVar("x"))}, (EVar("y"))})}
}
func createUnaryOp(name string) ScDefn {
	return ScDefn{Name(name), []Name{"x"}, CoreExpr(EAp{EVar(name), EVar("x")})}
}

func primitiveScs() []ScDefn {
	scdefn := []ScDefn{}

	for _, name := range binaryOperators {
		scdefn = append(scdefn, createBinaryOp(name))
	}

	for _, sc := range unaryOperators {
		scdefn = append(scdefn, createUnaryOp(sc))
	}

	return scdefn
}
func selFunName(i int, r int) Name {
	str := "select-" + strconv.Itoa(i) + "-" + strconv.Itoa(r)
	return Name(str)
}

type builDyadic struct {
	Name string
	Inst Instruction
}

var builtinDyadicInt []builDyadic = []builDyadic{
	builDyadic{"+", Add{}},
	builDyadic{"-", Sub{}},
	builDyadic{"*", Mul{}},
	builDyadic{"/", Div{}},
	builDyadic{"%", Mod{}},
}

var builtinDyadicBool []builDyadic = []builDyadic{
	builDyadic{"==", Eq{}},
	builDyadic{"!=", Ne{}},
	builDyadic{"<", Lt{}},
	builDyadic{"<=", Le{}},
	builDyadic{">", Gt{}},
	builDyadic{">=", Ge{}},
}

type builtinDyadic struct {
	bd []builDyadic
}

var built builtinDyadic = builtinDyadic{addbuilDyadic()}

func addbuilDyadic() []builDyadic { //Done
	tmpbuiltinDyadic := []builDyadic{}
	tmpbuiltinDyadic = append(tmpbuiltinDyadic, builtinDyadicBool...)
	tmpbuiltinDyadic = append(tmpbuiltinDyadic, builtinDyadicInt...)
	return tmpbuiltinDyadic
}

func aHasKey(bUilt builtinDyadic, name string) bool {
	for _, element := range built.bd {
		if element.Name == name {
			return true
		}
	}
	return false
}

func aLookup(bUilt builtinDyadic, name string) Instruction {
	for _, element := range built.bd {
		if element.Name == name {
			return element.Inst
		}
	}
	return Error("Impossible Dude")
}

func preCompiledScs() []GmCompiledSC { //Done
	acc := []GmCompiledSC{}
	r := []int{1, 2, 3, 4, 5}
	for _, i := range r {
		acc = genSelFuncs(acc, i)
	}
	return acc
}

func genSelFuncs(acc []GmCompiledSC, r int) []GmCompiledSC { //Done
	tmpacc := []GmCompiledSC{}
	for i := 0; i < r; i++ {
		tmpacc = genSelFunc(r, tmpacc, i)
	}
	return append(acc, tmpacc...)
}

func genSelFunc(r int, acc []GmCompiledSC, i int) []GmCompiledSC {
	sc := append(acc, GmCompiledSC{selFunName(r, i), 1, GmCode{Push(0), Eval{}, Select{r, i}, Update(1), Pop(1), Unwind{}}})
	return sc
}
