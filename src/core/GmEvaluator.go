package core	

import (
	"fmt"
	"strconv"
	// "reflect"
)


func EvalState(gmState GmState) []GmState { //Done
	//result := gmState
	if gmFinal(gmState) == true {
		return []GmState{gmState}
	}  else {
		fmt.Println("Dispatching: ", gmState.gmc)
		return append([]GmState{gmState}, EvalState(doAdmin(step(gmState)))...)
	}
}

func doAdmin(gmState GmState) GmState { //Done
	return GmState{gmState.gmo, gmState.gmc, gmState.gms, gmState.gmd, gmState.gmvstack, gmState.gmh, gmState.gmg, statIncSteps(gmState.gmst)}
}

func statIncSteps(gmst GmStats) GmStats { //Done
	return gmst + 1
}

func gmFinal(gmState GmState) bool { //Done
	fmt.Println("GmCode Len: ", len(gmState.gmc), "Stack: ", gmState.gms)
	if len(gmState.gmc) == 0 {
		fmt.Println("Got it Final GmCode")
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
			fmt.Println("Push ", int(instr.(Push)))
			return push(int(instr.(Push)), gmState)
		case Pop:
			fmt.Println("Pop")
			return pop(int(instr.(Pop)), gmState)
		case Pushglobal:
			fmt.Println("Pushglobal")
			return pushglobal(string(instr.(Pushglobal)), gmState)
		case Pushbasic:
			fmt.Println("PushBasic")
			return pushBasic(int(instr.(Pushbasic)), gmState)
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
			fmt.Println("Eval2")
			return eval2(gmState)
		case Add:
			fmt.Println("Add")
			return add(gmState)
		case Sub:
			fmt.Println("Sub")
			return sub(gmState)
		case Div:
			fmt.Println("Div")
			return div(gmState)
		case Mul:
			fmt.Println("Mul")
			return mul(gmState)
		case Mod:
			fmt.Println("Mod")
			return mod(gmState)
		case Eq:
			fmt.Println("==")
			return eq(gmState)
		case Lt:
			fmt.Println("Less than <")
			return lt(gmState)
		case Gt:
			fmt.Println("Greater >")
			return gt(gmState)
		case Get:
			fmt.Println("Get")
			return get(gmState)
		case MkInt:
			fmt.Println("MkInt")
			return mkInt(gmState)
		case MkBool:
			fmt.Println("MkBool")
			return mkBool(gmState)

		case CasejumpSimple:
			fmt.Println("CasejumpSimple")
			return casejump(CasejumpSimple(instr.(CasejumpSimple)), gmState)

		default:
			fmt.Println("Default")
			// fmt.Println(instr.(type))
			return mod(gmState)

	}
}

func casejump(objs []CasejumpSimpleObj, gmState GmState) GmState {
	fmt.Println("Inside CasejumpSimple")
	heap := gmState.gmh
	node := heap.HLookup(gmState.gms.TopOfStack())
	fmt.Println("Before findMatchingBranch GMC: ", gmState.gmc)
	gmState.gmc = append(findMatchingBranch(objs, node), gmState.gmc...)
	fmt.Println("After findMatchingBranch GMC: ", gmState.gmc)
	return gmState
}

func findMatchingBranch(objs []CasejumpSimpleObj, node Node) GmCode {
	if len(objs) <= 0 {
		return GmCode{}
	} 
	obj := objs[0]
	fmt.Println()
	if obj.Int == -1 {
		return obj.gmC
	}
	switch node.(type) {
		case NNum:
			fmt.Println("findMatchingBranch NNum: ", obj.Int, " == ", int(node.(NNum)))
			if int(node.(NNum)) == obj.Int {
				fmt.Println("Return GmCode: ", obj.gmC)
				return obj.gmC
			} else {
				return findMatchingBranch(objs[1:], node)
			}

		case NChar:
			fmt.Println("findMatchingBranch NChar", obj.Int, " == ", int(node.(NChar)))
			if int(node.(NChar)) == obj.Int {
				return obj.gmC
			} else {
				return findMatchingBranch(objs[1:], node)
			}

		case NConstr:
			fmt.Println("findMatchingBranch NConstr", obj.Int, " == ", int(node.(NConstr).Tag))
			if int(node.(NConstr).Tag) == obj.Int {
				return obj.gmC
			} else {
				return findMatchingBranch(objs[1:], node)
			}

		case NGlobal:
			tmp := (node.(NGlobal)).GmC[0]
			fmt.Println("findMatchingBranch NGlobal", obj.Int)
			switch tmp.(type) {
				case Pack:
					if tmp.(Pack).tag == obj.Int {
						return obj.gmC
					} else {
						return findMatchingBranch(objs[1:], node)
					}

				default:
					return findMatchingBranch(objs[1:], node)
			}

		default:
			fmt.Println("Not possible inside findMatchingBranch Default")
			return GmCode{}
	}
}

func mkInt(gmState GmState) GmState {
	n := int(gmState.gmvstack[0])
	return mkObj(n, NNum(n), gmState)
}

func mkBool(gmState GmState) GmState {
	n := int(gmState.gmvstack[0])
	return mkObj(n, NConstr{n, []Addr{} }, gmState)
}

func mkObj(n int, node Node, gmState GmState) GmState {
	addr := gmState.gmh.HAlloc(node)
	gmState.gms.PushStack(addr)
	gmState.gmvstack = gmState.gmvstack[1:] //Doubt about Direction to retrive Data
	return gmState
}

func unwind(gmState GmState) GmState { //Done
	heap := gmState.gmh
	fmt.Println("GmStack: ",gmState.gms)
	addr := gmState.gms.TopOfStack()
	fmt.Println("Address:", addr)
	node := heap.HLookup(addr)
	fmt.Println("Reading node from: ", addr)
	// gmState.gms.PushStack(Addr(node.(NInd)))
	fmt.Println("Node:", node)
	fmt.Println("Heap: ", heap)
	if  node == nil {
		fmt.Println("End of all GmStates")
		return GmState{}
	}
	return newState(node, gmState)
}

func newState(node Node, gmState GmState) GmState { 
	fmt.Println("Inside NewState")
	fmt.Println("Stack: ", gmState.gms)
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

			fmt.Println(node.(NGlobal).Nargs, ">", len(stack.Addrs) - 1/* stack.index */)
												
			if node.(NGlobal).Nargs > len(stack.Addrs) {
				fmt.Println("Inside if of NGlobal")
				fmt.Println("Dump:",dump)
				if len(dump) == 0 {
					fmt.Println("Not enough arguments on the stack")
					return GmState{}
				} else {
					dumpElement := dump[0]
					is := dumpElement.gms
					ss := stack.BottomOfStack()
					is.PushStack(ss)
					fmt.Println(dumpElement.gmc)
					fmt.Println("Get vstack from dump: ", dumpElement.gmvstack)
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
			// xyz := stack.PopStack()
			fmt.Println("NInd Addr: ",(node.(NInd)))
			fmt.Println("Heap item: ",gmState.gmh.HLookup(Addr(node.(NInd))))
			stack.PushStack(Addr(node.(NInd)))
			return GmState{gmState.gmo, GmCode{Unwind{}}, stack, gmState.gmd, gmState.gmvstack, gmState.gmh, gmState.gmg, gmState.gmst}
	}
	return GmState{}
}


func rearrange(n int, gmh GmHeap, gms GmStack) GmStack { //DOne Inefficiently
	if n < 0 {
		return gms
	} 
	fmt.Println("Stack Before TailStack: ", gms)
	tail := gms.TailStack()
	fmt.Println("TailStack: ", tail, "\n n: ", n)
	take := tail.TakeNStack(n)
	fmt.Println("TakeNStack: ",take)
	var addrss []Addr
	i := -1

	for _, addr := range take.Addrs {
		fmt.Println("Checking value at, ", addr)
		node := gmh.HLookup(Addr(addr))

		if node != nil {
			switch node.(type) {
				case NAp:
					i = i + 1
					addrss = append(addrss, getArg(node))
					fmt.Println("Heap Node: ",node,"\nHeap Addr: ", addrss[i])
				default:
					fmt.Println("Heap Node: ", node)
			}
		}
	}
	fmt.Println("Hello n, ", n)
	fmt.Println("len(gms.Addrs), ", len(gms.Addrs))
	fmt.Println(addrss, gms.Addrs)
	addrss = append(addrss, gms.Addrs[n:]...)

	fmt.Println("addresses after rearrange, ",  addrss)
	// for j := gms.Index -n ; j < gms.Index; j++ {
	// 	i = i + 1
	// 	addrss[i] = gms.Addrs[j]
	// 	fmt.Println("i: ",i," j: ",j)
	// }
	// for j := n; j < len(gms.Addrs); j++ {
	// 	i = i + 1
	// 	// addrss[i] = gms.Addrs[j]
	// 	addrss = append(addrss, gms.Addrs[j])
	// 	fmt.Println("i: ",i," j: ",j)
	// }

	st := GmStack{addrss}
	fmt.Println("rearrange returns stack: ", st)
	return st
}

func getArg(node Node) Addr{ //Done
	fmt.Println("get arg: ", node.(NAp))
	return Addr(node.(NAp).Body)
}

func unwindDump(gmState GmState)  GmState { //DONE:
	addr := gmState.gms.TopOfStack()
	dumpElement := gmState.gmd[0]
	gmState.gms = dumpElement.gms
	gmState.gms.PushStack(addr)
	fmt.Println("UnwindDump: Pushing ",addr, " inside stack")
	fmt.Println("Dump GmCode: ", dumpElement.gmc)
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
	fmt.Println("GmGlobalsLookup Addr: ", addr)
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
	fmt.Println("Address of NAp: ", addrDash, a1, a2)
	fmt.Println("Node: ", NAp{a1, a2})
	gmState.gms.PushStack(addrDash)
	return gmState
}

func push(n int, gmState GmState) GmState { //Done: Doubt addr taken from stack and pushed in it.
	fmt.Println("stack before push ", n, " ", gmState.gms)
	argAddr := gmState.gms.AddrsByIndexOf(n)
	gmState.gms.PushStack(argAddr)
	fmt.Println("stack after push ", n, " ", gmState.gms)
	return gmState
}

func update(n int, gmState GmState) GmState { //Done
	fmt.Println(n)
	fmt.Println(gmState.gms)
	redexRoot := gmState.gms.AddrsByIndexOf(n)
	a := gmState.gms.PopStack()
	fmt.Println("Readexroot: ", redexRoot)
	nidaddrs := gmState.gmh.HAlloc(NInd(a))
	//Stack n+1 condition
	gmState.gms.PushStack(nidaddrs)
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
	// for i := 0; i <=as.Index; i++ {
	// 	stackDash.PushStack(as.Addrs[i])
	// }
	// for i := 0; i <= gmState.gms.Index; i++ {
	// 	stackDash.PushStack(gmState.gms.Addrs[i])
	// }
	stackDash.Addrs = append(stackDash.Addrs, as.Addrs...)
	gmState.gms = stackDash
	gmState.gmh = heapDash
	return gmState
}

func allocNodes(n int, gmh GmHeap) (GmHeap, GmStack){ //Done
	if n == 0 {
		return gmh, GmStack{[]Addr{}}
	}
	heap0, as := allocNodes(n-1, gmh)
	a := heap0.HAlloc(NInd(Addr(0)))
	as.PushStack(a)
	return heap0, as	
}

func eval2(gmState GmState) GmState { //DOne
	vstack := gmState.gmvstack
	a := gmState.gms.PopStack()
	stack := gmState.gms
	code := gmState.gmc
	fmt.Println("Inside Eval2(GmCode): ", gmState.gmc)
	dumpDash := GmDump{GmDumpItem{code, stack, vstack}}
	fmt.Println("Here Dump is added in GmDump: ", dumpDash)
	dumpDash = append(dumpDash, gmState.gmd...)
	gmState.gmd = dumpDash
	gmState.gmc = GmCode{Unwind{}}
	gmState.gms = InitStackWithAddr(a)
	return gmState
}

func get(gmState GmState) GmState { // Done 
	vstack :=  gmState.gmvstack
	fmt.Println("Before Get vstack: ", vstack)
	a := gmState.gms.PopStack()	
	node := gmState.gmh.HLookup(a)
	switch  node.(type) {
		case NNum:
			vstack = append(vstack, int(node.(NNum)))
		case NConstr:
			vstack = append(vstack, int(node.(NConstr).Tag))
	}
	gmState.gmvstack = vstack
	fmt.Println("AFter Get vstack: ", gmState.gmvstack)
	return gmState
}

func pushBasic(n int ,gmState GmState) GmState { //Done
	gmState.gmvstack = append(gmState.gmvstack, n)
	return gmState
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
	fmt.Println("Inside binOp vstack: ", vstack)
	fmt.Println(int(vstack[len(vstack)-2])," > ", 0)
	if int(vstack[len(vstack)-2]) > 0 {
		newVS := []int{calculate(op, int(vstack[len(vstack)-1]), int(vstack[len(vstack)-2]))}
		gmState.gmvstack = append(newVS, vstack[0:len(vstack)-2]...)
		fmt.Println("gmvstack: ", gmState.gmvstack)
		return gmState
	}
	return GmState{}
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
		case "%":
			return v1 % v2
		case "==":
			if v1 == v2 { return 1 } else { return 0 }
		case "<":
			if v1 < v2 { return 1 } else { return 0	}
		case ">":
			if v1 > v2 { return 1 } else { return 0	}
		default:
			return 0
	
	}
}

//------------------------------------Extra PArt----------------------------------------
//--------------------------------------------------------------------------------------

func pack(t int, n int, gmState GmState) GmState { // Adding Data to Constructors
	stack := gmState.gms
	take := gmState.gms.TakeNStack(n)
	addr := gmState.gmh.HAlloc(NConstr{t, take.Addrs})
	stack.DropStack(n)
	stack.PushStack(addr)
	gmState.gms = stack
	return gmState
}

func select2(r int, i int, gmState GmState) GmState { //Done with Confusion
	heap := gmState.gmh
	a := gmState.gms.PopStack()
	node := heap.HLookup(a)
	stackElem := node.(NConstr).Arity[i]
	gmState.gms.PushStack(stackElem)
	return gmState
}

func split2(n int, gmState GmState) GmState { //Some Mistakes but Complete
	heap := gmState.gmh
	a := gmState.gms.PopStack()
	node := heap.HLookup(a)
	if n == len(node.(NConstr).Arity) {
		// stackDash := GmStack{[10]Addr{node.(NConstr).Addrs, gmState.gms.Addrs}, len(node.(NConstr).Addrs) + len(gmState.gms)}
		// gmState.gms = stackDash
		return gmState
	} else {
		fmt.Println("Incorrect number of constructor parameters.")
		return GmState{}
	}
}

