package llvmgen

import "llvm.org/llvm/bindings/go/llvm"

func createGetItemPtr(module llvm.Module) llvm.Value {
	getItemPtr := llvm.AddFunction(module, "getItemPtr", llvm.FunctionType(
		llvm.PointerType(llvm.PointerType(llvm.Int64Type(), 0), 0),
		[]llvm.Type{
			llvm.Int64Type(),
		},
		false))

	getItemPtr.Param(0).SetName("n")
	getItemPtrBody := llvm.AddBasicBlock(getItemPtr, "")
	IRBuilder.SetInsertPointAtEnd(getItemPtrBody)
	ret := IRBuilder.CreateGEP(
		module.NamedGlobal("stack"),
		[]llvm.Value{
			llvm.ConstInt(llvm.Int64Type(), 0, false),
			getItemPtr.Param(0),
		},
		"item")
	IRBuilder.CreateRet(ret)
	IRBuilder.ClearInsertionPoint()

	SourceGlobals["getItemPtr"] = getItemPtr
	return getItemPtr
}

func createPush(module llvm.Module) llvm.Value {

	getItemPtr := SourceGlobals["getItemPtr"]
	incSp := SourceGlobals["incSp"]
	sp := SourceGlobals["sp"]

	push := llvm.AddFunction(module, "push", llvm.FunctionType(llvm.VoidType(), []llvm.Type{
		llvm.PointerType(llvm.Int64Type(), 0),
	}, false))

	addr := push.Param(0)
	addr.SetName("addr")

	pushBody := llvm.AddBasicBlock(push, "")
	IRBuilder.SetInsertPointAtEnd(pushBody)
	n := IRBuilder.CreateLoad(module.NamedGlobal("sp"), "n")
	ptop := IRBuilder.CreateCall(
		getItemPtr,
		[]llvm.Value{n},
		"ptop",
	)
	IRBuilder.CreateStore(addr, ptop)
	IRBuilder.CreateCall(
		incSp,
		[]llvm.Value{sp},
		"",
	)
	IRBuilder.CreateRetVoid()
	IRBuilder.ClearInsertionPoint()

	SourceGlobals["push"] = push
	return push
}

func createPop(module llvm.Module) llvm.Value {
	getTopPtr := SourceGlobals["getTopPtr"]
	decSp := SourceGlobals["decSp"]
	sp := SourceGlobals["sp"]

	pop := llvm.AddFunction(module, "pop", llvm.FunctionType(
		llvm.VoidType(),
		[]llvm.Type{},
		false,
	))

	popBody := llvm.AddBasicBlock(pop, "")
	IRBuilder.SetInsertPointAtEnd(popBody)
	ptop := IRBuilder.CreateCall(
		getTopPtr,
		[]llvm.Value{},
		"ptop",
	)
	addr := IRBuilder.CreateLoad(ptop , "addr")
	IRBuilder.CreateCall(
		decSp,
		[]llvm.Value{sp},
		"",
	)
	IRBuilder.CreateRet(addr)
	IRBuilder.ClearInsertionPoint()

	return pop
}

func createPopn(module llvm.Module) llvm.Value {
	sp := SourceGlobals["sp"]

	popn := llvm.AddFunction(module, "popn", llvm.FunctionType(
		llvm.VoidType(),
		[]llvm.Type{llvm.Int64Type()},
		false,
	))

	n := popn.Param(0)
	n.SetName("n")

	popBody := llvm.AddBasicBlock(popn, "")
	IRBuilder.SetInsertPointAtEnd(popBody)
	vsp := IRBuilder.CreateLoad(sp, "vsp")
	vsp1 := IRBuilder.CreateSub(vsp, n, "vsp1")
	IRBuilder.CreateStore(vsp1, sp)
	IRBuilder.CreateRetVoid()
	IRBuilder.ClearInsertionPoint()

	return popn
}

func createGetTopPtr(module llvm.Module) llvm.Value {

	getItemPtr := SourceGlobals["getItemPtr"]

	getTopPtr := llvm.AddFunction(module, "getTopPtr", llvm.FunctionType(
		llvm.PointerType(llvm.PointerType(llvm.Int64Type(), 0), 0),
		[]llvm.Type{},
		false,
	))
	getTopPtrBody := llvm.AddBasicBlock(getTopPtr, "")

	IRBuilder.SetInsertPointAtEnd(getTopPtrBody)
	n := IRBuilder.CreateLoad(module.NamedGlobal("sp"), "n")
	n1 := IRBuilder.CreateSub(n, llvm.ConstInt(llvm.Int64Type(), 1, false), "n1")
	topPtr := IRBuilder.CreateCall(
		getItemPtr,
		[]llvm.Value{n1},
		"topPtr",
	)
	IRBuilder.CreateRet(topPtr)

	IRBuilder.ClearInsertionPoint()

	SourceGlobals["getTopPtr"] = getTopPtr
	return getTopPtr
}

func createIncSp(module llvm.Module) llvm.Value {
	incSp := llvm.AddFunction(module, "incSp", llvm.FunctionType(llvm.VoidType(), []llvm.Type{
		llvm.PointerType(llvm.Int64Type(), 0),
	}, false))

	sp := incSp.Param(0)
	sp.SetName("sp")

	incSpBody := llvm.AddBasicBlock(incSp, "")
	IRBuilder.SetInsertPointAtEnd(incSpBody)
	n := IRBuilder.CreateLoad(module.NamedGlobal("sp"), "n")
	n1 := IRBuilder.CreateAdd(n, llvm.ConstInt(llvm.Int64Type(), 1, false), "n1")
	IRBuilder.CreateStore(n1, sp)
	IRBuilder.CreateRetVoid()
	IRBuilder.ClearInsertionPoint()

	SourceGlobals["incSp"] = incSp
	return incSp
}

func createDecSp(module llvm.Module) llvm.Value {
	decSp := llvm.AddFunction(module, "decSp", llvm.FunctionType(llvm.VoidType(), []llvm.Type{
		llvm.PointerType(llvm.Int64Type(), 0),
	}, false))

	sp := decSp.Param(0)
	sp.SetName("sp")

	decSpBody := llvm.AddBasicBlock(decSp, "")
	IRBuilder.SetInsertPointAtEnd(decSpBody)
	n := IRBuilder.CreateLoad(module.NamedGlobal("sp"), "n")
	n1 := IRBuilder.CreateSub(n, llvm.ConstInt(llvm.Int64Type(), 1, false), "n1")
	IRBuilder.CreateStore(n1, sp)
	IRBuilder.CreateRetVoid()
	IRBuilder.ClearInsertionPoint()

	SourceGlobals["decSp"] = decSp
	return decSp
}

func createRuntimeStackOps(module llvm.Module) {
	createGetItemPtr(module)
	createGetTopPtr(module)
	createPush(module)
	createPop(module)
	createPopn(module)
}

func createGenericStackOps(module llvm.Module) {
	createIncSp(module)
	createDecSp(module)
}

