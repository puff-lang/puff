package core

import (
	"fmt"
)

type GmState struct{
	gmc GmCode 			//Current instruction stream
	gms GmStack			//Current stack
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

type Slide int
func (e Slide) isInstruction() {}

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


type GmHeap struct {
	hNode Node
	nargs  int
	instn []Instruction
	index Addr
}

func HInitial() GmHeap {
	var h GmHeap
	h.index = -1
	return h
}

func (h *GmHeap) HAlloc(node Node) Addr {
	h.index = h.index + 1
	h.hNode = node;
	h.instn = []Instruction{};
	return h.index
}

func getHeap(gState GmState) GmHeap {
	return gState.gmh
}

func putHeap(gmh GmHeap, gState GmState) GmState {
	gState.gmh = gmh
	return gState
}



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
//Part of GmState implementation is over.
//--------------------------------------------------------------------------

func Compile(p Program) GmState {
	var stats GmStats = 0
	heap, globals := buildInitialHeap(p)
	return GmState{initialCode(), []Addr{}, heap, globals, stats}
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
	return GmCode{Pushglobal("main"), Unwind{}}
}


type GmCompiledSC struct{
	Name   Name
	Length int
	body   GmCode
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

// type GmCompiler = func(CoreExpr, GmEnvironment) GmCode

//Creates code which instnst the expr e in env ρ, for a SC of arity d, and then proceeds to unwind the resulting stack
func compilerR(cexp CoreExpr, env GmEnvironment) GmCode {
	inst := []Instruction{}
	cC := compileC(cexp,env)
	for _,obj := range cC {
		inst = append(inst, obj)
	}
	inst = append(inst, Slide(len(env) + 1))
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
    }
	return GmCode{}
}

func argOffset(n int, env GmEnvironment) GmEnvironment {	
	for _,obj := range env {
		obj.Int = obj.Int + n
	}
	return env
}