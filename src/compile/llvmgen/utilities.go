package llvmgen

import "llvm.org/llvm/bindings/go/llvm"

func createGetTag(module llvm.Module) llvm.Value {

	bodyBuilder := func (Builder llvm.Builder, f llvm.Value) {
		tag := Builder.CreateLoad(f.Param(0), "tag")
		Builder.CreateRet(tag)
	}

	return createFunction(
		module,
		"getTag",
		llvm.Int64Type(),
		[]llvm.Type{llvm.PointerType(llvm.Int64Type(), 0)},
		[]string{"addr"},
		bodyBuilder,
	)
}

func createNextPtr(module llvm.Module) llvm.Value {

	bodyBuilder := func (Builder llvm.Builder, f llvm.Value) {
		v1 := Builder.CreatePtrToInt(f.Param(1), llvm.Int64Type(), "v1")
		v2 := Builder.CreateMul(f.Param(0), llvm.ConstInt(llvm.Int64Type(), 8, false), "v2")
		v3 := Builder.CreateAdd(v2, v1, "v3")
		v4 := Builder.CreateIntToPtr(v3, llvm.PointerType(llvm.Int8Type(), 0), "v4")
		Builder.CreateRet(v4)
	}

	return createFunction(
		module,
		"nextPtr",
		llvm.PointerType(llvm.Int8Type(), 0),
		[]llvm.Type{
			llvm.Int64Type(),
			llvm.PointerType(llvm.Int64Type(), 0),
		},
		[]string{"n", "ptr"},
		bodyBuilder,
	)
}

func createGetPtrFuncs(module llvm.Module, name string, num uint64, castTo llvm.Type, nextPtr llvm.Value) llvm.Value {

	bodyBuilder := func (Builder llvm.Builder, f llvm.Value) {
		p8 := Builder.CreateCall(
			nextPtr,
			[]llvm.Value{
				llvm.ConstInt(llvm.Int64Type(), num, false),
				f.Param(0),
			},
			"p8",
		)
		p := Builder.CreateBitCast(p8, castTo, "p")
		Builder.CreateRet(p)
	}

	return createFunction(
		module,
		"get" + name + "Ptr",
		castTo,
		[]llvm.Type{
			llvm.PointerType(llvm.Int64Type(), 0),
		},
		[]string{"addr"},
		bodyBuilder,
	)
}

func createUtilityFunctions(module llvm.Module) {
	createGetTag(module)
	nextPtr := createNextPtr(module)

	createGetPtrFuncs(module, "Num", 1, llvm.PointerType(llvm.Int64Type(), 0), nextPtr)
	createGetPtrFuncs(module, "Fun", 1, llvm.PointerType(llvm.PointerType(llvm.Int64Type(), 0), 0), nextPtr)
	createGetPtrFuncs(module, "Arg", 2, llvm.PointerType(llvm.PointerType(llvm.Int64Type(), 0), 0), nextPtr)
	createGetPtrFuncs(module, "Arity", 1, llvm.PointerType(llvm.Int64Type(), 0), nextPtr)
	createGetPtrFuncs(module, "Code", 2, llvm.PointerType(llvm.PointerType(llvm.FunctionType(llvm.VoidType(), []llvm.Type{}, false), 0), 0), nextPtr)
	createGetPtrFuncs(module, "Addr", 1, llvm.PointerType(llvm.PointerType(llvm.Int64Type(), 0), 0), nextPtr)
	createGetPtrFuncs(module, "ConstrTag", 1, llvm.PointerType(llvm.Int64Type(), 0), nextPtr)
	createGetPtrFuncs(module, "ConstrArity", 2, llvm.PointerType(llvm.Int64Type(), 0), nextPtr)
	createGetPtrFuncs(module, "ConstrArgs", 3, llvm.PointerType(llvm.PointerType(llvm.Int64Type(), 0), 0), nextPtr)
}
