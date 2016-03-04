package core


type GmState struct{
	gmc gmCode 			//Current instruction stream
	gms GmStack			//Current stack
	gmh GmHeap			//Heap of Nodes
	gmg GmGlobals		//Global Addresses in heap
	gmst GmStats		//Statitics
}


// type GmCode and is simply a list of instructions.
type Instruction interface{
	isInstruction()
}
type Unwind bool
func (e Unwind) isInstruction() {}

type Pushglobal string
func (e Pushglobal) isInstruction() {}

type Pushint int
func (e Pushint) isInstruction() {}

type Push int
func (e Push) isInstruction() {}

type Mkap bool
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

type NAp *Expr *Expr // Applications
func (e NAp) isNode() {}

type NGlobal int //Globals(contain no of arg that global expects & the code sequence to be exec when the global has enough argms)
func (e NGlobal) isNode() {}


type GmHeap struct{
	hNode Node
	nargs  int
	insts []Instruction
	index int
}

func HInitial() GmHeap{
	var h GmHeap
	h.index = -1
	return h
}

func (h *Heap) HAlloc(hNode Node, nargs int, instn []Instruction) {
	h.index = h.index + 1
	h.hNode = hNode;
	h.instn = instn;
}

func getHeap(gState GmState) GmHeap{
	return gState.gmh
}
func putHeap(gmh GmCode, gState GmState) GmState {
	gState.gmh = gmh
	return gState
}



//Implementation of GmGlobals for the GmState 
type GmGlobals struct{
	Name string
	Addr int
}

func getGlobals(gState GmState) GmGlobals{
	return gState.gmg
}

//Implementation of GmStats for GmState
type GmStats int
func getStats(gState GmState) GmStats{
	return gState.gmst
}
func putStats(gmst GmStats, gState GmState) GmStats{
	gState.gmst = gmst
	return gState
}
//Part of GmState implementation is over.
//--------------------------------------------------------------------------

func Compile(p Program) GmState {
	heap, globals := buildInitialHeap(p)
	return GmState{initialCode, [], heap, globals, statInitial}
}


func buildInitialHeap(p Program) (GmHeap, GmGlobals) {
	var compiled []GmCompiledSC
	gmHeap := HInitial()

	for _, sc := range p {
		compiled = append(compiled, sc)
	}
	// mapAccuml allocateSc hInitial compiled
	for _, compiledSc := range compiled {
		allocateSc(gmHeap, compiledSc)	
	}
}

//---------------------------------------------------------------------

//This structure resembles with the structure of GmGlobal. We can use GmGlobal instead of this.
type Name string
type Addr int
type Object struct{
	name Name
	addr Addr
}
//-------------------------------------------------------------------
//We must define the type for passing function as parameter
type allocates func(GmHeap, GmCompiledSC) (GmHeap, Object) 

// allocateSc implements allocates, returning GmHeap & Object
func allocateSc(gmh GmHeap, gCSC GmCompiledSC) (GmHeap, Object){
	gmh.HAlloc()
}

func mapAccuml(f allocates, acc GmHeap, list []GmCompiledSC) {
	acc1, xdash := f(acc, list.head())
	acc2, xsdash := mapAccuml(f, acc1, list.tail());
	return acc2, xsdash.concat(xdash);
}

func initialCode() GmCode {
	gmCode := [2]Instruction{Pushglobal "main", Unwind}
	return gmCode
}


type GmCompiledSC struct{
	Name string
	Length int
	body []Instruction
}

//Each SuperCombinator is compiled using compileSc which implements SC scheme
func compileSc(name string, env []string, cexp CoreExpr) GmCompiledSC {
	var gmCSC = GmCompiledSC
	gmCSC.name = name
	gmCSC.Length = length(env)
	for _, gmCSC.body := range env {
		
	}
}

type GmEnvironment struct{
	Name string
	Int int
}

//Creates code which instnst the expr e in env ρ, for a SC of arity d, and then proceeds to unwind the resulting stack
func compilerR(cexp CoreExpr, env GmEnvironment) GmCode {


	
}

//generates code which creates the graph of e in env ρ,leaving a pointer to it on top of the stack
func compileC(cexp CoreExpr, env GmEnvironment) GmCode{


}