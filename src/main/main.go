package main

import (
	"fmt"
    "os"
    "io"
	"io/ioutil"
    "bufio"
    "core"
    "compile"
    "strings"
)

func replLoop() {
    reader := bufio.NewReader(os.Stdin)
    // H := jit.addModule(module)
    var globals string
    for {
        fmt.Print("puff> ")
        text, err := reader.ReadString('\n')

        if err == io.EOF {
            break
        }

        if strings.HasPrefix(text, "fn ") {
            globals = globals + "\n" + text
        } else {
            program := compile.Translate(globals + "\n" + "fn main() => " + text)
            core.ShowStates(core.Compile(program))
        }
    }
}

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
    program := compile.Translate(string(b))

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
    action := os.Args[1]

    switch action {
    case "run":
        fileName := os.Args[2]
        core.ShowStates(compileProgramFromFile(fileName))
    case "repl":
        fmt.Println("Starting puff REPL")
        replLoop()
        fmt.Println("\nGoodbye! Thanks for using puff.")
    default:
        fileName := os.Args[2]
        compile.SaveLLVMIR(compile.GenLLVMIR(compileProgramFromFile(fileName)))
    }
}

// fn add(x, y) => x + y
// fn addOne(x) => add(x, 2)
// fn main() => let addOne = add(1) in addOne(10)