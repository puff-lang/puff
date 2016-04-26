package main

import (
	"compile"
	"core"
	"fmt"
	"io/ioutil"
	"os"
	"repl"
)

var helpText = `
puff - The puff source code manager

Usage:

    puff command [arguments]

If no command is provided then the
default action is to start the REPL.

The commands are:

    run    Run the puff program
    build  Compile puff program and create executable
    help   Show this help text
`

func compileProgramFromFile(fileName string) core.GmState {
	b, err := ioutil.ReadFile(fileName)

	fmt.Println(string(b))
	if err != nil {
		panic(err)
	}
	// Generate the LLVM-IR code for Input file
	/*
	   contentProgram := core.Compile(core.Program{
	           core.ScDefn{"main", []core.Name{}, compile.Translate(string(b))},
	       })
	*/
	program := compile.Translate(string(b), fileName)

	mainFound := false
	for _, sc := range program {
		if string(sc.Name) == "main" {
			mainFound = true
			break
		}
	}

	if mainFound == false {
		panic("No main function found in program. Aborting.")
	}

	return core.Compile(program)
}

func main() {
	var action string = ""

	if len(os.Args) > 1 {
		action = os.Args[1]
	}

	switch action {
	case "help":
		fmt.Println(helpText)
	case "run":
		if len(os.Args) < 3 {
			fmt.Println("run command requires a filename as argument")
			os.Exit(1)
		}
		fileName := os.Args[2]
		core.ShowStates(compileProgramFromFile(fileName))
	case "build":
		if len(os.Args) < 3 {
			fmt.Println("build command requires a filename as argument")
			os.Exit(1)
		}
		fileName := os.Args[2]
		compile.SaveLLVMIR(compile.GenLLVMIR(compileProgramFromFile(fileName)))
	default:
		repl.Start()
	}
}
