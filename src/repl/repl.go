package main

import (
	"bufio"
	"compile"
	"fmt"
	"io"
	// "llvm.org/llvm/bindings/go/llvm"
	"os"
	"core"
)

// func MainLoop(module llvm.Module, jit llvm.ExecutionEngine) {
func MainLoop() {

	reader := bufio.NewReader(os.Stdin)
	// H := jit.addModule(module)
	for {
		fmt.Print("puff> ")
		text, err := reader.ReadString('\n')

		if err == io.EOF {
			break
		}

		if text == "4" {
			fmt.Print("4")
			continue
		}

		// llvm.compile.Compile(text, module).Dump()
		// compiledSc := core.CompileSc(core.ScDefn{"main", []core.Name{}, compile.Translate(text)})
		// fmt.Println(compiledSc)
		// core.PrintBody(compiledSc.Body())
		fmt.Println(core.Compile(core.Program{
			core.ScDefn{"main", []core.Name{}, compile.Translate(text)},
		}))
	}
}

func main() {

	// TheModule := llvm.NewModule("Awesome JIT")
	// TheJIT, _ := llvm.NewExecutionEngine(TheModule)

	// MainLoop(TheModule, TheJIT)
	MainLoop()

	fmt.Println("\nGoodbye! Thanks for using puff.")
}
