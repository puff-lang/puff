package main

import (
	"fmt"
    "os"
	"io/ioutil"
    "core"
    "compile"
)

func main() {
    action := os.Args[1]
    fileName := os.Args[2]

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

    switch action {
    case "run":
        core.ShowStates(contentProgram)
    default:
        compile.SaveLLVMIR(compile.GenLLVMIR(contentProgram))
    }
}

// fn add(x, y) => x + y
// fn addOne(x) => add(x, 2)
// fn main() => let addOne = add(1) in addOne(10)