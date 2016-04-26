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

func replLoop() {
    fmt.Println(`
                      __  __
         _ __  _   _ / _|/ _|
        | '_ \| | | | |_| |_
        | |_) | |_| |  _|  _|
        | .__/ \__,_|_| |_|
        |_|
    `)
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
            program := compile.Translate(globals + "\n" + "fn main() => " + text, "repl")
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
        fmt.Println("Starting puff REPL")
        replLoop()
        fmt.Println("\nGoodbye! Thanks for using puff.")
    }
}

// fn add(x, y) => x + y
// fn addOne(x) => add(x, 2)
// fn main() => let addOne = add(1) in addOne(10)