package main

import (
	"fmt"
	"bufio"
	"io"
	"os"
	"compile"
	"llvm.org/llvm/bindings/go/llvm"
)

func MainLoop(module llvm.Module, jit llmv.ExecutionEngine) {
	reader := bufio.NewReader(os.Stdin)
	H := jit.addModule(module)
	for {
		fmt.Print("puff> ")
		text, err := reader.ReadString('\n')

		if err == io.EOF {
			break
		}

		if text == "" {
			continue
		}

		llvm.compile.Compile(text, module).Dump()
	}
}

func main() {

	TheModule := llvm.NewModule("Awesome JIT")
	TheJIT, _ := llvm.NewExecutionEngine(TheModule)

	MainLoop(TheModule, TheJIT)

	fmt.Println("\nGoodbye! Thanks for using puff.")
}