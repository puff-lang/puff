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
    /*
    contentProgram := core.Compile(core.Program{
            core.ScDefn{"main", []core.Name{}, compile.Translate(string(b))},
        })
    */
    program := compile.Translate(string(b))

    mainFound := false
    for _, sc := range program {
        if string(sc.Name) == "main" {
            mainFound = true
            break
        }
    }

    if mainFound == false {
        fmt.Println("No main function found in program. Aborting.")
        return
    }

    contentProgram := core.Compile(program)

    // compile.SaveLLVMIR(compile.GenLLVMIR(contentProgram))
    result := core.EvalState(contentProgram)
    fmt.Println(result)
}

// fn add(x, y) => x + y
// fn addOne(x) => add(x, 2)
// fn main() => let addOne = add(1) in addOne(10)