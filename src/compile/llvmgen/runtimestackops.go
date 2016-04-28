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
	IRBuilder.CreateRetVoid()
	IRBuilder.ClearInsertionPoint()

	SourceGlobals["push"] = push
	return push
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

func createRuntimeStackOperations(module llvm.Module) {
	createGetItemPtr(module)
	createPush(module)
	createGetTopPtr(module)
}

