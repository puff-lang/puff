package llvmgen

import (
	 "llvm.org/llvm/bindings/go/llvm"
)

var SourceGlobals = make(map[string]llvm.Value)

var IRBuilder = llvm.NewBuilder()

func createGlobals(module llvm.Module) {
	vstackType := llvm.ArrayType(llvm.Int64Type(), 1000)
	vstack := llvm.AddGlobal(module, vstackType, "vstack")
	vstack.SetInitializer(llvm.Undef(vstackType))
	SourceGlobals["vstack"] = vstack

	vsp := llvm.AddGlobal(module, llvm.Int64Type(), "vsp")
	vsp.SetInitializer(llvm.Undef(llvm.Int64Type()))
	SourceGlobals["vsp"] = vsp

	stackType := llvm.ArrayType(llvm.PointerType(llvm.Int64Type(), 0), 1000)
	stack := llvm.AddGlobal(module, stackType, "stack")
	stack.SetInitializer(llvm.Undef(stackType))
	SourceGlobals["stack"] = stack

	sp := llvm.AddGlobal(module, llvm.Int64Type(), "sp")
	sp.SetInitializer(llvm.Undef(llvm.Int64Type()))
	SourceGlobals["sp"] = sp
}

func GenerateLLVMCode() llvm.Module {
	TheModule := llvm.NewModule("main")

	createGlobals(TheModule)
	createGenericStackOps(TheModule)
	createRuntimeStackOps(TheModule)
	createRuntimeVStackOps(TheModule)

	return TheModule
}


