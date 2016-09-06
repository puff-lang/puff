package llvmgen

import (
	 "llvm.org/llvm/bindings/go/llvm"
)

var SourceGlobals = make(map[string]llvm.Value)

var IRBuilder = llvm.NewBuilder()

func createDeclarations(module llvm.Module) {
	malloc := llvm.AddGlobal(module, llvm.FunctionType(
		llvm.PointerType(llvm.Int8Type(), 0),
		[]llvm.Type{llvm.Int64Type()},
		false,
	), "malloc")
	malloc.SetLinkage(llvm.ExternalLinkage)
}

func createInt64Const(module llvm.Module, name string, value uint64) {
	constant := llvm.AddGlobal(module, llvm.Int64Type(), name)
	constant.SetInitializer(llvm.ConstInt(llvm.Int64Type(), value, false))
}

func createConstants(module llvm.Module) {
	createInt64Const(module, "NUM_TAG", 1)
	createInt64Const(module, "AP_TAG", 2)
	createInt64Const(module, "GLOBAL_TAG", 3)
	createInt64Const(module, "IND_TAG", 4)
	createInt64Const(module, "CONSTR_TAG", 5)
}

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

type BodyBuilder func (llvm.Builder, llvm.Value)

func createFunction(
	module llvm.Module,
	name string,
	returnType llvm.Type,
	paramTypes []llvm.Type,
	paramNames []string,
	createBody BodyBuilder) llvm.Value {

	function := llvm.AddFunction(module, name, llvm.FunctionType(
		returnType,
		paramTypes,
		false,
	))

	for i, paramName := range paramNames {
		n := function.Param(i)
		n.SetName(paramName)
	}

	funcBody := llvm.AddBasicBlock(function, "")
	IRBuilder.SetInsertPointAtEnd(funcBody)
	createBody(IRBuilder, function)
	IRBuilder.ClearInsertionPoint()

	SourceGlobals[name] = function
	return function
}

func GenerateLLVMCode() llvm.Module {
	TheModule := llvm.NewModule("main")
	// TheModule.SetTarget("i686-unknown-linux")

	createDeclarations(TheModule)
	createConstants(TheModule)
	createGlobals(TheModule)
	createGenericStackOps(TheModule)
	createRuntimeStackOps(TheModule)
	createRuntimeVStackOps(TheModule)
	createUtilityFunctions(TheModule)
	createHeapOps(TheModule)

	return TheModule
}


