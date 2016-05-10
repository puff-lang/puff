package core

import (
	"fmt"
	"reflect"
	"strconv"
)

func showNode(node Node, addr Addr, state GmState) string {
	switch node.(type) {
	case NNum:
		return strconv.Itoa(int(node.(NNum)))
	case NGlobal:
		gmg := GetGlobals(state)
		var name Name
		for _, obj := range gmg {
			if obj.Addr == addr {
				name = obj.Name
			}
		}
		return "<fun " + string(name) + ">"
	case NAp:
		return "(" + showNode(state.gmh.HLookup(node.(NAp).Left), node.(NAp).Left, state) + " " + showNode(state.gmh.HLookup(node.(NAp).Body), node.(NAp).Body, state) + ")"
	case NInd:
		addr := Addr(node.(NInd))
		if addr == -1 {
			return "NULL"
		} else {
			return /* "NInd " + */ showNode(state.gmh.HLookup(addr), addr, state)
		}
	case NConstr:
		return "CONSTR: " + string((node.(NConstr)).Tag)
	case NMarked:
		return "marked"
	default:
		return "don't know"
	}
}

func ShowStates(gmState GmState) {
	states := EvalState(gmState)

	for _, state := range states {
		node := state.gmh.HLookup(state.gms.TopOfStack())
		fmt.Println(showNode(node, state.gms.TopOfStack(), state))
	}

	endState := states[len(states)-1]
	node := endState.gmh.HLookup(endState.gms.TopOfStack())
	fmt.Println("Output:", showNode(node, endState.gms.TopOfStack(), endState))
}

func EvalState(gmState GmState) []GmState { //Done
	//result := gmState
	if gmFinal(gmState) == true {
		return []GmState{gmState}
	} else {
		return append([]GmState{gmState}, EvalState(doAdmin(step(gmState)))...)
	}
}

func doAdmin(gmState GmState) GmState { //Done
	gmState.gmst = statIncSteps(gmState.gmst)
	return gmState
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
	gmState.gmc = gmState.gmc[1:]
	fmt.Println(gmState.gms)
	return dispatch(i, gmState)
}

func dispatch(instr Instruction, gmState GmState) GmState { //Done
	switch instr.(type) {
	case Update:
		fmt.Println("Update", int(instr.(Update)))
		return update(int(instr.(Update)), gmState)
	case Push:
		fmt.Println("Push ", int(instr.(Push)))
		return push(int(instr.(Push)), gmState)
	case Pop:
		fmt.Println("Pop", int(instr.(Pop)))
		return pop(int(instr.(Pop)), gmState)
	case Pushglobal:
		fmt.Println("Pushglobal", string(instr.(Pushglobal)))
		return pushglobal(string(instr.(Pushglobal)), gmState)
	case Pushbasic:
		fmt.Println("PushBasic", int(instr.(Pushbasic)))
		return pushBasic(int(instr.(Pushbasic)), gmState)
	case Mkap:
		fmt.Println("Mkap")
		return mkap(gmState)
	case Unwind:
		fmt.Println("Unwind")
		return unwind(gmState)
	case Pushint:
		fmt.Println("Pushint", int(instr.(Pushint)))
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

	// case CasejumpSimple:
	// 	fmt.Println("CasejumpSimple")
	// 	return casejump(CasejumpSimple(instr.(CasejumpSimple)), gmState)
	case CasejumpConstr:
		fmt.Println("CasejumpConstr")
		return casejump(instr.(CasejumpConstr), gmState)

	case Pushconstr:
		fmt.Println("Pushconstr")
		n := instr.(Pushconstr)
		return pushconstr(n.Tag, n.Arity, gmState)

	case Pack:
		fmt.Println("Pack")
		n := instr.(Pack)
		return pack(n.Tag, n.Arity, gmState)

	default:
		fmt.Println("Dispatch for", reflect.TypeOf(instr), "instruction on implemented.")
		panic("Exiting")
	}
}

func casejump(objs []CasejumpObj, gmState GmState) GmState {
	fmt.Println("Inside CasejumpSimple")
	heap := gmState.gmh
	node := heap.HLookup(gmState.gms.TopOfStack())
	fmt.Println("Before findMatchingBranch GMC: ", gmState.gmc)
	gmState.gmc = append(findMatchingBranch(objs, node), gmState.gmc...)
	fmt.Println("After findMatchingBranch GMC: ", gmState.gmc)
	return gmState
}

func findMatchingBranch(objs []CasejumpObj, node Node) GmCode {
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
			if tmp.(Pack).Tag == obj.Int {
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
	return mkObj(n, NConstr{n, []Addr{}}, gmState)
}

func mkObj(n int, node Node, gmState GmState) GmState {
	addr := gmState.gmh.HAlloc(node)
	gmState.gms.PushStack(addr)
	gmState.gmvstack = gmState.gmvstack[1:] //Doubt about Direction to retrive Data
	return gmState
}

func unwind(gmState GmState) GmState { //Done
	heap := gmState.gmh
	addr := gmState.gms.TopOfStack()
	node := heap.HLookup(addr)
	// gmState.gms.PushStack(Addr(node.(NInd)))
	if node == nil {
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
		fmt.Println("Dispatching NNum", showNode(node, gmState.gms.TopOfStack(), gmState))
		return unwindDump(gmState)
	case NChar:
		return unwindDump(gmState)
	case NConstr:
		return unwindDump(gmState)
	case NAp:
		fmt.Println("Dispatching NAp")
		addr := Addr(node.(NAp).Left)
		gmState.gms.PushStack(addr)
		gmState.gmc = GmCode{Unwind{}}
		return gmState
	case NGlobal:
		stack := gmState.gms
		heap := gmState.gmh
		dump := gmState.gmd
		fmt.Println("Dispatching NGlobal", showNode(node, gmState.gms.TopOfStack(), gmState),
			"Nargs", node.(NGlobal).Nargs, "Stack length", len(stack.Addrs))

		if node.(NGlobal).Nargs > len(stack.Addrs)-1 {
			if len(dump) == 0 {
				fmt.Println("Not enough arguments on the stack")
				return GmState{}
			} else {
				dumpElement := dump[0]
				gmState.gmc = dumpElement.gmc
				gmState.gms = dumpElement.gms
				gmState.gms.PushStack(stack.BottomOfStack())
				gmState.gmvstack = dumpElement.gmvstack
				return gmState
			}
		} else {
			gmState.gms = rearrange(node.(NGlobal).Nargs, heap, stack)
			gmState.gmc = node.(NGlobal).GmC
			return gmState
		}
	// Require to execute tail function in which except head element all other elements of stack get returned & concatenated with node
	case NInd:
		fmt.Println("Dispatching NInd")
		fmt.Println("Heap", gmState.gmh)
		gmState.gms.PopStack()
		gmState.gms.PushStack(Addr(node.(NInd)))
		gmState.gmc = GmCode{Unwind{}}
		return gmState
	}
	return GmState{}
}

func rearrange(n int, gmh GmHeap, gms GmStack) GmStack { //DOne Inefficiently
	tail := gms.TailStack()
	fmt.Println("TailStack", tail)
	take := tail.TakeNStack(n)
	fmt.Println("take", n, take)
	var addrss []Addr
	i := -1

	for _, addr := range take.Addrs {
		node := gmh.HLookup(Addr(addr))

		if node != nil {
			switch node.(type) {
			case NAp:
				i = i + 1
				addrss = append(addrss, getArg(node))
			default:
				fmt.Println("Heap Node: ", node)
			}
		}
	}
	fmt.Println("len(gms.Addrs), ", len(gms.Addrs))
	fmt.Println(addrss, gms.Addrs)
	addrss = append(addrss, gms.Addrs[n:]...)

	fmt.Println("addresses after rearrange, ", addrss)
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

func getArg(node Node) Addr { //Done
	fmt.Println("get arg: ", node.(NAp))
	return Addr(node.(NAp).Body)
}

func unwindDump(gmState GmState) GmState { //DONE:
	fmt.Println("Stack before unwindDump", gmState.gms)
	addr := gmState.gms.TopOfStack()
	dumpElement := gmState.gmd[0]

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
	nm := Name(str)
	addr := GlobalsLookup(gmState.gmg, nm)
	if addr == -1 {
		addrDash := gmState.gmh.HAlloc(node)
		gmState.gms.PushStack(addrDash)
		gmState.gmg = append(GmGlobals{Object{nm, addrDash}}, gmState.gmg...)
		return gmState
	} else {
		gmState.gms.PushStack(addr)
		return gmState
	}
}

func pushint(n int, gmState GmState) GmState { //Done
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
	redexRoot := gmState.gms.AddrsByIndexOf(n)
	fmt.Println("Readexroot: ", redexRoot)
	fmt.Println("Addr for NInd", a)
	// nidaddrs := gmState.gmh.HAlloc(NInd(a))
	gmState.gmh.HUpdate(redexRoot, NInd(a))
	// Stack n+1 condition
	// gmState.gms.PushStack(nidaddrs)
	fmt.Println("Stack after update", gmState.gms)
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
	stackDash.Addrs = append(as.Addrs, gmState.gms.Addrs...)
	gmState.gms = stackDash
	gmState.gmh = heapDash
	return gmState
}

func allocNodes(n int, gmh GmHeap) (GmHeap, GmStack) { //Done
	if n == 0 {
		return gmh, GmStack{[]Addr{}}
	}
	heap0, as := allocNodes(n-1, gmh)
	a := heap0.HAlloc(NInd(Addr(-1)))
	as.PushStack(a)
	return heap0, as
}

func eval2(gmState GmState) GmState { //DOne
	vstack := gmState.gmvstack
	a := gmState.gms.PopStack()
	as := gmState.gms
	code := gmState.gmc

	gmState.gmd = append(GmDump{GmDumpItem{code, as, vstack}}, gmState.gmd...)
	gmState.gmc = GmCode{Unwind{}}
	gmState.gms = InitStackWithAddr(a)

	fmt.Println("Dump", gmState.gmd)

	return gmState
}

func get(gmState GmState) GmState { // Done
	vstack := gmState.gmvstack
	fmt.Println("Before Get vstack: ", vstack)
	a := gmState.gms.PopStack()
	node := gmState.gmh.HLookup(a)
	switch node.(type) {
	case NNum:
		vstack = append([]int{int(node.(NNum))}, vstack...)
	case NConstr:
		vstack = append([]int{int(node.(NConstr).Tag)}, vstack...)
	}
	gmState.gmvstack = vstack
	fmt.Println("AFter Get vstack: ", gmState.gmvstack)
	return gmState
}

func pushBasic(n int, gmState GmState) GmState { //Done
	gmState.gmvstack = append([]int{n}, gmState.gmvstack...)
	return gmState
}

func pushconstr(tag int, arity int, gmState GmState) GmState {
	globals := gmState.gmg
	name := "Pack{" + strconv.Itoa(tag) + "," + strconv.Itoa(arity) + "}"
	addr := GlobalsLookup(globals, Name(name))
	if addr != Addr(-1) {
		gmState.gms.PushStack(addr)
	} else {
		naddr := gmState.gmh.HAlloc(NGlobal{arity, GmCode{Pack{tag, arity}, Update(0), Unwind{}}})
		gmState.gms.PushStack(naddr)
		gmState.gmg = append([]Object{Object{Name(name), naddr}}, gmState.gmg...)
	}

	return gmState
}

func add(gmState GmState) GmState { //Done
	return arithmetic2("+", gmState)
}
func sub(gmState GmState) GmState { //Done
	return arithmetic2("-", gmState)
}
func mul(gmState GmState) GmState { //Done
	return arithmetic2("*", gmState)
}
func div(gmState GmState) GmState { //Done
	return arithmetic2("/", gmState)
}
func mod(gmState GmState) GmState { //Done
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

func arithmetic2(op string, gmState GmState) GmState { //Done
	return binOp(op, gmState)
}

func relational2(op string, gmState GmState) GmState {
	return binOp(op, gmState)
}

func binOp(op string, gmState GmState) GmState { //DOne
	vstack := gmState.gmvstack
	fmt.Println("Inside binOp op:", op)
	fmt.Println("Inside binOp vstack: ", vstack)
	fmt.Println(len(vstack), "=>", 2)
	if len(vstack) > 1 {
		newVS := []int{calculate(op, int(vstack[0]), int(vstack[1]))}
		gmState.gmvstack = append(newVS, vstack[2:]...)
		fmt.Println("gmvstack: ", gmState.gmvstack)
		return gmState
	} else {
		panic("Not enough arguments on vstack to perform binary operation")
	}
	return GmState{}
}

func calculate(op string, v1 int, v2 int) int { //Done
	switch op {
	case "+":
		return v1 + v2
	case "-":
		return v1 - v2
	case "*":
		return v1 * v2
	case "/":
		return v1 / v2
	case "%":
		return v1 % v2
	case "==":
		if v1 == v2 {
			return 1
		} else {
			return 0
		}
	case "<":
		if v1 < v2 {
			return 1
		} else {
			return 0
		}
	case ">":
		if v1 > v2 {
			return 1
		} else {
			return 0
		}
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
