package repl

import (
	"bufio"
	"compile"
	"fmt"
	"io"
	"os"
	"core"
    "strings"
)

// func MainLoop(module llvm.Module, jit llvm.ExecutionEngine) {
func loop() {
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

func Start() {
    fmt.Println("Starting puff REPL")
    loop()
    fmt.Println("\nGoodbye! Thanks for using puff.")
}