package core

import (
	"fmt"
	"reflect"
)

type GmState struct{
	gmc GmCode 			//Current instruction stream
	gms GmStack			//Current stack
	gmd GmDump			//current Dump
	gmh GmHeap			//Heap of Nodes
	gmg GmGlobals		//Global Addresses in heap
	gmst GmStats		//Statitics
}


// type GmCode and is simply a list of instructions.
type Instruction interface{
	isInstruction()
}
type Unwind struct{}
func (e Unwind) isInstruction() {}

type Pushglobal string
func (e Pushglobal) isInstruction() {}

type Pushint int
func (e Pushint) isInstruction() {}

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

type Cond struct{
	gm1 GmCode
	gm2 GmCode
}
func (e Cond) isInstruction() {}

type GmCode []Instruction

func getCode(gState GmState) GmCode{
	return gState.gmc
}

func putCode(gmc GmCode, gState GmState) GmState {
	gState.gmc = gmc
	return gState
} 


//GmStack Implementation required for the GmState
type Addr int
type GmStack []Addr

func getStack(gState GmState) GmStack {
	return gState.gms
}
func putStack(gms GmStack, gState GmState) GmState {
	gState.gms = gms
	return gState
}
//--------------------------------------------------------------------------
//GmDump Implementation required for the GmState
type GmDumpItem struct {
	gmc GmCode
	gms GmStack
}

type GmDump []GmDumpItem

func getDump(gState GmState) GmDump {
	return gState.gmd
}

func putDump(gmd GmDump, gState GmState) GmState{
	gState.gmd = gmd
	return gState
}



//--------------------------------------------------------------------------
//GmHeap Implementation required for the GmState
//minimal G-machine have only three types of nodes
type Node interface {
	isNode()
}
type NNum int //Numbers
func (e NNum) isNode() {}

type NAp struct {
	Left CoreExpr
	Body CoreExpr
}
func (e NAp) isNode() {}

type NGlobal struct { //Globals(contain no of arg that global expects & the code sequence to be exec when the global has enough argms)
	nargs  int
	gmCode GmCode
}
func (e NGlobal) isNode() {}

type NInd Addr
func (e NInd) isNode() {}




type GmHeap struct {
	hNode [10]Node
	// nargs  int
	instn []Instruction
	index Addr
}

func HInitial() GmHeap {
	var h GmHeap
	h.index = -1
	return h
}

func (h *GmHeap) HNull() Addr{
	h.index = 0
	return h.index
}

func (h *GmHeap) HAlloc(node Node) Addr {
	h.index = h.index + 1
	h.hNode[h.index] = node;
	h.instn = []Instruction{};
	return h.index
}

// func (h *GmHeap) HLookup(addr Addr) Node{
// 	for i := 0; i <= h.index; i++ {
// 		if h.addr == addr {
// 			return h.hNode[i]
// 		}
// 	}
// 	return NNum{}
// }

func getHeap(gState GmState) GmHeap {
	return gState.gmh
}

func putHeap(gmh GmHeap, gState GmState) GmState {
	gState.gmh = gmh
	return gState
}

//{[        {0 [4 test {} 1 {}]} {1 [0 2 {}]}   ] 0 [] 1}

//Implementation of GmGlobals for the GmState 
type Object struct{
	Name Name
	addr Addr
}
type  GmGlobals []Object

func getGlobals(gState GmState) GmGlobals{
	return gState.gmg
}

//Implementation of GmStats for GmState
type GmStats int
func getStats(gState GmState) GmStats{
	return gState.gmst
}

func putStats(gmst GmStats, gState GmState) GmState{
	gState.gmst = gmst
	return gState
}

func initialDump() GmDump{
	return GmDump{GmDumpItem{GmCode{},GmStack{}}}
}
//Part of GmState implementation is over.
//--------------------------------------------------------------------------
type compiledPrimitives []GmCompiledSC

var compPrim = compiledPrimitives{
	GmCompiledSC{"+", 2, GmCode{Push(1), Eval{}, Push(1), Eval{}, Add{}, Update(2), Pop(2), Unwind{}}},
	GmCompiledSC{"-", 2, GmCode{Push(1), Eval{}, Push(1), Eval{}, Sub{}, Update(2), Pop(2), Unwind{}}},
	GmCompiledSC{"*", 2, GmCode{Push(1), Eval{}, Push(1), Eval{}, Mul{}, Update(2), Pop(2), Unwind{}}},
	GmCompiledSC{"/", 2, GmCode{Push(1), Eval{}, Push(1), Eval{}, Div{}, Update(2), Pop(2), Unwind{}}},
	GmCompiledSC{"negate", 2, GmCode{Push(0), Eval{}, Neg{}, Update(1), Pop(1), Unwind{}}},
	GmCompiledSC{"==", 2, GmCode{Push(1), Eval{}, Push(1), Eval{}, Eq{}, Update(2), Pop(2), Unwind{}}},
	GmCompiledSC{"~=", 2, GmCode{Push(1), Eval{}, Push(1), Eval{}, Ne{}, Update(2), Pop(2), Unwind{}}},
	GmCompiledSC{"<", 2, GmCode{Push(1), Eval{}, Push(1), Eval{}, Lt{}, Update(2), Pop(2), Unwind{}}},
	GmCompiledSC{"<=", 2, GmCode{Push(1), Eval{}, Push(1), Eval{}, Le{}, Update(2), Pop(2), Unwind{}}},
	GmCompiledSC{">", 2, GmCode{Push(1), Eval{}, Push(1), Eval{}, Gt{}, Update(2), Pop(2), Unwind{}}},
	GmCompiledSC{">=", 2, GmCode{Push(1), Eval{}, Push(1), Eval{}, Ge{}, Update(2), Pop(2), Unwind{}}},
	GmCompiledSC{"if", 3, GmCode{Push(0), Eval{}, Cond{GmCode{Push(1)}, GmCode{Push(2)}}, Update(3), Pop(3), Unwind{}}},
}


func Compile(p Program) GmState {
	var stats GmStats = 0
	heap, globals := buildInitialHeap(p)
	return GmState{initialCode(), []Addr{}, initialDump(), heap, globals, stats}
}


func buildInitialHeap(p Program) (GmHeap, GmGlobals) {
	var compiled []GmCompiledSC
	gmHeap := HInitial()

	for _, sc := range p {
		compiled = append(compiled, compileSc(sc))
	}
	// mapAccuml allocateSc hInitial compiled
	// for _, compiledSc := range compiled {
	// 	allocateSc(gmHeap, compiledSc)	
	// }
	return mapAccuml(allocateSc, gmHeap, compiled)
}

//-------------------------------------------------------------------
//We must define the type for passing function as parameter
type allocates func(GmHeap, GmCompiledSC) (GmHeap, Object) 

// allocateSc implements allocates, returning GmHeap & Object
func allocateSc(gmh GmHeap, gCSC GmCompiledSC) (GmHeap, Object) {
	addr := gmh.HAlloc(NGlobal{gCSC.Length, gCSC.body})
	return gmh, Object{gCSC.Name, addr}
}

func mapAccuml(f allocates, acc GmHeap, list []GmCompiledSC) (GmHeap, GmGlobals) {
	acc1 := acc
	xsdash := GmGlobals{}
	var xdash Object

	for _, sc := range list {
		acc1, xdash = f(acc1, sc)
		xsdash = append(GmGlobals{xdash}, xsdash...)
	}
	return acc1, xsdash
}

func initialCode() GmCode {
	return GmCode{Pushglobal("main"), Eval{}}
}


type GmCompiledSC struct{
	Name   Name
	Length int
	body   GmCode
}

func (sc GmCompiledSC) Body() GmCode {
	return sc.body
}


//Each SuperCombinator is compiled using compileSc which implements SC scheme
func compileSc(sc ScDefn) GmCompiledSC {
	var gmE = GmEnvironment{}

	for i,eString := range sc.Args {
		gmE = append(gmE, Environment{eString, i})
	}
	fmt.Println("hello")

	return GmCompiledSC{sc.Name, len(sc.Args), compilerR(sc.Expr, gmE)}
}

//Each SuperCombinator is compiled using compileSc which implements SC scheme
func CompileSc(sc ScDefn) GmCompiledSC {
	var gmE = GmEnvironment{}

	for i,eString := range sc.Args {
		gmE = append(gmE, Environment{eString, i})
	}
	fmt.Println("hello")

	return GmCompiledSC{sc.Name, len(sc.Args), compilerR(sc.Expr, gmE)}
}

type Environment struct{
	Name Name
	Int int
}

type GmEnvironment []Environment

func elem(name Name, assoc GmEnvironment) int {
	for _,obj := range assoc {
		if obj.Name == name  {
			return obj.Int
		}
	}
	return -1 //Default Value: null string
}

type GmCompiler func(CoreExpr, GmEnvironment) (GmCode)

//Creates code which instnst the expr e in env ρ, for a SC of arity d, and then proceeds to unwind the resulting stack
func compilerR(cexp CoreExpr, env GmEnvironment) GmCode {
	inst := []Instruction{}
	cC := compileC(cexp,env)
	for _,obj := range cC {
		inst = append(inst, obj)
	}
	length := len(env)
	inst = append(inst, Update(length))
	inst = append(inst, Pop(length))
	//inst = append(inst, Slide(len(env) + 1))
	inst = append(inst, Unwind{})
	return inst	
}

//generates code which creates the graph of e in env ρ,leaving a pointer to it on top of the stack
func compileC(cexp CoreExpr, env GmEnvironment) GmCode {
	switch cexp.(type) {
        case EVar:
        	expr := cexp.(EVar)
        	n := elem(Name(expr), env)
        	if n != -1 {
        		return GmCode{Push(n)}
        	} else {
        		return GmCode{Pushglobal(Name(expr))}
        	}

		case ENum:
			expr := cexp.(ENum)
			if expr.IsInt {
				return GmCode{Pushint(expr.Int64)}
			} else if expr.IsUint {
				return GmCode{Pushint(expr.Uint64)}
			}
			return GmCode{Pushint(42)} // TODO
		case EAp:
			expr := cexp.(EAp)
			var gmC = GmCode{}
			gmC = append(gmC, compileC(expr.Body, env)...)
			gmC = append(gmC, compileC(expr.Left, argOffset(1, env))...)
			gmC = append(gmC, Mkap{})
			return gmC
		case ELet:
			expr := cexp.(ELet)
			if expr.IsRec {
				return compileLetrec(compileC, expr.Defns, expr.Body, env)
			} else {
				return compileLet(compileC, expr.Defns, expr.Body, env)
			}
    }
	return GmCode{}
}


func compileLet(comp GmCompiler, defs []Defn, expr CoreExpr, env GmEnvironment) GmCode {
	envdash := compileArgs(defs, env) // Creating New Environment
	gmC := GmCode{}
	gmC = append(gmC, compileLetDash(defs, envdash)...)
	gmC = append(gmC,  comp(expr, envdash)...)
	return append(gmC,Slide(len(defs)))
}

func compileLetrec(comp GmCompiler, defs []Defn, expr CoreExpr, env GmEnvironment) GmCode {
	envdash := compileArgs(defs, env) // Creating New Environment
	gmC := GmCode{Alloc(len(defs))}
	gmC = append(gmC, compileLetDash(defs, envdash)...)
	gmC = append(gmC, Update(0))
	gmC = append(gmC,  comp(expr, envdash)...)
	return append(gmC,Slide(len(defs)))
}

func compileLetDash(defns []Defn, env GmEnvironment) GmCode {
	envdash := env
	gmC := GmCode{}
	for _, defn := range defns {
		gmC = append(gmC, compileC(defn.Expr, envdash)...)
		envdash = argOffset(1, envdash)
	}
	return gmC
}

func compileArgs(defns []Defn, env GmEnvironment) (GmEnvironment) {
	n := len(defns)
	var gmE GmEnvironment
	for _, defn := range defns {
		tmpEnv := Environment{defn.Var, n-1}
		gmE = append(gmE, tmpEnv)
		n = n - 1
	}
	return append(gmE, argOffset(len(defns), env)...)
}

func argOffset(n int, env GmEnvironment) GmEnvironment {	
	var gmE GmEnvironment
	for _,obj := range env {
		tmpEnv := Environment{obj.Name, obj.Int + n}
		gmE = append(gmE, tmpEnv)
	}
	return gmE
}

func PrintBody(body GmCode) {
	for _, inst := range body {
		fmt.Print(reflect.TypeOf(inst), inst, "  ")
	}
	fmt.Println()
}