package main

import (
	"fmt"
	"io/ioutil"
    "core"
    "compile"
)

func main() {
     b, err := ioutil.ReadFile("demo.puff")

    fmt.Println(string(b))
    if err != nil {
        panic(err)
    }
    // Generate the LLVM-IR code for Input file
    contentProgram := core.Compile(core.Program{
            core.ScDefn{"main", []core.Name{}, compile.Translate(string(b))},
        })
    // compile.SaveLLVMIR(compile.GenLLVMIR(contentProgram))
    result := core.EvalState(contentProgram)
    fmt.Println(result)
}

