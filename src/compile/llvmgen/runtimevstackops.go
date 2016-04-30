package llvmgen

import "llvm.org/llvm/bindings/go/llvm"

func createGetItemVPtr(module llvm.Module) llvm.Value {
	getItemVPtr := llvm.AddFunction(module, "getItemVPtr", llvm.FunctionType(
		llvm.PointerType(llvm.Int64Type(), 0),
		[]llvm.Type{llvm.Int64Type()},
		false,
	))

	n := getItemVPtr.Param(0)
	n.SetName("n")

	getItemVPtrBody := llvm.AddBasicBlock(getItemVPtr, "")
	IRBuilder.SetInsertPointAtEnd(getItemVPtrBody)
	ptr := IRBuilder.CreateGEP(
		module.NamedGlobal("vstack"),
		[]llvm.Value{
			llvm.ConstInt(llvm.Int64Type(), 0, false),
			n,
		},
		"ptr")
	IRBuilder.CreateRet(ptr)
	IRBuilder.ClearInsertionPoint()

	SourceGlobals["getItemVPtr"] = getItemVPtr
	return getItemVPtr

}

func createTopVPtr(module llvm.Module) llvm.Value {

	getItemVPtr := SourceGlobals["getItemVPtr"]
	vsp := SourceGlobals["vsp"]

	getTopVPtr := llvm.AddFunction(module, "getTopVPtr", llvm.FunctionType(
		llvm.PointerType(llvm.Int64Type(), 0),
		[]llvm.Type{},
		false,
	))

	getTopVPtrBody := llvm.AddBasicBlock(getTopVPtr, "")
	IRBuilder.SetInsertPointAtEnd(getTopVPtrBody)
	n := IRBuilder.CreateLoad(vsp, "n")
	n1 := IRBuilder.CreateSub(n, llvm.ConstInt(llvm.Int64Type(), 1, false), "n1")
	item := IRBuilder.CreateCall(
		getItemVPtr,
		[]llvm.Value{n1},
		"item",
	)
	IRBuilder.CreateRet(item)
	IRBuilder.ClearInsertionPoint()

	SourceGlobals["getTopVPtr"] = getTopVPtr
	return getTopVPtr
}

func createPopV(module llvm.Module) llvm.Value {

	getTopVPtr := SourceGlobals["getTopVPtr"]
	decSp := SourceGlobals["decSp"]
	vsp := SourceGlobals["vsp"]

	popV := llvm.AddFunction(module, "popV", llvm.FunctionType(
		llvm.Int64Type(),
		[]llvm.Type{},
		false,
	))

	popVBody := llvm.AddBasicBlock(popV, "")
	IRBuilder.SetInsertPointAtEnd(popVBody)
	ptop := IRBuilder.CreateCall(
		getTopVPtr,
		[]llvm.Value{},
		"ptop",
	)
	val := IRBuilder.CreateLoad(ptop, "val")
	IRBuilder.CreateCall(
		decSp,
		[]llvm.Value{vsp},
		"",
	)
	IRBuilder.CreateRet(val)
	IRBuilder.ClearInsertionPoint()

	SourceGlobals["popV"] = popV
	return popV
}

func createPushV(module llvm.Module) llvm.Value {
	vsp := SourceGlobals["vsp"]
	getItemVPtr := SourceGlobals["getItemVPtr"]
	incSp := SourceGlobals["incSp"]

	bodyBuilder := func (Builder llvm.Builder, f llvm.Value) {
		n := Builder.CreateLoad(vsp, "n")
		ptop := Builder.CreateCall(
			getItemVPtr,
			[]llvm.Value{n},
			"ptop",
		)
		Builder.CreateStore(f.Param(0), ptop)
		Builder.CreateCall(
			incSp,
			[]llvm.Value{vsp},
			"",
		)
		Builder.CreateRetVoid()
	}

	return createFunction(
		module,
		"pushV",
		llvm.VoidType(),
		[]llvm.Type{llvm.Int64Type()},
		[]string{"val"},
		bodyBuilder,
	)
}

func createRuntimeVStackOps(module llvm.Module) {
	createGetItemVPtr(module)
	createTopVPtr(module)
	createPopV(module)
	createPushV(module)
}
