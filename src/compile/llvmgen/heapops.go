package llvmgen

import "llvm.org/llvm/bindings/go/llvm"

func createHAllocNum(module llvm.Module) llvm.Value {

	bodyBuilder := func (Builder llvm.Builder, f llvm.Value) {
		ptr := Builder.CreateCall(
			module.NamedGlobal("malloc"),
			[]llvm.Value{llvm.ConstInt(llvm.Int64Type(), 16, false)},
			"ptr",
		)
		ptag := Builder.CreateBitCast(ptr, llvm.PointerType(llvm.Int64Type(), 0), "ptag")
		pval := Builder.CreateCall(
			module.NamedFunction("getNumPtr"),
			[]llvm.Value{ptag},
			"pval",
		)

		numtag := Builder.CreateLoad(module.NamedGlobal("NUM_TAG"), "numtag")
		Builder.CreateStore(numtag, ptag)
		Builder.CreateStore(f.Param(0), pval)

		Builder.CreateRet(ptag)
	}

	return createFunction(
		module,
		"hAllocNum",
		llvm.PointerType(llvm.Int64Type(), 0),
		[]llvm.Type{llvm.Int64Type()},
		[]string{"n"},
		bodyBuilder,
	)

}

func createHAllocAp(module llvm.Module) llvm.Value {

	bodyBuilder := func (Builder llvm.Builder, f llvm.Value) {
		ptr := Builder.CreateCall(
			module.NamedGlobal("malloc"),
			[]llvm.Value{llvm.ConstInt(llvm.Int64Type(), 24, false)},
			"ptr",
		)
		ptag := Builder.CreateBitCast(ptr, llvm.PointerType(llvm.Int64Type(), 0), "ptag")
		pfun := Builder.CreateCall(
			module.NamedFunction("getFunPtr"),
			[]llvm.Value{ptag},
			"pfun",
		)
		parg := Builder.CreateCall(
			module.NamedFunction("getArgPtr"),
			[]llvm.Value{ptag},
			"parg",
		)

		aptag := Builder.CreateLoad(module.NamedGlobal("AP_TAG"), "aptag")
		Builder.CreateStore(aptag, ptag)
		Builder.CreateStore(f.Param(0), pfun)
		Builder.CreateStore(f.Param(1), parg)

		Builder.CreateRet(ptag)
	}

	return createFunction(
		module,
		"hAllocAp",
		llvm.PointerType(llvm.Int64Type(), 0),
		[]llvm.Type{
			llvm.PointerType(llvm.Int64Type(), 0),
			llvm.PointerType(llvm.Int64Type(), 0),
		},
		[]string{"a1", "a2"},
		bodyBuilder,
	)

}

func createHAllocGlobal(module llvm.Module) llvm.Value {

	bodyBuilder := func (Builder llvm.Builder, f llvm.Value) {
		ptr := Builder.CreateCall(
			module.NamedGlobal("malloc"),
			[]llvm.Value{llvm.ConstInt(llvm.Int64Type(), 24, false)},
			"ptr",
		)
		ptag := Builder.CreateBitCast(ptr, llvm.PointerType(llvm.Int64Type(), 0), "ptag")
		parity := Builder.CreateCall(
			module.NamedFunction("getArityPtr"),
			[]llvm.Value{ptag},
			"parity",
		)
		pcode := Builder.CreateCall(
			module.NamedFunction("getCodePtr"),
			[]llvm.Value{ptag},
			"pcode",
		)

		globaltag := Builder.CreateLoad(module.NamedGlobal("GLOBAL_TAG"), "globaltag")
		Builder.CreateStore(globaltag, ptag)
		Builder.CreateStore(f.Param(0), parity)
		Builder.CreateStore(f.Param(1), pcode)

		Builder.CreateRet(ptag)
	}

	return createFunction(
		module,
		"hAllocGlobal",
		llvm.PointerType(llvm.Int64Type(), 0),
		[]llvm.Type{
			llvm.Int64Type(),
			llvm.PointerType(llvm.FunctionType(llvm.VoidType(), []llvm.Type{}, false), 0),
		},
		[]string{"arity", "funPtr"},
		bodyBuilder,
	)

}

func createHAllocInd(module llvm.Module) llvm.Value {

	bodyBuilder := func (Builder llvm.Builder, f llvm.Value) {
		ptr := Builder.CreateCall(
			module.NamedGlobal("malloc"),
			[]llvm.Value{llvm.ConstInt(llvm.Int64Type(), 16, false)},
			"ptr",
		)
		ptag := Builder.CreateBitCast(ptr, llvm.PointerType(llvm.Int64Type(), 0), "ptag")
		paddr := Builder.CreateCall(
			module.NamedFunction("getAddrPtr"),
			[]llvm.Value{ptag},
			"paddr",
		)

		indtag := Builder.CreateLoad(module.NamedGlobal("IND_TAG"), "indtag")
		Builder.CreateStore(indtag, ptag)
		Builder.CreateStore(f.Param(0), paddr)

		Builder.CreateRet(ptag)
	}

	return createFunction(
		module,
		"hAllocInd",
		llvm.PointerType(llvm.Int64Type(), 0),
		[]llvm.Type{
			llvm.PointerType(llvm.Int64Type(), 0),
		},
		[]string{"addr"},
		bodyBuilder,
	)

}

func createHeapOps(module llvm.Module) {
	createHAllocNum(module)
	createHAllocAp(module)
	createHAllocGlobal(module)
	createHAllocInd(module)
}
