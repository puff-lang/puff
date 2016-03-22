package core

import (
	"fmt"
	"strconv"
	// "reflect"
)


func EvalState(gmState GmState) []GmState { //Done
	var restState []GmState
	if gmFinal(gmState) == true {
		restState = []GmState{}
	}  else {
		fmt.Println("Dispatching: ", gmState.gmc)
		restState = EvalState(doAdmin(step(gmState)))
	}
	result := []GmState{gmState}
	return append(result, restState...)
}


func doAdmin(gmState GmState) GmState { //Done
	return GmState{gmState.gmo, gmState.gmc, gmState.gms, gmState.gmd, gmState.gmvstack, gmState.gmh, gmState.gmg, statIncSteps(gmState.gmst)}
}

func statIncSteps(gmst GmStats) GmStats { //Done
	return gmst + 1
}
func gmFinal(gmState GmState) bool { //Done
	fmt.Println("GMCODE Len: ", len(gmState.gmc))
	if len(gmState.gmc) == 0 {
		return true
	}
	return false
}
	
func step(gmState GmState) GmState { // Done
	i := gmState.gmc[0]
	fmt.Println("Instruction: ", i)
	gmState.gmc = gmState.gmc[1:]
	fmt.Println(gmState.gms)
	return dispatch(i, gmState)
}

func dispatch(instr Instruction, gmState GmState) GmState { //Done
	switch instr.(type) {
		case Update:
			fmt.Println("Update")
			return update(int(instr.(Update)), gmState)
		case Push:
			fmt.Println("Push")
			return push(int(instr.(Push)), gmState)
		case Pop:
			fmt.Println("Pop")
			return pop(int(instr.(Pop)), gmState)
		case Pushglobal:
			fmt.Println("Pushglobal")
			return pushglobal(string(instr.(Pushglobal)), gmState)
		case Mkap:
			fmt.Println("Mkap")
			return mkap(gmState)
		case Unwind:
			fmt.Println("Unwind")
			return unwind(gmState)
		case Pushint:
			fmt.Println("Pushint")
			return pushint(int(instr.(Pushint)), gmState)
		case Alloc:
			fmt.Println("Alloc")
			return alloc(int(instr.(Alloc)), gmState)
		case Slide:
			return slide(int(instr.(Slide)), gmState)
		case Eval:
			return eval2(gmState)
		case Add:
			fmt.Println("Add")
			return add(gmState)
		case Sub:
			return sub(gmState)
		case Div:
			return div(gmState)
		case Mul:
			return mul(gmState)
		default:
			fmt.Println("Default")
			return mod(gmState)

	}
}

func unwind(gmState GmState) GmState { //Done
	heap := gmState.gmh
	fmt.Println(gmState.gms)
	addr := gmState.gms.TopOfStack()
	fmt.Println("Address:", addr)
	return newState(heap.HLookup(addr), gmState)
}

func newState(node Node, gmState GmState) GmState { //Error
	fmt.Println("Inside NewState")
	switch node.(type) {
		case NNum:
			fmt.Println("Inide NNum")
			return unwindDump(gmState)
		case NChar:
			return unwindDump(gmState)
		case NConstr:
			return unwindDump(gmState)
		case NAp:
			fmt.Println("Inside NAp")
			addr := Addr(node.(NAp).Left)
			gmState.gms.PushStack(addr)
			gmState.gmc = GmCode{Unwind{}}
			return gmState
		case NGlobal:
			fmt.Println("Inside NGlobal")
			stack := gmState.gms
			heap := gmState.gmh
			dump := gmState.gmd

			fmt.Println("Node")
			fmt.Println(node)
												
			if node.(NGlobal).Nargs > len(stack.Addrs)-1 {
				fmt.Println("Inside if of NGlobal")
				if len(dump) == 0 {
					fmt.Println("Not enough arguments on the stack")
					return GmState{}
				} else {
					dumpElement := dump[0:1][0]
					is := dumpElement.gms
					ss := stack.BottomOfStack()
					is.PushStack(ss)
					return GmState{gmState.gmo, (dumpElement.gmc), is, gmState.gmd, dumpElement.gmvstack, gmState.gmh, gmState.gmg, gmState.gmst}
				}
			} else {
				fmt.Println("Inside else of NGlobal")
				gmstack := rearrange(node.(NGlobal).Nargs, heap, stack)
				fmt.Println("Done with rearrange")
				return GmState{gmState.gmo, node.(NGlobal).GmC, gmstack, gmState.gmd, gmState.gmvstack, gmState.gmh, gmState.gmg, gmState.gmst}
			}
		// Require to execute tail function in which except head element all other elements of stack get returned & concatenated with node
		case NInd:
			fmt.Println("Inside NInd")
			stack := gmState.gms
			stack.TopOfStack()
			stack.PushStack(Addr(node.(NInd)))
			return GmState{gmState.gmo, GmCode{Unwind{}}, stack, gmState.gmd, gmState.gmvstack, gmState.gmh, gmState.gmg, gmState.gmst}
	}
	return GmState{}
}

func unwindDump(gmState GmState)  GmState { //DONE:
	addr := gmState.gms.TopOfStack()
	dumpElement := gmState.gmd[0:1][0]
	gmState.gms = dumpElement.gms
	gmState.gms.PushStack(addr)
	gmState.gmc = dumpElement.gmc
	gmState.gmvstack = dumpElement.gmvstack
	gmState.gmd = gmState.gmd[1:]
	return gmState
}

func pushglobal(name string, gmState GmState) GmState { //Done
	nm := Name(name)
	addr := GlobalsLookup(gmState.gmg, nm)
	if addr == -1 {
		return GmState{}
	}
	gmState.gms.PushStack(addr)
	return GmState{gmState.gmo, gmState.gmc, gmState.gms, gmState.gmd, gmState.gmvstack, gmState.gmh, gmState.gmg, gmState.gmst}
}

func pushgen(n int, str string, node Node, gmState GmState) GmState { //Done
	heap := gmState.gmh
	stack := gmState.gms
	globals := gmState.gmg
	nm := Name(str)
	addr := GlobalsLookup(globals, nm)
	if addr == -1 {
		fmt.Println("Inside Pushgen")
		addrDash := heap.HAlloc(node)
		stack.PushStack(addrDash)
		globalsDash := append(GmGlobals{Object{nm, addrDash}}, globals...)
		return GmState{gmState.gmo, gmState.gmc, stack, gmState.gmd, gmState.gmvstack, heap, globalsDash, gmState.gmst}
	} else {
		stack.PushStack(addr)
		return GmState{gmState.gmo, gmState.gmc, stack, gmState.gmd, gmState.gmvstack, gmState.gmh, gmState.gmg, gmState.gmst}
	}
}

func pushint(n int, gmState GmState) GmState { //Done
	fmt.Println("in Pushint")
	str := strconv.Itoa(n)
	return pushgen(n, str, NNum(n), gmState)
}
func pushchar(c int, gmState GmState) GmState { //Done
	str := strconv.Itoa(c)
	return pushgen(c, str, NChar(c), gmState)
}

func mkap(gmState GmState) GmState { //Done
	a1 := gmState.gms.PopStack()
	a2 := gmState.gms.PopStack()
	addrDash := gmState.gmh.HAlloc(NAp{a1, a2}) //Doubt About Passing type ENum or EVar
	gmState.gms.PushStack(addrDash)
	return gmState
}

func push(n int, gmState GmState) GmState { //Done: Doubt addr taken from stack and pushed in it.
	argAddr := gmState.gms.AddrsByIndexOf(n)
	gmState.gms.PushStack(argAddr)
	return gmState
}

func update(n int, gmState GmState) GmState { //Done
	a := gmState.gms.PopStack()
	fmt.Println(n)
	fmt.Println(gmState.gms)
	redexRoot := gmState.gms.AddrsByIndexOf(n)
	fmt.Println("Readexroot: ", redexRoot)
	gmState.gmh.HUpdate(redexRoot, NInd(a)) 
	return gmState
}

func pop(n int, gmState GmState) GmState { //Done
	gmState.gms.DropStack(n)
	return gmState
}

func slide(n int, gmState GmState) GmState { //Done
	a := gmState.gms.PopStack()
	gmState.gms.DropStack(n)
	gmState.gms.PushStack(a)
	return gmState
}

func alloc(n int, gmState GmState) GmState { //Done
	heapDash, as := allocNodes(n, gmState.gmh)
	stackDash := InitStack()
	for i := 0; i <=as.Index; i++ {
		stackDash.PushStack(as.Addrs[i])
	}
	for i := 0; i <= gmState.gms.Index; i++ {
		stackDash.PushStack(gmState.gms.Addrs[i])
	}
	gmState.gms = stackDash
	gmState.gmh = heapDash
	return gmState
}

func allocNodes(n int, gmh GmHeap) (GmHeap, GmStack){ //Done
	if n == 0 {
		return gmh, GmStack{[10]Addr{},-1}
	}
	heap0, as := allocNodes(n-1, gmh)
	a := heap0.HAlloc(NInd(Addr(0)))
	as.PushStack(a)
	return heap0, as	
}

func eval2(gmState GmState) GmState { //DOne
	vstack := gmState.gmvstack
	a := gmState.gms.PopStack()
	code := gmState.gmc
	dumpDash := GmDump{GmDumpItem{code, gmState.gms, vstack}}
	dumpDash = append(dumpDash, gmState.gmd...)
	gmState.gmd = dumpDash
	gmState.gmc = GmCode{Unwind{}}
	gmState.gms = InitStackWithAddr(a)
	return gmState
}

func rearrange(n int, gmh GmHeap, gms GmStack) GmStack { //DOne Inefficiently
	tail := gms.TailStack()
	take := tail.TakeNStack(n)
	var addrss [10]Addr
	i := -1
	for i := Addr(0); i <= gmh.index; i++ {
		if gmh.hNode[i] == nil {
			break
		}
		heapaddrs := getArg(gmh.hNode[i])
		fmt.Println("Done with arg: ", heapaddrs)
		if take.StackLookup(heapaddrs) {
			i = i + 1
			addrss[i] = heapaddrs
			fmt.Println("Heap Addr: ",heapaddrs)
		}
	}
	return GmStack{addrss, i}
}


func getArg(node Node) Addr{ //Done
	fmt.Println(node.(NGlobal))
	return Addr(node.(NGlobal).Nargs)
}

func add(gmState GmState) GmState{ //Done
	return arithmetic2("+", gmState)
}
func sub(gmState GmState) GmState{ //Done
	return arithmetic2("-", gmState)
}
func mul(gmState GmState) GmState{ //Done
	return arithmetic2("*", gmState)
}
func div(gmState GmState) GmState{ //Done
	return arithmetic2("/", gmState)
}
func mod(gmState GmState) GmState{ //Done
	return arithmetic2("%", gmState)
}
func eq(gmState GmState) GmState { //Done
	return relational2("==", gmState)
}
func ne(gmState GmState) GmState { //Done
	return relational2("/=", gmState)
}
func lt(gmState GmState) GmState { //Done
	return relational2("<", gmState)
}
func le(gmState GmState) GmState { //Done
	return relational2("<=", gmState)
}
func gt(gmState GmState) GmState { //Done
	return relational2(">", gmState)
}
func ge(gmState GmState) GmState { //Done
	return relational2(">=", gmState)
}

func arithmetic2(op string, gmState GmState) GmState{ //Done
	return binOp(op, gmState)
}

func relational2(op string, gmState GmState) GmState{
	return binOp(op, gmState)
}

func binOp(op string, gmState GmState) GmState{ //DOne
	vstack := gmState.gmvstack
	newVS := []int{calculate(op,vstack[len(vstack)-1],vstack[len(vstack)-2])}
	gmState.gmvstack = append(newVS, vstack[0:len(vstack)-2]...)
	return gmState
}

func calculate(op string, v1 int, v2 int) int{ //Done
	switch op {
		case "+":
			return v1 + v2
		case "-":
			return v1 -v2
		case "*":
			return v1 * v2
		case "/":
			return v1 / v2
		default:
			return v1 % v2
	
	}
}

//------------------------------------Extra PArt----------------------------------------
//--------------------------------------------------------------------------------------

// func get(gmState GmState) GmState { // Done with some errors
// 	vstack :=  gmState.gmvstack
// 	a := gmState.gms.PopStack()	
// 	node := gmState.gmh.HLookup(a)
// 	switch  node.(type) {
// 		case NNum:
// 			vstack = append(vstack, node.(NNum))
// 		case NConstr:
// 			vstack = append(vstack, node.(NConstr).Addrs...)
// 	}
// 	gmState.gmvstack = vstack
// 	return gmState
// }


func pushBasic(n int ,gmState GmState) GmState { //Done
	gmState.gmvstack = append(gmState.gmvstack, n)
	return gmState
}

func pack(t int, n int, gmState GmState) GmState {
	take := gmState.gms.TakeNStack(n)
	addr := gmState.gmh.HAlloc(NConstr{t, take.Addrs})
	gmState.gms.DropStack(n)
	gmState.gms.PushStack(addr)
	return gmState
}

func select2(r int, i int, gmState GmState) GmState { //Done with Confusion
	heap := gmState.gmh
	a := gmState.gms.PopStack()
	node := heap.HLookup(a)
	stackElem := node.(NConstr).Addrs[i]
	gmState.gms.PushStack(stackElem)
	return gmState
}

func split2(n int, gmState GmState) GmState { //Some Mistakes but Complete
	heap := gmState.gmh
	a := gmState.gms.PopStack()
	node := heap.HLookup(a)
	if n == len(node.(NConstr).Addrs) {
		// stackDash := GmStack{[10]Addr{node.(NConstr).Addrs, gmState.gms.Addrs}, len(node.(NConstr).Addrs) + len(gmState.gms)}
		// gmState.gms = stackDash
		return gmState
	} else {
		fmt.Println("Incorrect number of constructor parameters.")
		return GmState{}
	}
}

// func error2(msg string, gmState GmState) GmState {

// }