package compile

import (
	"core"
	"fmt"
	// "llvm.org/llvm/bindings/go/llvm"
	"parse"
)

var builtins = map[string]interface{}{
	"printf": fmt.Printf,
}

// func Compile(src string, TheModule llvm.Module) llvm.Value {
// 	// tree, err := parse.Parse("Program", src, "", "", make(map[string]*parse.Tree), builtins)
// 	tree, err := parse.New("Program").Parse(src, "", "", make(map[string]*parse.Tree), builtins)

// 	// for node := range tree.Root.Nodes {
// 	// 	compileNode(node, TheModule)
// 	// }

// 	// return tree.Root.String();
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// 	node := tree.Root.Nodes[0]
// 	return compileNode(node, TheModule)
// }

func Translate(src string, context string) core.Program {
	tree, err := parse.New(context).Parse(src, "", "", make(map[string]*parse.Tree), builtins)

	if err != nil {
		fmt.Println(err)
	}

	prg := core.Program{}
	for _, node := range tree.Root.Nodes {
		prg = append(prg, translateNode(node)...)
	}
	return prg
}

func run(str string) string { //Almost Done
	// contentProgram := core.Compile(core.Program{
	//            core.ScDefn{"main", []core.Name{}, Translate(string(str))},
	//    	})

	program := Translate(string(str), "Program")

	mainFound := false
	for _, sc := range program {
		if string(sc.Name) == "main" {
			mainFound = true
			break
		}
	}

	if mainFound == false {
		fmt.Println("No main function found in program. Aborting.")
		return ""
	}

	contentProgram := core.Compile(program)

	result := core.EvalState(contentProgram) // []GmState
	fmt.Println(result)
	return string("result")
}
